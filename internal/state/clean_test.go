package state

import (
	"context"
	"path/filepath"
	"testing"
)

func TestCleanReservationProtectsReferences(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	s, err := Open(ctx, filepath.Join(root, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	gens := make([]Generation, 4)
	previous := ""
	var lease *Lease
	for i, id := range []string{"one", "two", "three", "four"} {
		g := Generation{ID: id, Directory: filepath.Join(root, id), InputDigest: id, PythonExecutable: filepath.Join(root, id, "python")}
		if err = s.Publish(ctx, g, previous); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			lease, err = s.AcquireActive(ctx)
			if err != nil {
				t.Fatal(err)
			}
		}
		gens[i], previous = g, id
	}
	// Protect an old generation by an unfinished operation sharing its directory.
	if err = s.BeginOperation(ctx, "preparing", gens[1].Directory, "d"); err != nil {
		t.Fatal(err)
	}
	items, err := s.CleanCandidates(ctx, "", 256)
	if err != nil || len(items) != 0 {
		t.Fatalf("protected candidates %+v %v", items, err)
	}
	for _, g := range gens {
		if marked, err := s.MarkDeleting(ctx, g.ID, g.Directory); err != nil || marked {
			t.Fatalf("reserved protected %s %v", g.ID, err)
		}
	}
	if err = s.ReleaseLease(ctx, lease.ID); err != nil {
		t.Fatal(err)
	}
	items, err = s.CleanCandidates(ctx, "", 1)
	if err != nil || len(items) != 1 || items[0].ID != "one" {
		t.Fatalf("released candidate %+v %v", items, err)
	}
	if marked, err := s.MarkDeleting(ctx, "one", filepath.Join(root, "wrong")); err != nil || marked {
		t.Fatal("accepted stale path")
	}
	if marked, err := s.MarkDeleting(ctx, "one", gens[0].Directory); err != nil || !marked {
		t.Fatalf("reservation %v %v", marked, err)
	}
	// A second connection observes the durable marker and cannot resurrect it.
	other, err := Open(ctx, filepath.Join(root, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	if err = other.BeginOperation(ctx, "late-operation", gens[0].Directory, "d"); err == nil {
		t.Fatal("started operation on deleting generation")
	}
	if _, err = other.db.ExecContext(ctx, `UPDATE active SET previous_id='one' WHERE singleton=1`); err == nil {
		t.Fatal("reactivated deleting generation")
	}
	if _, err = other.db.ExecContext(ctx, `INSERT INTO leases(id,generation_id,supervisor_pid) VALUES('late','one',1)`); err == nil {
		t.Fatal("leased deleting generation")
	}
	if marked, err := other.MarkDeleting(ctx, "one", gens[0].Directory); err != nil || !marked {
		t.Fatal("lost durable deletion reservation")
	}
	if err = other.FinishDeleting(ctx, "four", gens[3].Directory); err == nil {
		t.Fatal("finalized active generation")
	}
	if err = other.FinishDeleting(ctx, "one", gens[0].Directory); err != nil {
		t.Fatal(err)
	}
	var remaining int
	if err = s.db.QueryRowContext(ctx, `SELECT count(*) FROM generation_python WHERE generation_id='one'`).Scan(&remaining); err != nil || remaining != 0 {
		t.Fatal("Python entry not removed transactionally")
	}
	active, err := s.Active(ctx)
	if err != nil || active.ID != "four" {
		t.Fatal("clean changed active")
	}
	prior, err := s.Previous(ctx)
	if err != nil || prior.ID != "three" {
		t.Fatal("clean changed rollback target")
	}
	if err = s.FailOperation(ctx, "preparing"); err != nil {
		t.Fatal(err)
	}
	items, err = s.CleanCandidates(ctx, "", 256)
	if err != nil || len(items) != 1 || items[0].ID != "two" {
		t.Fatalf("failed operation candidate %+v %v", items, err)
	}
}
