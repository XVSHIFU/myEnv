package core

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"

	"myenv/internal/config"
)

func TestPackageManagePythonPreviewWindowsAlias(t *testing.T) {
	venv, _, sentinel := packagePythonPreviewFixture(t)
	alias := filepath.Join(t.TempDir(), "venv-alias")
	powershell, err := exec.LookPath("pwsh")
	if err != nil {
		t.Fatal("PowerShell is required to create the isolated junction fixture")
	}
	command := exec.Command(powershell, "-NoProfile", "-NonInteractive", "-Command", "$ErrorActionPreference='Stop'; New-Item -ItemType Junction -Path $env:MYENV_PACKAGE_ALIAS -Target $env:MYENV_PACKAGE_VENV | Out-Null")
	command.Env = append(os.Environ(), "MYENV_PACKAGE_ALIAS="+alias, "MYENV_PACKAGE_VENV="+venv)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("create isolated junction: %v %s", err, output)
	}
	t.Cleanup(func() {
		if err := os.Remove(alias); err != nil {
			t.Error(err)
		}
	})
	entry := filepath.Join(alias, "Scripts", "python.exe")
	group := inspectPythonPackages(context.Background(), entry, "interpreter", "")
	canonical, err := config.ResolveExistingPath(venv)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Python metadata root=%q canonical selected root=%q", group.Root, canonical)
	plan, err := (&Service{}).PlanPackages(context.Background(), PackageRequest{Target: PackageTarget{Ecosystem: "python", Scope: "interpreter", Root: alias, Interpreter: entry}, Operation: "remove", Items: []PackageItemRequest{{Name: "idna"}}})
	if err != nil {
		t.Fatal(err)
	}
	if !sameInventoryPath(plan.Request.Target.Root, canonical) {
		t.Fatalf("selected root was not canonical: %+v", plan.Request.Target)
	}
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Fatal("read-only alias preview executed Python site hooks")
	}
	t.Run("short-path", func(t *testing.T) {
		long, err := windows.UTF16PtrFromString(venv)
		if err != nil {
			t.Fatal(err)
		}
		buffer := make([]uint16, 32768)
		n, err := windows.GetShortPathName(long, &buffer[0], uint32(len(buffer)))
		if err != nil {
			t.Fatal(err)
		}
		if n >= uint32(len(buffer)) {
			t.Fatal("short path exceeds fixed buffer")
		}
		short := windows.UTF16ToString(buffer[:n])
		if sameInventoryPath(short, venv) {
			t.Skip("8.3 path names unavailable on this volume; junction case still verifies canonical identity")
		}
		shortEntry := filepath.Join(short, "Scripts", "python.exe")
		plan, err := (&Service{}).PlanPackages(context.Background(), PackageRequest{Target: PackageTarget{Ecosystem: "python", Scope: "interpreter", Root: short, Interpreter: shortEntry}, Operation: "remove", Items: []PackageItemRequest{{Name: "idna"}}})
		if err != nil {
			t.Fatal(err)
		}
		if !sameInventoryPath(plan.Request.Target.Root, canonical) {
			t.Fatalf("short root was not canonical: %+v", plan.Request.Target)
		}
		if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
			t.Fatal("read-only short-path preview executed Python site hooks")
		}
		t.Logf("8.3 path=%q canonical root=%q", short, plan.Request.Target.Root)
	})
	t.Run("different-environment", func(t *testing.T) {
		other := t.TempDir()
		_, err := (&Service{}).PlanPackages(context.Background(), PackageRequest{Target: PackageTarget{Ecosystem: "python", Scope: "interpreter", Root: other, Interpreter: entry}, Operation: "remove", Items: []PackageItemRequest{{Name: "idna"}}})
		if err == nil {
			t.Fatal("unrelated environment accepted")
		}
	})
}
