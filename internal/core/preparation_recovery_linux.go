package core

import (
	"context"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"

	"myenv/internal/state"
)

func recoverPreparationChildren(ctx context.Context, store *state.Store, work string, result *CleanResult) (int64, error) {
	var recovered int64
	after := ""
	for {
		owners, err := store.PreparingChildOperations(ctx, after)
		if err != nil {
			return recovered, err
		}
		if len(owners) == 0 {
			return recovered, nil
		}
		for _, owner := range owners {
			after = owner.ID
			if err := ctx.Err(); err != nil {
				return recovered, err
			}
			if !managedGenerationID(owner.ID) || owner.PID <= 1 || !validLinuxOwnerIdentity(owner.Identity) {
				continue
			}
			gone, err := state.SupervisorExited(owner.PID, owner.Identity)
			if err != nil || !gone {
				continue
			}
			complete, err := linuxPreparationComplete(ctx, store, work, owner, result)
			if err != nil {
				return recovered, err
			}
			if !complete {
				continue
			}
			ok, err := store.RecoverPreparationOwner(ctx, owner)
			if err != nil {
				return recovered, err
			}
			if ok {
				recovered++
				result.Changed = true
			}
		}
	}
}

// The caller has verified owner death. No directory is needed when the
// complete-registration protocol proves that no child was ever launched.
func linuxPreparationComplete(ctx context.Context, store *state.Store, work string, owner state.PreparationOwner, result *CleanResult) (bool, error) {
	if owner.Tracked {
		children, err := store.PreparationChildren(ctx, owner.ID, "")
		if err != nil {
			return false, err
		}
		if len(children) == 0 {
			return true, nil
		}
	}
	root, err := os.OpenRoot(filepath.Join(work, "operations", owner.ID))
	if err != nil {
		return false, nil
	}
	complete, err := recoverPreparationReceipts(ctx, store, root, owner, result)
	return complete, errors.Join(err, root.Close())
}

// A nil result performs a read-only check with the identical receipt rules.
func recoverPreparationReceipts(ctx context.Context, store *state.Store, root *os.Root, owner state.PreparationOwner, result *CleanResult) (bool, error) {
	after := ""
	all, seen := true, false
	for {
		children, err := store.PreparationChildren(ctx, owner.ID, after)
		if err != nil {
			return false, err
		}
		if len(children) == 0 {
			return all && seen, nil
		}
		for _, child := range children {
			seen = true
			after = child.ID
			if err := ctx.Err(); err != nil {
				return false, err
			}
			if child.Completed {
				continue
			}
			if !managedGenerationID(child.ID) {
				all = false
				continue
			}
			token, err := hex.DecodeString(child.Token)
			if err != nil || len(token) != 32 || hex.EncodeToString(token) != child.Token {
				all = false
				continue
			}
			file, err := openCompletionReceipt(root, child.ID+".complete")
			if err != nil {
				all = false
				continue
			}
			info, err := file.Stat()
			if err != nil || !info.Mode().IsRegular() {
				all = false
				file.Close()
				continue
			}
			expected := "myenv-tree-complete-v1:" + child.Token + "\n"
			data, readErr := io.ReadAll(io.LimitReader(file, int64(len(expected)+1)))
			closeErr := file.Close()
			if readErr != nil || closeErr != nil || string(data) != expected {
				all = false
				continue
			}
			if result == nil {
				continue
			}
			changed, err := store.RecoverPreparationChild(ctx, owner, child)
			result.Changed = result.Changed || changed
			if err != nil {
				return false, err
			}
		}
	}
}
