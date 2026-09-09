package state

import (
	"context"
	"path/filepath"
	"testing"
)

func TestRollbackReferences(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()
	s, err := Open(ctx, filepath.Join(root, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if g, err := s.Previous(ctx); err != nil || g != nil {
		t.Fatalf("unexpected previous %+v %v", g, err)
	}
	prior := ""
	for _, id := range []string{"one", "two"} {
		g := Generation{ID: id, Directory: filepath.Join(root, id), InputDigest: id, NodeExecutable: filepath.Join(root, id, "node")}
		if err = s.Publish(ctx, g, prior); err != nil {
			t.Fatal(err)
		}
		prior = id
	}
	lease, err := s.AcquireActive(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer s.ReleaseLease(ctx, lease.ID)
	if err = s.Rollback(ctx, "stale", "one"); err == nil {
		t.Fatal("accepted stale current")
	}
	if err = s.Rollback(ctx, "two", "one"); err != nil {
		t.Fatal(err)
	}
	active, err := s.Active(ctx)
	if err != nil || active.ID != "one" {
		t.Fatalf("active %+v %v", active, err)
	}
	previous, err := s.Previous(ctx)
	if err != nil || previous.ID != "two" {
		t.Fatalf("previous %+v %v", previous, err)
	}
	var leased string
	if err = s.db.QueryRowContext(ctx, `SELECT generation_id FROM leases WHERE id=?`, lease.ID).Scan(&leased); err != nil || leased != "two" {
		t.Fatalf("changed running lease %s %v", leased, err)
	}
}
