package config

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// GenerationDigest measures generation-owned files, not external symlink
// targets. Completion/evidence markers and Python bytecode caches are excluded.
// It is for installation/deep checks, never the ordinary status/run path.
func GenerationDigest(ctx context.Context, directory string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return "", err
	}
	defer root.Close()
	hash := sha256.New()
	encoder := json.NewEncoder(hash)
	buffer := make([]byte, 64<<10)
	var total int64
	entries := 0
	type item struct {
		Path, Kind, Value string
		Mode              uint32
	}
	var visit func(string, int) error
	visit = func(path string, depth int) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		entries++
		if entries > 1000000 || depth > 128 {
			return fmt.Errorf("generation evidence traversal limit exceeded")
		}
		info, err := root.Lstat(path)
		if err != nil {
			return err
		}
		record := item{Path: filepath.ToSlash(path), Mode: uint32(info.Mode().Perm())}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			record.Kind = "link"
			record.Value, err = root.Readlink(path)
			if err != nil {
				return err
			}
		case info.Mode().IsRegular():
			record.Kind = "file"
			file, err := root.Open(path)
			if err != nil {
				return err
			}
			content := sha256.New()
			for {
				if err = ctx.Err(); err != nil {
					file.Close()
					return err
				}
				n, e := file.Read(buffer)
				total += int64(n)
				if total > 8<<30 {
					file.Close()
					return fmt.Errorf("generation evidence byte limit exceeded")
				}
				if n > 0 {
					content.Write(buffer[:n])
				}
				if e == io.EOF {
					break
				}
				if e != nil {
					file.Close()
					return e
				}
			}
			if err = file.Close(); err != nil {
				return err
			}
			record.Value = hex.EncodeToString(content.Sum(nil))
		case info.IsDir():
			record.Kind = "directory"
		default:
			return fmt.Errorf("unsupported generation file type at %s", path)
		}
		if err = encoder.Encode(record); err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		file, err := root.Open(path)
		if err != nil {
			return err
		}
		children, err := file.ReadDir(4097)
		closeErr := file.Close()
		if err != nil && err != io.EOF {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		if len(children) > 4096 {
			return fmt.Errorf("generation directory evidence limit exceeded")
		}
		sort.Slice(children, func(i, j int) bool { return children[i].Name() < children[j].Name() })
		for _, child := range children {
			name := child.Name()
			if name == "__pycache__" && child.IsDir() {
				continue
			}
			if path == "." && (name == "complete" || name == "evidence.json") {
				continue
			}
			if err = visit(filepath.Join(path, name), depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	if err = visit(".", 0); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
