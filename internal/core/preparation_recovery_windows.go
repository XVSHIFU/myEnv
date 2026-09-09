package core

import (
	"context"
	"os"
	"path/filepath"

	"myenv/internal/state"
)

func windowsPreparationChildren(ctx context.Context, store *state.Store, owner state.PreparationOwner, result *CleanResult) (bool, error) {
	after := ""
	all, seen := true, false
	for {
		children, err := store.PreparationChildren(ctx, owner.ID, after)
		if err != nil {
			return false, err
		}
		if len(children) == 0 {
			if !seen && owner.Tracked {
				// There were no launches. Reuse the strict birth/session parser
				// and owner-exit check; an unexpected matching Job also protects.
				gone, err := windowsLeaseReclaimable(state.LeaseRecord{ID: owner.ID, PID: owner.PID, Identity: owner.Identity})
				return gone && err == nil, nil
			}
			return all && seen, nil
		}
		for _, child := range children {
			after = child.ID
			seen = true
			if err := ctx.Err(); err != nil {
				return false, err
			}
			gone, err := windowsLeaseReclaimable(state.LeaseRecord{ID: child.ID, PID: owner.PID, Identity: owner.Identity})
			if err != nil || !gone || child.Token != "" {
				all = false
				continue
			}
			if child.Completed || result == nil {
				continue
			}
			changed, err := store.RecoverPreparationJob(ctx, owner, child)
			result.Changed = result.Changed || changed
			if err != nil {
				return false, err
			}
			if !changed {
				all = false
			}
		}
	}
}

func recoverPreparationChildren(ctx context.Context, store *state.Store, _ string, result *CleanResult) (int64, error) {
	after := ""
	var recovered int64
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
			complete, err := windowsPreparationChildren(ctx, store, owner, result)
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
			complete, err := windowsPreparationChildren(ctx, store, owner, nil)
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
