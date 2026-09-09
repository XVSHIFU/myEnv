package backend

import (
	"context"
	"encoding/json"
	"os"
	"testing"
)

func TestVerifyRetainedNode(t *testing.T) {
	record := os.Getenv("MYENV_TEST_PREPARED_RECORD")
	if record == "" {
		t.Skip("explicit retained runtime validation")
	}
	data, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	var prepared struct {
		Artifact   NodeArtifact
		Executable string
	}
	if err = json.Unmarshal(data, &prepared); err != nil {
		t.Fatal(err)
	}
	if err = VerifyNode(context.Background(), prepared.Executable, prepared.Artifact); err != nil {
		t.Fatal(err)
	}
	prepared.Artifact.Version = "0.0.0"
	if err = VerifyNode(context.Background(), prepared.Executable, prepared.Artifact); err == nil {
		t.Fatal("accepted mismatched runtime version")
	}
}
