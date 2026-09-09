package core

import (
	"context"
	"os"
	"path/filepath"

	"myenv/internal/state"
)

func cleanFailed(ctx context.Context, store *state.Store, anchor *os.Root, project, work string, dryRun bool, report func(CleanItem) error, result *CleanResult) error {
	after := ""
	for {
		var items []state.Preparation
		var err error
		if dryRun {
			items, err = store.PreviewPreparationCandidates(ctx, after, 128)
		} else {
			items, err = store.PreparationCandidates(ctx, after, 128)
		}
		if err != nil {
			return err
		}
		if len(items) == 0 {
			if dryRun {
				return previewPreparationChildren(ctx, store, anchor, project, work, report, result)
			}
			return nil
		}
		for _, g := range items {
			after = g.ID
			name, err := generationName(filepath.Join(work, "generations"), g.Generation)
			if err != nil {
				return err
			}
			item := CleanItem{Kind: "failed_preparation", ID: g.ID, Directory: g.Directory}
			if g.Status == "complete" {
				item.Kind = "orphaned_operation"
			}
			// Both locations are derived from the validated ID, never arbitrary
			// paths from operation metadata. Missing parents are already clean.
			for _, parent := range []string{"generations", "operations"} {
				bytes, err := failedFiles(ctx, anchor, project, work, parent, name, false)
				if err != nil {
					return err
				}
				item.Bytes += bytes
			}
			if !dryRun {
				marked, err := store.MarkPreparationDeleting(ctx, g.ID, g.Directory)
				if err != nil {
					return err
				}
				if !marked {
					continue
				}
				result.Changed = true
				for _, parent := range []string{"generations", "operations"} {
					if _, err = failedFiles(ctx, anchor, project, work, parent, name, true); err != nil {
						return err
					}
				}
				result.Removed++
				if err = store.FinishPreparationDeleting(ctx, g.ID, g.Directory); err != nil {
					return err
				}
				item.Removed = true
			}
			result.Candidates++
			result.Bytes += item.Bytes
			if report != nil {
				if err = report(item); err != nil {
					return err
				}
			}
		}
	}
}

func failedFiles(ctx context.Context, anchor *os.Root, project, work, parent, name string, remove bool) (int64, error) {
	rel, err := filepath.Rel(project, filepath.Join(work, parent))
	if err != nil {
		return 0, err
	}
	files, err := anchor.OpenRoot(rel)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	defer files.Close()
	if remove {
		return 0, removeGeneration(ctx, files, name)
	}
	return generationBytes(ctx, files, name)
}
