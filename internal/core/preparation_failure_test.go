package core

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"myenv/internal/runner"
	"myenv/internal/state"
)

func TestPreparationFailureCompletionPolicy(t *testing.T) {
	for _, tc := range []struct {
		name, platform string
		uncertain      bool
		wantCandidate  bool
	}{
		{"windows_confirmed", "windows-amd64", false, true},
		{"linux_confirmed", "linux-amd64-glibc", false, true},
		{"windows_unknown", "windows-amd64", true, false},
		{"macos_unproven", "darwin-arm64", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			root := t.TempDir()
			unlock, err := state.LockWorkspace(ctx, filepath.Join(root, "modify.lock"))
			if err != nil {
				t.Fatal(err)
			}
			defer unlock()
			s, err := state.Open(ctx, filepath.Join(root, "state.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			if err := s.BeginGuardedOperation(ctx, "pending", filepath.Join(root, "generation"), "digest"); err != nil {
				t.Fatal(err)
			}
			cause := errors.New("backend failed")
			failure := cause
			if tc.uncertain {
				failure = errors.Join(cause, runner.ErrTreeUnconfirmed)
			}
			err = finishFailedPreparation(s, "pending", tc.platform, failure)
			if !errors.Is(err, cause) || errors.Is(err, runner.ErrTreeUnconfirmed) == tc.wantCandidate {
				t.Fatalf("failure identity: %v", err)
			}
			if _, err := s.RecoverCompletedPreparations(ctx); err != nil {
				t.Fatal(err)
			}
			items, err := s.PreparationCandidates(ctx, "", 128)
			want := 0
			if tc.wantCandidate {
				want = 1
			}
			if err != nil || len(items) != want {
				t.Fatalf("candidates=%+v error=%v", items, err)
			}
		})
	}
}
