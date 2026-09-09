package cli

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// This opt-in diagnostic does not change registry or installed compiler files.
func TestSDKMSVCTelemetryProbe(t *testing.T) {
	root := os.Getenv("MYENV_SDK_TEST_ROOT")
	if root == "" || runtime.GOOS != "windows" {
		t.Skip("isolated SDK trial required")
	}
	t.Setenv("VSCMD_SKIP_SENDTELEMETRY", "1")
	project := filepath.Join(root, "rust")
	out, err := os.Create(filepath.Join(project, "msvc-probe.log"))
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	args := []string{"-C", project, "run", "rustc", "main.rs", "-o", "hello-telemetry.exe"}
	if linker := os.Getenv("MYENV_SDK_TEST_LINKER"); linker != "" {
		args = append(args, "-C", "linker="+linker)
	}
	code := executeContext(ctx, args, strings.NewReader(""), out, out, "probe", filepath.Join(root, "namespace-rust"))
	if code != 0 {
		t.Fatalf("compiler tree did not complete: exit %d, context %v", code, ctx.Err())
	}
}
