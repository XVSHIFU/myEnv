package core

import (
	"context"
	"errors"
	"myenv/internal/config"
	"myenv/internal/state"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNodeCacheCleanLockProtection(t *testing.T) {
	root := t.TempDir()
	storage := config.UserStorage{Data: filepath.Join(root, "data"), Cache: filepath.Join(root, "cache")}
	directory := filepath.Join(storage.Cache, "node-archives")
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatal(err)
	}
	name := strings.Repeat("a", 64)
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte("in use"), 0600); err != nil {
		t.Fatal(err)
	}
	unlock, err := state.LockWorkspace(context.Background(), path+".lock")
	if err != nil {
		t.Fatal(err)
	}
	locked := true
	defer func() {
		if locked {
			unlock()
		}
	}()
	s := &Service{Storage: &storage}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	result, err := s.CleanNodeCache(ctx, false, nil)
	cancel()
	if !errors.Is(err, context.DeadlineExceeded) || result.Changed || result.Removed != 0 {
		t.Fatalf("locked cleanup %+v %v", result, err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "in use" {
		t.Fatal("deleted held cache", err)
	}
	// Dry-run does not require write ownership or remove the object while held.
	preview, err := s.CleanNodeCache(context.Background(), true, nil)
	if err != nil || preview.Candidates != 1 || preview.Changed {
		t.Fatalf("held preview %+v %v", preview, err)
	}
	if err = unlock(); err != nil {
		t.Fatal(err)
	}
	locked = false
	result, err = s.CleanNodeCache(context.Background(), false, nil)
	if err != nil || result.Removed != 1 || !result.Changed {
		t.Fatalf("released cleanup %+v %v", result, err)
	}
	if _, err = os.Stat(path + ".lock"); err != nil {
		t.Fatal("removed shared lock identity", err)
	}
}

func TestNodeCacheCleanPreservesColocatedRuntimeStorage(t *testing.T) {
	root := t.TempDir()
	// Windows data and cache roots may resolve to the same application path.
	storage := config.UserStorage{Data: root, Cache: root}
	name := strings.Repeat("a", 64)
	var protected []string
	for _, directory := range []string{"runtimes", "backends", "locks", "uv"} {
		path := filepath.Join(root, directory, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("retained"), 0600); err != nil {
			t.Fatal(err)
		}
		protected = append(protected, path)
	}
	archive := filepath.Join(root, "node-archives", name)
	if err := os.MkdirAll(filepath.Dir(archive), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archive, []byte("transport"), 0600); err != nil {
		t.Fatal(err)
	}
	s := &Service{Storage: &storage}
	for _, dry := range []bool{true, false} {
		result, err := s.CleanNodeCache(context.Background(), dry, nil)
		if err != nil || result.Candidates != 1 || result.Bytes != 9 || result.Changed == dry {
			t.Fatalf("dry=%v result=%+v err=%v", dry, result, err)
		}
		for _, path := range protected {
			data, err := os.ReadFile(path)
			if err != nil || string(data) != "retained" {
				t.Fatalf("changed shared resource %s: %q %v", path, data, err)
			}
		}
	}
}

func TestNodeCacheCleanupCancelBetweenItems(t *testing.T) {
	root := t.TempDir()
	storage := config.UserStorage{Data: filepath.Join(root, "data"), Cache: filepath.Join(root, "cache")}
	directory := filepath.Join(storage.Cache, "node-archives")
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatal(err)
	}
	for _, digit := range []string{"a", "b"} {
		if err := os.WriteFile(filepath.Join(directory, strings.Repeat(digit, 64)), []byte("cache"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := &Service{Storage: &storage}
	reported := ""
	result, err := s.CleanNodeCache(ctx, false, func(item CleanItem) error { reported = item.ID; cancel(); return nil })
	if !errors.Is(err, context.Canceled) || result.Removed != 1 || !result.Changed {
		t.Fatalf("partial cancel %+v %v", result, err)
	}
	for _, digit := range []string{"a", "b"} {
		name := strings.Repeat(digit, 64)
		_, err := os.Stat(filepath.Join(directory, name))
		if name == reported {
			if !os.IsNotExist(err) {
				t.Fatal("reported removal did not happen")
			}
		} else if err != nil {
			t.Fatal("cancellation removed next item", err)
		}
	}
	if result, err = s.CleanNodeCache(ctx, false, nil); !errors.Is(err, context.Canceled) || result.Changed {
		t.Fatal("canceled retry modified cache", err)
	}
}
