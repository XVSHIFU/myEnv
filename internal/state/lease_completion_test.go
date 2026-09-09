package state

import (
	"context"
	"encoding/hex"
	"path/filepath"
	"strings"
	"testing"
)

func TestAcquireActiveCompletionAtomic(t *testing.T) {
	ctx := context.Background()
	for _, fail := range []bool{false, true} {
		name := "commit"
		if fail {
			name = "rollback"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			s, err := Open(ctx, filepath.Join(root, "state.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			if err := s.Publish(ctx, Generation{ID: "one", Directory: root, InputDigest: "one", NodeExecutable: filepath.Join(root, "node")}, ""); err != nil {
				t.Fatal(err)
			}
			if fail {
				if _, err := s.db.Exec(`CREATE TRIGGER reject_completion BEFORE INSERT ON lease_completion BEGIN SELECT RAISE(ABORT,'injected token failure'); END`); err != nil {
					t.Fatal(err)
				}
			}
			lease, err := s.AcquireActiveWithCompletion(ctx)
			if fail {
				if err == nil || lease != nil {
					t.Fatal("accepted failed binding", lease, err)
				}
				for _, table := range []string{"leases", "lease_identity", "lease_completion"} {
					var count int
					if err := s.db.QueryRow("SELECT count(*) FROM " + table).Scan(&count); err != nil || count != 0 {
						t.Fatal("partial registration", table, count, err)
					}
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := hex.DecodeString(lease.CompletionToken)
			if err != nil || len(decoded) != 32 {
				t.Fatal("invalid recovery token", err)
			}
			reader, err := OpenReadOnly(ctx, filepath.Join(root, "state.db"))
			if err != nil {
				t.Fatal(err)
			}
			records, err := reader.LeaseRecords(ctx, "", 128)
			reader.Close()
			if err != nil || len(records) != 1 || records[0].CompletionToken != lease.CompletionToken {
				t.Fatal("binding not committed", records, err)
			}
			if err := s.ReleaseLease(ctx, lease.ID); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestLeaseCompletionBindingCAS(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	s, err := Open(ctx, filepath.Join(root, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err = s.Publish(ctx, Generation{ID: "one", Directory: root, InputDigest: "one", NodeExecutable: filepath.Join(root, "node")}, ""); err != nil {
		t.Fatal(err)
	}
	lease, err := s.AcquireActive(ctx)
	if err != nil {
		t.Fatal(err)
	}
	before, err := s.LeaseRecords(ctx, "", 128)
	if err != nil {
		t.Fatal(err)
	}
	token := strings.Repeat("a", 64)
	if err = s.BindLeaseCompletion(ctx, lease.ID, token); err != nil {
		t.Fatal(err)
	}
	if err = s.BindLeaseCompletion(ctx, lease.ID, strings.Repeat("b", 64)); err == nil {
		t.Fatal("overwrote binding")
	}
	if removed, err := s.DeleteObservedLease(ctx, before[0]); err != nil || removed {
		t.Fatal("stale unbound observation deleted lease", removed, err)
	}
	after, err := s.LeaseRecords(ctx, "", 128)
	if err != nil || len(after) != 1 || after[0].CompletionToken != token {
		t.Fatal("binding not visible", after, err)
	}
	if removed, err := s.DeleteObservedLease(ctx, after[0]); err != nil || !removed {
		t.Fatal("matching bound observation not removed", removed, err)
	}
	var count int
	if err = s.db.QueryRow(`SELECT count(*) FROM lease_completion`).Scan(&count); err != nil || count != 0 {
		t.Fatal("binding did not cascade", count, err)
	}
	if err = s.BindLeaseCompletion(ctx, lease.ID, token); err == nil {
		t.Fatal("bound missing lease")
	}
}
