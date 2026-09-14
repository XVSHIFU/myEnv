package cli

import (
	"context"
	"os"
	"testing"
)

func TestAstralRetainedCatalog(t *testing.T) {
	namespace := os.Getenv("MYENV_CATALOG_TEST_NAMESPACE")
	if namespace == "" {
		t.Skip("requires retained isolated uv installation")
	}
	result, err := QueryVersions(context.Background(), VersionQuery{Tool: "python", Namespace: namespace})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Releases) < 10 {
		t.Fatalf("unexpected small catalog: %d", len(result.Releases))
	}
	for _, r := range result.Releases {
		if r.Provider != "astral" || r.Kind != "archive" || r.URL == "" {
			t.Fatalf("not installable: %+v", r)
		}
	}
	t.Logf("%s: %d installable stable versions", result.Platform, len(result.Releases))
}
