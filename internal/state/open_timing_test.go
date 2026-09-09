package state

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

func TestMeasureInitializedStoreOpen(t *testing.T) {
	output := os.Getenv("MYENV_TEST_STORE_OPEN_TIMINGS")
	if output == "" {
		t.Skip("set MYENV_TEST_STORE_OPEN_TIMINGS")
	}
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "state.db")
	s, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	samples := map[string][]float64{"writable": {}, "readonly": {}}
	for i := 0; i < 51; i++ {
		for j := 0; j < 2; j++ {
			readOnly := (i+j)%2 == 0
			name := "writable"
			if readOnly {
				name = "readonly"
			}
			start := time.Now()
			store, err := openStore(ctx, path, readOnly)
			if err != nil {
				t.Fatal(err)
			}
			closeErr := store.Close()
			elapsed := float64(time.Since(start).Nanoseconds()) / 1e6
			if closeErr != nil {
				t.Fatal(closeErr)
			}
			samples[name] = append(samples[name], elapsed)
		}
	}
	report := map[string]any{"method": "51 alternating paired initialized-store opens; includes Close; first separate; remaining 50 nearest-rank p95; in-process diagnostic, not CLI gate"}
	for name, values := range samples {
		ordered := append([]float64(nil), values[1:]...)
		sort.Float64s(ordered)
		report[name] = map[string]any{"first_ms": values[0], "samples_ms": values[1:], "median_ms": ordered[24], "p95_ms": ordered[47]}
		t.Logf("%s median=%.4fms p95=%.4fms", name, ordered[24], ordered[47])
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.Write(append(data, '\n'))
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		t.Fatal(err, closeErr)
	}
}
