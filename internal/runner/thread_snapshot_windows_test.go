package runner

import (
	"runtime"
	"testing"
	"unsafe"
)

func TestSnapshotThreadEntryABI(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("first-release Windows ABI is amd64")
	}
	var entry snapshotThreadEntry
	if unsafe.Sizeof(entry) != 120 || unsafe.Offsetof(entry.ThreadID) != 20 || unsafe.Offsetof(entry.CreateTime) != 52 || unsafe.Offsetof(entry.ContextRecord) != 112 {
		t.Fatal("PSS_THREAD_ENTRY no longer matches Windows amd64 ABI")
	}
}
