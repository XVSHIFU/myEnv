package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"myenv/internal/backend"
	"myenv/internal/state"
	"os"
	"path/filepath"
	"strings"
)

// nodeArchive copies a verified shared object into operation-owned staging.
// The shared object is never hard-linked to writable project content.
func (s *Service) nodeArchive(ctx context.Context, node *backend.Node, artifact backend.NodeArtifact, operation string) (string, error) {
	if s.Storage == nil {
		return node.Download(ctx, artifact, operation)
	}
	digest, err := hex.DecodeString(artifact.SHA256)
	if err != nil || len(digest) != sha256.Size {
		return "", fmt.Errorf("invalid locked SHA256")
	}
	storage, err := s.storagePaths("")
	if err != nil {
		return "", err
	}
	directory := filepath.Join(storage.Cache, "node-archives")
	if err = os.MkdirAll(directory, 0700); err != nil {
		return "", err
	}
	key := strings.ToLower(artifact.SHA256)
	unlock, err := state.LockWorkspace(ctx, filepath.Join(directory, key+".lock"))
	if err != nil {
		return "", err
	}
	defer unlock()
	path := filepath.Join(directory, key)
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		download, err := node.Download(ctx, artifact, directory)
		if err != nil {
			return "", err
		}
		defer os.Remove(download)
		if err = os.Rename(download, path); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	} else if !info.Mode().IsRegular() {
		return "", fmt.Errorf("invalid shared Node archive type")
	}
	input, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer input.Close()
	output, err := os.CreateTemp(operation, "node-cache-*")
	if err != nil {
		return "", err
	}
	success := false
	defer func() {
		output.Close()
		if !success {
			os.Remove(output.Name())
		}
	}()
	hash := sha256.New()
	buffer := make([]byte, 64<<10)
	var size int64
	for {
		if err = ctx.Err(); err != nil {
			return "", err
		}
		n, readErr := input.Read(buffer)
		if n > 0 {
			size += int64(n)
			if size > 512<<20 {
				return "", fmt.Errorf("shared Node archive exceeds size limit")
			}
			if _, err = output.Write(buffer[:n]); err != nil {
				return "", err
			}
			hash.Write(buffer[:n])
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", readErr
		}
	}
	if hex.EncodeToString(hash.Sum(nil)) != key {
		return "", fmt.Errorf("CHECKSUM_MISMATCH: shared Node archive differs from expected digest")
	}
	if err = output.Sync(); err != nil {
		return "", err
	}
	if err = output.Close(); err != nil {
		return "", err
	}
	success = true
	return output.Name(), nil
}
