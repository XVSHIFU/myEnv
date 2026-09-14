//go:build windows

package backend

import (
	"context"
	"errors"
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSDKRenameTemporaryHoldAndCancel(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		t.Run(map[bool]string{false: "released", true: "canceled"}[cancel], func(t *testing.T) {
			root := t.TempDir()
			source := filepath.Join(root, "sdk")
			dest := filepath.Join(root, "ready")
			if e := os.Mkdir(source, 0700); e != nil {
				t.Fatal(e)
			}
			name, _ := windows.UTF16PtrFromString(source)
			h, e := windows.CreateFile(name, windows.GENERIC_READ, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
			if e != nil {
				t.Fatal(e)
			}
			ctx, stop := context.WithTimeout(context.Background(), time.Second)
			defer stop()
			closed := make(chan struct{})
			go func() {
				defer close(closed)
				time.Sleep(100 * time.Millisecond)
				if cancel {
					stop()
					time.Sleep(50 * time.Millisecond)
				}
				windows.CloseHandle(h)
			}()
			e = renameSDK(ctx, source, dest)
			<-closed
			if cancel {
				if !errors.Is(e, context.Canceled) {
					t.Fatalf("cancel: %v", e)
				}
				if _, e = os.Stat(source); e != nil {
					t.Fatal("canceled source lost")
				}
			} else if e != nil {
				t.Fatal(e)
			}
		})
	}
}
