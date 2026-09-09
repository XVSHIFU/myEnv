package core

import (
	"path/filepath"
	"strings"

	"myenv/internal/config"
)

func (s *Service) commandHint(hint string) string {
	if !s.Profile {
		return hint
	}
	return strings.NewReplacer("myenv sync", "myenv sync --global", "myenv run", "myenv run --global", "myenv clean", "myenv clean --global", "use run --current", "use myenv run --global --current").Replace(hint)
}

func (s *Service) resolveRoot(directory string) (string, error) {
	if !s.Profile {
		return config.Discover(directory)
	}
	path, err := config.ProfilePath(s.UserConfigDirectory)
	if err != nil {
		return "", err
	}
	return filepath.Dir(path), nil
}

func (s *Service) declarationPath(root string) string {
	if s.Profile {
		return filepath.Join(root, "profile.yaml")
	}
	return filepath.Join(root, "myenv.yaml")
}

func (s *Service) loadDeclaration(root string) (*config.Config, error) {
	if s.Profile {
		return config.LoadProfile(s.declarationPath(root))
	}
	return config.Load(s.declarationPath(root))
}

func (s *Service) workDirectory(root string) string {
	if s.Profile {
		return filepath.Join(root, ".myenv-profile")
	}
	return filepath.Join(root, ".myenv")
}

func (s *Service) desiredLockPath(root string) string {
	if s.Profile {
		return filepath.Join(root, "profile.lock")
	}
	return filepath.Join(root, "myenv.lock")
}
