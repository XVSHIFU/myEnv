package backend

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// Download writes a unique staging file. The caller owns publication and cleanup.
// The returned file has been flushed and its bytes match the pre-resolved digest.
func (n *Node) Download(ctx context.Context, artifact NodeArtifact, directory string) (path string, err error) {
	if n.SegmentedSDK {
		return downloadSDK(ctx, n.Client, artifact.URL, artifact.SHA256, directory)
	}
	return downloadArtifact(ctx, n.Client, artifact.URL, artifact.SHA256, directory)
}

func downloadArtifact(ctx context.Context, client *http.Client, url, expectedSHA, directory string) (path string, err error) {
	digest, e := hex.DecodeString(expectedSHA)
	if e != nil || len(digest) != sha256.Size {
		return "", fmt.Errorf("invalid locked SHA256")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", &requestFailure{cause: err, invalidURL: true}
	}
	response, err := client.Do(req)
	if err != nil {
		return "", &requestFailure{cause: err}
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("DOWNLOAD_FAILED: HTTP %d", response.StatusCode)
	}
	const maxArchive = 512 << 20
	if response.ContentLength > maxArchive {
		return "", fmt.Errorf("DOWNLOAD_FAILED: archive exceeds size limit")
	}
	file, err := os.CreateTemp(directory, "artifact-download-*")
	if err != nil {
		return "", err
	}
	filename := file.Name()
	defer func() {
		file.Close()
		if err != nil {
			os.Remove(filename)
		}
	}()
	hash := sha256.New()
	count, err := io.CopyBuffer(io.MultiWriter(file, hash), io.LimitReader(response.Body, maxArchive+1), make([]byte, 64<<10))
	if err != nil {
		return "", &requestFailure{cause: err}
	}
	if count > maxArchive {
		return "", fmt.Errorf("DOWNLOAD_FAILED: archive exceeds size limit")
	}
	if hex.EncodeToString(hash.Sum(nil)) != strings.ToLower(expectedSHA) {
		return "", fmt.Errorf("CHECKSUM_MISMATCH: downloaded archive differs from expected digest")
	}
	if err = file.Sync(); err != nil {
		return "", err
	}
	if err = file.Close(); err != nil {
		return "", err
	}
	return filename, nil
}
