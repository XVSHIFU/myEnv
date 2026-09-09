package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type failingNativeIO struct{ err error }

func (f failingNativeIO) Read([]byte) (int, error)  { return 0, f.err }
func (f failingNativeIO) Write([]byte) (int, error) { return 0, f.err }

type checkedNativeOutput struct {
	active  atomic.Int32
	overlap atomic.Bool
	data    bytes.Buffer
}

func (w *checkedNativeOutput) Write(p []byte) (int, error) {
	if w.active.Add(1) != 1 {
		w.overlap.Store(true)
	}
	defer w.active.Add(-1)
	time.Sleep(time.Millisecond)
	return w.data.Write(p)
}

func TestWindowsNativeIOFailures(t *testing.T) {
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
	for _, mode := range []string{"reader", "writer", "shared-writer"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			failure := errors.New("test I/O failure")
			p := Process{Executable: prepared.Executable}
			var output checkedNativeOutput
			switch mode {
			case "reader":
				p.Stdin = failingNativeIO{failure}
				p.Args = []string{"-e", `require('fs').readFileSync(0);process.exit(0)`}
			case "writer":
				p.Stdout = failingNativeIO{failure}
				p.Args = []string{"-e", `require('fs').writeSync(1,'output');process.exit(0)`}
			case "shared-writer":
				p.Stdout, p.Stderr = &output, &output
				p.Args = []string{"-e", `const fs=require('fs');for(let i=0;i<100;i++){fs.writeSync(1,'o');fs.writeSync(2,'e')}process.exit(0)`}
			}
			code, err := Execute(ctx, p)
			if ctx.Err() != nil {
				t.Fatal("I/O test exceeded deadline")
			}
			if mode == "shared-writer" {
				if code != 0 || err != nil || output.overlap.Load() || output.data.String() != strings.Repeat("oe", 100) {
					t.Fatalf("shared output: code=%d err=%v overlap=%t bytes=%d", code, err, output.overlap.Load(), output.data.Len())
				}
			} else if code != 1 || !errors.Is(err, failure) || errors.Is(err, ErrTreeUnconfirmed) {
				t.Fatalf("I/O error lost or false unknown completion: %d %v", code, err)
			}
		})
	}
}
