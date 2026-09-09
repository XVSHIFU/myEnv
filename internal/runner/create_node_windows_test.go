package runner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"golang.org/x/sys/windows"
)

func TestCreateProcessInJobNodeIO(t *testing.T) {
	record := os.Getenv("MYENV_TEST_PREPARED_RECORD")
	if record == "" {
		t.Skip("requires retained Node")
	}
	data, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	var prepared struct{ Executable string }
	if err := json.Unmarshal(data, &prepared); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	var files []*os.File
	var inherited []windows.Handle
	for _, name := range []string{"input", "output", "error"} {
		file, err := os.Create(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		files = append(files, file)
		var handle windows.Handle
		if err := windows.DuplicateHandle(windows.CurrentProcess(), windows.Handle(file.Fd()), windows.CurrentProcess(), &handle, 0, true, windows.DUPLICATE_SAME_ACCESS); err != nil {
			t.Fatal(err)
		}
		defer windows.CloseHandle(handle)
		inherited = append(inherited, handle)
	}
	if _, err := files[0].WriteString("input 中文"); err != nil {
		t.Fatal(err)
	}
	if _, err := files[0].Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	job, err := createSupervisionJob("")
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(job)
	args := []string{"", "space value", `quote"value`, `trail\`, "中文", "--json"}
	script := `const fs=require('fs');process.stdout.write(JSON.stringify({args:process.argv.slice(1),cwd:process.cwd(),env:process.env.MYENV_NATIVE_TEST,input:fs.readFileSync(0,'utf8')}));process.stderr.write('diagnostic');process.exit(17)`
	p := Process{Executable: prepared.Executable, Args: append([]string{"-e", script}, args...), Directory: root, Environment: []string{"MYENV_NATIVE_TEST=旧", "myenv_native_test=新 值"}}
	startup := windows.StartupInfo{Flags: windows.STARTF_USESTDHANDLES, StdInput: inherited[0], StdOutput: inherited[1], StdErr: inherited[2]}
	process, err := createProcessInJob(p, job, startup, inherited)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(process.Process)
	defer windows.CloseHandle(process.Thread)
	defer windows.TerminateProcess(process.Process, 1)
	if _, err := windows.ResumeThread(process.Thread); err != nil {
		t.Fatal(err)
	}
	if event, err := windows.WaitForSingleObject(process.Process, 5000); err != nil || event != windows.WAIT_OBJECT_0 {
		t.Fatalf("wait: %d %v", event, err)
	}
	var code uint32
	if err := windows.GetExitCodeProcess(process.Process, &code); err != nil || code != 17 {
		t.Fatalf("exit: %d %v", code, err)
	}
	data, err = os.ReadFile(filepath.Join(root, "output"))
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Args            []string
		Cwd, Env, Input string
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("output %q: %v", data, err)
	}
	if !reflect.DeepEqual(got.Args, args) || got.Cwd != root || got.Env != "新 值" || got.Input != "input 中文" {
		t.Fatalf("changed child inputs: %+v", got)
	}
	data, err = os.ReadFile(filepath.Join(root, "error"))
	if err != nil || string(data) != "diagnostic" {
		t.Fatalf("stderr: %q %v", data, err)
	}
}
