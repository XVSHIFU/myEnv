package backend

import (
	"context"
	"fmt"
	"myenv/internal/config"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PrepareNodeZIP materializes a verified archive at its final unique directory.
// Publication remains the responsibility of core and the state transaction.
func PrepareNodeZIP(ctx context.Context, archive, directory string, artifact NodeArtifact) (string, error) {
	if artifact.Platform != "windows-amd64" {
		return "", fmt.Errorf("ZIP preparation requires windows-amd64")
	}
	constraint, err := exactNodeVersion(artifact.Version)
	if err != nil {
		return "", err
	}
	if err = ExtractZIP(archive, directory); err != nil {
		return "", err
	}
	executable := filepath.Join(directory, "node-v"+constraint+"-win-x64", "node.exe")
	if err = VerifyNode(ctx, executable, artifact); err != nil {
		return "", err
	}
	return executable, nil
}

// PrepareNodeTarGZ prepares the official Unix layout at its final directory.
func PrepareNodeTarGZ(ctx context.Context, archive, directory string, artifact NodeArtifact) (string, error) {
	var suffix string
	switch artifact.Platform {
	case "linux-amd64-glibc":
		suffix = "linux-x64"
	case "darwin-arm64":
		suffix = "darwin-arm64"
	default:
		return "", fmt.Errorf("unsupported Node tar platform %q", artifact.Platform)
	}
	version, err := exactNodeVersion(artifact.Version)
	if err != nil {
		return "", err
	}
	if err = ExtractTarGZ(archive, directory); err != nil {
		return "", err
	}
	executable := filepath.Join(directory, "node-v"+version+"-"+suffix, "bin", "node")
	if err = VerifyNode(ctx, executable, artifact); err != nil {
		return "", err
	}
	return executable, nil
}

func exactNodeVersion(version string) (string, error) {
	if config.NodeChannel(version) != "" {
		return version, nil
	}
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid exact Node version")
	}
	for _, p := range parts {
		if p == "" {
			return "", fmt.Errorf("invalid exact Node version")
		}
		for _, c := range p {
			if c < '0' || c > '9' {
				return "", fmt.Errorf("invalid exact Node version")
			}
		}
	}
	return version, nil
}

// VerifyNode executes only the selected absolute entry point, never PATH lookup.
func VerifyNode(ctx context.Context, executable string, artifact NodeArtifact) error {
	if !filepath.IsAbs(executable) {
		return fmt.Errorf("Node executable must be absolute")
	}
	info, err := os.Lstat(executable)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("Node entry point is not a regular file")
	}
	output := &boundedOutput{limit: 4096}
	command := exec.CommandContext(ctx, executable, "--version")
	command.Stdout = output
	command.Stderr = output
	if err = command.Run(); err != nil {
		return fmt.Errorf("Node verification failed: %w: %s", err, output.data)
	}
	if strings.TrimSpace(string(output.data)) != "v"+artifact.Version {
		return fmt.Errorf("Node version differs from lock")
	}
	return nil
}

type boundedOutput struct {
	data  []byte
	limit int
}

func (b *boundedOutput) Write(p []byte) (int, error) {
	n := len(p)
	remaining := b.limit - len(b.data)
	if remaining > 0 {
		if len(p) > remaining {
			p = p[:remaining]
		}
		b.data = append(b.data, p...)
	}
	return n, nil
}
