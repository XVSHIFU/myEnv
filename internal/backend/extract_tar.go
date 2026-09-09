package backend

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// ExtractTarGZ materializes files before links, so no archive write follows a
// link. Only relative links to extracted regular files are supported.
func ExtractTarGZ(archive, destination string) error {
	if err := os.Mkdir(destination, 0700); err != nil {
		return err
	}
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	reader := tar.NewReader(gz)
	type link struct{ name, target string }
	var links []link
	files := make(map[string]bool)
	seen := make(map[string]bool)
	var total int64
	buffer := make([]byte, 64<<10)
	for count := 0; ; count++ {
		h, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if count >= 100000 {
			return fmt.Errorf("archive has too many entries")
		}
		name, err := archivePath(h.Name)
		if err != nil {
			return err
		}
		if seen[name] {
			return fmt.Errorf("duplicate archive path %q", name)
		}
		seen[name] = true
		if h.Size < 0 || h.Size > 2<<30 || total > 2<<30-h.Size {
			return fmt.Errorf("archive expanded size exceeds 2 GiB")
		}
		total += h.Size
		target := filepath.Join(destination, filepath.FromSlash(name))
		switch h.Typeflag {
		case tar.TypeDir:
			if err = os.MkdirAll(target, 0700); err != nil {
				return err
			}
		case tar.TypeReg:
			if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
				return err
			}
			mode := os.FileMode(0600)
			if h.Mode&0111 != 0 {
				mode = 0700
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
			if err != nil {
				return err
			}
			n, copyErr := io.CopyBuffer(out, reader, buffer)
			syncErr := out.Sync()
			closeErr := out.Close()
			if copyErr != nil {
				return copyErr
			}
			if syncErr != nil {
				return syncErr
			}
			if closeErr != nil {
				return closeErr
			}
			if n != h.Size {
				return fmt.Errorf("archive size mismatch for %s", name)
			}
			files[name] = true
		case tar.TypeSymlink:
			if h.Linkname == "" || strings.ContainsAny(h.Linkname, "\\:\x00") || path.IsAbs(h.Linkname) {
				return fmt.Errorf("unsafe archive link %q", name)
			}
			resolved, err := archivePath(path.Join(path.Dir(name), h.Linkname))
			if err != nil {
				return err
			}
			links = append(links, link{name, resolved})
		default:
			return fmt.Errorf("unsupported archive entry %q", name)
		}
	}
	// Consume the gzip trailer to verify its CRC even after tar's end marker.
	if n, err := io.CopyBuffer(io.Discard, io.LimitReader(gz, (1<<20)+1), buffer); err != nil {
		return err
	} else if n > 1<<20 {
		return fmt.Errorf("excessive trailing archive data")
	}
	for _, l := range links {
		if !files[l.target] {
			return fmt.Errorf("archive link %q does not target an extracted regular file", l.name)
		}
		target := filepath.Join(destination, filepath.FromSlash(l.name))
		if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		relative, err := filepath.Rel(filepath.Dir(target), filepath.Join(destination, filepath.FromSlash(l.target)))
		if err != nil {
			return err
		}
		if err = os.Symlink(relative, target); err != nil {
			return err
		}
	}
	return nil
}
