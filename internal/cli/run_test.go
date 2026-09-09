package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"myenv/internal/backend"
	"myenv/internal/config"
	"myenv/internal/core"
	"myenv/internal/state"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type retainedNodeRun struct {
	Executable string
	Artifact   backend.NodeArtifact
}

func prepareRetainedNodeRun(t testing.TB) (string, retainedNodeRun) {
	t.Helper()
	record := os.Getenv("MYENV_TEST_PREPARED_RECORD")
	if record == "" {
		t.Skip("requires retained Node")
	}
	data, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	var prepared retainedNodeRun
	if err = json.Unmarshal(data, &prepared); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	work := filepath.Join(root, ".myenv")
	if err = os.Mkdir(work, 0700); err != nil {
		t.Fatal(err)
	}
	c := &config.Config{Schema: 1, Tools: map[string]string{"node": "22"}}
	if err = os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("schema: 1\ntools: {node: \"22\"}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = config.WriteSnapshot(work, c); err != nil {
		t.Fatal(err)
	}
	digest, _ := config.Digest(c)
	a := prepared.Artifact
	lock := &config.Lock{Schema: 1, ConfigDigest: digest, Platforms: map[string]config.PlatformLock{a.Platform: {Tools: map[string]config.RuntimeLock{"node": {Version: a.Version, Backend: a.Backend, Evidence: a.Evidence, URL: a.URL, SHA256: a.SHA256, NPM: a.NPM}}}}}
	for _, directory := range []string{root, work} {
		if err = config.WriteNewLock(filepath.Join(directory, "myenv.lock"), lock); err != nil {
			t.Fatal(err)
		}
	}
	if err = config.MarkComplete(work, digest); err != nil {
		t.Fatal(err)
	}
	store, err := state.Open(context.Background(), filepath.Join(work, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Publish(context.Background(), state.Generation{ID: "test", Directory: work, InputDigest: digest, NodeExecutable: prepared.Executable}, ""); err != nil {
		t.Fatal(err)
	}
	store.Close()
	return root, prepared
}

func TestRunRetainedNode(t *testing.T) {
	root, prepared := prepareRetainedNodeRun(t)
	a := prepared.Artifact
	var err error
	var out, diagnostic bytes.Buffer
	code := Execute([]string{"-C", root, "run", "node", "-e", "process.stdout.write(process.argv[1]);process.exit(17)", "--", "--json"}, bytes.NewReader(nil), &out, &diagnostic, "test")
	if code != 17 || out.String() != "--json" || diagnostic.Len() != 0 {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, out.String(), diagnostic.String())
	}
	for _, command := range []string{"npm", "npx"} {
		out.Reset()
		diagnostic.Reset()
		code = Execute([]string{"-C", root, "run", command, "--version"}, bytes.NewReader(nil), &out, &diagnostic, "test")
		if code != 0 || strings.TrimSpace(out.String()) != prepared.Artifact.NPM || diagnostic.Len() != 0 {
			t.Fatalf("%s exit=%d stdout=%q stderr=%q", command, code, out.String(), diagnostic.String())
		}
	}
	out.Reset()
	diagnostic.Reset()
	code = Execute([]string{"-C", root, "doctor", "--json"}, bytes.NewReader(nil), &out, &diagnostic, "test")
	var diagnosis struct {
		OK, Changed bool
		Data        core.Diagnosis
	}
	if err = json.Unmarshal(out.Bytes(), &diagnosis); err != nil {
		t.Fatal(err)
	}
	if code != 0 || !diagnosis.OK || diagnosis.Changed || diagnosis.Data.CheckLevel != "quick" || diagnosis.Data.Environment != "ready" || diagnosis.Data.AppliedNode != prepared.Executable {
		t.Fatalf("doctor code %d: %s %s", code, out.String(), diagnostic.String())
	}
	actualPath, pathErr := os.Stat(diagnosis.Data.RunPathNode)
	expectedPath, expectedErr := os.Stat(prepared.Executable)
	if pathErr != nil || expectedErr != nil || !os.SameFile(actualPath, expectedPath) {
		t.Fatal("doctor resolved a different run executable")
	}
	if err = os.WriteFile(filepath.Join(root, "myenv.lock"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	diagnostic.Reset()
	code = Execute([]string{"-C", root, "run", "node", "--version"}, bytes.NewReader(nil), &out, &diagnostic, "test")
	if code != 1 || out.Len() != 0 {
		t.Fatalf("accepted changed lock: %d %s", code, out.String())
	}
	out.Reset()
	diagnostic.Reset()
	code = Execute([]string{"-C", root, "run", "--current", "node", "--version"}, bytes.NewReader(nil), &out, &diagnostic, "test")
	if code != 0 || strings.TrimSpace(out.String()) != "v"+a.Version {
		t.Fatalf("current rejected retained runtime: %d %s %s", code, out.String(), diagnostic.String())
	}
}
