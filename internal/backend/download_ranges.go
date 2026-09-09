package backend

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// SDK downloads use bounded ranges because large full-body responses can stall
// on intermediary caches. Every response range and the final digest are checked.
func downloadSDK(ctx context.Context, client *http.Client, address, digest, directory string) (result string, err error) {
	expected, e := hex.DecodeString(digest)
	if e != nil || len(expected) != 32 {
		return "", fmt.Errorf("invalid SDK digest")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	file, err := os.CreateTemp(directory, "sdk-download-*")
	if err != nil {
		return "", err
	}
	defer func() {
		file.Close()
		if err != nil {
			os.Remove(file.Name())
		}
	}()
	const chunk int64 = 4 << 20
	const maximum int64 = 512 << 20
	hash := sha256.New()
	writer := io.MultiWriter(file, hash)
	var offset, total int64
	for {
		req, e := http.NewRequestWithContext(ctx, "GET", address, nil)
		if e != nil {
			return "", e
		}
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, offset+chunk-1))
		req.Header.Set("Accept-Encoding", "identity")
		response, e := client.Do(req)
		if e != nil {
			return "", &requestFailure{cause: e}
		}
		if response.StatusCode == http.StatusOK && offset == 0 {
			n, e := io.CopyBuffer(writer, io.LimitReader(response.Body, maximum+1), make([]byte, 64<<10))
			response.Body.Close()
			if e != nil {
				return "", e
			}
			if n > maximum {
				return "", fmt.Errorf("SDK archive exceeds size limit")
			}
			break
		}
		if response.StatusCode != http.StatusPartialContent {
			response.Body.Close()
			return "", fmt.Errorf("SDK range download: HTTP %d", response.StatusCode)
		}
		start, end, size, e := parseContentRange(response.Header.Get("Content-Range"))
		if e != nil || start != offset || end < start || end >= size || end >= offset+chunk || size > maximum || (total != 0 && total != size) {
			response.Body.Close()
			return "", fmt.Errorf("invalid SDK Content-Range")
		}
		total = size
		n, e := io.CopyBuffer(writer, io.LimitReader(response.Body, end-start+2), make([]byte, 64<<10))
		response.Body.Close()
		if e != nil {
			return "", e
		}
		if n != end-start+1 {
			return "", fmt.Errorf("SDK response length differs from Content-Range")
		}
		offset += n
		if offset == total {
			break
		}
	}
	if hex.EncodeToString(hash.Sum(nil)) != strings.ToLower(digest) {
		return "", fmt.Errorf("CHECKSUM_MISMATCH: SDK archive differs from official digest")
	}
	if err = file.Sync(); err != nil {
		return "", err
	}
	if err = file.Close(); err != nil {
		return "", err
	}
	return file.Name(), nil
}

func parseContentRange(s string) (start, end, total int64, err error) {
	fail := func() (int64, int64, int64, error) { return 0, 0, 0, fmt.Errorf("invalid content range") }
	if !strings.HasPrefix(s, "bytes ") {
		return fail()
	}
	bounds, size, ok := strings.Cut(strings.TrimPrefix(s, "bytes "), "/")
	if !ok {
		return fail()
	}
	a, b, ok := strings.Cut(bounds, "-")
	if !ok {
		return fail()
	}
	start, err = strconv.ParseInt(a, 10, 64)
	if err != nil || start < 0 {
		return fail()
	}
	end, err = strconv.ParseInt(b, 10, 64)
	if err != nil {
		return fail()
	}
	total, err = strconv.ParseInt(size, 10, 64)
	if err != nil || total <= 0 {
		return fail()
	}
	return
}
