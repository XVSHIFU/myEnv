package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"

	"myenv/internal/runner"
	"myenv/internal/state"
)

func prepareChildCompletion(ctx context.Context, directory string, store *state.Store, operation, child string, process *runner.Process) (func(bool) error, error) {
	if !managedGenerationID(child) {
		return nil, fmt.Errorf("invalid preparation child identity")
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	name := child + ".complete"
	file, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		root.Close()
		return nil, err
	}
	var random [32]byte
	_, err = rand.Read(random[:])
	token := hex.EncodeToString(random[:])
	if err == nil {
		err = store.BindOperationChildCompletion(ctx, operation, child, token)
	}
	if err != nil {
		return nil, errors.Join(err, file.Close(), root.Remove(name), root.Close())
	}
	process.Completion = &runner.TreeCompletion{File: file, Token: token}
	return func(complete bool) error {
		closeErr := file.Close()
		var removeErr error
		if complete {
			removeErr = root.Remove(name)
		}
		return errors.Join(closeErr, removeErr, root.Close())
	}, nil
}
