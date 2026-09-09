package cli

import (
	"io"
	"strings"
	"testing"
)

// This isolates CLI construction and formatting from package initialization,
// executable loading and process creation. It is not an end-to-end p95 gate.
func BenchmarkInformationalCLI(b *testing.B) {
	for _, flag := range []string{"--version", "--help"} {
		b.Run(flag, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if code := Execute([]string{flag}, strings.NewReader(""), io.Discard, io.Discard, "benchmark"); code != 0 {
					b.Fatalf("%s exit %d", flag, code)
				}
			}
		})
	}
}
