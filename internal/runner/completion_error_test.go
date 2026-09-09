package runner

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"
)

func TestDirectCompletionErrorRetainsProtection(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	failure := errors.New("injected completion accounting failure")
	closed := false
	code, err := executeDirectWithPreparation(context.Background(), Process{
		Executable:  executable,
		Args:        []string{"-test.run=^TestCompletionErrorChild$"},
		Environment: append(os.Environ(), "MYENV_COMPLETION_ERROR_CHILD=1"),
	}, func(ctx context.Context, command *exec.Cmd, id string) (*supervision, error) {
		supervisor := completionBoundaryFixture()
		finish, close := supervisor.Finish, supervisor.Close
		supervisor.Finish = func() error {
			// The real child is fully cleaned up; simulate unavailable accounting
			// evidence at the caller-facing completion boundary.
			if err := finish(); err != nil {
				return err
			}
			return failure
		}
		supervisor.Close = func() { close(); closed = true }
		return supervisor, nil
	})
	if code != 1 || !errors.Is(err, ErrTreeUnconfirmed) || !errors.Is(err, failure) || !closed {
		t.Fatalf("code=%d error=%v closed=%t", code, err, closed)
	}
}

func TestCompletionErrorChild(t *testing.T) {
	if os.Getenv("MYENV_COMPLETION_ERROR_CHILD") != "1" {
		t.Skip("subprocess helper")
	}
}

func TestDirectStartFailureCompletion(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name                   string
		uncertain, beforeStart bool
	}{
		{"confirmed", false, false}, {"unconfirmed", true, false}, {"before_start", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			uncertain := tc.uncertain
			startFailure := errors.New("injected supervisor start failure")
			finishFailure := errors.New("injected completion failure")
			finished, closed := false, false
			code, err := executeDirectWithPreparation(context.Background(), Process{
				Executable: executable, Args: []string{"-test.run=^TestCompletionErrorChild$"},
				Environment: append(os.Environ(), "MYENV_COMPLETION_ERROR_CHILD=1"),
			}, func(ctx context.Context, command *exec.Cmd, id string) (*supervision, error) {
				s := completionBoundaryFixture()
				start, finish, close := s.Start, s.Finish, s.Close
				s.Start = func() error {
					if tc.beforeStart {
						return startFailure
					}
					if err := start(); err != nil {
						return err
					}
					return startFailure
				}
				s.Finish = func() error {
					finished = true
					if err := finish(); err != nil {
						return err
					}
					if uncertain {
						return finishFailure
					}
					return nil
				}
				s.Close = func() { close(); closed = true }
				return s, nil
			})
			if code != 1 || !errors.Is(err, startFailure) || !finished || !closed || errors.Is(err, ErrTreeUnconfirmed) != uncertain || errors.Is(err, finishFailure) != uncertain {
				t.Fatalf("code=%d error=%v finished=%t closed=%t", code, err, finished, closed)
			}
		})
	}
}

// These tests exercise executeDirect's completion/error boundary, independently
// of the platform supervisor. The subprocess fixture never creates descendants;
// executeDirect itself waits for (or kills and waits for) this single process.
// Real platform tree accounting is covered by the platform integration tests.
func completionBoundaryFixture() *supervision {
	return &supervision{Start: func() error { return nil }, Finish: func() error { return nil }, Close: func() {}}
}
