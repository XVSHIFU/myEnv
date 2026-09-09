package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"myenv/internal/backend"
	"myenv/internal/config"
	"myenv/internal/state"
)

func TestRollbackRetainedNode(t *testing.T) {
	record := os.Getenv("MYENV_TEST_PREPARED_RECORD")
	if record == "" {
		t.Skip("requires retained official Node")
	}
	data, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	var prepared struct {
		Executable string
		Artifact   backend.NodeArtifact
	}
	if err = json.Unmarshal(data, &prepared); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	work := filepath.Join(root, ".myenv")
	if err = os.Mkdir(work, 0700); err != nil {
		t.Fatal(err)
	}
	store, err := state.Open(context.Background(), filepath.Join(work, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	prior := ""
	var desiredLock []byte
	for _, id := range []string{"old", "new"} {
		directory := filepath.Join(work, id)
		if err = os.Mkdir(directory, 0700); err != nil {
			t.Fatal(err)
		}
		c := &config.Config{Schema: 1, Tools: map[string]string{"node": "22"}, Env: map[string]string{"MYENV_ROLLBACK_VALUE": id}}
		digest, _ := config.Digest(c)
		if err = config.WriteSnapshot(directory, c); err != nil {
			t.Fatal(err)
		}
		a := prepared.Artifact
		lock := &config.Lock{Schema: 1, ConfigDigest: digest, Platforms: map[string]config.PlatformLock{a.Platform: {Tools: map[string]config.RuntimeLock{"node": {Version: a.Version, Backend: a.Backend, Evidence: a.Evidence, URL: a.URL, SHA256: a.SHA256, NPM: a.NPM}}}}}
		if err = config.WriteNewLock(filepath.Join(directory, "myenv.lock"), lock); err != nil {
			t.Fatal(err)
		}
		if err = config.MarkComplete(directory, digest); err != nil {
			t.Fatal(err)
		}
		if err = store.Publish(context.Background(), state.Generation{ID: id, Directory: directory, InputDigest: digest, NodeExecutable: prepared.Executable}, prior); err != nil {
			t.Fatal(err)
		}
		prior = id
		desiredLock, err = os.ReadFile(filepath.Join(directory, "myenv.lock"))
		if err != nil {
			t.Fatal(err)
		}
	}
	declaration := []byte("schema: 1\ntools: {node: \"22\"}\nenv: {MYENV_ROLLBACK_VALUE: new}\n")
	for name, data := range map[string][]byte{"myenv.yaml": declaration, "myenv.lock": desiredLock} {
		if err = os.WriteFile(filepath.Join(root, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	var out, diagnostic bytes.Buffer
	code := Execute([]string{"-C", root, "rollback", "--json"}, bytes.NewReader(nil), &out, &diagnostic, "test")
	var response struct {
		OK, Changed bool
		Data        state.Generation
	}
	if err = json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if code != 0 || !response.OK || !response.Changed || response.Data.ID != "old" || diagnostic.Len() != 0 {
		t.Fatalf("rollback code %d: %s %s", code, out.String(), diagnostic.String())
	}
	for name, want := range map[string][]byte{"myenv.yaml": declaration, "myenv.lock": desiredLock} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || !bytes.Equal(data, want) {
			t.Fatalf("changed %s: %v", name, err)
		}
	}
	out.Reset()
	diagnostic.Reset()
	code = Execute([]string{"-C", root, "run", "node", "--version"}, bytes.NewReader(nil), &out, &diagnostic, "test")
	if code != 1 || out.Len() != 0 {
		t.Fatalf("ordinary run accepted drift: %d %s", code, out.String())
	}
	out.Reset()
	diagnostic.Reset()
	code = Execute([]string{"-C", root, "run", "--current", "node", "-e", "process.stdout.write(process.env.MYENV_ROLLBACK_VALUE);process.exit(17)"}, bytes.NewReader(nil), &out, &diagnostic, "test")
	if code != 17 || out.String() != "old" || diagnostic.Len() != 0 {
		t.Fatalf("current exit %d: %s %s", code, out.String(), diagnostic.String())
	}
}
