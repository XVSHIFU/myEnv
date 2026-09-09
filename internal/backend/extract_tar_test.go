package backend

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestExtractTarGZ(t *testing.T) {
	for _, scenario := range []string{"regular", "traversal", "absolute-link", "escaping-link", "missing-link", "hardlink", "duplicate", "crc", "valid-link"} {
		t.Run(scenario, func(t *testing.T) {
			if scenario == "valid-link" && runtime.GOOS == "windows" {
				t.Skip("Windows symlink privilege is not assumed")
			}
			root := t.TempDir()
			archive := filepath.Join(root, "node.tar.gz")
			f, err := os.Create(archive)
			if err != nil {
				t.Fatal(err)
			}
			gz := gzip.NewWriter(f)
			tw := tar.NewWriter(gz)
			name := "node/bin/node"
			if scenario == "traversal" {
				name = "../escape"
			}
			if err = tw.WriteHeader(&tar.Header{Name: name, Mode: 0755, Size: 7, Typeflag: tar.TypeReg}); err != nil {
				t.Fatal(err)
			}
			if _, err = tw.Write([]byte("payload")); err != nil {
				t.Fatal(err)
			}
			h := &tar.Header{Name: "node/bin/npm", Typeflag: tar.TypeSymlink, Linkname: "node"}
			switch scenario {
			case "absolute-link":
				h.Linkname = "/outside"
			case "escaping-link":
				h.Linkname = "../../../outside"
			case "missing-link":
				h.Linkname = "missing"
			case "hardlink":
				h.Typeflag = tar.TypeLink
			case "duplicate":
				h.Name = name
			}
			if scenario != "regular" && scenario != "traversal" && scenario != "crc" {
				if err = tw.WriteHeader(h); err != nil {
					t.Fatal(err)
				}
			}
			if err = tw.Close(); err != nil {
				t.Fatal(err)
			}
			if err = gz.Close(); err != nil {
				t.Fatal(err)
			}
			if err = f.Close(); err != nil {
				t.Fatal(err)
			}
			if scenario == "crc" {
				data, err := os.ReadFile(archive)
				if err != nil {
					t.Fatal(err)
				}
				data[len(data)-8] ^= 0xff
				if err = os.WriteFile(archive, data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			out := filepath.Join(root, "out")
			err = ExtractTarGZ(archive, out)
			if scenario != "regular" && scenario != "valid-link" {
				if err == nil {
					t.Fatal("accepted invalid archive")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			entry := filepath.Join(out, "node", "bin", "node")
			if scenario == "valid-link" {
				entry = filepath.Join(out, "node", "bin", "npm")
			}
			data, err := os.ReadFile(entry)
			if err != nil || string(data) != "payload" {
				t.Fatalf("contents %q: %v", data, err)
			}
			info, err := os.Stat(entry)
			if err != nil {
				t.Fatal(err)
			}
			if runtime.GOOS != "windows" && info.Mode()&0100 == 0 {
				t.Fatal("lost executable permission")
			}
			if err = ExtractTarGZ(archive, out); err == nil {
				t.Fatal("reused destination")
			}
		})
	}
}
