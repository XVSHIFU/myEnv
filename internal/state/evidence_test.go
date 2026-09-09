package state

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvidencePublicationAtomic(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	s, err := Open(ctx, filepath.Join(root, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	g := Generation{ID: "one", Directory: filepath.Join(root, "one"), InputDigest: "d", PythonExecutable: filepath.Join(root, "one", "python")}
	digest := strings.Repeat("a", 64)
	if err = s.PublishWithEvidence(ctx, g, "", digest); err != nil {
		t.Fatal(err)
	}
	if value, err := s.GenerationEvidence(ctx, g.ID); err != nil || value != digest {
		t.Fatal("baseline missing", value, err)
	}
	other := g
	other.ID = "two"
	other.Directory = filepath.Join(root, "two")
	if err = s.PublishWithEvidence(ctx, other, "stale", digest); err == nil {
		t.Fatal("stale publish accepted")
	}
	if value, err := s.GenerationEvidence(ctx, other.ID); err != nil || value != "" {
		t.Fatal("failed publication leaked evidence", err)
	}
	if err = s.Publish(ctx, other, "one"); err != nil {
		t.Fatal(err)
	}
	if value, err := s.GenerationEvidence(ctx, other.ID); err != nil || value != "" {
		t.Fatal("invented legacy baseline", err)
	}
}
