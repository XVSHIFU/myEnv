package state

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestHasLeasesWithoutOptionalMetadata(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "state.db")
	s, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if present, err := s.HasLeases(ctx); err != nil || present {
		t.Fatalf("empty: %v %v", present, err)
	}
	if err := s.Publish(ctx, Generation{ID: "first", Directory: filepath.Dir(path), InputDigest: "d", NodeExecutable: filepath.Join(filepath.Dir(path), "node")}, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AcquireActive(ctx); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{"DROP TABLE lease_completion", "DROP TABLE lease_identity"} {
		if _, err := s.db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := OpenReadOnly(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	present, queryErr := reader.HasLeases(ctx)
	closeErr := reader.Close()
	if queryErr != nil || closeErr != nil || !present {
		t.Fatalf("legacy protection: %v %v %v", present, queryErr, closeErr)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("read mutated database: %v", err)
	}
}

func TestLeaseRecoveryObservationAndPaging(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	s, err := Open(ctx, filepath.Join(root, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	g := Generation{ID: "first", Directory: filepath.Join(root, "first"), InputDigest: "d", PythonExecutable: filepath.Join(root, "first", "python")}
	if err = s.Publish(ctx, g, ""); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if _, err = s.AcquireActive(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = s.db.Exec(`INSERT INTO leases(id,generation_id,supervisor_pid) VALUES('legacy','first',1)`); err != nil {
		t.Fatal(err)
	}
	after := ""
	seen := map[string]bool{}
	var observed LeaseRecord
	for {
		rows, err := s.GenerationLeaseRecords(ctx, "first", after, 1)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) == 0 {
			break
		}
		if len(rows) != 1 || seen[rows[0].ID] {
			t.Fatal("pagination duplicate")
		}
		r := rows[0]
		seen[r.ID] = true
		after = r.ID
		if r.Identity == "" {
			if removed, err := s.DeleteObservedLease(ctx, r); err != nil || removed {
				t.Fatal("removed unidentified lease", err)
			}
		} else {
			observed = r
		}
	}
	if len(seen) != 4 {
		t.Fatalf("lost leases: %v", seen)
	}
	for _, mutate := range []func(*LeaseRecord){func(r *LeaseRecord) { r.PID++ }, func(r *LeaseRecord) { r.Identity += "changed" }, func(r *LeaseRecord) { r.GenerationID = "other" }} {
		stale := observed
		mutate(&stale)
		if removed, err := s.DeleteObservedLease(ctx, stale); err != nil || removed {
			t.Fatal("deleted mismatched observation", err)
		}
	}
	// State only verifies the observed row; process/tree authorization belongs
	// to the core caller. This fixture exercises that conditional write boundary.
	if removed, err := s.DeleteObservedLease(ctx, observed); err != nil || !removed {
		t.Fatal("matching observation not removed", err)
	}
	rows, err := s.LeaseRecords(ctx, "", 128)
	if err != nil || len(rows) != 3 {
		t.Fatalf("remaining %+v %v", rows, err)
	}
	rows, err = s.GenerationLeaseRecords(ctx, "other", "", 1)
	if err != nil || len(rows) != 0 {
		t.Fatal("cross-generation lease page", err)
	}
}
