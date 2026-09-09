package backend

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestUVMirrorRetainsPinnedDigest(t *testing.T) {
	requests := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.URL.Path
		fmt.Fprint(w, "untrusted replacement bytes")
	}))
	defer server.Close()
	root := t.TempDir()
	release, archive, err := DownloadUVFromMirror(context.Background(), "windows-amd64", root, server.URL+"/uv/")
	fixed, fixedErr := FixedUVRelease("windows-amd64")
	if fixedErr != nil || err == nil || archive != "" || release.SHA256 != fixed.SHA256 || release.Version != fixed.Version {
		t.Fatalf("mirror weakened pin: %+v %q %v", release, archive, err)
	}
	select {
	case got := <-requests:
		if want := "/uv/" + UVVersion + "/uv-x86_64-pc-windows-msvc.zip"; got != want {
			t.Fatalf("request: %q want %q", got, want)
		}
	default:
		t.Fatal("mirror was not requested")
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("unverified archive retained: %v %v", entries, err)
	}
}
