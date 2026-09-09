package core

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSyncAlreadyCanceledPreservesProject(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		t.Run(map[bool]string{false: "apply", true: "preview"}[dryRun], func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("schema: 1\ntools: {node: '22'}\n"), 0600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			result, err := (&Service{}).Sync(ctx, SyncRequest{Directory: root, DryRun: dryRun})
			if !errors.Is(err, context.Canceled) || result.Changed || result.LockChanged || result.NativeLockChanged || result.Generation != nil || result.Plan != nil {
				t.Fatalf("canceled sync: %+v %v", result, err)
			}
			entries, err := os.ReadDir(root)
			if err != nil || len(entries) != 1 || entries[0].Name() != "myenv.yaml" {
				t.Fatalf("canceled sync created state: %v %v", entries, err)
			}
		})
	}
}
