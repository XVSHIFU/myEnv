package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"myenv/internal/backend"
	"myenv/internal/runner"
)

type packageTestRegistry struct{ latest string }

func (r packageTestRegistry) Package(_ context.Context, ecosystem, name string) (backend.RegistryPackage, error) {
	return backend.RegistryPackage{State: "found", Name: name, Ecosystem: ecosystem, Latest: r.latest}, nil
}
func (r packageTestRegistry) Version(_ context.Context, ecosystem, name, version string) (backend.RegistryPackageVersion, error) {
	return backend.RegistryPackageVersion{State: "found", Name: name, Ecosystem: ecosystem, Version: version, Available: true, URL: "https://registry.npmjs.org/" + name}, nil
}

func TestPackageManageRejectInstallSpecifications(t *testing.T) {
	for _, name := range []string{"../outside", "@scope/../../outside", "--prefix", "foo;whoami", "https://example.com/a", "pkg@1", "foo bar", "a\\b", "@scope/"} {
		if validManagePackage("node", name) {
			t.Fatalf("accepted package expression %q", name)
		}
	}
	for _, version := range []string{"latest", "^1", "1;whoami", "file:../x", "https://example.com/pkg", "1 --prefix X"} {
		if validManageVersion(version) {
			t.Fatalf("accepted expression %q", version)
		}
	}
	for _, pair := range [][2]string{{"node", "@openai/codex"}, {"node", "is-number"}, {"python", "typing_extensions"}} {
		if !validManagePackage(pair[0], pair[1]) {
			t.Fatal(pair)
		}
	}
	if !validManageVersion("1!2.0") {
		t.Fatal("PEP440 epoch rejected")
	}
}

func TestPackageManageCommandScopeAndPolicies(t *testing.T) {
	root := t.TempDir()
	node := filepath.Join(root, "node.exe")
	manager := packageManager{Name: "npm", Executable: node, Script: filepath.Join(root, "npm-cli.js")}
	target := PackageTarget{Ecosystem: "node", Scope: "project", Root: root, Directory: root}
	cmd, err := manager.command(target, "install", PackageChange{Name: "@scope/pkg", Version: "1.2.3", Kind: "development"})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(cmd.Args, " ")
	for _, required := range []string{"--ignore-scripts", "--save-exact", "--save-dev", "--workspaces=false", "--@scope:registry=https://registry.npmjs.org", "@scope/pkg@1.2.3"} {
		if !strings.Contains(joined, required) {
			t.Fatal(joined)
		}
	}
	if cmd.Directory != root || cmd.Executable != node {
		t.Fatal(cmd)
	}
	manager.Name = "pip"
	manager.Script = ""
	target = PackageTarget{Ecosystem: "python", Scope: "interpreter", Root: root, Interpreter: node}
	cmd, err = manager.command(target, "install", PackageChange{Name: "idna", Version: "3.10"})
	if err != nil {
		t.Fatal(err)
	}
	joined = strings.Join(cmd.Args, " ")
	for _, required := range []string{"-I -m pip", "--isolated", "--no-input", "--only-binary=:all:", "--index-url https://pypi.org/simple", "idna==3.10"} {
		if !strings.Contains(joined, required) {
			t.Fatal(joined)
		}
	}
	t.Setenv("PIP_TARGET", filepath.Join(root, "wrong"))
	t.Setenv("NODE_OPTIONS", "--require=wrong.js")
	t.Setenv("NPM_CONFIG_PREFIX", filepath.Join(root, "wrong"))
	env, err := packageEnvironment(target, manager)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range env {
		if strings.HasPrefix(entry, "PIP_TARGET=") || strings.HasPrefix(entry, "NODE_OPTIONS=") || strings.HasPrefix(entry, "NPM_CONFIG_PREFIX=") {
			t.Fatalf("scope override survived %q", entry)
		}
	}
}

func TestPackageManageLatestPreservesExactNativeLocks(t *testing.T) {
	for _, manager := range []string{"npm", "pnpm"} {
		t.Run(manager, func(t *testing.T) {
			root := t.TempDir()
			inventoryFixture(t, root, "package.json", `{"name":"fixture","dependencies":{"is-number":"7.0.0"},"private":true}`)
			if manager == "npm" {
				inventoryFixture(t, root, "package-lock.json", `{"lockfileVersion":3,"packages":{"":{"dependencies":{"is-number":"7.0.0"}},"node_modules/is-number":{"version":"7.0.0","integrity":"sha512-kept"}}}`)
			} else {
				inventoryFixture(t, root, "pnpm-lock.yaml", "lockfileVersion: '9.0'\nimporters:\n  .:\n    dependencies:\n      is-number:\n        specifier: 7.0.0\n        version: 7.0.0\npackages:\n  is-number@7.0.0:\n    resolution: {integrity: sha512-kept}\n")
			}
			if err := setNodeLatestExpectation(PackageTarget{Directory: root, Manager: manager}, PackageChange{Name: "is-number", Version: "7.0.0", Kind: "dependency"}); err != nil {
				t.Fatal(err)
			}
			pkg, err := readNodeManifest(filepath.Join(root, "package.json"))
			if err != nil || pkg.Dependencies["is-number"] != "latest" {
				t.Fatal(pkg, err)
			}
			lock := "package-lock.json"
			if manager == "pnpm" {
				lock = "pnpm-lock.yaml"
			}
			data, _ := os.ReadFile(filepath.Join(root, lock))
			if !strings.Contains(string(data), "latest") || !strings.Contains(string(data), "7.0.0") || !strings.Contains(string(data), "sha512-kept") {
				t.Fatal(string(data))
			}
		})
	}
}

func TestPackageManageLatestUsesActiveShrinkwrap(t *testing.T) {
	root := t.TempDir()
	inventoryFixture(t, root, "package.json", `{"dependencies":{"is-number":"7.0.0"}}`)
	inventoryFixture(t, root, "npm-shrinkwrap.json", `{"lockfileVersion":3,"packages":{"":{"dependencies":{"is-number":"7.0.0"}},"node_modules/is-number":{"version":"7.0.0"}}}`)
	ignored := `{"lockfileVersion":3,"packages":{"":{"dependencies":{"is-number":"6.0.0"}}}}`
	inventoryFixture(t, root, "package-lock.json", ignored)
	if err := setNodeLatestExpectation(PackageTarget{Directory: root, Manager: "npm"}, PackageChange{Name: "is-number", Version: "7.0.0", Kind: "dependency"}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "package-lock.json"))
	if string(data) != ignored {
		t.Fatal("ignored package-lock changed")
	}
	data, _ = os.ReadFile(filepath.Join(root, "npm-shrinkwrap.json"))
	if !strings.Contains(string(data), "latest") {
		t.Fatal(string(data))
	}
}

func packageFixtureManager(t *testing.T) (PackageTarget, *Service) {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("existing Node needed for isolated process fixture")
	}
	root := t.TempDir()
	project := filepath.Join(root, "project")
	inventoryFixture(t, project, "package.json", `{"name":"fixture","version":"1.0.0","dependencies":{}}`)
	manifest := inventoryFixture(t, root, "manager/node_modules/npm/package.json", `{"name":"npm","version":"11.0.0"}`)
	script := `const fs=require('fs'),path=require('path');const a=process.argv.slice(2);if(a.includes('--version')){console.log('11.0.0');process.exit(0)}
const root=process.cwd();if(a[0]==='root'){console.log(path.join(root,process.platform==='win32'?'node_modules':'lib/node_modules'));process.exit(0)}
const spec=a[a.length-1], at=spec.lastIndexOf('@'), name=a[0]==='uninstall'?spec:spec.slice(0,at), ver=spec.slice(at+1);if(name==='fail-package'){console.error('intentional fixture failure');process.exit(17)}
const file=path.join(root,'package.json'),m=JSON.parse(fs.readFileSync(file));m.dependencies=m.dependencies||{};const dir=path.join(root,'node_modules',name);if(a[0]==='uninstall'){delete m.dependencies[name];fs.rmSync(dir,{recursive:true,force:true})}else{m.dependencies[name]=ver;fs.mkdirSync(dir,{recursive:true});fs.writeFileSync(path.join(dir,'package.json'),JSON.stringify({name,version:ver}))}fs.writeFileSync(file,JSON.stringify(m));fs.writeFileSync(path.join(root,'package-lock.json'),JSON.stringify({lockfileVersion:3,packages:{'':{dependencies:m.dependencies}}}));console.log('fixture modified '+name);`
	inventoryFixture(t, root, "manager/node_modules/npm/bin/npm-cli.js", script)
	return PackageTarget{Ecosystem: "node", Scope: "project", Root: project, Directory: project, Manager: "npm", ManagerPath: manifest, Interpreter: node}, &Service{}
}

func TestPackageManageBindsLatestAndRejectsStalePlan(t *testing.T) {
	target, s := packageFixtureManager(t)
	request := PackageRequest{Target: target, Operation: "install", Items: []PackageItemRequest{{Name: "is-number", Desired: "latest"}}}
	plan, err := s.planPackages(context.Background(), request, packageTestRegistry{latest: "7.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	if request.Items[0].Desired != "latest" || request.Target.ManagerPath != target.ManagerPath {
		t.Fatal("request aliased")
	}
	if plan.Changes[0].Version != "7.0.0" || !strings.Contains(strings.Join(plan.Commands[0].Args, " "), "is-number@7.0.0") {
		t.Fatal(plan)
	}
	bytes, _ := json.Marshal(plan)
	var retained PackagePlan
	if err = json.Unmarshal(bytes, &retained); err != nil {
		t.Fatal(err)
	}
	result, err := s.ApplyPackages(context.Background(), retained)
	if err != nil {
		t.Fatal(err, result.Log)
	}
	if len(result.Outcomes) != 1 || result.Outcomes[0].State != "applied" {
		t.Fatal(result)
	}
	if current := findManagedPackage(result.Group, "is-number"); current.Requested != "latest" || current.Version != "7.0.0" {
		t.Fatal(current)
	}
	if _, err = s.ApplyPackages(context.Background(), retained); err == nil || !strings.Contains(err.Error(), "INPUT_CHANGED") {
		t.Fatalf("replayed stale plan: %v", err)
	}
	request.Items[0].Desired = "6.0.0"
	plan, err = s.planPackages(context.Background(), request, packageTestRegistry{latest: "7.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(target.Root, "package.json")
	data, _ := os.ReadFile(file)
	os.WriteFile(file, append(data, ' '), 0600)
	if _, err = s.ApplyPackages(context.Background(), plan); err == nil || !strings.Contains(err.Error(), "INPUT_CHANGED") {
		t.Fatalf("changed declaration accepted: %v", err)
	}
}

func TestPackageManageReportsPartialBatch(t *testing.T) {
	target, s := packageFixtureManager(t)
	request := PackageRequest{Target: target, Operation: "install", Items: []PackageItemRequest{{Name: "is-number", Desired: "7.0.0"}, {Name: "fail-package", Desired: "1.0.0"}, {Name: "not-started", Desired: "1.0.0"}}}
	plan, err := s.planPackages(context.Background(), request, packageTestRegistry{})
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.ApplyPackages(context.Background(), plan)
	if err == nil {
		t.Fatal("fixture manager failure hidden")
	}
	if len(result.Outcomes) != 3 || result.Outcomes[0].State != "applied" || result.Outcomes[1].State != "not_applied" || result.Outcomes[2].State != "not_applied" {
		t.Fatal(result)
	}
	if !strings.Contains(result.Log, "intentional fixture failure") {
		t.Fatal(result.Log)
	}
}

func TestPackageManageProtectsManagedPythonAndUnknownUV(t *testing.T) {
	root := t.TempDir()
	venv := filepath.Join(root, ".myenv", "generations", "a", "venv")
	os.MkdirAll(venv, 0700)
	_, _, _, err := (&Service{}).packageTarget(context.Background(), PackageTarget{Ecosystem: "python", Scope: "interpreter", Root: venv, Interpreter: filepath.Join(venv, "python.exe")})
	if err == nil || !strings.Contains(err.Error(), "不能原地修改") {
		t.Fatal(err)
	}
	if _, err := packageToolTarget(context.Background(), PackageTarget{Ecosystem: "node", Scope: "tool", Tool: "pnpm", ManagerPath: filepath.Join(root, "unknown.cmd")}); err == nil {
		t.Fatal("unknown owner accepted")
	}
}

func TestPackageManagePythonPreviewNeverRunsSiteHooks(t *testing.T) {
	python, err := exec.LookPath("python")
	if err != nil {
		t.Skip("existing Python required for isolated metadata fixture")
	}
	root := t.TempDir()
	venv := filepath.Join(root, "venv")
	if output, err := probeInventoryCommand(context.Background(), python, []string{"-I", "-m", "venv", "--without-pip", venv}, root, 8192); err != nil {
		t.Fatal(err, output)
	}
	entry := filepath.Join(venv, "bin", "python")
	site := ""
	if runtime.GOOS == "windows" {
		entry = filepath.Join(venv, "Scripts", "python.exe")
		site = filepath.Join(venv, "Lib", "site-packages")
	} else {
		entries, _, _ := inventoryDirectoryEntries(filepath.Join(venv, "lib"), 8)
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), "python") {
				site = filepath.Join(venv, "lib", e.Name(), "site-packages")
			}
		}
	}
	if site == "" {
		t.Fatal("site-packages fixture not found")
	}
	sentinel := filepath.Join(root, "preview-ran-hook")
	hook := "import pathlib;pathlib.Path(" + fmt.Sprintf("%q", sentinel) + ").write_text('unexpected')\n"
	inventoryFixture(t, site, "sitecustomize.py", hook)
	inventoryFixture(t, site, "preview.pth", hook)
	inventoryFixture(t, site, "pip-25.0.dist-info/METADATA", "Metadata-Version: 2.1\nName: pip\nVersion: 25.0\n")
	inventoryFixture(t, site, "pip/__main__.py", "raise RuntimeError('preview must not import pip')\n")
	inventoryFixture(t, site, "idna-3.10.dist-info/METADATA", "Metadata-Version: 2.1\nName: idna\nVersion: 3.10\n")
	_, err = (&Service{}).PlanPackages(context.Background(), PackageRequest{Target: PackageTarget{Ecosystem: "python", Scope: "interpreter", Root: venv, Interpreter: entry}, Operation: "remove", Items: []PackageItemRequest{{Name: "idna"}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Fatal("read-only preview executed Python site hooks")
	}
}

// These opt-in tests use only dedicated temporary projects/venvs. The manager
// comes from the host, while every install/switch/remove targets the fixture.
func TestPackageManageRealNodeLifecycle(t *testing.T) {
	if os.Getenv("MYENV_TEST_PACKAGE_REAL") != "1" {
		t.Skip("set MYENV_TEST_PACKAGE_REAL=1 for official registry integration")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	for _, manager := range []string{"npm", "pnpm"} {
		t.Run(manager, func(t *testing.T) {
			if _, err := exec.LookPath(manager); err != nil {
				t.Skip("manager not installed")
			}
			root := t.TempDir()
			inventoryFixture(t, root, "package.json", `{"name":"myenv-isolated-package-test","private":true,"version":"1.0.0","dependencies":{}}`)
			target := PackageTarget{Ecosystem: "node", Scope: "project", Root: root, Directory: root, Manager: manager, Interpreter: node}
			s := &Service{}
			for _, action := range []struct{ operation, version string }{{"install", "6.0.0"}, {"switch", "7.0.0"}, {"update", "latest"}, {"remove", ""}} {
				ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
				kind := "dependency"
				if action.operation == "install" {
					kind = "development"
				}
				plan, err := s.PlanPackages(ctx, PackageRequest{Target: target, Operation: action.operation, Items: []PackageItemRequest{{Name: "is-number", Desired: action.version, Kind: kind}}})
				if err != nil {
					cancel()
					t.Fatal(err)
				}
				result, err := s.ApplyPackages(ctx, plan)
				cancel()
				if err != nil {
					t.Fatal(err, result.Log)
				}
				t.Logf("%s %s: %+v", manager, action.operation, result.Outcomes)
			}
		})
	}
}

func TestPackageManageRealPythonLifecycle(t *testing.T) {
	if os.Getenv("MYENV_TEST_PACKAGE_REAL") != "1" {
		t.Skip("set MYENV_TEST_PACKAGE_REAL=1 for official registry integration")
	}
	python, err := exec.LookPath("python")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	venv := filepath.Join(root, "venv")
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	out := &inventoryOutput{limit: 8192}
	code, err := runner.Execute(ctx, runner.Process{Background: true, Executable: python, Args: []string{"-I", "-m", "venv", venv}, Directory: root, Environment: os.Environ(), Stdout: out, Stderr: out})
	if err != nil || code != 0 {
		t.Fatal(err, code, out.text.String())
	}
	entry := filepath.Join(venv, "bin", "python")
	if runtime.GOOS == "windows" {
		entry = filepath.Join(venv, "Scripts", "python.exe")
	}
	target := PackageTarget{Ecosystem: "python", Scope: "interpreter", Root: venv, Interpreter: entry}
	s := &Service{}
	for _, action := range []struct{ operation, version string }{{"install", "3.9"}, {"switch", "3.10"}, {"remove", ""}} {
		plan, err := s.PlanPackages(ctx, PackageRequest{Target: target, Operation: action.operation, Items: []PackageItemRequest{{Name: "idna", Desired: action.version}}})
		if err != nil {
			t.Fatal(err)
		}
		result, err := s.ApplyPackages(ctx, plan)
		if err != nil {
			t.Fatal(err, result.Log)
		}
		t.Logf("pip %s: %+v", action.operation, result.Outcomes)
	}
	if t.Failed() {
		fmt.Println("all package mutations targeted", root)
	}
}

func TestPackageManageRealGlobalLifecycle(t *testing.T) {
	if os.Getenv("MYENV_TEST_PACKAGE_REAL") != "1" {
		t.Skip("set MYENV_TEST_PACKAGE_REAL=1 for official registry integration")
	}
	prefix := t.TempDir()
	root := filepath.Join(prefix, "node_modules")
	if runtime.GOOS != "windows" {
		root = filepath.Join(prefix, "lib", "node_modules")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	target := PackageTarget{Ecosystem: "node", Scope: "global_package_root", Root: root, Manager: "npm"}
	s := &Service{}
	for _, action := range []struct{ operation, version string }{{"install", "6.0.0"}, {"switch", "7.0.0"}, {"remove", ""}} {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		plan, err := s.PlanPackages(ctx, PackageRequest{Target: target, Operation: action.operation, Items: []PackageItemRequest{{Name: "is-number", Desired: action.version}}})
		if err != nil {
			cancel()
			t.Fatal(err)
		}
		result, err := s.ApplyPackages(ctx, plan)
		cancel()
		if err != nil {
			t.Fatal(err, result.Log)
		}
		t.Logf("npm isolatedglobal %s: %+v", action.operation, result.Outcomes)
	}
}

func TestPackageManageRealToolPlans(t *testing.T) {
	if os.Getenv("MYENV_TEST_PACKAGE_REAL") != "1" {
		t.Skip("set MYENV_TEST_PACKAGE_REAL=1 for readonly official tool plans")
	}
	// Planning reads the actual manager's owner but never executes the plans.
	s := &Service{}
	for _, tool := range []string{"npm", "pnpm", "uv"} {
		t.Run(tool, func(t *testing.T) {
			entry, err := exec.LookPath(tool)
			if err != nil {
				t.Skip("not installed")
			}
			eco := "node"
			if tool == "uv" {
				eco = "python"
			}
			plan, err := s.PlanPackages(context.Background(), PackageRequest{Target: PackageTarget{Ecosystem: eco, Scope: "tool", Tool: tool, ManagerPath: entry}, Operation: "update", Items: []PackageItemRequest{{Name: tool, Desired: "latest"}}})
			if err != nil {
				t.Fatal(err)
			}
			if len(plan.Commands) != 1 || plan.Changes[0].Version == "" {
				t.Fatal(plan)
			}
			t.Logf("read-only %s owner=%s target=%s version=%s", tool, plan.Request.Target.Manager, plan.Request.Target.Root, plan.Changes[0].Version)
		})
	}
}
