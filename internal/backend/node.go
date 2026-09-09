// Package backend prepares runtimes using their explicit release sources.
package backend

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"myenv/internal/config"
)

type NodeArtifact struct {
	Version  string `json:"version"`
	NPM      string `json:"npm"`
	Platform string `json:"platform"`
	URL      string `json:"url"`
	SHA256   string `json:"sha256"`
	Backend  string `json:"backend"`
	Evidence string `json:"evidence"`
}

type Node struct {
	SegmentedSDK bool
	Client       *http.Client
	BaseURL      string
}

func NewNode() *Node {
	return &Node{Client: &http.Client{Timeout: 5 * time.Minute}, BaseURL: "https://nodejs.org/dist"}
}

func (n *Node) metadata(ctx context.Context, path string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(n.BaseURL, "/")+path, nil)
	if err != nil {
		return nil, &requestFailure{cause: err, invalidURL: true}
	}
	response, err := n.Client.Do(req)
	if err != nil {
		return nil, &requestFailure{cause: err}
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DOWNLOAD_FAILED: HTTP %d for %s", response.StatusCode, path)
	}
	b, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, &requestFailure{cause: err}
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("metadata exceeds %d bytes", limit)
	}
	return b, nil
}

func (n *Node) Resolve(ctx context.Context, selector, platform string) (NodeArtifact, error) {
	if channel := config.NodeChannel(selector); channel != "" {
		copy := *n
		copy.BaseURL = "https://nodejs.org/download/" + channel
		n = &copy
	}
	var result NodeArtifact
	target, extension, fileKey := "", "", ""
	switch platform {
	case "windows-amd64":
		target, extension, fileKey = "win-x64", "zip", "win-x64-zip"
	case "linux-amd64-glibc":
		target, extension, fileKey = "linux-x64", "tar.gz", "linux-x64"
	case "darwin-arm64":
		target, extension, fileKey = "darwin-arm64", "tar.gz", "osx-arm64-tar"
	default:
		return result, fmt.Errorf("unsupported Node platform %q", platform)
	}
	constraint, err := config.ParseConstraint("node", selector)
	if err != nil {
		return result, err
	}
	data, err := n.metadata(ctx, "/index.json", 4<<20)
	if err != nil {
		return result, err
	}
	var releases []struct {
		Version string   `json:"version"`
		NPM     string   `json:"npm"`
		Files   []string `json:"files"`
	}
	if err = json.Unmarshal(data, &releases); err != nil {
		return result, fmt.Errorf("invalid Node index: %w", err)
	}
	for _, release := range releases {
		version := strings.TrimPrefix(release.Version, "v")
		if !constraint.Contains(version) {
			continue
		}
		available := false
		for _, file := range release.Files {
			if file == fileKey {
				available = true
				break
			}
		}
		if !available || !newer(version, result.Version) {
			continue
		}
		result = NodeArtifact{Version: version, NPM: release.NPM, Platform: platform, Backend: "node-official-v1", Evidence: "artifact"}
	}
	if result.Version == "" {
		return result, fmt.Errorf("no Node release satisfies %q on %s", selector, platform)
	}
	filename := "node-v" + result.Version + "-" + target + "." + extension
	checksums, err := n.metadata(ctx, "/v"+result.Version+"/SHASUMS256.txt", 1<<20)
	if err != nil {
		return NodeArtifact{}, err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(checksums)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 2 || strings.TrimPrefix(fields[1], "*") != filename {
			continue
		}
		digest, err := hex.DecodeString(fields[0])
		if err != nil || len(digest) != 32 {
			return NodeArtifact{}, fmt.Errorf("invalid checksum for %s", filename)
		}
		if result.SHA256 != "" {
			return NodeArtifact{}, fmt.Errorf("duplicate checksum for %s", filename)
		}
		result.SHA256 = strings.ToLower(fields[0])
	}
	if err = scanner.Err(); err != nil {
		return NodeArtifact{}, err
	}
	if result.SHA256 == "" {
		return NodeArtifact{}, fmt.Errorf("missing checksum for %s", filename)
	}
	result.URL = strings.TrimRight(n.BaseURL, "/") + "/v" + result.Version + "/" + filename
	return result, nil
}

func newer(a, b string) bool {
	if b == "" {
		return true
	}
	aa, bb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < 3; i++ {
		x, _ := strconv.Atoi(aa[i])
		y, _ := strconv.Atoi(bb[i])
		if x != y {
			return x > y
		}
	}
	return false
}
