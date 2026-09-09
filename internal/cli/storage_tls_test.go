package cli

import (
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRuntimeServiceClosesCustomTLSConnections(t *testing.T) {
	certificate, root := mirrorCertificate(t)
	closed := make(chan struct{}, 1)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "ok") }))
	server.TLS = &tls.Config{Certificates: []tls.Certificate{certificate}}
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateClosed {
			select {
			case closed <- struct{}{}:
			default:
			}
		}
	}
	server.StartTLS()
	defer server.Close()
	file := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(file, root, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SSL_CERT_FILE", file)
	service, err := runtimeService(false, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer closeRuntimeService(service)
	response, err := service.Node.Client.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.Copy(io.Discard, response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-closed:
		t.Fatal("connection closed before explicit cleanup")
	default:
	}
	closeRuntimeService(service)
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("custom TLS connection retained after cleanup")
	}
}
