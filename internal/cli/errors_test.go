package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestErrorOutputContract(t *testing.T) {
	for _, tt := range []struct {
		args []string
		exit int
		code string
	}{
		{[]string{"init", "--invalid", "--json"}, 2, "USAGE_ERROR"},
		{[]string{"-C", filepath.Join(t.TempDir(), "missing"), "--json"}, 1, "IO_ERROR"},
	} {
		var out, diag bytes.Buffer
		exit := Execute(tt.args, bytes.NewReader(nil), &out, &diag, "test")
		var r result
		if err := json.Unmarshal(out.Bytes(), &r); err != nil {
			t.Fatalf("invalid JSON %q: %v", out.String(), err)
		}
		if exit != tt.exit || r.Error == nil || r.Error.Code != tt.code || r.OK || diag.Len() != 0 {
			t.Fatalf("exit=%d result=%+v stderr=%s", exit, r, diag.String())
		}
	}
}
