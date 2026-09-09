package state

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestReadOnlyOpenFailures(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	contents := []byte("not a SQLite database")
	if err := os.WriteFile(path, contents, 0600); err != nil {
		t.Fatal(err)
	}
	if s, err := OpenReadOnly(context.Background(), path); err == nil {
		s.Close()
		t.Fatal("opened corrupt database")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if s, err := OpenReadOnly(ctx, path); !errors.Is(err, context.Canceled) {
		if s != nil {
			s.Close()
		}
		t.Fatalf("canceled open: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(contents) {
		t.Fatalf("failed open changed database: %v", err)
	}
}

func TestReadOnlyStore(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "state.db")
	if s, err := OpenReadOnly(ctx, path); err == nil {
		s.Close()
		t.Fatal("opened missing database")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("read-only open created database: %v", err)
	}
	s, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	s, err = OpenReadOnly(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if active, err := s.Active(ctx); err != nil || active != nil {
		t.Fatalf("read active: %+v %v", active, err)
	}
	if _, err := s.db.ExecContext(ctx, "DELETE FROM active"); err == nil {
		t.Fatal("read-only store accepted mutation")
	}
}
