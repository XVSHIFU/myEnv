package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"myenv/internal/backend"
	"myenv/internal/config"
	"myenv/internal/runner"
	"myenv/internal/runtrace"
	"myenv/internal/state"
)

type RunEnvironment struct {
	Completion      *runner.TreeCompletion
	TreeID          string
	Generation      state.Generation
	Config          *config.Config
	Release         func() error
	retainLease     func() error
	completionToken string
}

// Finish closes the state connection and releases protection only when the
// runner has not reported an unconfirmed process tree.
func (r *RunEnvironment) Finish(runErr error) error {
	if errors.Is(runErr, runner.ErrTreeUnconfirmed) {
		return r.retainLease()
	}
	return r.Release()
}

func leasedRunEnvironment(store *state.Store, lease *state.Lease) *RunEnvironment {
	return &RunEnvironment{
		TreeID: lease.ID, Generation: lease.Generation, retainLease: store.Close, completionToken: lease.CompletionToken,
		Release: func() error {
			runtrace.Mark("lease_release_begin")
			defer runtrace.Mark("lease_release_end")
			return errors.Join(store.ReleaseLease(context.Background(), lease.ID), store.Close())
		},
	}
}

// SelectRun never installs. The lease protects the selected generation until
// the caller finishes supervising the command and calls Finish with its error.
func (s *Service) SelectRun(ctx context.Context, directory string, current bool) (*RunEnvironment, error) {
	runtrace.Mark("selection_begin")
	root, err := s.resolveRoot(directory)
	if err != nil {
		return nil, err
	}
	c, err := s.loadDeclaration(root)
	if err != nil {
		return nil, err
	}
	database := filepath.Join(s.workDirectory(root), "state.db")
	runtrace.Mark("declaration_loaded")
	if _, err = os.Stat(database); err != nil {
		return nil, fmt.Errorf("%s: %w", s.commandHint("ENV_NOT_READY: no applied environment; run myenv sync"), err)
	}
	store, err := openRunStore(ctx, database)
	if err != nil {
		return nil, &os.PathError{Op: "open environment state", Path: database, Err: err}
	}
	runtrace.Mark("state_opened")
	lease, err := acquireRunLease(ctx, store)
	runtrace.Mark("lease_acquired")
	if err != nil {
		store.Close()
		if errors.Is(err, state.ErrNoAppliedGeneration) {
			return nil, err
		}
		return nil, &os.PathError{Op: "acquire environment lease", Path: database, Err: err}
	}
	selected := leasedRunEnvironment(store, lease)
	fail := func(err error) (*RunEnvironment, error) {
		return nil, errors.Join(err, selected.Release())
	}
	digest, err := config.Digest(c)
	if err != nil {
		return fail(err)
	}
	if !current && digest != lease.Generation.InputDigest {
		return fail(fmt.Errorf("%s", s.commandHint("ENV_NOT_READY: configuration differs from the applied environment; run myenv sync or use run --current")))
	}
	platform, err := runner.Platform()
	if err != nil {
		return fail(err)
	}
	appliedLock, appliedErr := config.ReadLock(filepath.Join(lease.Generation.Directory, "myenv.lock"))
	if !current {
		desired, err := config.ReadLock(s.desiredLockPath(root))
		inputs, inputsErr := pythonInputs(root, c, false)
		if err != nil || appliedErr != nil || inputsErr != nil || desired.ConfigDigest != digest || appliedLock.ConfigDigest != digest || !reflect.DeepEqual(appliedLock.Platforms[platform], desired.Platforms[platform]) || !lockHasTools(desired.Platforms[platform].Tools, c.Tools) || !reflect.DeepEqual(inputs, appliedLock.Platforms[platform].Python) {
			return fail(fmt.Errorf("%s", s.commandHint("ENV_NOT_READY: runtime lock differs from the applied environment; run myenv sync or use run --current")))
		}
	}
	if err = config.CheckComplete(lease.Generation.Directory, lease.Generation.InputDigest); err != nil {
		return fail(fmt.Errorf("ENV_NOT_READY: incomplete generation: %w", err))
	}
	applied, err := config.ReadSnapshot(lease.Generation.Directory, root)
	if err != nil {
		return fail(fmt.Errorf("ENV_NOT_READY: read applied configuration: %w", err))
	}
	appliedDigest, err := config.Digest(applied)
	if err != nil {
		return fail(err)
	}
	if appliedDigest != lease.Generation.InputDigest {
		return fail(fmt.Errorf("ENV_NOT_READY: applied configuration snapshot differs from generation record"))
	}
	if appliedErr != nil || appliedLock.ConfigDigest != appliedDigest || !platformLockHealthy(appliedLock.Platforms[platform], applied) {
		return fail(fmt.Errorf("ENV_NOT_READY: applied runtime lock is incomplete"))
	}
	for tool := range applied.Tools {
		entry := lease.Generation.NodeExecutable
		if tool == "python" {
			entry = lease.Generation.PythonExecutable
		}
		if tool == "java" || tool == "go" || tool == "rust" {
			if e := backend.CheckSDKEntries(lease.Generation.Directory, tool, platform); e != nil {
				return fail(e)
			}
			entry = backend.SDKEntry(lease.Generation.Directory, tool, platform)
		}
		if entry == "" {
			return fail(fmt.Errorf("ENV_NOT_READY: missing applied %s entry", tool))
		}
		var info os.FileInfo
		if tool == "node" {
			info, err = os.Lstat(entry)
		} else {
			info, err = os.Stat(entry)
		}
		if err != nil || !info.Mode().IsRegular() {
			return fail(fmt.Errorf("ENV_NOT_READY: applied %s entry is unavailable", tool))
		}
	}
	selected.Config = applied
	runtrace.Mark("validation_done")
	if err := prepareRunCompletion(ctx, s.workDirectory(root), store, selected); err != nil {
		return fail(err)
	}
	runtrace.Mark("receipt_prepared")
	return selected, nil
}
