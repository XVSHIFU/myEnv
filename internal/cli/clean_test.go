package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"myenv/internal/core"
	"myenv/internal/state"
)

func TestCleanCLIResult(t *testing.T) {
	for _, unsafe := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "partial-failure"}[unsafe], func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("schema: 1\ntools: {python: '3.12'}\n"), 0600); err != nil {
				t.Fatal(err)
			}
			work := filepath.Join(root, ".myenv")
			if err := os.Mkdir(work, 0700); err != nil {
				t.Fatal(err)
			}
			store, err := state.Open(context.Background(), filepath.Join(work, "state.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			previous := ""
			for _, id := range []string{"00000000000000000000000000000001", "00000000000000000000000000000002", "00000000000000000000000000000003", "00000000000000000000000000000004"} {
				dir := filepath.Join(work, "generations", id)
				if unsafe && id == "00000000000000000000000000000002" {
					dir = filepath.Join(root, "outside-generations")
				}
				if err = os.MkdirAll(dir, 0700); err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(filepath.Join(dir, "python"), []byte("fixture"), 0600); err != nil {
					t.Fatal(err)
				}
				if err = store.Publish(context.Background(), state.Generation{ID: id, Directory: dir, InputDigest: id, PythonExecutable: filepath.Join(dir, "python")}, previous); err != nil {
					t.Fatal(err)
				}
				previous = id
			}
			for _, dry := range []bool{true, false} {
				args := []string{"-C", root, "clean", "--json"}
				if dry {
					args = append(args, "--dry-run")
				}
				var out, diagnostic bytes.Buffer
				code := Execute(args, bytes.NewReader(nil), &out, &diagnostic, "test")
				var response struct {
					OK, Changed bool
					Error       *failure
					Data        struct {
						Items   []core.CleanItem
						Summary core.CleanResult
					}
				}
				decoder := json.NewDecoder(&out)
				if err = decoder.Decode(&response); err != nil {
					t.Fatal("invalid JSON", err, out.String())
				}
				if err = decoder.Decode(new(any)); err != io.EOF {
					t.Fatal("more than one JSON result", err)
				}
				if response.OK == unsafe || response.Changed == dry || diagnostic.Len() != 0 {
					t.Fatalf("response %+v stderr=%s", response, diagnostic.String())
				}
				want := 2
				if unsafe {
					want = 1
					if code != 1 || response.Error == nil || response.Error.Code != "CLEAN_FAILED" {
						t.Fatalf("failure %+v exit %d", response, code)
					}
				} else if code != 0 {
					t.Fatalf("exit %d", code)
				}
				if len(response.Data.Items) != want {
					t.Fatalf("items %+v", response.Data.Items)
				}
				if dry && response.Data.Summary.Removed != 0 {
					t.Fatal("dry-run removed files")
				}
				if !dry && response.Data.Summary.Removed != want {
					t.Fatalf("removed %+v", response.Data.Summary)
				}
				if dry {
					if _, err = os.Stat(filepath.Join(work, "generations", "00000000000000000000000000000001", "python")); err != nil {
						t.Fatal(err)
					}
				}
			}
			for _, id := range []string{"00000000000000000000000000000003", "00000000000000000000000000000004"} {
				if _, err = os.Stat(filepath.Join(work, "generations", id, "python")); err != nil {
					t.Fatal("removed protected generation", err)
				}
			}
			if unsafe {
				if _, err = os.Stat(filepath.Join(root, "outside-generations", "python")); err != nil {
					t.Fatal("removed outside file", err)
				}
			}
		})
	}
}

func TestCleanNodeCacheCLI(t *testing.T) {
	namespace := t.TempDir()
	service, err := runtimeService(false, namespace)
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(service.Storage.Cache, "node-archives")
	if err = os.MkdirAll(directory, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, strings.Repeat("a", 64))
	if err = os.WriteFile(path, []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(directory, "unrelated")
	if err = os.WriteFile(keep, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, dry := range []bool{true, false} {
		args := []string{"clean", "--cache", "node", "--json"}
		if dry {
			args = append(args, "--dry-run")
		}
		var out, diagnostic bytes.Buffer
		code := execute(args, bytes.NewReader(nil), &out, &diagnostic, "test", namespace)
		var response result
		if err = json.Unmarshal(out.Bytes(), &response); err != nil {
			t.Fatal(err, out.String())
		}
		if code != 0 || !response.OK || response.Changed == dry || diagnostic.Len() != 0 {
			t.Fatalf("cache result %+v %d %s", response, code, diagnostic.String())
		}
		if dry {
			if _, err = os.Stat(path); err != nil {
				t.Fatal("preview removed cache", err)
			}
		}
	}
	if _, err = os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("cache not removed", err)
	}
	if _, err = os.Stat(keep); err != nil {
		t.Fatal("unrelated file removed", err)
	}
}
