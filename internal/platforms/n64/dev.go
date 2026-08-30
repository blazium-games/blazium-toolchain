package n64

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"runtime"

	"github.com/blazium-games/blazium-toolchain/internal/cache"
)

// officialAresURL is the spawn-only Ares binary zip for this host.
// Never a PIF/BIOS dump. Linux has no GitHub zip in the v148 release.
func officialAresURL() string {
	return officialAresURLFor(runtime.GOOS, runtime.GOARCH)
}

// aresRunArgs are v148 --dump-all-settings keys (not the UI labels).
func aresRunArgs(rom string) []string {
	return []string{
		"--system", "Nintendo 64",
		"--no-file-prompt",
		"--kiosk",
		"--setting", "General/HomebrewMode=true",
		"--setting", "Nintendo64/ExpansionPak=true",
		rom,
	}
}

func officialAresURLFor(goos, goarch string) string {
	if goos == "windows" && (goarch == "amd64" || goarch == "386") {
		return "https://github.com/ares-emulator/ares/releases/download/v148/ares-windows-x64.zip"
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
		if log != nil {
			fmt.Fprintln(log, "note: no official Ares zip for this host; install ares and set ARES_EXE")
		}
		return nil
	}
	dest := filepath.Join(cache.PlatformDir(prefix, ID), "ares")
	if walkNamed(dest, hostNames("ares")...) != "" {
		return nil
	}
	if log != nil {
		fmt.Fprintf(log, "fetching Ares (spawn only, no PIF) from %s\n", url)
	}
	if err := t.fetcher().FetchZip(ctx, url, "", dest, log); err != nil {
		if walkNamed(dest, hostNames("ares")...) != "" {
			if log != nil {
				fmt.Fprintf(log, "extract reported %v; ares is present, continuing\n", err)
			}
			return nil
		}
		return fmt.Errorf("ares: %w", err)
	}
	if walkNamed(dest, hostNames("ares")...) == "" {
		return fmt.Errorf("ares zip extracted but ares executable not found in %s", dest)
	}
	return nil
}
