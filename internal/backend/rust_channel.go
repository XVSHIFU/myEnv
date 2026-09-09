package backend

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/pelletier/go-toml/v2"
	"myenv/internal/config"
	"strings"
	"time"
)

// Rust publishes channel manifests, not an enumerable index of every nightly.
// A dated selector queries that exact official manifest and never falls back.
func (c Catalog) RustChannel(ctx context.Context, selector, platform string) (CatalogRelease, error) {
	var out CatalogRelease
	if _, e := config.ParseConstraint("rust", selector); e != nil {
		return out, e
	}
	channel, date, _ := strings.Cut(selector, "-")
	if channel != "beta" && channel != "nightly" {
		return out, fmt.Errorf("expected beta or nightly[-YYYY-MM-DD]")
	}
	base := "https://static.rust-lang.org/dist/"
	if date != "" {
		if _, e := time.Parse("2006-01-02", date); e != nil {
			return out, e
		}
		base += date + "/"
	}
	data, e := c.get(ctx, base+"channel-rust-"+channel+".toml", 8<<20)
	if e != nil {
		return out, e
	}
	var manifest struct {
		Date string
		Pkg  map[string]struct {
			Version string
			Target  map[string]struct {
				Available bool
				URL, Hash string
			}
		}
	}
	if e = toml.Unmarshal(data, &manifest); e != nil {
		return out, e
	}
	if _, e = time.Parse("2006-01-02", manifest.Date); e != nil {
		return out, e
	}
	if date != "" && date != manifest.Date {
		return out, fmt.Errorf("Rust manifest date differs from requested date")
	}
	target := "x86_64-unknown-linux-gnu"
	if platform == "windows-amd64" {
		target = "x86_64-pc-windows-msvc"
	} else if platform != "linux-amd64-glibc" {
		return out, fmt.Errorf("unsupported Rust platform")
	}
	pkg := manifest.Pkg["rust"]
	archive := pkg.Target[target]
	if !archive.Available {
		return out, fmt.Errorf("Rust %s has no full distribution for %s", selector, target)
	}
	if e = ValidateSDKOrigin("rust", archive.URL); e != nil {
		return out, e
	}
	if !strings.HasPrefix(archive.URL, "https://static.rust-lang.org/dist/"+manifest.Date+"/") {
		return out, fmt.Errorf("Rust channel archive must be dated")
	}
	sum, e := hex.DecodeString(archive.Hash)
	if e != nil || len(sum) != 32 {
		return out, fmt.Errorf("invalid Rust manifest hash")
	}
	fields := strings.Fields(pkg.Version)
	if len(fields) == 0 {
		return out, fmt.Errorf("missing Rust runtime version")
	}
	return CatalogRelease{Tool: "rust", Version: channel + "-" + manifest.Date, RuntimeVersion: fields[0], Provider: "Rust Project", Platform: platform, URL: archive.URL, SHA256: archive.Hash, Channel: "preview", Kind: "archive"}, nil
}
