package core

import (
	"context"
	"myenv/internal/config"
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveDefaultLastTool(t *testing.T) {
	root := t.TempDir()
	profile, err := config.ProfilePath(root)
	if err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Dir(profile), 0700)
	if err = os.WriteFile(profile, []byte("# retain this comment\nschema: 1\ntools: {go: '1.26.6'}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	s := Service{Profile: true, UserConfigDirectory: root}
	removed, err := s.RemoveDefault(context.Background(), "go")
	if err != nil {
		t.Fatal(err)
	}
	if !removed.DeclarationChanged || !removed.Changed {
		t.Fatal("no published empty profile")
	}
	c, err := config.LoadProfile(profile)
	if err != nil || len(c.Tools) != 0 {
		t.Fatalf("profile: %+v %v", c, err)
	}
	status, err := s.Status(context.Background(), "")
	if err != nil || status.Environment != "ready" {
		t.Fatalf("status: %+v %v", status, err)
	}
	second, err := s.RemoveDefault(context.Background(), "go")
	if err != nil || second.Changed {
		t.Fatalf("non-idempotent remove: %+v %v", second, err)
	}
	if _, err = config.Parse([]byte("schema: 1\ntools: {}\n"), root); err == nil {
		t.Fatal("empty project unexpectedly allowed")
	}
}
