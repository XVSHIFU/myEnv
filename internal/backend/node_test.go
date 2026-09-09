package backend

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestNodeResolve(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.json":
			fmt.Fprint(w, `[{"version":"v22.1.0","npm":"10.1.0","files":["win-x64-zip"]},{"version":"v24.0.0","files":["win-x64-zip"]},{"version":"v22.9.0","npm":"10.9.0","files":["win-x64-zip"]}]`)
		case "/v22.9.0/SHASUMS256.txt":
			fmt.Fprintf(w, "%s  node-v22.9.0-win-x64.zip\n", strings.Repeat("a", 64))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	n := &Node{Client: server.Client(), BaseURL: server.URL}
	artifact, err := n.Resolve(context.Background(), "22", "windows-amd64")
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Version != "22.9.0" || artifact.NPM != "10.9.0" || artifact.SHA256 != strings.Repeat("a", 64) {
		t.Fatalf("wrong resolution: %+v", artifact)
	}
	if _, err = n.Resolve(context.Background(), "22", "musl"); err == nil {
		t.Fatal("accepted unsupported platform")
	}
}

func TestNodeOfficialMetadata(t *testing.T) {
	if os.Getenv("MYENV_TEST_NETWORK") != "1" {
		t.Skip("explicit network integration gate")
	}
	artifact, err := NewNode().Resolve(context.Background(), "22", "windows-amd64")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("resolved %+v", artifact)
}
