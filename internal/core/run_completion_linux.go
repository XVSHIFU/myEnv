package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"myenv/internal/runner"
	"myenv/internal/state"
)

func init() {
	leaseReclaimable = linuxLeaseReclaimable
	removeRecoveredReceipt = func(work string, r state.LeaseRecord) error {
		if !managedGenerationID(r.ID) {
			return fmt.Errorf("invalid lease receipt identity")
		}
		root, err := os.OpenRoot(work)
		if err != nil {
			return err
		}
		defer root.Close()
		err = root.Remove(leaseReceiptName(r.ID))
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
}

func leaseReceiptName(id string) string { return filepath.Join("lease-receipts", id+".complete") }

func acquireRunLease(ctx context.Context, store *state.Store) (*state.Lease, error) {
	return store.AcquireActiveWithCompletion(ctx)
}

func openRunStore(ctx context.Context, path string) (*state.Store, error) {
	return state.OpenForRun(ctx, path)
}

func prepareRunCompletion(ctx context.Context, work string, store *state.Store, selected *RunEnvironment) error {
	if !managedGenerationID(selected.TreeID) {
		return fmt.Errorf("invalid lease receipt identity")
	}
	root, err := os.OpenRoot(work)
	if err != nil {
		return err
	}
	if err = root.Mkdir("lease-receipts", 0700); err != nil && !os.IsExist(err) {
		root.Close()
		return err
	}
	name := leaseReceiptName(selected.TreeID)
	file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		root.Close()
		return err
	}
	token := selected.completionToken
	if token == "" {
		// Keep the explicitly acquired legacy lease path for recovery callers.
		bytes := make([]byte, 32)
		if _, err = rand.Read(bytes); err == nil {
			token = hex.EncodeToString(bytes)
			err = store.BindLeaseCompletion(ctx, selected.TreeID, token)
		}
	}
	if err != nil {
		file.Close()
		root.Remove(name)
		root.Close()
		return err
	}
	selected.Completion = &runner.TreeCompletion{File: file, Token: token}
	release, retain := selected.Release, selected.retainLease
	selected.Release = func() error {
		releaseErr := release()
		closeErr := file.Close()
		var removeErr error
		if releaseErr == nil {
			removeErr = root.Remove(name)
		}
		return errors.Join(releaseErr, closeErr, removeErr, root.Close())
	}
	selected.retainLease = func() error { return errors.Join(retain(), file.Close(), root.Close()) }
	return nil
}

func linuxLeaseReclaimable(work string, r state.LeaseRecord) (bool, error) {
	if !managedGenerationID(r.ID) || r.PID <= 1 {
		return false, nil
	}
	if !validLinuxOwnerIdentity(r.Identity) {
		return false, nil
	}
	token, err := hex.DecodeString(r.CompletionToken)
	if err != nil || len(token) != 32 || hex.EncodeToString(token) != r.CompletionToken {
		return false, nil
	}
	exited, err := state.SupervisorExited(r.PID, r.Identity)
	if err != nil || !exited {
		return false, err
	}
	root, err := os.OpenRoot(work)
	if err != nil {
		return false, err
	}
	defer root.Close()
	// NOFOLLOW rejects a replaced leaf; NONBLOCK prevents a FIFO replacement
	// from hanging clean before the regular-file check can run.
	file, err := openCompletionReceipt(root, leaseReceiptName(r.ID))
	if err != nil {
		return false, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("lease receipt is not a regular file")
	}
	expected := "myenv-tree-complete-v1:" + r.CompletionToken + "\n"
	data, err := io.ReadAll(io.LimitReader(file, int64(len(expected)+1)))
	if err != nil {
		return false, err
	}
	return string(data) == expected, nil
}
