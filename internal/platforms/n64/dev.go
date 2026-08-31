package n64

import (
	"context"
	"io"
	"path/filepath"
	"runtime"

	"github.com/blazium-games/blazium-toolchain/internal/cache"
	"github.com/blazium-games/blazium-toolchain/internal/report"
	"github.com/blazium-games/blazium-toolchain/internal/settings"
)

// officialAresURL is the spawn-only Ares binary zip for this host.
// Never a PIF/BIOS dump. Linux has no GitHub zip in the v148 release.
func officialAresURL() string {
	return officialAresURLFor(runtime.GOOS, runtime.GOARCH)
}

// aresRunArgs are v148 --dump-all-settings keys (not the UI labels).
// expansionPak is true for the locked 8 MiB machine; false only for --rdram 4.
func aresRunArgs(rom string, expansionPak bool) []string {
	pak := "true"
	if !expansionPak {
		pak = "false"
	}
	return []string{
		"--system", "Nintendo 64",
		"--no-file-prompt",
		"--kiosk",
		"--setting", "General/HomebrewMode=true",
		"--setting", "General/AutoSaveMemory=true",
		"--setting", "Nintendo64/ExpansionPak=" + pak,
		rom,
	}
}

func officialAresURLFor(goos, goarch string) string {
	if goos == "windows" && (goarch == "amd64" || goarch == "386") {
		return settings.Current().N64.Fetch.AresWindows
	}
	return ""
}

func (t *Tool) ensureDev(ctx context.Context, prefix string, log io.Writer) error {
	env, _ := t.discover(prefix)
	if fileExists(env["ARES_EXE"]) {
		return nil
	}
	url := officialAresURL()
	if url == "" {
		report.Line(log, report.Note, "no official Ares zip for this host; install ares and set ARES_EXE")
		return nil
	}
	dest := filepath.Join(cache.PlatformDir(prefix, ID), "ares")
	if walkNamed(dest, hostNames("ares")...) != "" {
		return nil
	}
	report.Linef(log, report.Fetching, "Ares  %s", url)
	if err := t.fetcher().FetchZip(ctx, url, "", dest, log); err != nil {
		if walkNamed(dest, hostNames("ares")...) != "" {
			report.Line(log, report.Skip, "Ares extract incomplete but executable found")
			return nil
		}
		return report.Fail("ares: "+err.Error(), "blazium-toolchain n64 setup --profile dev")
	}
	if walkNamed(dest, hostNames("ares")...) == "" {
		return report.Fail("Ares zip extracted but ares executable not found in "+dest, "")
	}
	return nil
}
