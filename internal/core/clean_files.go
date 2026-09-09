package core

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"myenv/internal/state"
)

// generationName accepts only an immediate child of the managed generation
// directory. Database paths are untrusted input, never removal authority.
func generationName(parent string, g state.Generation) (string, error) {
	if !managedGenerationID(g.ID) || g.Directory != filepath.Join(parent, g.ID) {
		return "", fmt.Errorf("CLEAN_UNSAFE_PATH: invalid generation directory %q", g.Directory)
	}
	return g.ID, nil
}

// Sync creates exactly 16 random bytes encoded as lowercase hex. Requiring that
// canonical spelling excludes Windows case, trailing-dot, device and 8.3 aliases.
func managedGenerationID(id string) bool {
	if len(id) != 32 {
		return false
	}
	for _, c := range id {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

// generationBytes uses bounded directory batches and does not follow links.
// Sizes are logical file bytes, not an estimate of filesystem allocated blocks.
func generationBytes(ctx context.Context, root *os.Root, name string) (int64, error) {
	var total int64
	entries := 0
	var visit func(string, int) error
	visit = func(path string, depth int) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		entries++
		if entries > 1000000 || depth > 128 {
			return fmt.Errorf("clean directory traversal limit exceeded")
		}
		info, err := root.Lstat(path)
		if os.IsNotExist(err) && path == name {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		if info.Mode().IsRegular() {
			total += info.Size()
			return nil
		}
		if !info.IsDir() {
			return fmt.Errorf("unsupported managed file type at %s", path)
		}
		file, err := root.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		for {
			batch, err := file.ReadDir(128)
			if err != nil && err != io.EOF {
				return err
			}
			for _, entry := range batch {
				if e := visit(filepath.Join(path, entry.Name()), depth+1); e != nil {
					return e
				}
			}
			if err == io.EOF {
				return nil
			}
		}
	}
	err := visit(name, 0)
	return total, err
}

// removeGeneration is called with the modification lock and a durable deleting
// reservation held. Root keeps recursive removal inside the opened directory,
// including if links are replaced while the removal is in progress.
func removeGeneration(ctx context.Context, root *os.Root, name string) error {
	if !managedGenerationID(name) {
		return fmt.Errorf("CLEAN_UNSAFE_PATH: invalid generation name")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return root.RemoveAll(name)
}
