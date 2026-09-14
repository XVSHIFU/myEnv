package backend

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestDownloadProgressActualBytes(t *testing.T) {
	payload := strings.Repeat("network bytes", 11000)
	for _, known := range []bool{true, false} {
		t.Run(fmt.Sprint("known_total=", known), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if known {
					w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
				}
				w.WriteHeader(http.StatusOK)
				w.(http.Flusher).Flush()
				_, _ = fmt.Fprint(w, payload)
			}))
			defer server.Close()
			var events []DownloadProgress
			ctx := WithDownloadProgress(context.Background(), func(p DownloadProgress) { events = append(events, p) })
			artifact := NodeArtifact{URL: server.URL + "/node.zip?secret=must-not-be-a-label", SHA256: fmt.Sprintf("%x", sha256.Sum256([]byte(payload)))}
			filename, err := (&Node{Client: server.Client()}).Download(ctx, artifact, t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(filename)
			if err != nil || string(data) != payload {
				t.Fatalf("artifact changed: %v", err)
			}
			assertDownloadProgress(t, events, int64(len(payload)), known, "node.zip")
		})
	}
}

func assertDownloadProgress(t *testing.T, events []DownloadProgress, size int64, known bool, label string) {
	t.Helper()
	if len(events) < 3 || events[0].Completed != 0 || events[0].Done {
		t.Fatalf("missing real start/bytes/clear events: %+v", events)
	}
	var previous int64
	for i, event := range events {
		if event.ID == 0 || event.ID != events[0].ID || event.Label != label || event.Completed < previous || event.Completed > size {
			t.Fatalf("invalid event %d: %+v", i, event)
		}
		if event.Total != 0 && (!known || event.Total != size) {
			t.Fatalf("unreliable total: %+v", event)
		}
		if known && event.Total != size {
			t.Fatalf("missing reliable total: %+v", event)
		}
		if event.Done != (i == len(events)-1) {
			t.Fatalf("transfer cleared before its final event: %+v", event)
		}
		previous = event.Completed
	}
	if previous != size {
		t.Fatalf("reported %d, actually read %d", previous, size)
	}
}

func TestDownloadProgressCancelAndErrorClear(t *testing.T) {
	for _, cancelDownload := range []bool{true, false} {
		t.Run(fmt.Sprint("cancel=", cancelDownload), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Length", "100")
				_, _ = fmt.Fprint(w, "partial")
				w.(http.Flusher).Flush()
				if cancelDownload {
					<-r.Context().Done()
				}
			}))
			defer server.Close()
			var events []DownloadProgress
			ctx = WithDownloadProgress(ctx, func(p DownloadProgress) {
				events = append(events, p)
				if cancelDownload && p.Completed > 0 {
					cancel()
				}
			})
			directory := t.TempDir()
			_, err := downloadArtifact(ctx, server.Client(), server.URL+"/uv.zip", strings.Repeat("0", 64), directory)
			if err == nil || (cancelDownload && !errors.Is(err, context.Canceled)) {
				t.Fatalf("download should fail: %v", err)
			}
			if len(events) < 3 || !events[len(events)-1].Done || events[len(events)-1].Completed != int64(len("partial")) {
				t.Fatalf("failed transfer not cleared with actual bytes: %+v", events)
			}
			entries, readErr := os.ReadDir(directory)
			if readErr != nil || len(entries) != 0 {
				t.Fatalf("partial retained: %v %v", entries, readErr)
			}
		})
	}
}

func TestSDKDownloadProgressRangesAndFallback(t *testing.T) {
	payload := strings.Repeat("s", (4<<20)+17)
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte(payload)))
	for _, ranges := range []bool{true, false} {
		for _, known := range []bool{true, false} {
			t.Run(fmt.Sprintf("ranges=%t/known=%t", ranges, known), func(t *testing.T) {
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					start, end, status := 0, len(payload), http.StatusOK
					if ranges {
						var requestedEnd int
						if _, err := fmt.Sscanf(r.Header.Get("Range"), "bytes=%d-%d", &start, &requestedEnd); err != nil {
							t.Error(err)
							w.WriteHeader(http.StatusBadRequest)
							return
						}
						end = min(end, requestedEnd+1)
						status = http.StatusPartialContent
						w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end-1, len(payload)))
					}
					if known {
						w.Header().Set("Content-Length", strconv.Itoa(end-start))
					}
					w.WriteHeader(status)
					w.(http.Flusher).Flush()
					_, _ = fmt.Fprint(w, payload[start:end])
				}))
				defer server.Close()
				var events []DownloadProgress
				ctx := WithDownloadProgress(context.Background(), func(p DownloadProgress) { events = append(events, p) })
				if _, err := downloadSDK(ctx, server.Client(), server.URL+"/sdk.tar.gz", digest, t.TempDir()); err != nil {
					t.Fatal(err)
				}
				assertDownloadProgress(t, events, int64(len(payload)), known, "sdk.tar.gz")
				if (ranges && calls != 2) || (!ranges && calls != 1) {
					t.Fatalf("unexpected duplicate requests: %d", calls)
				}
			})
		}
	}
}

func TestDownloadProgressUnreliableTotals(t *testing.T) {
	for _, response := range []*http.Response{
		{ContentLength: -1},
		{ContentLength: 0},
		{ContentLength: 20, Uncompressed: true},
		{ContentLength: 20, Header: http.Header{"Content-Encoding": {"gzip"}}},
	} {
		if total := responseDownloadTotal(response); total != 0 {
			t.Fatalf("reported unreliable content length: %d", total)
		}
	}
}
