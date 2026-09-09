package cli

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"myenv/internal/runner"
	"myenv/internal/state"
)

func TestRunFailureContextHint(t *testing.T) {
	t.Setenv("MYENV_LANG", "en")
	for _, global := range []bool{false, true} {
		for _, uncertain := range []bool{false, true} {
			cause := context.Canceled
			if uncertain {
				cause = errors.Join(cause, runner.ErrTreeUnconfirmed)
			}
			failure := &runFailure{cause: cause, global: global}
			hint := failure.nextAction()
			if strings.Contains(hint, "myenv doctor --global") != global || strings.Contains(hint, "protection is retained") != uncertain || !errors.Is(failure, context.Canceled) || errors.Is(failure, runner.ErrTreeUnconfirmed) != uncertain {
				t.Fatalf("context global=%v uncertain=%v: %s %v", global, uncertain, hint, failure)
			}
		}
	}
}

func TestRunLeaseReleaseFailure(t *testing.T) {
	t.Setenv("MYENV_LANG", "en")
	for _, childCode := range []int{0, 7} {
		t.Run(fmt.Sprint(childCode), func(t *testing.T) {
			root, _ := prepareRetainedNodeRun(t)
			db, err := sql.Open("sqlite", filepath.Join(root, ".myenv", "state.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if _, err := db.Exec(`CREATE TRIGGER reject_release BEFORE DELETE ON leases BEGIN SELECT RAISE(ABORT, 'test lease release failure'); END`); err != nil {
				t.Fatal(err)
			}
			var out, diagnostic bytes.Buffer
			exit := Execute([]string{"-C", root, "run", "node", "-e", fmt.Sprintf("process.stdout.write('executed'); process.exit(%d)", childCode)}, bytes.NewReader(nil), &out, &diagnostic, "test")
			if exit != childCode || out.String() != "executed" || !strings.Contains(diagnostic.String(), "IO_ERROR: could not finalize process lease:") || !strings.Contains(diagnostic.String(), "test lease release failure") {
				t.Fatalf("exit=%d stdout=%q stderr=%q", exit, out.String(), diagnostic.String())
			}
			var count int
			if err := db.QueryRow(`SELECT count(*) FROM leases l JOIN lease_identity i ON i.lease_id=l.id`).Scan(&count); err != nil || count != 1 {
				t.Fatalf("failed release lost protection: count=%d err=%v", count, err)
			}
		})
	}
}

func TestRunExecutorFailureExit(t *testing.T) {
	t.Setenv("MYENV_LANG", "en")
	root, prepared := prepareRetainedNodeRun(t)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(cwd, prepared.Executable)
	if err != nil || filepath.IsAbs(relative) {
		t.Fatalf("relative entry: %q %v", relative, err)
	}
	db, err := sql.Open("sqlite", filepath.Join(root, ".myenv", "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`UPDATE generations SET node_executable=?`, relative); err != nil {
		t.Fatal(err)
	}
	var out, diagnostic bytes.Buffer
	exit := Execute([]string{"-C", root, "run", "node"}, bytes.NewReader(nil), &out, &diagnostic, "test")
	if exit != 1 || out.Len() != 0 || !strings.HasPrefix(diagnostic.String(), "RUN_FAILED:") {
		t.Fatalf("exit=%d stdout=%q stderr=%q", exit, out.String(), diagnostic.String())
	}
	var leases int
	if err := db.QueryRow(`SELECT count(*) FROM leases`).Scan(&leases); err != nil || leases != 0 {
		t.Fatalf("prelaunch failure leaked lease: %d %v", leases, err)
	}
}

func TestRunMissingEnvironmentExit(t *testing.T) {
	t.Setenv("MYENV_LANG", "en")
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("schema: 1\ntools: {node: \"22\"}"), 0600); err != nil {
		t.Fatal(err)
	}
	var out, diagnostic bytes.Buffer
	exit := Execute([]string{"-C", root, "run", "node", "--json"}, bytes.NewReader(nil), &out, &diagnostic, "test")
	if exit != 1 || out.Len() != 0 || !strings.HasPrefix(diagnostic.String(), "ENV_NOT_READY:") {
		t.Fatalf("exit %d stdout=%q stderr=%q", exit, out.String(), diagnostic.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".myenv")); !os.IsNotExist(err) {
		t.Fatal("run initialized environment state")
	}
}

func TestRunLeaseFailureExit(t *testing.T) {
	t.Setenv("MYENV_LANG", "en")
	for _, applied := range []bool{false, true} {
		name, want := "no_active_generation", "ENV_NOT_READY:"
		if applied {
			name, want = "lease_insert_failure", "IO_ERROR:"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("schema: 1\ntools: {node: \"22\"}\n"), 0600); err != nil {
				t.Fatal(err)
			}
			work := filepath.Join(root, ".myenv")
			if err := os.Mkdir(work, 0700); err != nil {
				t.Fatal(err)
			}
			database := filepath.Join(work, "state.db")
			store, err := state.Open(context.Background(), database)
			if err != nil {
				t.Fatal(err)
			}
			if applied {
				err = store.Publish(context.Background(), state.Generation{ID: "fixture", Directory: work, InputDigest: "unused", NodeExecutable: filepath.Join(work, "node.exe")}, "")
			}
			closeErr := store.Close()
			if err != nil || closeErr != nil {
				t.Fatal(err, closeErr)
			}
			db, err := sql.Open("sqlite", database)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if applied {
				if _, err = db.Exec(`CREATE TRIGGER reject_lease BEFORE INSERT ON leases BEGIN SELECT RAISE(ABORT, 'test lease write failure'); END`); err != nil {
					t.Fatal(err)
				}
			}
			var out, diagnostic bytes.Buffer
			exit := Execute([]string{"-C", root, "run", "node"}, bytes.NewReader(nil), &out, &diagnostic, "test")
			if exit != 1 || out.Len() != 0 || !strings.HasPrefix(diagnostic.String(), want) {
				t.Fatalf("exit %d stdout=%q stderr=%q", exit, out.String(), diagnostic.String())
			}
			if applied && !strings.Contains(diagnostic.String(), "test lease write failure") {
				t.Fatal("lost underlying database error", diagnostic.String())
			}
			var count int
			if err := db.QueryRow(`SELECT count(*) FROM leases`).Scan(&count); err != nil || count != 0 {
				t.Fatalf("failed selection left leases: %d %v", count, err)
			}
		})
	}
}

func TestRunCorruptStateExit(t *testing.T) {
	t.Setenv("MYENV_LANG", "en")
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("schema: 1\ntools: {node: \"22\"}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	work := filepath.Join(root, ".myenv")
	if err := os.Mkdir(work, 0700); err != nil {
		t.Fatal(err)
	}
	database := filepath.Join(work, "state.db")
	original := bytes.Repeat([]byte("invalid database\n"), 64)
	if err := os.WriteFile(database, original, 0600); err != nil {
		t.Fatal(err)
	}
	var out, diagnostic bytes.Buffer
	exit := Execute([]string{"-C", root, "run", "node"}, bytes.NewReader(nil), &out, &diagnostic, "test")
	if exit != 1 || out.Len() != 0 || !strings.HasPrefix(diagnostic.String(), "IO_ERROR:") || !strings.Contains(diagnostic.String(), database) {
		t.Fatalf("exit %d stdout=%q stderr=%q", exit, out.String(), diagnostic.String())
	}
	remaining, err := os.ReadFile(database)
	if err != nil || !bytes.Equal(remaining, original) {
		t.Fatalf("failed run changed corrupt database: %v", err)
	}
}
