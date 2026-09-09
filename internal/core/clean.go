package core

import (
	"context"
	"os"
	"path/filepath"

	"myenv/internal/state"
)

type CleanItem struct {
	Kind      string `json:"kind"`
	ID        string `json:"id"`
	Directory string `json:"directory"`
	Bytes     int64  `json:"bytes"`
	Removed   bool   `json:"removed"`
}

type CleanResult struct {
	RecoverableLeases     int   `json:"recoverable_leases"`
	RecoveredLeases       int   `json:"recovered_leases"`
	RecoveredPreparations int64 `json:"recovered_preparations"`
	UnknownLeases         int   `json:"unknown_leases"`
	Changed               bool  `json:"changed"`
	Candidates            int   `json:"candidates"`
	Removed               int   `json:"removed"`
	Bytes                 int64 `json:"bytes"`
}

// Clean streams candidate results so a long history does not require an
// unbounded in-memory report. Dry-run never initializes or migrates state.
func (s *Service) Clean(ctx context.Context, directory string, dryRun bool, report func(CleanItem) error) (CleanResult, error) {
	var result CleanResult
	if err := ctx.Err(); err != nil {
		return result, err
	}
	project, err := s.resolveRoot(directory)
	if err != nil {
		return result, err
	}
	work := s.workDirectory(project)
	database := filepath.Join(work, "state.db")
	if _, err = os.Stat(database); os.IsNotExist(err) {
		return result, nil
	} else if err != nil {
		return result, err
	}
	if !dryRun {
		unlock, err := state.LockWorkspace(ctx, filepath.Join(work, "modify.lock"))
		if err != nil {
			return result, err
		}
		defer unlock()
	}
	var store *state.Store
	if dryRun {
		store, err = state.OpenReadOnly(ctx, database)
	} else {
		store, err = state.Open(ctx, database)
	}
	if err != nil {
		return result, err
	}
	defer store.Close()
	if !dryRun {
		result.RecoveredPreparations, err = recoverPreparationChildren(ctx, store, work, &result)
		if err != nil {
			return result, err
		}
		var completed int64
		completed, err = store.RecoverCompletedPreparations(ctx)
		result.RecoveredPreparations += completed
		result.Changed = result.Changed || result.RecoveredPreparations != 0
		if err != nil {
			return result, err
		}
	}
	if err = recoverCleanLeases(ctx, store, work, &result, dryRun); err != nil {
		return result, err
	}
	if err = cleanLeaseReceipts(ctx, store, work, dryRun, report, &result); err != nil {
		return result, err
	}
	anchor, err := os.OpenRoot(project)
	if err != nil {
		return result, err
	}
	defer anchor.Close()
	after := ""
	for {
		var items []state.Generation
		if dryRun {
			items, err = store.PreviewCandidates(ctx, after, 128)
		} else {
			items, err = store.CleanCandidates(ctx, after, 128)
		}
		if err != nil {
			return result, err
		}
		if len(items) == 0 {
			err = cleanFailed(ctx, store, anchor, project, work, dryRun, report, &result)
			return result, err
		}
		for _, g := range items {
			after = g.ID
			if dryRun {
				allowed, err := previewLeasesAllow(ctx, store, work, g.ID)
				if err != nil {
					return result, err
				}
				if !allowed {
					continue
				}
			}
			name, err := generationName(filepath.Join(work, "generations"), g)
			if err != nil {
				return result, err
			}
			bytes, err := failedFiles(ctx, anchor, project, work, "generations", name, false)
			if err != nil {
				return result, err
			}
			operationBytes, err := failedFiles(ctx, anchor, project, work, "operations", name, false)
			if err != nil {
				return result, err
			}
			bytes += operationBytes
			item := CleanItem{Kind: "generation", ID: g.ID, Directory: g.Directory, Bytes: bytes}
			if !dryRun {
				marked, err := store.MarkDeleting(ctx, g.ID, g.Directory)
				if err != nil {
					return result, err
				}
				if !marked {
					continue
				}
				// Reservation is a durable mutation; deletion may also partially
				// succeed before RemoveAll reports an error.
				result.Changed = true
				if _, err = failedFiles(ctx, anchor, project, work, "generations", name, true); err != nil {
					return result, err
				}
				if _, err = failedFiles(ctx, anchor, project, work, "operations", name, true); err != nil {
					return result, err
				}
				// File removal is already externally visible even if finalization fails.
				result.Removed++
				if err = store.FinishDeleting(ctx, g.ID, g.Directory); err != nil {
					return result, err
				}
				item.Removed = true
			}
			result.Candidates++
			result.Bytes += bytes
			if report != nil {
				if err = report(item); err != nil {
					return result, err
				}
			}
		}
	}
}
