package backend

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestSDKRangeDownload(t *testing.T) {
	payload := strings.Repeat("a", (4<<20)+17)
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte(payload)))
	for _, broken := range []string{"", "offset", "digest", "length"} {
		t.Run(broken, func(t *testing.T) {
			calls := 0
			client := &http.Client{Transport: catalogTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				startText, _, _ := strings.Cut(strings.TrimPrefix(req.Header.Get("Range"), "bytes="), "-")
				start, _ := strconv.Atoi(startText)
				end := start + (4 << 20)
				if end > len(payload) {
					end = len(payload)
				}
				body := payload[start:end]
				reported := start
				if broken == "offset" {
					reported++
				}
				if broken == "length" {
					body = body[:len(body)-1]
				}
				return &http.Response{StatusCode: 206, Header: http.Header{"Content-Range": {fmt.Sprintf("bytes %d-%d/%d", reported, end-1, len(payload))}}, Body: io.NopCloser(strings.NewReader(body))}, nil
			})}
			sum := digest
			if broken == "digest" {
				sum = strings.Repeat("0", 64)
			}
			directory := t.TempDir()
			path, err := downloadSDK(context.Background(), client, "https://go.dev/dl/example", sum, directory)
			if broken != "" {
				if err == nil {
					t.Fatal("accepted invalid response")
				}
				files, _ := os.ReadDir(directory)
				if len(files) != 0 {
					t.Fatal("failed staging retained")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if calls != 2 {
				t.Fatal("did not fetch complete ranges")
			}
			data, _ := os.ReadFile(path)
			if string(data) != payload {
				t.Fatal("wrong payload")
			}
		})
	}
}
