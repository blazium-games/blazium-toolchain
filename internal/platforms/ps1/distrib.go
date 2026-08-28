package ps1

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/blazium-games/blazium-toolchain/internal/fetch"
	"github.com/blazium-games/blazium-toolchain/internal/platforms"
)

const (
	distribRoot         = "https://distrib.app"
	pcsxWinCLICatalog   = "https://distrib.app/storage/manifests/pcsx-redux/dev-win-cli-x64/manifest.json"
	pcsxWinCLIInfoBase  = "https://distrib.app/storage/manifests/pcsx-redux/dev-win-cli-x64/"
	pcsxPinnedWinCLIZip = "https://distrib.app/storage/assets/265/a79/ea7/640c8fce00aca4a92d8638443a0533597780c73adddabc747cb3a50/pcsx-redux-nightly-25100.20260827.8-x64-cli.zip"
)

type jsonGetter interface {
	GetJSON(ctx context.Context, url string, dest any) error
}

type catalogFile struct {
	Builds []struct {
		ID int `json:"id"`
	} `json:"builds"`
}

type buildFile struct {
	Path string `json:"path"`
}

func (t *Tool) pcsxZipURL(ctx context.Context) (string, error) {
	if t.CLIURL != "" {
		return t.CLIURL, nil
	}
	if runtime.GOOS != "windows" {
		return "", fmt.Errorf("%w: pcsx-redux AppDistrib is Windows-only. Install pcsx-redux or pcsx-redux-cli, put it on PATH, or set PCSX_EXE", platforms.ErrMissingTool)
	}
	url, err := t.resolveAppDistribZip(ctx, pcsxWinCLICatalog, pcsxWinCLIInfoBase)
	if err == nil && url != "" {
		return url, nil
	}
	return pcsxPinnedWinCLIZip, nil
}

func (t *Tool) resolveAppDistribZip(ctx context.Context, catalogURL, infoBase string) (string, error) {
	g := t.jsonGet()
	var cat catalogFile
	if err := g.GetJSON(ctx, catalogURL, &cat); err != nil {
		return "", err
	}
	if len(cat.Builds) == 0 || cat.Builds[0].ID == 0 {
		return "", fmt.Errorf("empty AppDistrib catalog")
	}
	var man buildFile
	u := fmt.Sprintf("%smanifest-%d.json", infoBase, cat.Builds[0].ID)
	if err := g.GetJSON(ctx, u, &man); err != nil {
		return "", err
	}
	path := strings.TrimSpace(man.Path)
	if path == "" {
		return "", fmt.Errorf("catalog build %d has no path", cat.Builds[0].ID)
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path, nil
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return distribRoot + path, nil
}

func (t *Tool) jsonGet() jsonGetter {
	if t.JSON != nil {
		return t.JSON
	}
	return fetch.HTTP{}
}
