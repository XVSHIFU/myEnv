package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// UserStorage holds application directories, separate from project generations
// and the user configuration/profile namespace. Resolving it creates no files.
type UserStorage struct {
	Data  string
	Cache string
}

// ResolveUserStorage accepts OS-root overrides for isolated integrations. Each
// returned path includes the myenv application directory, like ProfilePath.
func ResolveUserStorage(dataRoot, cacheRoot string) (UserStorage, error) {
	var err error
	if dataRoot == "" {
		dataRoot, err = userDataRoot(runtime.GOOS, os.Getenv, os.UserHomeDir)
		if err != nil {
			return UserStorage{}, err
		}
	}
	if cacheRoot == "" {
		cacheRoot, err = os.UserCacheDir()
		if err != nil {
			return UserStorage{}, err
		}
	}
	if !filepath.IsAbs(dataRoot) || !filepath.IsAbs(cacheRoot) {
		return UserStorage{}, fmt.Errorf("user data and cache directories must be absolute")
	}
	return UserStorage{Data: filepath.Join(dataRoot, "myenv"), Cache: filepath.Join(cacheRoot, "myenv")}, nil
}

func userDataRoot(system string, env func(string) string, home func() (string, error)) (string, error) {
	var path string
	switch system {
	case "windows":
		path = env("LOCALAPPDATA")
		if path == "" {
			return "", fmt.Errorf("LOCALAPPDATA is required for user runtime storage")
		}
	case "linux":
		path = env("XDG_DATA_HOME")
		if path == "" {
			base, err := home()
			if err != nil {
				return "", err
			}
			path = filepath.Join(base, ".local", "share")
		}
	case "darwin":
		base, err := home()
		if err != nil {
			return "", err
		}
		path = filepath.Join(base, "Library", "Application Support")
	default:
		return "", fmt.Errorf("unsupported user storage platform %q", system)
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("user data directory must be absolute")
	}
	return path, nil
}
