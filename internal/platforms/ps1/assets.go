package ps1

import (
	"context"
	"io"
	"runtime"

	"github.com/blazium-games/blazium-toolchain/internal/embedfs"
)

// ZipAsset is an official unmodified archive this CLI may fetch.
type ZipAsset struct {
	ID     string
	URL    string
	SHA256 string
	Dest   string // relative to <prefix>/ps1/
}

// ZipFetcher downloads and unpacks a zip.
type ZipFetcher interface {
	FetchZip(ctx context.Context, url, sha256, destDir string, log io.Writer) error
}

func zipAssets(pins []embedfs.ZipPin) []ZipAsset {
	out := make([]ZipAsset, len(pins))
	for i, p := range pins {
		out[i] = ZipAsset{ID: p.ID, URL: p.URL, SHA256: p.SHA256, Dest: p.Dest}
	}
	return out
}

func compileAssetsFor(goos string) []ZipAsset {
	p := embedfs.MustPins()
	return zipAssets(p.Compile[goos])
}

func defaultCompileAssets() []ZipAsset {
	return compileAssetsFor(runtime.GOOS)
}

func hostBuildAssets() []ZipAsset {
	p := embedfs.MustPins()
	return zipAssets(p.HostBuild[runtime.GOOS])
}
