package runner

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestCancelStoppedProcessTree(t *testing.T) {
	root := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	receipt, err := os.OpenFile(filepath.Join(root, "complete"), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer receipt.Close()
	token := strings.Repeat("ab", 32)
	type outcome struct {
		code int
		err  error
	}
	done := make(chan outcome, 1)
	go func() {
		code, err := Execute(ctx, Process{Executable: "/bin/sh", Directory: root, Environment: os.Environ(), Completion: &TreeCompletion{File: receipt, Token: token}, Args: []string{"-c", `echo $$ > leader; /bin/sh -c 'echo $$ > descendant; kill -STOP $$; exit 0' & kill -STOP $$; wait`}})
		done <- outcome{code, err}
	}()
	// Always cancel and join the runner before closing its receipt or fixture.
	joined := false
	defer func() {
		cancel()
		if !joined {
			<-done
		}
	}()
	var processPaths []string
	for _, name := range []string{"leader", "descendant"} {
		for {
			data, err := os.ReadFile(filepath.Join(root, name))
			pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
			if err == nil && pid > 1 {
				path := filepath.Join("/proc", strconv.Itoa(pid))
				stat, err := os.ReadFile(filepath.Join(path, "stat"))
				if err != nil {
					t.Fatal(err)
				}
				end := strings.LastIndexByte(string(stat), ')')
				if end < 0 {
					t.Fatal("invalid process stat")
				}
				fields := strings.Fields(string(stat[end+1:]))
				if len(fields) > 0 && fields[0] == "T" {
					processPaths = append(processPaths, path)
					break
				}
			}
			if ctx.Err() != nil {
				t.Fatal("tree did not stop", name)
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
	cancel()
	result := <-done
	joined = true
	if result.err != nil || result.code == 0 {
		t.Fatalf("cancellation: %+v", result)
	}
	for _, path := range processPaths {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("unreaped stopped process: %s %v", path, err)
		}
	}
	data, err := os.ReadFile(receipt.Name())
	if err != nil || string(data) != "myenv-tree-complete-v1:"+token+"\n" {
		t.Fatalf("completion proof: %q %v", data, err)
	}
}
