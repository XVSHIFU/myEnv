package core

import (
	"context"
	"os"
	"path/filepath"

	"myenv/internal/state"
)

func previewPreparationChildren(ctx context.Context, store *state.Store, anchor *os.Root, project, work string, report func(CleanItem) error, result *CleanResult) error {
	after := ""
	for {
		owners, err := store.PreparingChildOperations(ctx, after)
		if err != nil {
			return err
		}
		if len(owners) == 0 {
			return nil
		}
		for _, owner := range owners {
			after = owner.ID
			if err := ctx.Err(); err != nil {
				return err
			}
			if !managedGenerationID(owner.ID) || owner.PID <= 1 || !validLinuxOwnerIdentity(owner.Identity) {
				continue
			}
			gone, err := state.SupervisorExited(owner.PID, owner.Identity)
			if err != nil || !gone {
				continue
			}
			complete, err := linuxPreparationComplete(ctx, store, work, owner, nil)
			if err != nil {
				return err
			}
			if !complete {
				continue
			}
			name, err := generationName(filepath.Join(work, "generations"), state.Generation{ID: owner.ID, Directory: owner.Directory})
			if err != nil {
				return err
			}
			item := CleanItem{Kind: "failed_preparation", ID: owner.ID, Directory: owner.Directory}
			for _, parent := range []string{"generations", "operations"} {
				n, err := failedFiles(ctx, anchor, project, work, parent, name, false)
				if err != nil {
					return err
				}
				item.Bytes += n
			}
			result.Candidates++
			result.Bytes += item.Bytes
			if report != nil {
				if err := report(item); err != nil {
					return err
				}
			}
		}
	}
}
