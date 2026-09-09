package cli

import (
	"bytes"
	"myenv/internal/core"
	"strings"
	"testing"
)

func TestProgressDisplayClearsBeforeOutputAndStops(t *testing.T) {
	var out bytes.Buffer
	p := newProgressDisplay(&out)
	p.phase("正在验证")
	p.Write([]byte("warning\n"))
	p.phase("正在应用")
	final := &progressOutput{display: p, out: &out}
	final.Write([]byte("done\n"))
	p.close()
	if !strings.Contains(out.String(), "\r\x1b[2Kwarning\n") || !strings.HasSuffix(out.String(), "\r\x1b[2Kdone\n") {
		t.Fatal(out.String())
	}
	select {
	case <-p.done:
	default:
		t.Fatal("animation did not stop")
	}
}

func TestProgressPlainWriterContainsNoControls(t *testing.T) {
	var out bytes.Buffer
	syncProgress(&out, true)(core.PhaseSDK)
	if strings.ContainsAny(out.String(), "\x1b\r") || !strings.HasSuffix(out.String(), "\n") {
		t.Fatal(out.String())
	}
	if paint(&out, "32", "可用") != "可用" {
		t.Fatal("redirected output colored")
	}
	t.Setenv("NO_COLOR", "1")
	if terminalANSI(&out) {
		t.Fatal("NO_COLOR ignored")
	}
}
