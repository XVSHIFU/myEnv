package backend

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// DownloadClient uses an explicit PEM bundle as the sole trust source, matching
// uv's SSL_CERT_FILE override. It never changes the process-wide transport.
func DownloadClient(certificateFile string) (*http.Client, error) {
	client := &http.Client{Timeout: 5 * time.Minute}
	if certificateFile == "" {
		return client, nil
	}
	if !filepath.IsAbs(certificateFile) {
		return nil, fmt.Errorf("SSL_CERT_FILE must be an absolute PEM file path")
	}
	info, err := os.Stat(certificateFile)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > 4<<20 {
		return nil, fmt.Errorf("SSL_CERT_FILE must be a regular PEM file no larger than 4 MiB")
	}
	file, err := os.Open(certificateFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, (4<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > 4<<20 {
		return nil, fmt.Errorf("SSL_CERT_FILE exceeds 4 MiB")
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(data) {
		return nil, fmt.Errorf("SSL_CERT_FILE contains no valid PEM certificates")
	}
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok || base == nil {
		return nil, fmt.Errorf("custom CA requires a configurable HTTP transport")
	}
	transport := base.Clone()
	transport.TLSClientConfig = &tls.Config{RootCAs: roots}
	client.Transport = transport
	return client, nil
}
