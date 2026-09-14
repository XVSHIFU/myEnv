package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func inventoryFixture(t *testing.T, root, name, value string) string {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0700); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestInventoryPathResolutionSeparatesOtherInstallations(t *testing.T) {
	root := t.TempDir()
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	first := inventoryFixture(t, root, "first/node"+suffix, "not executed")
	second := inventoryFixture(t, root, "second/node"+suffix, "not executed")
	t.Setenv("PATH", filepath.Dir(first)+string(os.PathListSeparator)+filepath.Dir(second))
	commands := resolveInventoryCommands()
	for _, command := range commands {
		if command.Tool == "node" {
			if command.Path != first || command.State != "unverified" {
				t.Fatalf("wrong PATH winner: %+v", command)
			}
		}
		if command.Tool == "python" && command.State != "not_found" {
			t.Fatalf("invented Python resolution: %+v", command)
		}
	}
	service := Service{UserConfigDirectory: filepath.Join(root, "user")}
	inventory, err := service.Inventory(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if inventory.Context.Kind != "application_process" {
		t.Fatal(inventory.Context)
	}
	for _, command := range inventory.Commands {
		if command.Tool == "node" && (command.Path != first || command.InstallationID == "") {
			t.Fatalf("missing resolution association: %+v", command)
		}
	}
	found := false
	for _, installation := range inventory.Installations {
		if installation.Path == second {
			found = true
		}
	}
	if !found {
		t.Fatal("non-winning installation hidden")
	}
}

func TestInventoryNodeProjectPackagesAreScopedAndReadOnly(t *testing.T) {
	root := t.TempDir()
	manifest := `{"packageManager":"pnpm@10.20.0","dependencies":{"@test/one":"^2","absent":"~3","../../escape":"*"},"devDependencies":{"test":"^1"}}`
	path := inventoryFixture(t, root, "package.json", manifest)
	inventoryFixture(t, root, "node_modules/@test/one/package.json", `{"name":"@test/one","version":"2.1.0"}`)
	inventoryFixture(t, root, "node_modules/test/package.json", `{"name":"test","version":"1.0.0"}`)
	group := inspectNodeProject(root)
	if group.Scope != "project" || group.Root != root || group.Manager != "pnpm@10.20.0" || len(group.Packages) != 4 {
		t.Fatalf("unexpected group: %+v", group)
	}
	for _, pkg := range group.Packages {
		switch pkg.Name {
		case "@test/one":
			if pkg.Version != "2.1.0" || pkg.State != "installed" || pkg.Requested != "^2" {
				t.Fatal(pkg)
			}
		case "absent":
			if pkg.State != "declared" || pkg.Version != "" {
				t.Fatal(pkg)
			}
		case "../../escape":
			if pkg.State != "invalid" || pkg.Path != "" {
				t.Fatal(pkg)
			}
		case "test":
			if pkg.Kind != "development" {
				t.Fatal(pkg)
			}
		}
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != manifest {
		t.Fatal("inspection mutated declaration", err)
	}
	if _, err := os.Stat(filepath.Join(root, "pnpm-lock.yaml")); !os.IsNotExist(err) {
		t.Fatal("inspection created lock")
	}
}

func TestInventoryGlobalPackagesDoNotMixProjects(t *testing.T) {
	root := t.TempDir()
	inventoryFixture(t, root, "global/node_modules/npm/package.json", `{"name":"npm","version":"11.0.0"}`)
	inventoryFixture(t, root, "global/node_modules/@test/tool/package.json", `{"name":"@test/tool","version":"2.0.0"}`)
	inventoryFixture(t, root, "project/node_modules/private/package.json", `{"name":"private","version":"3.0.0"}`)
	group := inspectNodeGlobal(filepath.Join(root, "global", "node_modules"))
	if group.Scope != "global_package_root" || len(group.Packages) != 2 {
		t.Fatal(group)
	}
	for _, pkg := range group.Packages {
		if pkg.Name == "private" {
			t.Fatal("project leaked into global group")
		}
	}
}

func TestInventoryPythonProjectDoesNotAssumeSystemInterpreter(t *testing.T) {
	root := t.TempDir()
	inventoryFixture(t, root, "pyproject.toml", "[project]\nname='sample'\ndependencies=['requests>=2', 'my_pkg[extra] ; python_version >= \"3.11\"']\n")
	group := inspectPythonProject(context.Background(), root, false)
	if group.State != "declared" || group.Interpreter != "" || len(group.Packages) != 2 || group.Packages[1].Name != "my_pkg" {
		t.Fatalf("unexpected declaration: %+v", group)
	}
	if !strings.Contains(group.Problem, "not assumed") {
		t.Fatal("missing scope explanation")
	}
	entry := "venv/bin/python"
	if runtime.GOOS == "windows" {
		entry = "venv/Scripts/python.exe"
	}
	interpreter := inventoryFixture(t, root, entry, "never execute shallow inspection")
	inventoryFixture(t, root, "venv/pyvenv.cfg", "home = fixture")
	group = inspectPythonProject(context.Background(), root, false)
	if group.Interpreter != interpreter {
		t.Fatalf("selected another interpreter: %+v", group)
	}
	managed := inventoryFixture(t, root, "managed/python"+filepath.Ext(interpreter), "not executed")
	group = inspectPythonProject(context.Background(), root, false, managed)
	if group.Interpreter != managed {
		t.Fatalf("current managed project interpreter was ignored: %+v", group)
	}
}

func TestInventoryMetadataLimitsAndRequirements(t *testing.T) {
	root := t.TempDir()
	path := inventoryFixture(t, root, "too-large.json", strings.Repeat(" ", 1024*1024+1))
	if _, err := readInventoryFile(path); err == nil {
		t.Fatal("unbounded metadata accepted")
	}
	inventoryFixture(t, root, "requirements.txt", "# comment\nrequests>=2\n-r ../outside.txt\n--index-url https://example.invalid\n")
	group := inspectPythonProject(context.Background(), root, false)
	if len(group.Packages) != 1 || group.Packages[0].Name != "requests" || group.Coverage == "" {
		t.Fatalf("included or executed non-direct requirements: %+v", group)
	}
}

func TestInventoryProbeBoundedAndCancelable(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("MYENV_INVENTORY_PROBE_TEST", "1")
	output, err := probeInventoryCommand(context.Background(), executable, []string{"-test.run=^TestInventoryProbeHelper$"}, "", 16)
	if err == nil || !strings.Contains(err.Error(), "exceeded") || len(output) > 16 {
		t.Fatalf("output not bounded: %q %v", output, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := probeInventoryCommand(ctx, executable, []string{"-test.run=^TestInventoryProbeHelper$"}, "", 8192); err != context.Canceled {
		t.Fatalf("cancellation lost: %v", err)
	}
}

func TestInventoryProbeHelper(t *testing.T) {
	if os.Getenv("MYENV_INVENTORY_PROBE_TEST") != "1" {
		return
	}
	fmt.Print(strings.Repeat("v24.18.0\n", 64))
	os.Exit(0)
}

func TestInventoryPythonMetadataDoesNotRunSiteHooks(t *testing.T) {
	python := os.Getenv("MYENV_TEST_INVENTORY_PYTHON")
	if python == "" {
		t.Skip("set MYENV_TEST_INVENTORY_PYTHON to an existing Python interpreter; no downloads")
	}
	root := t.TempDir()
	venv := filepath.Join(root, "venv")
	if output, err := probeInventoryCommand(context.Background(), python, []string{"-I", "-m", "venv", "--without-pip", venv}, root, 8192); err != nil {
		t.Fatalf("isolated fixture creation: %v %s", err, output)
	}
	entry, site := filepath.Join(venv, "bin", "python"), ""
	if runtime.GOOS == "windows" {
		entry, site = filepath.Join(venv, "Scripts", "python.exe"), filepath.Join(venv, "Lib", "site-packages")
	} else {
		entries, _, err := inventoryDirectoryEntries(filepath.Join(venv, "lib"), 8)
		if err != nil {
			t.Fatal(err)
		}
		for _, child := range entries {
			if strings.HasPrefix(child.Name(), "python") {
				site = filepath.Join(venv, "lib", child.Name(), "site-packages")
				break
			}
		}
	}
	if site == "" {
		t.Fatal("fixture site-packages not found")
	}
	sentinel := filepath.Join(root, "site-hook-ran")
	inventoryFixture(t, site, "sitecustomize.py", "from pathlib import Path\nPath("+fmt.Sprintf("%q", sentinel)+").write_text('unexpected')\n")
	inventoryFixture(t, site, "test_inventory-1.2.3.dist-info/METADATA", "Metadata-Version: 2.1\nName: test-inventory\nVersion: 1.2.3\n")
	group := inspectPythonPackages(context.Background(), entry, "project", root)
	if group.State != "available" || !sameInventoryPath(group.Root, venv) {
		t.Fatalf("wrong interpreter metadata: %+v", group)
	}
	found := false
	for _, pkg := range group.Packages {
		if pkg.Name == "test-inventory" && pkg.Version == "1.2.3" {
			found = true
		}
	}
	if !found {
		t.Fatalf("fixture distribution omitted: %+v", group)
	}
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Fatal("inspection ran site customization")
	}
}
