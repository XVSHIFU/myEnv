package state

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

// Keep the production FULL durability setting. The phase metrics are means,
// not percentiles; each iteration registers and then releases a real lease.
func BenchmarkLeaseLifecycle(b *testing.B) {
	ctx := context.Background()
	root := b.TempDir()
	s, err := Open(ctx, filepath.Join(root, "state.db"))
	if err != nil {
		b.Fatal(err)
	}
	defer s.Close()
	if err := s.Publish(ctx, Generation{ID: "active", Directory: filepath.Join(root, "active"), InputDigest: "input", NodeExecutable: filepath.Join(root, "active", "node.exe")}, ""); err != nil {
		b.Fatal(err)
	}
	var acquire, release time.Duration
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		lease, err := s.AcquireActive(ctx)
		acquire += time.Since(start)
		if err != nil {
			b.Fatal(err)
		}
		start = time.Now()
		err = s.ReleaseLease(ctx, lease.ID)
		release += time.Since(start)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	b.ReportMetric(float64(acquire.Nanoseconds())/float64(b.N), "acquire-ns/op")
	b.ReportMetric(float64(release.Nanoseconds())/float64(b.N), "release-ns/op")
	if present, err := s.HasLeases(ctx); err != nil || present {
		b.Fatalf("lease cleanup: present=%t error=%v", present, err)
	}
}

func BenchmarkGenerationLeasePage(b *testing.B) {
	ctx := context.Background()
	root := b.TempDir()
	s, err := Open(ctx, filepath.Join(root, "state.db"))
	if err != nil {
		b.Fatal(err)
	}
	defer s.Close()
	previous := ""
	for _, id := range []string{"target", "other"} {
		if err = s.Publish(ctx, Generation{ID: id, Directory: filepath.Join(root, id), InputDigest: id, PythonExecutable: filepath.Join(root, id, "python")}, previous); err != nil {
			b.Fatal(err)
		}
		previous = id
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < 10000; i++ {
		generation := "other"
		if i < 128 {
			generation = "target"
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO leases(id,generation_id,supervisor_pid) VALUES(?,?,1)`, fmt.Sprintf("%032x", i), generation); err != nil {
			tx.Rollback()
			b.Fatal(err)
		}
	}
	if err = tx.Commit(); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows, err := s.GenerationLeaseRecords(ctx, "target", "", 128)
		if err != nil || len(rows) != 128 {
			b.Fatal("page", len(rows), err)
		}
	}
}
