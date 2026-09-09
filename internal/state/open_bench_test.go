package state

import (
	"context"
	"path/filepath"
	"testing"
)

// This measures opening an existing local database, including connection close.
// Read-only is a diagnostic baseline, not a replacement for writable run leases.
func BenchmarkOpenExistingStore(b *testing.B) {
	ctx := context.Background()
	path := filepath.Join(b.TempDir(), "state.db")
	store, err := Open(ctx, path)
	if err != nil {
		b.Fatal(err)
	}
	if err := store.Close(); err != nil {
		b.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		open func(context.Context, string) (*Store, error)
	}{
		{"writable", Open},
		{"readonly", OpenReadOnly},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				store, err := tc.open(ctx, path)
				if err != nil {
					b.Fatal(err)
				}
				if err := store.Close(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
