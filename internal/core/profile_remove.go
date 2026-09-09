package core

import (
	"context"
	"fmt"
	"myenv/internal/config"
	"myenv/internal/state"
	"os"
	"path/filepath"
)

// RemoveDefault publishes a profile without the selected runtime. Old generations
// remain subject to the existing previous-generation and live-lease protections.
func (s *Service) RemoveDefault(ctx context.Context, tool string) (UseResult, error) {
	var result UseResult
	if !s.Profile || !config.SupportedTool(tool) {
		return result, fmt.Errorf("remove requires a supported tool in the current user profile")
	}
	root, err := s.resolveRoot("")
	if err != nil {
		return result, err
	}
	if _, err = s.loadDeclaration(root); err != nil {
		return result, err
	}
	if err = ctx.Err(); err != nil {
		return result, err
	}
	if err = os.MkdirAll(s.workDirectory(root), 0700); err != nil {
		return result, err
	}
	unlock, err := state.LockWorkspace(ctx, filepath.Join(s.workDirectory(root), "modify.lock"))
	if err != nil {
		return result, err
	}
	changed, err := config.RemoveProfileTool(s.declarationPath(root), tool)
	var digest string
	if err == nil {
		c, e := s.loadDeclaration(root)
		err = e
		if err == nil {
			digest, err = config.Digest(c)
		}
	}
	unlock()
	result.DeclarationChanged = changed
	if err != nil {
		return result, err
	}
	result.SyncResult, err = s.Sync(ctx, SyncRequest{ExpectedDigest: digest})
	return result, err
}
