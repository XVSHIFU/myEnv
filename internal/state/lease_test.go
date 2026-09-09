package state

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestLeaseRetainsSelectedGeneration(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()
	s, err := Open(ctx, filepath.Join(root, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err = s.AcquireActive(ctx); !errors.Is(err, ErrNoAppliedGeneration) {
		t.Fatalf("missing environment error: %v", err)
	}
	first := Generation{ID: "first", Directory: filepath.Join(root, "first"), InputDigest: "a", NodeExecutable: filepath.Join(root, "first", "node.exe")}
	if err = s.Publish(ctx, first, ""); err != nil {
		t.Fatal(err)
	}
	lease, err := s.AcquireActive(ctx)
	if err != nil {
		t.Fatal(err)
	}
	next := Generation{ID: "next", Directory: filepath.Join(root, "next"), InputDigest: "b", NodeExecutable: filepath.Join(root, "next", "node.exe")}
	if err = s.Publish(ctx, next, "first"); err != nil {
		t.Fatal(err)
	}
	var birth string
	if err = s.db.QueryRow(`SELECT process_identity FROM lease_identity WHERE lease_id=?`, lease.ID).Scan(&birth); err != nil || birth == "" {
		t.Fatalf("missing process identity %q %v", birth, err)
	}
	wantBirth, err := supervisorIdentity()
	if err != nil || birth != wantBirth {
		t.Fatalf("unstable process identity %q %q %v", birth, wantBirth, err)
	}
	var protected string
	if err = s.db.QueryRow(`SELECT generation_id FROM leases WHERE id=?`, lease.ID).Scan(&protected); err != nil || protected != "first" {
		t.Fatalf("lease changed with active generation: %s %v", protected, err)
	}
	if _, err = s.db.Exec(`UPDATE lease_identity SET process_identity='different-start' WHERE lease_id=?`, lease.ID); err != nil {
		t.Fatal(err)
	}
	if err = s.ReleaseLease(ctx, lease.ID); !errors.Is(err, ErrLeaseNotOwned) {
		t.Fatalf("unowned release reported success: %v", err)
	}
	var retained int
	if err = s.db.QueryRow(`SELECT count(*) FROM leases WHERE id=?`, lease.ID).Scan(&retained); err != nil || retained != 1 {
		t.Fatal("released another process identity", err)
	}
	if _, err = s.db.Exec(`UPDATE lease_identity SET process_identity=? WHERE lease_id=?`, birth, lease.ID); err != nil {
		t.Fatal(err)
	}
	if err = s.ReleaseLease(ctx, lease.ID); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = s.db.QueryRow(`SELECT count(*) FROM leases`).Scan(&count); err != nil || count != 0 {
		t.Fatal("lease not released")
	}
	if err = s.db.QueryRow(`SELECT count(*) FROM lease_identity`).Scan(&count); err != nil || count != 0 {
		t.Fatal("lease identity not released", err)
	}
	if err = s.ReleaseLease(ctx, lease.ID); !errors.Is(err, ErrLeaseNotOwned) {
		t.Fatalf("duplicate release reported success: %v", err)
	}
}
