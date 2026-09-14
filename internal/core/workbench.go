package core

import (
	"context"
	"myenv/internal/config"
	"myenv/internal/runner"
	"os"
	"path/filepath"
)

// Workbench is a read-only view of declaration and applied lock, not a second resolver.
type Workbench struct {
	Status    *Status                       `json:"status,omitempty"`
	Empty     bool                          `json:"empty"`
	Directory string                        `json:"directory"`
	Platform  string                        `json:"platform"`
	Desired   map[string]string             `json:"desired"`
	Applied   map[string]config.RuntimeLock `json:"applied"`
	Locked    map[string]config.RuntimeLock `json:"locked"`
	Path      string                        `json:"path"`
	Digest    string                        `json:"digest"`
}

func (s *Service) Workbench(ctx context.Context, directory string) (Workbench, error) {
	root, err := filepath.Abs(directory)
	if s.Profile {
		root, err = s.resolveRoot(directory)
	}
	if err != nil {
		return Workbench{}, err
	}
	platform, err := runner.Platform()
	if err != nil {
		return Workbench{}, err
	}
	w := Workbench{Directory: root, Platform: platform, Desired: map[string]string{}, Applied: map[string]config.RuntimeLock{}, Locked: map[string]config.RuntimeLock{}}
	if _, err := os.Stat(s.declarationPath(root)); os.IsNotExist(err) {
		w.Empty = true
		return w, nil
	} else if err != nil {
		return w, err
	}
	status, err := s.Status(ctx, root)
	w.Status = status
	if err != nil {
		return w, err
	}
	if status.Tools != nil {
		w.Desired = status.Tools
	}
	cfg, err := s.loadDeclaration(root)
	if err != nil {
		return w, err
	}
	w.Digest, err = config.Digest(cfg)
	if err != nil {
		return w, err
	}
	lock, err := config.ReadLock(s.desiredLockPath(root))
	if err == nil {
		if tools := lock.Platforms[platform].Tools; tools != nil {
			w.Locked = tools
		}
	} else if !os.IsNotExist(err) {
		return w, err
	}
	if status.Generation != nil {
		w.Path = status.Generation.Directory
		lock, err := config.ReadLock(filepath.Join(w.Path, "myenv.lock"))
		if err != nil {
			return w, err
		}
		if tools := lock.Platforms[platform].Tools; tools != nil {
			w.Applied = tools
		}
	}
	return w, nil
}
