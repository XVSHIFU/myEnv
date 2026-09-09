package backend

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

type failingTransport struct{ cause error }

func (f failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, f.cause
}

func TestRequestFailureDoesNotExposeCredentials(t *testing.T) {
	cause := errors.New("proxy https://proxy-user:proxy-secret@example.test/?token=proxy-token failed")
	n := &Node{Client: &http.Client{Transport: failingTransport{cause}}, BaseURL: "https://source-user:source-secret@example.test/?token=source-token"}
	for _, metadata := range []bool{false, true} {
		var err error
		if metadata {
			_, err = n.metadata(context.Background(), "/index.json", 1024)
		} else {
			_, err = n.Download(context.Background(), NodeArtifact{URL: n.BaseURL, SHA256: strings.Repeat("0", 64)}, t.TempDir())
		}
		if err == nil || err.Error() != "DOWNLOAD_FAILED: request failed; check network, proxy and TLS settings" {
			t.Fatalf("unexpected diagnostic: %v", err)
		}
		if !errors.Is(err, cause) {
			t.Fatal("lost error classification chain")
		}
	}
	for _, cause := range []error{context.Canceled, context.DeadlineExceeded} {
		n.Client.Transport = failingTransport{cause}
		_, err := n.Download(context.Background(), NodeArtifact{URL: n.BaseURL, SHA256: strings.Repeat("0", 64)}, t.TempDir())
		if !errors.Is(err, cause) || strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "token") {
			t.Fatalf("context failure: %v", err)
		}
	}
	n.BaseURL = "https://example.test/%invalid?secret=private-token"
	_, err := n.metadata(context.Background(), "/index.json", 1024)
	if err == nil || err.Error() != "DOWNLOAD_FAILED: invalid download request; check the configured source URL" {
		t.Fatalf("invalid URL diagnostic: %v", err)
	}
}

type failingBody struct {
	cause  error
	closed bool
}

func (b *failingBody) Read([]byte) (int, error) { return 0, b.cause }
func (b *failingBody) Close() error             { b.closed = true; return nil }

type bodyFailureTransport struct{ body *failingBody }

func (f bodyFailureTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: 200, Body: f.body, ContentLength: -1, Header: make(http.Header)}, nil
}

func TestResponseBodyFailureDoesNotExposeCredentials(t *testing.T) {
	for _, metadata := range []bool{true, false} {
		cause := errors.New("read https://user:private-password@example.test/?token=private-token failed")
		body := &failingBody{cause: cause}
		n := &Node{Client: &http.Client{Transport: bodyFailureTransport{body}}, BaseURL: "https://example.test"}
		directory := t.TempDir()
		var err error
		if metadata {
			_, err = n.metadata(context.Background(), "/index.json", 1024)
		} else {
			_, err = n.Download(context.Background(), NodeArtifact{URL: n.BaseURL, SHA256: strings.Repeat("0", 64)}, directory)
		}
		if !errors.Is(err, cause) || strings.Contains(err.Error(), "private-") || !strings.HasPrefix(err.Error(), "DOWNLOAD_FAILED:") {
			t.Fatalf("body diagnostic: %v", err)
		}
		if !body.closed {
			t.Fatal("response body not closed")
		}
		entries, readErr := os.ReadDir(directory)
		if readErr != nil || len(entries) != 0 {
			t.Fatalf("failed transfer retained staging: %v %v", entries, readErr)
		}
	}
}

func TestTruncatedHTTPResponseCleanup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("partial"))
	}))
	defer server.Close()
	node := &Node{Client: server.Client(), BaseURL: server.URL}
	for _, metadata := range []bool{true, false} {
		directory := t.TempDir()
		var err error
		if metadata {
			_, err = node.metadata(context.Background(), "/index.json", 1024)
		} else {
			_, err = node.Download(context.Background(), NodeArtifact{URL: server.URL + "/archive", SHA256: strings.Repeat("0", 64)}, directory)
		}
		if !errors.Is(err, io.ErrUnexpectedEOF) || !strings.HasPrefix(err.Error(), "DOWNLOAD_FAILED:") {
			t.Fatalf("truncated response: %v", err)
		}
		entries, readErr := os.ReadDir(directory)
		if readErr != nil || len(entries) != 0 {
			t.Fatalf("partial archive retained: %v %v", entries, readErr)
		}
	}
}

func TestHTTPBodyCancellationCleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("partial"))
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer server.Close()
	directory := t.TempDir()
	client := server.Client()
	client.Transport = cancelAfterBodyReadTransport{base: client.Transport, cancel: cancel}
	_, err := downloadArtifact(ctx, client, server.URL, strings.Repeat("0", 64), directory)
	if !errors.Is(err, context.Canceled) || !strings.Contains(err.Error(), "request canceled") {
		t.Fatalf("cancel classification: %v", err)
	}
	entries, readErr := os.ReadDir(directory)
	if readErr != nil || len(entries) != 0 {
		t.Fatalf("canceled download retained staging: %v %v", entries, readErr)
	}
}

type cancelAfterBodyReadTransport struct {
	base   http.RoundTripper
	cancel context.CancelFunc
}

func (t cancelAfterBodyReadTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := t.base.RoundTrip(r)
	if err == nil {
		response.Body = &cancelAfterBodyRead{ReadCloser: response.Body, cancel: t.cancel}
	}
	return response, err
}

type cancelAfterBodyRead struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (r *cancelAfterBodyRead) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	if n > 0 {
		r.cancel()
	}
	return n, err
}
