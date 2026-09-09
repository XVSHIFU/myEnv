package runner

import (
	"crypto/rand"
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"testing"
)

func TestNamedJobOwnership(t *testing.T) {
	var seed [16]byte
	if _, err := rand.Read(seed[:]); err != nil {
		t.Fatal(err)
	}
	id := fmt.Sprintf("%x", seed)
	first, err := createSupervisionJob(id)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(first)
	var session uint32
	if err = windows.ProcessIdToSessionId(uint32(os.Getpid()), &session); err != nil {
		t.Fatal(err)
	}
	if gone, err := JobTreeGone(id, session); err != nil || gone {
		t.Fatal("existing job lost protection", err)
	}
	if gone, err := JobTreeGone(id, session+1); err == nil || gone {
		t.Fatal("cross-session absence accepted")
	}
	missing := fmt.Sprintf("%032x", seed[0:16])
	// Alter one hex digit to obtain a distinct name without creating it.
	if missing[0] == '0' {
		missing = "1" + missing[1:]
	} else {
		missing = "0" + missing[1:]
	}
	if gone, err := JobTreeGone(missing, session); err != nil || !gone {
		t.Fatal("missing job not observed", err)
	}
	if other, err := createSupervisionJob(id); err == nil {
		windows.CloseHandle(other)
		t.Fatal("adopted existing job")
	}
	for _, bad := range []string{"..", `Global\other`, "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"} {
		if job, err := createSupervisionJob(bad); err == nil {
			windows.CloseHandle(job)
			t.Fatal("accepted invalid tree identity")
		}
	}
}
