package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"myenv/internal/config"
	"myenv/internal/state"
)

type UseResult struct {
	DeclarationChanged bool `json:"declaration_changed"`
	SyncResult
}

type UseRequest struct {
	Directory, Selection string
	AllowBuild           bool
	// Preview persists the explicitly requested declaration but does not apply a generation.
	Preview      bool
	ConfirmBuild func(string) (bool, error)
	Progress     func(SyncPhase)
}

func (s *Service) Use(ctx context.Context, directory, selection string) (UseResult, error) {
	return s.UseWithRequest(ctx, UseRequest{Directory: directory, Selection: selection})
}

func (s *Service) UseWithRequest(ctx context.Context, request UseRequest) (UseResult, error) {
	var result UseResult
	if err := ctx.Err(); err != nil {
		return result, err
	}
	tool, version, ok := strings.Cut(request.Selection, "@")
	if !ok || !config.SupportedTool(tool) || version == "" {
		return result, fmt.Errorf("use requires a supported tool@<version>")
	}
	if _, err := config.ParseConstraint(tool, version); err != nil {
		return result, err
	}
	root, err := s.resolveRoot(request.Directory)
	if err != nil {
		return result, err
	}
	_, err = s.loadDeclaration(root)
	if err != nil && !(s.Profile && os.IsNotExist(err)) {
		return result, err
	}
	work := s.workDirectory(root)
	if err = ctx.Err(); err != nil {
		return result, err
	}
	if err = os.MkdirAll(work, 0700); err != nil {
		return result, err
	}
	unlock, err := state.LockWorkspace(ctx, filepath.Join(work, "modify.lock"))
	if err != nil {
		return result, err
	}
	if err = ctx.Err(); err != nil {
		unlock()
		return result, err
	}
	if s.Profile {
		result.DeclarationChanged, err = config.SetProfileTool(s.declarationPath(root), tool, version)
	} else {
		result.DeclarationChanged, err = config.SetTool(s.declarationPath(root), tool, version)
	}
	var expectedDigest string
	if err == nil {
		var edited *config.Config
		edited, err = s.loadDeclaration(root)
		if err == nil && edited.Tools[tool] != version {
			err = fmt.Errorf("INPUT_CHANGED: requested declaration was replaced after use edited it")
		}
		if err == nil {
			expectedDigest, err = config.Digest(edited)
		}
	}
	unlock()
	if err != nil {
		return result, err
	}
	result.SyncResult, err = s.Sync(ctx, SyncRequest{Directory: root, ExpectedDigest: expectedDigest, DryRun: request.Preview, AllowBuild: request.AllowBuild, ConfirmBuild: request.ConfirmBuild, Progress: request.Progress})
	if err != nil {
		return result, fmt.Errorf("%w; use declaration_changed=%t lock_changed=%t native_lock_changed=%t; active environment unchanged; %s", err, result.DeclarationChanged, result.LockChanged, result.NativeLockChanged, s.commandHint("retry myenv sync or use myenv run --current"))
	}
	return result, nil
}
