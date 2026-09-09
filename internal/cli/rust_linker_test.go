package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRustLinkerPreservesExplicitAndExternal(t *testing.T) {
	root := t.TempDir()
	sdk := filepath.Join(root, "rust")
	exe := filepath.Join(sdk, "bin", "rustc.exe")
	linker := filepath.Join(sdk, "lib", "rustlib", "x86_64-pc-windows-msvc", "bin", "rust-lld.exe")
	os.MkdirAll(filepath.Dir(linker), 0700)
	os.WriteFile(linker, []byte("fixture"), 0600)
	args := []string{"main.rs"}
	got, e := rustLinker("auto", exe, sdk, root, args, map[string]string{})
	if e != nil || len(got) != 3 || got[2] != "linker="+linker {
		t.Fatalf("%v %v", got, e)
	}
	explicit := []string{"main.rs", "-C", "linker=custom.exe"}
	got, e = rustLinker("auto", exe, sdk, root, explicit, map[string]string{})
	if e != nil || !reflect.DeepEqual(got, explicit) {
		t.Fatal(got, e)
	}
	got, e = rustLinker("auto", filepath.Join(root, "external", "rustc.exe"), sdk, root, args, map[string]string{})
	if e != nil || !reflect.DeepEqual(got, args) {
		t.Fatal(got, e)
	}
	got, e = rustLinker("system", exe, sdk, root, args, map[string]string{})
	if e != nil || !reflect.DeepEqual(got, args) {
		t.Fatal(got, e)
	}
	cross := []string{"main.rs", "--target=x86_64-unknown-linux-gnu"}
	got, e = rustLinker("auto", exe, sdk, root, cross, map[string]string{})
	if e != nil || !reflect.DeepEqual(got, cross) {
		t.Fatal("cross target changed", got, e)
	}
	got, e = rustLinker("auto", exe, sdk, root, []string{"--", "main.rs"}, map[string]string{})
	if e != nil || len(got) != 4 || got[2] != "--" {
		t.Fatal("options appended after separator", got, e)
	}
}

func TestRustCargoConfigIsAuthoritative(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".cargo"), 0700)
	os.WriteFile(filepath.Join(root, ".cargo", "config.toml"), []byte("[target.x86_64-pc-windows-msvc]\nlinker='custom'\n"), 0600)
	if !cargoLinkerConfigured(root, nil) {
		t.Fatal("overrode project linker")
	}
}
