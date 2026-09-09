package backend

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// PrepareUV rechecks the pinned release bytes before extracting and executing.
// Release provenance is verified as part of supplying the fixed distribution.
func PrepareUV(ctx context.Context, archive, directory, platform string) (UV, error) {
	var result UV
	release, err := FixedUVRelease(platform)
	if err != nil {
		return result, err
	}
	f, err := os.Open(archive)
	if err != nil {
		return result, err
	}
	hash := sha256.New()
	n, err := io.CopyBuffer(hash, io.LimitReader(f, (512<<20)+1), make([]byte, 64<<10))
	f.Close()
	if err != nil {
		return result, err
	}
	if n > 512<<20 || hex.EncodeToString(hash.Sum(nil)) != release.SHA256 {
		return result, fmt.Errorf("CHECKSUM_MISMATCH: uv archive differs from pinned release")
	}
	if platform == "windows-amd64" {
		err = ExtractZIP(archive, directory)
		result.Executable = filepath.Join(directory, "uv.exe")
	} else {
		err = ExtractTarGZ(archive, directory)
		name := strings.TrimSuffix(filepath.Base(release.URL), ".tar.gz")
		result.Executable = filepath.Join(directory, name, "uv")
	}
	if err != nil {
		return UV{}, err
	}
	if err = result.VerifyVersion(ctx); err != nil {
		return UV{}, err
	}
	return result, nil
}
