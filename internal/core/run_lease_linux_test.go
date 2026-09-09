package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"myenv/internal/runner"
	"myenv/internal/state"
)

func TestRunFinishProtectsUnconfirmedTree(t *testing.T) {
	for _, uncertain := range []bool{false, true} {
		t.Run(fmt.Sprint(uncertain), func(t *testing.T) {
			ctx := context.Background()
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("schema: 1\ntools: {node: '22'}\n"), 0600); err != nil {
				t.Fatal(err)
			}
			work := filepath.Join(root, ".myenv")
			if err := os.Mkdir(work, 0700); err != nil {
				t.Fatal(err)
			}
			database := filepath.Join(work, "state.db")
			store, err := state.Open(ctx, database)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			var selected *RunEnvironment
			previous := ""
			for i := 1; i <= 4; i++ {
				id := fmt.Sprintf("%032x", i)
				dir := filepath.Join(work, "generations", id)
				if err = os.MkdirAll(dir, 0700); err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(filepath.Join(dir, "payload"), []byte("retained"), 0600); err != nil {
					t.Fatal(err)
				}
				if err = store.Publish(ctx, state.Generation{ID: id, Directory: dir, InputDigest: id, NodeExecutable: filepath.Join(dir, "node")}, previous); err != nil {
					t.Fatal(err)
				}
				if i == 1 {
					lease, err := acquireRunLease(ctx, store)
					if err != nil {
						t.Fatal(err)
					}
					selected = leasedRunEnvironment(store, lease)
				}
				previous = id
			}
			if err = prepareRunCompletion(ctx, work, store, selected); err != nil {
				t.Fatal(err)
			}
			script := "exit 17"
			if uncertain {
				script = "setsid /bin/sh -c 'printf ready > started; sleep 0.5; cat payload > survived' </dev/null >/dev/null 2>&1 & while [ ! -e started ]; do sleep 0.01; done; kill -KILL $PPID; exit 0"
				// Even on assertion failure, let the short fixture finish before
				// removing its working directory; no signaling orphan PIDs.
				defer func() {
					deadline := time.Now().Add(2 * time.Second)
					for time.Now().Before(deadline) {
						if _, err := os.Stat(filepath.Join(selected.Generation.Directory, "survived")); err == nil {
							return
						}
						time.Sleep(10 * time.Millisecond)
					}
				}()
			}
			code, runErr := runner.Execute(ctx, runner.Process{Completion: selected.Completion, TreeID: selected.TreeID, Executable: "/bin/sh", Args: []string{"-c", script}, Directory: selected.Generation.Directory, Environment: os.Environ()})
			if errors.Is(runErr, runner.ErrTreeUnconfirmed) != uncertain || (!uncertain && (runErr != nil || code != 17)) {
				t.Fatalf("runner result: %d %v", code, runErr)
			}
			if err = selected.Finish(runErr); err != nil {
				t.Fatal(err)
			}
			receipt, receiptErr := os.ReadFile(filepath.Join(work, leaseReceiptName(selected.TreeID)))
			if uncertain && (receiptErr != nil || len(receipt) != 0) {
				t.Fatal("uncertain receipt not retained empty", receiptErr)
			}
			if !uncertain && !os.IsNotExist(receiptErr) {
				t.Fatal("normal finish retained receipt", receiptErr)
			}
			other, err := state.OpenReadOnly(ctx, database)
			if err != nil {
				t.Fatal(err)
			}
			records, err := other.GenerationLeaseRecords(ctx, selected.Generation.ID, "", 128)
			other.Close()
			want := 0
			if uncertain {
				want = 1
			}
			if err != nil || len(records) != want {
				t.Fatalf("durable leases: %+v %v", records, err)
			}
			for _, dry := range []bool{true, false} {
				result, err := (&Service{}).Clean(ctx, root, dry, nil)
				if err != nil || result.Candidates != 2-want {
					t.Fatalf("clean dry=%v: %+v %v", dry, result, err)
				}
			}
			_, err = os.Stat(filepath.Join(selected.Generation.Directory, "payload"))
			if uncertain && err != nil {
				t.Fatal("clean removed uncertain generation", err)
			}
			if !uncertain && !os.IsNotExist(err) {
				t.Fatal("completed generation was not cleaned", err)
			}
			if uncertain {
				deadline := time.Now().Add(2 * time.Second)
				for time.Now().Before(deadline) {
					data, err := os.ReadFile(filepath.Join(selected.Generation.Directory, "survived"))
					if err == nil && string(data) == "retained" {
						return
					}
					time.Sleep(10 * time.Millisecond)
				}
				t.Fatal("orphan could not read the retained environment")
			}
		})
	}
}
