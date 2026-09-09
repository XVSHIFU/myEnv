package core

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"myenv/internal/state"
)

func TestCanceledDiagnosisBeforeDiscovery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	missing := filepath.Join(t.TempDir(), "missing")
	s := &Service{}
	if _, err := s.Status(ctx, missing); !errors.Is(err, context.Canceled) {
		t.Fatalf("status: %v", err)
	}
	if _, err := s.Doctor(ctx, missing, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("doctor: %v", err)
	}
	if _, err := s.DoctorDeep(ctx, missing, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("deep doctor: %v", err)
	}
}

func TestDoctorDeepMissingEvidence(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("schema: 1\ntools: {node: '22'}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	s := &Service{}
	r, err := s.DoctorDeep(ctx, root, nil)
	if err != nil || r.ContentEvidence != "missing" || r.RunProtection != "none" {
		t.Fatalf("uninitialized: %+v %v", r, err)
	}
	work := filepath.Join(root, ".myenv")
	if _, err := os.Stat(work); !os.IsNotExist(err) {
		t.Fatal("doctor initialized state", err)
	}
	if err := os.Mkdir(work, 0700); err != nil {
		t.Fatal(err)
	}
	db := filepath.Join(work, "state.db")
	store, err := state.Open(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	err = store.Publish(ctx, state.Generation{ID: "legacy", Directory: filepath.Join(work, "legacy"), InputDigest: "old", NodeExecutable: filepath.Join(work, "legacy", "node")}, "")
	if err == nil {
		_, err = store.AcquireActive(ctx)
	}
	store.Close()
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(db)
	if err != nil {
		t.Fatal(err)
	}
	r, err = s.DoctorDeep(ctx, root, nil)
	if err != nil || r.ContentEvidence != "missing" || r.EvidenceDetail == "" || r.RunProtection != "present" || r.ProtectionDetail == "" {
		t.Fatalf("legacy: %+v %v", r, err)
	}
	after, err := os.ReadFile(db)
	if err != nil || string(before) != string(after) {
		t.Fatal("doctor changed database", err)
	}
}
