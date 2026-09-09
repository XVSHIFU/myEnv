package runner

import (
	"encoding/json"
	"io"
	"os"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestCreateProcessInJobPipes(t *testing.T) {
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
	inputRead, inputWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer inputRead.Close()
	defer inputWrite.Close()
	outputRead, outputWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer outputRead.Close()
	defer outputWrite.Close()
	errorFile, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer errorFile.Close()
	stdio, err := duplicateNativeStdio([3]*os.File{inputRead, outputWrite, errorFile})
	if err != nil {
		t.Fatal(err)
	}
	defer stdio.close()
	job, err := createSupervisionJob("")
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(job)
	process, err := createProcessInJob(Process{Executable: prepared.Executable, Args: []string{"-e", `process.stdin.pipe(process.stdout)`}}, job, stdio.startup, stdio.handles)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(process.Process)
	defer windows.CloseHandle(process.Thread)
	defer windows.TerminateProcess(process.Process, 1)
	stdio.close()
	inputRead.Close()
	outputWrite.Close()
	if _, err := windows.ResumeThread(process.Thread); err != nil {
		t.Fatal(err)
	}
	const payload = "anonymous pipe 中文\n"
	if _, err := io.WriteString(inputWrite, payload); err != nil {
		t.Fatal(err)
	}
	inputWrite.Close()
	type result struct {
		data []byte
		err  error
	}
	done := make(chan result, 1)
	go func() { data, err := io.ReadAll(outputRead); done <- result{data, err} }()
	select {
	case got := <-done:
		if got.err != nil || string(got.data) != payload {
			t.Fatalf("pipe: %q %v", got.data, got.err)
		}
	case <-time.After(5 * time.Second):
		outputRead.Close()
		<-done
		t.Fatal("pipe EOF missing")
	}
	if event, err := windows.WaitForSingleObject(process.Process, 3000); err != nil || event != windows.WAIT_OBJECT_0 {
		t.Fatalf("wait: %d %v", event, err)
	}
	var code uint32
	if err := windows.GetExitCodeProcess(process.Process, &code); err != nil || code != 0 {
		t.Fatalf("exit: %d %v", code, err)
	}
}
