package backend

import (
	"context"
	"fmt"
	"os"
	"path"

	"myenv/internal/config"
)

type UVRelease struct {
	Version  string `json:"version"`
	Platform string `json:"platform"`
	URL      string `json:"url"`
	SHA256   string `json:"sha256"`
}

// FixedUVRelease embeds hashes from the official 0.11.26 release checksum files.
// There is no latest-version lookup or remote checksum substitution at runtime.
func FixedUVRelease(platform string) (UVRelease, error) {
	var filename, digest string
	switch platform {
	case "windows-amd64":
		filename = "uv-x86_64-pc-windows-msvc.zip"
		digest = "4e1278ede866be6c0bf32d2f466cc6de7a9fb399ecf20c9ce2d186e52424be47"
	case "linux-amd64-glibc":
		filename = "uv-x86_64-unknown-linux-gnu.tar.gz"
		digest = "6426a73c3837e6e2483ee344cbc00f36394d179afcba6183cb77437e67db4af0"
	case "darwin-arm64":
		filename = "uv-aarch64-apple-darwin.tar.gz"
		digest = "8f7fbf1708399b921857bce71e1d60f0d3ccf52a30caebc1c1a2f175dce13ab6"
	default:
		return UVRelease{}, fmt.Errorf("unsupported uv platform %q", platform)
	}
	return UVRelease{Version: UVVersion, Platform: platform, URL: "https://releases.astral.sh/github/uv/releases/download/" + UVVersion + "/" + filename, SHA256: digest}, nil
}

func DownloadUV(ctx context.Context, platform, directory string) (UVRelease, string, error) {
	return DownloadUVFromMirror(ctx, platform, directory, "")
}

func DownloadUVFromMirror(ctx context.Context, platform, directory, mirror string) (UVRelease, string, error) {
	release, err := FixedUVRelease(platform)
	if err != nil {
		return release, "", err
	}
	base, err := config.MirrorBaseURL(mirror)
	if err != nil {
		return release, "", err
	}
	if base != "" {
		release.URL = base + "/" + release.Version + "/" + path.Base(release.URL)
	}
	client, err := DownloadClient(os.Getenv("SSL_CERT_FILE"))
	if err != nil {
		return release, "", err
	}
	defer client.CloseIdleConnections()
	archive, err := downloadArtifact(ctx, client, release.URL, release.SHA256, directory)
	return release, archive, err
}
