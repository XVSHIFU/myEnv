package core

import (
	"context"
	"os"
	"runtime/trace"
	"testing"
)

func traceMixedStatus(t *testing.T, service *Service, root, generation string) {
	t.Helper()
	path := os.Getenv("MYENV_TEST_STATUS_TRACE")
	if path == "" {
		return
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := trace.Start(file); err != nil {
		t.Fatal(err)
	}
	defer trace.Stop()
	for i := 0; i < 51; i++ {
		status, err := service.Status(context.Background(), root)
		if err != nil || status.Environment != "ready" || status.Generation == nil || status.Generation.ID != generation {
			t.Fatalf("traced status: %+v %v", status, err)
		}
	}
}
