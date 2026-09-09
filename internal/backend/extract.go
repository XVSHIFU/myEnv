package backend

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// ExtractZIP requires a fresh destination and never follows archive links.
// Failed destinations remain owned by the caller's incomplete operation.
func ExtractZIP(archive, destination string) error {
	if err := os.Mkdir(destination, 0700); err != nil {
		return err
	}
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer reader.Close()
	if len(reader.File) > 100000 {
		return fmt.Errorf("archive has too many entries")
	}
	var total uint64
	for _, entry := range reader.File {
		relative, err := archivePath(entry.Name)
		if err != nil {
			return err
		}
		if entry.Mode()&os.ModeSymlink != 0 || (!entry.Mode().IsRegular() && !entry.FileInfo().IsDir()) {
			return fmt.Errorf("unsupported archive entry %q", entry.Name)
		}
		if entry.UncompressedSize64 > 2<<30 || total > 2<<30-entry.UncompressedSize64 {
			return fmt.Errorf("archive expanded size exceeds 2 GiB")
		}
		total += entry.UncompressedSize64
		target := filepath.Join(destination, filepath.FromSlash(relative))
		if entry.FileInfo().IsDir() {
			if err = os.MkdirAll(target, 0700); err != nil {
				return err
			}
			continue
		}
		if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		input, err := entry.Open()
		if err != nil {
			return err
		}
		output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			input.Close()
			return err
		}
		count, copyErr := io.CopyBuffer(output, io.LimitReader(input, int64(entry.UncompressedSize64)+1), make([]byte, 64<<10))
		inputErr := input.Close()
		syncErr := output.Sync()
		closeErr := output.Close()
		if copyErr != nil {
			return copyErr
		}
		if inputErr != nil {
			return inputErr
		}
		if syncErr != nil {
			return syncErr
		}
		if closeErr != nil {
			return closeErr
		}
		if uint64(count) != entry.UncompressedSize64 {
			return fmt.Errorf("archive size mismatch for %s", entry.Name)
		}
	}
	return nil
}

func archivePath(name string) (string, error) {
	if name == "" || strings.ContainsAny(name, "\\:\x00") || strings.HasPrefix(name, "/") {
		return "", fmt.Errorf("unsafe archive path %q", name)
	}
	clean := path.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("unsafe archive path %q", name)
	}
	for _, part := range strings.Split(name, "/") {
		if part == ".." || strings.HasSuffix(part, ".") || strings.HasSuffix(part, " ") {
			return "", fmt.Errorf("unsafe archive component %q", part)
		}
		base := strings.ToUpper(strings.SplitN(part, ".", 2)[0])
		if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9' {
			return "", fmt.Errorf("reserved archive component %q", part)
		}
	}
	return clean, nil
}
