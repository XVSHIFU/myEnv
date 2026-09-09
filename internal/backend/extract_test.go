package backend

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractZIPBoundary(t *testing.T) {
	for _, name := range []string{"../escape", "/absolute", "C:/escape", "node/../../escape", "node/CON.txt", "node/file:stream", "node/file.", "node/link", "node/bin/node.exe"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			archive := filepath.Join(root, "test.zip")
			file, err := os.Create(archive)
			if err != nil {
				t.Fatal(err)
			}
			writer := zip.NewWriter(file)
			header := &zip.FileHeader{Name: name, Method: zip.Deflate}
			if name == "node/link" {
				header.SetMode(os.ModeSymlink | 0777)
			}
			entry, err := writer.CreateHeader(header)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = entry.Write([]byte("payload")); err != nil {
				t.Fatal(err)
			}
			if err = writer.Close(); err != nil {
				t.Fatal(err)
			}
			if err = file.Close(); err != nil {
				t.Fatal(err)
			}
			destination := filepath.Join(root, "out")
			err = ExtractZIP(archive, destination)
			if name != "node/bin/node.exe" {
				if err == nil {
					t.Fatal("accepted unsafe archive")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(filepath.Join(destination, "node", "bin", "node.exe"))
			if err != nil || string(data) != "payload" {
				t.Fatalf("bad extraction %s %v", data, err)
			}
			if err = ExtractZIP(archive, destination); err == nil {
				t.Fatal("reused existing destination")
			}
		})
	}
}
