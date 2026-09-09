package core

import (
	"context"
	"myenv/internal/state"
)

// Platforms install this only when both supervisor and process-tree evidence
// can be checked. Unsupported platforms retain leases without guessing death.
var leaseReclaimable func(string, state.LeaseRecord) (bool, error)
var removeRecoveredReceipt func(string, state.LeaseRecord) error

func recoverCleanLeases(ctx context.Context, s *state.Store, work string, result *CleanResult, dryRun bool) error {
	if leaseReclaimable == nil {
		return nil
	}
	after := ""
	for {
		rows, err := s.LeaseRecords(ctx, after, 128)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		for _, r := range rows {
			after = r.ID
			if err := ctx.Err(); err != nil {
				return err
			}
			gone, err := leaseReclaimable(work, r)
			if err != nil {
				result.UnknownLeases++
				continue
			}
			if !gone {
				continue
			}
			result.RecoverableLeases++
			if dryRun {
				continue
			}
			removed, err := s.DeleteObservedLease(ctx, r)
			if err != nil {
				return err
			}
			if removed {
				result.RecoveredLeases++
				result.Changed = true
				if removeRecoveredReceipt != nil {
					if err := removeRecoveredReceipt(work, r); err != nil {
						return err
					}
				}
			}
		}
	}
}

func previewLeasesAllow(ctx context.Context, s *state.Store, work, generation string) (bool, error) {
	after := ""
	for {
		rows, err := s.GenerationLeaseRecords(ctx, generation, after, 128)
		if err != nil {
			return false, err
		}
		if len(rows) == 0 {
			return true, nil
		}
		for _, r := range rows {
			after = r.ID
			if leaseReclaimable == nil {
				return false, nil
			}
			gone, err := leaseReclaimable(work, r)
			if err != nil || !gone {
				return false, nil
			}
		}
	}
}
