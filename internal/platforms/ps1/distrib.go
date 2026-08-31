package ps1

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/blazium-games/blazium-toolchain/internal/fetch"
	"github.com/blazium-games/blazium-toolchain/internal/report"
	"github.com/blazium-games/blazium-toolchain/internal/settings"
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
		return "", report.Missing("pcsx-redux AppDistrib is Windows-only", "install pcsx-redux on PATH or set PCSX_EXE")
	}
	f := settings.Current().PS1.Fetch
	url, err := t.resolveAppDistribZip(ctx, f.PCSXCatalog, f.PCSXInfoBase)
	if err == nil && url != "" {
		return url, nil
	}
	return f.PCSXPinnedZip, nil
}

func (t *Tool) resolveAppDistribZip(ctx context.Context, catalogURL, infoBase string) (string, error) {
	g := t.jsonGet()
	var cat catalogFile
	if err := g.GetJSON(ctx, catalogURL, &cat); err != nil {
		return "", err
	}
	if len(cat.Builds) == 0 || cat.Builds[0].ID == 0 {
		return "", report.Fail("empty AppDistrib catalog", "check ps1.fetch.pcsx_catalog in blazium-toolchain.yml")
	}
	var man buildFile
	u := fmt.Sprintf("%smanifest-%d.json", infoBase, cat.Builds[0].ID)
	if err := g.GetJSON(ctx, u, &man); err != nil {
		return "", err
	}
	path := strings.TrimSpace(man.Path)
	if path == "" {
		return "", report.Fail(fmt.Sprintf("catalog build %d has no path", cat.Builds[0].ID), "check ps1.fetch.pcsx_info_base")
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		if err := settings.CheckFetchURL(path); err != nil {
			return "", err
		}
		return path, nil
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return settings.Current().PS1.Fetch.DistribRoot + path, nil
}

func (t *Tool) jsonGet() jsonGetter {
	if t.JSON != nil {
		return t.JSON
	}
	return fetch.HTTP{}
}
