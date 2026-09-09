package runner

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSupervisorDeadlineWithoutLifetimeEOF(t *testing.T) {
	executable := os.Args[0]
	if artifact := os.Getenv("MYENV_TEST_SUPERVISOR_EXECUTABLE"); artifact != "" {
		if !filepath.IsAbs(artifact) {
			t.Fatal("supervisor artifact must be absolute")
		}
		file, err := os.Open(artifact)
		if err != nil {
			t.Fatal(err)
		}
		hash := sha256.New()
		_, copyErr := io.Copy(hash, file)
		closeErr := file.Close()
		if copyErr != nil || closeErr != nil {
			t.Fatalf("hash artifact: %v %v", copyErr, closeErr)
		}
		t.Logf("supervisor artifact=%s sha256=%x", artifact, hash.Sum(nil))
		executable = artifact
	}
	for _, mode := range []string{"running", "expired", "receipt-readonly"} {
		t.Run(mode, func(t *testing.T) {
			expired := mode == "expired"
			root := t.TempDir()
			receiptPath := filepath.Join(root, "receipt")
			if err := os.WriteFile(receiptPath, nil, 0600); err != nil {
				t.Fatal(err)
			}
			flags := os.O_RDWR
			if mode == "receipt-readonly" {
				flags = os.O_RDONLY
			}
			receipt, err := os.OpenFile(receiptPath, flags, 0600)
			if err != nil {
				t.Fatal(err)
			}
			defer receipt.Close()
			var reads, writes []*os.File
			for i := 0; i < 3; i++ {
				r, w, err := os.Pipe()
				if err != nil {
					t.Fatal(err)
				}
				reads, writes = append(reads, r), append(writes, w)
				defer r.Close()
				defer w.Close()
			}
			deadline := time.Now().Add(300 * time.Millisecond)
			if expired {
				deadline = time.Now().Add(-time.Second)
			}
			request := supervisorRequest{Executable: "/bin/sh", Args: []string{"-c", "echo started > started; kill -STOP $$"}, Directory: root, Environment: os.Environ(), Deadline: deadline}
			request.CompletionToken = strings.Repeat("d", 64)
			if err := json.NewEncoder(writes[0]).Encode(request); err != nil {
				t.Fatal(err)
			}
			writes[0].Close()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, executable, supervisorArgument)
			command.ExtraFiles = []*os.File{reads[0], writes[1], reads[2], receipt}
			if err := command.Start(); err != nil {
				t.Fatal(err)
			}
			defer func() { _ = command.Process.Kill(); _ = command.Wait() }()
			reads[0].Close()
			writes[1].Close()
			reads[2].Close()
			// Keep writes[2] open: cancellation must originate in the supervisor.
			decoder := json.NewDecoder(reads[1])
			var ready, reply supervisorReply
			if err := decoder.Decode(&ready); err != nil || !ready.Ready {
				t.Fatalf("ready: %+v %v", ready, err)
			}
			if err := decoder.Decode(&reply); err != nil || !reply.Complete || reply.Code == 0 {
				t.Fatalf("completion: %+v %v", reply, err)
			}
			if err := command.Wait(); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(receiptPath)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "receipt-readonly" {
				if reply.Canceled || !strings.Contains(reply.Error, "bad file descriptor") || len(data) != 0 {
					t.Fatalf("deadline masked receipt failure: %+v %q", reply, data)
				}
			} else if !reply.Canceled || string(data) != "myenv-tree-complete-v1:"+request.CompletionToken+"\n" {
				t.Fatalf("deadline receipt: %+v %q", reply, data)
			}
			_, err = os.Stat(filepath.Join(root, "started"))
			if expired && !os.IsNotExist(err) {
				t.Fatalf("expired request launched: %v", err)
			}
			if !expired && err != nil {
				t.Fatalf("running fixture did not start: %v", err)
			}
		})
	}
}
