package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestUserStorageResolution(t *testing.T) {
	root := t.TempDir()
	data := filepath.Join(root, "data")
	cache := filepath.Join(root, "cache")
	paths, err := ResolveUserStorage(data, cache)
	if err != nil || paths.Data != filepath.Join(data, "myenv") || paths.Cache != filepath.Join(cache, "myenv") {
		t.Fatalf("paths %+v %v", paths, err)
	}
	for _, path := range []string{data, cache} {
		if _, err = os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("resolution created storage", path, err)
		}
	}
	for _, pair := range [][2]string{{"relative", cache}, {data, "relative"}} {
		if _, err = ResolveUserStorage(pair[0], pair[1]); err == nil {
			t.Fatal("relative storage accepted")
		}
	}
	home := func() (string, error) { return root, nil }
	for _, tt := range []struct{ system, key, value, want string }{
		{"windows", "LOCALAPPDATA", data, data},
		{"linux", "XDG_DATA_HOME", data, data},
		{"linux", "", "", filepath.Join(root, ".local", "share")},
		{"darwin", "", "", filepath.Join(root, "Library", "Application Support")},
	} {
		path, err := userDataRoot(tt.system, func(key string) string {
			if key == tt.key {
				return tt.value
			}
			return ""
		}, home)
		if err != nil || path != tt.want {
			t.Fatalf("%s: %s %v", tt.system, path, err)
		}
	}
	for _, system := range []string{"windows", "linux"} {
		if _, err = userDataRoot(system, func(string) string { return "relative" }, home); err == nil {
			t.Fatal("relative environment root accepted")
		}
	}
	if _, err = userDataRoot("windows", func(string) string { return "" }, home); err == nil {
		t.Fatal("missing local appdata accepted")
	}
	if _, err = userDataRoot("darwin", func(string) string { return "" }, func() (string, error) { return "", errors.New("no home") }); err == nil {
		t.Fatal("missing home ignored")
	}
}
