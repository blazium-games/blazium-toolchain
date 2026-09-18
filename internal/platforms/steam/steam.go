package steam

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/blazium-games/blazium-toolchain/internal/cache"
	"github.com/blazium-games/blazium-toolchain/internal/embedfs"
	"github.com/blazium-games/blazium-toolchain/internal/execx"
	"github.com/blazium-games/blazium-toolchain/internal/fetch"
	"github.com/blazium-games/blazium-toolchain/internal/platforms"
	"github.com/blazium-games/blazium-toolchain/internal/report"
)

const ID = "steam"

// ZipFetcher downloads and unpacks an official archive.
type ZipFetcher interface {
	FetchZip(ctx context.Context, url, sha256, destDir string, log io.Writer) error
}

// Tool caches Valve steamcmd for blazium-cli deploy. It does not upload.
type Tool struct {
	Runner  execx.Runner
	Fetcher ZipFetcher
}

func New() *Tool { return &Tool{} }

func (t *Tool) Info() platforms.Info {
	return platforms.Info{
		ID:          ID,
		Name:        "SteamCMD",
		Status:      platforms.StatusSupported,
		Commands:    []string{"setup", "env", "status"},
		Description: "Caches Valve steamcmd for blazium-cli deploy steam upload. Does not login or push builds.",
	}
}

func (t *Tool) fetcher() ZipFetcher {
	if t.Fetcher != nil {
		return t.Fetcher
	}
	return fetch.HTTP{}
}

func (t *Tool) runner() execx.Runner {
	if t.Runner != nil {
		return t.Runner
	}
	return execx.Host{}
}

func (t *Tool) Setup(ctx context.Context, opts platforms.SetupOptions) error {
	env := t.discover(opts.Prefix)
	if env["STEAMCMD"] == "" && !opts.Offline {
		if err := t.ensureSteamcmd(ctx, opts.Prefix, opts.Stdout); err != nil {
			return err
		}
		env = t.discover(opts.Prefix)
	}
	if opts.Offline && env["STEAMCMD"] == "" {
		return report.Offline("steamcmd missing under prefix", "drop --offline or run blazium-toolchain steam setup")
	}
	if env["STEAMCMD"] == "" {
		return report.Missing("steam setup needs steamcmd", "blazium-toolchain steam setup")
	}
	if !opts.Offline {
		t.bootstrap(ctx, env["STEAMCMD"], opts.Stdout, opts.Stderr)
	}
	st := cache.State{
		Platform: ID,
		Profile:  opts.Profile,
		Env:      env,
		Notes:    []string{"steamcmd cached under steam/steamcmd"},
	}
	if err := cache.WriteState(opts.Prefix, ID, st); err != nil {
		return err
	}
	report.Setup(opts.Stdout, ID, "steamcmd", cache.PlatformDir(opts.Prefix, ID), env)
	return nil
}

func (t *Tool) Env(opts platforms.CommonOptions) (platforms.EnvMap, error) {
	return t.discover(opts.Prefix), nil
}

func (t *Tool) Status(opts platforms.CommonOptions) (map[string]any, error) {
	env := t.discover(opts.Prefix)
	ready := env["STEAMCMD"] != ""
	return map[string]any{
		"platform": ID,
		"ready":    ready,
		"steamcmd": env["STEAMCMD"],
	}, nil
}

func (t *Tool) Build(_ context.Context, _ platforms.BuildOptions) error {
	return report.Usage("steam has no build; use blazium-cli deploy steam upload")
}

func (t *Tool) Run(_ context.Context, _ platforms.RunOptions) error {
	return report.Usage("steam has no run; use blazium-cli deploy steam upload")
}

func (t *Tool) ISO(_ context.Context, _ platforms.ISOOptions) error {
	return report.Usage("steam has no iso")
}

func steamcmdDest(prefix string) string {
	return filepath.Join(cache.PlatformDir(prefix, ID), "steamcmd")
}

func steamcmdPins(goos string) []embedfs.ZipPin {
	p := embedfs.MustPins()
	if p.Steam == nil {
		return nil
	}
	return p.Steam[goos]
}

func (t *Tool) ensureSteamcmd(ctx context.Context, prefix string, log io.Writer) error {
	pins := steamcmdPins(runtime.GOOS)
	if len(pins) == 0 {
		return report.Fail("no steamcmd pin for "+runtime.GOOS, "add a steam pin in pins.json")
	}
	dest := steamcmdDest(prefix)
	var last error
	for _, a := range pins {
		if err := t.fetcher().FetchZip(ctx, a.URL, a.SHA256, dest, log); err != nil {
			last = err
			continue
		}
		if walkSteamcmd(dest) != "" {
			return nil
		}
		last = report.Fail("steamcmd archive extracted but binary not found", "inspect "+dest)
	}
	if last != nil {
		return last
	}
	return report.Missing("steamcmd fetch failed", "blazium-toolchain steam setup")
}

func (t *Tool) bootstrap(ctx context.Context, bin string, stdout, stderr io.Writer) {
	if bin == "" {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	report.Line(stdout, report.Installing, "steamcmd +quit (first-run update)")
	_ = t.runner().Run(ctx, bin, []string{"+quit"}, stdout, stderr)
}

func (t *Tool) discover(prefix string) platforms.EnvMap {
	env := platforms.EnvMap{}
	if bin := walkSteamcmd(steamcmdDest(prefix)); bin != "" {
		env["STEAMCMD"] = absOr(bin)
	}
	return env
}

func walkSteamcmd(root string) string {
	prefer := []string{"steamcmd.sh", "steamcmd.exe", "steamcmd"}
	if runtime.GOOS == "windows" {
		prefer = []string{"steamcmd.exe", "steamcmd"}
	}
	for _, name := range prefer {
		p := filepath.Join(root, name)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return walkNamed(root, prefer...)
}

func walkNamed(root string, names ...string) string {
	if root == "" {
		return ""
	}
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		return ""
	}
	want := make(map[string]bool, len(names))
	for _, n := range names {
		want[strings.ToLower(n)] = true
	}
	var found string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if want[strings.ToLower(d.Name())] {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func absOr(path string) string {
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
}
