package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"myenv/internal/config"
)

// Only managed Windows Rust commands receive this documented default. The Job
// and descendant-completion contract remains unchanged.
func rustLinker(mode, executable, sdk, cwd string, argv []string, env map[string]string) ([]string, error) {
	if mode != "auto" && mode != "bundled" && mode != "system" {
		return nil, fmt.Errorf("rust-linker must be auto, bundled or system")
	}
	if mode == "system" {
		return argv, nil
	}
	name := strings.ToLower(filepath.Base(executable))
	if name != "rustc.exe" && name != "cargo.exe" {
		return argv, nil
	}
	if !strings.EqualFold(filepath.Clean(executable), filepath.Join(sdk, "bin", name)) {
		return argv, nil
	}
	explicit := false
	for i, a := range argv {
		if (a == "--target" && i+1 < len(argv) && argv[i+1] != "x86_64-pc-windows-msvc") || (strings.HasPrefix(a, "--target=") && a != "--target=x86_64-pc-windows-msvc") {
			return argv, nil
		}
		if name == "cargo.exe" && strings.Contains(a, "linker") {
			explicit = true
		}
		if strings.HasPrefix(a, "-Clinker=") || strings.HasPrefix(a, "-Clinker-flavor=") || (a == "-C" && i+1 < len(argv) && (strings.HasPrefix(argv[i+1], "linker=") || strings.HasPrefix(argv[i+1], "linker-flavor="))) {
			explicit = true
		}
	}
	if name == "cargo.exe" {
		explicit = explicit || cargoLinkerConfigured(cwd, env)
	}
	if explicit && mode == "auto" {
		return argv, nil
	}
	linker := filepath.Join(sdk, "lib", "rustlib", "x86_64-pc-windows-msvc", "bin", "rust-lld.exe")
	info, err := os.Lstat(linker)
	if err != nil || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("managed Rust bundled linker is unavailable; repair Rust or explicitly select --rust-linker=system")
	}
	if name == "cargo.exe" {
		env["CARGO_TARGET_X86_64_PC_WINDOWS_MSVC_LINKER"] = linker
		return argv, nil
	}
	at := len(argv)
	for i, a := range argv {
		if a == "--" {
			at = i
			break
		}
	}
	result := append([]string{}, argv[:at]...)
	result = append(result, "-C", "linker="+linker)
	result = append(result, argv[at:]...)
	return result, nil
}

func cargoLinkerConfigured(cwd string, env map[string]string) bool {
	for _, key := range []string{"CARGO_TARGET_X86_64_PC_WINDOWS_MSVC_LINKER", "RUSTC", "RUSTC_WRAPPER", "RUSTC_WORKSPACE_WRAPPER"} {
		if env[key] != "" || os.Getenv(key) != "" {
			return true
		}
	}
	for _, key := range []string{"RUSTFLAGS", "CARGO_ENCODED_RUSTFLAGS"} {
		if strings.Contains(env[key], "linker") || strings.Contains(os.Getenv(key), "linker") {
			return true
		}
	}
	// A project/user config is authoritative. Avoid overriding it without trying
	// to recreate Cargo's complete precedence/target-expression implementation.
	dirs := []string{}
	for d := cwd; ; d = filepath.Dir(d) {
		dirs = append(dirs, filepath.Join(d, ".cargo"))
		if filepath.Dir(d) == d {
			break
		}
	}
	home := env["CARGO_HOME"]
	if home == "" {
		home = os.Getenv("CARGO_HOME")
	}
	if home == "" {
		user, _ := os.UserHomeDir()
		home = filepath.Join(user, ".cargo")
	}
	dirs = append(dirs, home)
	for _, d := range dirs {
		for _, n := range []string{"config", "config.toml"} {
			b, e := config.ReadInput(filepath.Join(d, n))
			if e == nil && (strings.Contains(string(b), "linker") || strings.Contains(string(b), "rustflags") || strings.Contains(string(b), "rustc")) {
				return true
			}
			if e != nil && !os.IsNotExist(e) {
				return true
			}
		}
	}
	return false
}
