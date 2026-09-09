package backend

import "testing"

func TestSourceBuildDisabledDiagnostic(t *testing.T) {
	for _, test := range []struct {
		message         string
		needsPermission bool
	}{
		{"hint: Wheels are required for `example` because building from source is disabled for all packages (i.e., with `--no-build`)", true},
		{"error: Distribution `example==1.0.0` can't be installed because it is marked as `--no-build` but has no binary distribution", true},
		{"No solution found when resolving dependencies", false},
		{"Failed to build example: backend returned an error", false},
		{"failed to fetch URL: connection refused", false},
		{"invalid option --no-build", false},
	} {
		if sourceBuildDisabled(test.message) != test.needsPermission {
			t.Fatalf("unexpected classification for %q", test.message)
		}
	}
}
