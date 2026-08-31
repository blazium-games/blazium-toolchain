package interdvd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/blazium-games/blazium-toolchain/internal/cache"
	"github.com/blazium-games/blazium-toolchain/internal/embedfs"
	"github.com/blazium-games/blazium-toolchain/internal/execx"
	"github.com/blazium-games/blazium-toolchain/internal/fetch"
	"github.com/blazium-games/blazium-toolchain/internal/iso"
	"github.com/blazium-games/blazium-toolchain/internal/platforms"
	"github.com/blazium-games/blazium-toolchain/internal/report"
)

// ZipFetcher downloads and unpacks an official archive.
type ZipFetcher interface {
	FetchZip(ctx context.Context, url, sha256, destDir string, log io.Writer) error
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

func ffmpegAssetsFor(goos string) []embedfs.ZipPin {
	p := embedfs.MustPins()
	return p.InterDVD[goos]
}

// latestFFmpegPins are BtbN floating "latest" URLs. Daily autobuild tags
// expire after ~14 days; these stay valid.
func latestFFmpegPins(goos string) []embedfs.ZipPin {
	const root = "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/"
	switch goos {
	case "windows":
		return []embedfs.ZipPin{
			{ID: "ffmpeg-latest-8.1", URL: root + "ffmpeg-n8.1-latest-win64-gpl-8.1.zip", Dest: "ffmpeg"},
			{ID: "ffmpeg-latest-master", URL: root + "ffmpeg-master-latest-win64-gpl.zip", Dest: "ffmpeg"},
		}
	case "linux":
		return []embedfs.ZipPin{
			{ID: "ffmpeg-latest-8.1", URL: root + "ffmpeg-n8.1-latest-linux64-gpl-8.1.tar.xz", Dest: "ffmpeg"},
			{ID: "ffmpeg-latest-master", URL: root + "ffmpeg-master-latest-linux64-gpl.tar.xz", Dest: "ffmpeg"},
		}
	default:
		return nil
	}
}

func ffmpegFetchCandidates(goos string) []embedfs.ZipPin {
	seen := map[string]bool{}
	var out []embedfs.ZipPin
	for _, a := range append(append([]embedfs.ZipPin{}, ffmpegAssetsFor(goos)...), latestFFmpegPins(goos)...) {
		if a.URL == "" || seen[a.URL] {
			continue
		}
		seen[a.URL] = true
		out = append(out, a)
	}
	return out
}

func ffmpegDest(prefix string) string {
	return filepath.Join(cache.PlatformDir(prefix, ID), "ffmpeg")
}

func hostNames(base string) []string {
	if runtime.GOOS == "windows" {
		return []string{base, base + ".exe"}
	}
	return []string{base}
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func absOr(path string) string {
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
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
	if found == "" {
		return ""
	}
	return absOr(found)
}

func (t *Tool) discover(prefix string) map[string]string {
	env := platforms.EnvMap{
		"ISO_TOOL": "builtin",
		"SCHEMA":   iso.SchemaV1,
		"FFMPEG":   "",
		"FFPROBE":  "",
	}
	if st, err := cache.ReadState(prefix, ID); err == nil {
		for k, v := range st.Env {
			if v != "" {
				env[k] = v
			}
		}
	}
	root := cache.PlatformDir(prefix, ID)
	if env["FFMPEG"] == "" || !fileExists(env["FFMPEG"]) {
		if p := walkNamed(ffmpegDest(prefix), hostNames("ffmpeg")...); p != "" {
			env["FFMPEG"] = p
		} else if p := walkNamed(root, hostNames("ffmpeg")...); p != "" {
			env["FFMPEG"] = p
		}
	}
	if env["FFPROBE"] == "" || !fileExists(env["FFPROBE"]) {
		if p := walkNamed(ffmpegDest(prefix), hostNames("ffprobe")...); p != "" {
			env["FFPROBE"] = p
		} else if p := walkNamed(root, hostNames("ffprobe")...); p != "" {
			env["FFPROBE"] = p
		}
	}
	env["ISO_TOOL"] = "builtin"
	env["SCHEMA"] = iso.SchemaV1
	return env
}

func encodeReady(env map[string]string) bool {
	return env["FFMPEG"] != "" && fileExists(env["FFMPEG"]) && env["FFPROBE"] != "" && fileExists(env["FFPROBE"])
}

func (t *Tool) ensureFFmpeg(ctx context.Context, prefix string, log io.Writer) error {
	dest := ffmpegDest(prefix)
	if walkNamed(dest, hostNames("ffmpeg")...) != "" && walkNamed(dest, hostNames("ffprobe")...) != "" {
		return nil
	}
	assets := ffmpegFetchCandidates(runtime.GOOS)
	if len(assets) == 0 {
		return report.Missing("no interdvd ffmpeg pin for "+runtime.GOOS, "")
	}
	var last error
	for _, a := range assets {
		if log != nil {
			report.Linef(log, report.Fetching, "%s  %s", a.ID, a.URL)
		}
		if err := t.fetcher().FetchZip(ctx, a.URL, a.SHA256, dest, log); err != nil {
			last = err
			if log != nil {
				report.Line(log, report.Skip, a.ID+": "+err.Error())
			}
			continue
		}
		if walkNamed(dest, hostNames("ffmpeg")...) != "" && walkNamed(dest, hostNames("ffprobe")...) != "" {
			return nil
		}
		last = fmt.Errorf("archive extracted but ffmpeg/ffprobe not found in %s", dest)
	}
	if last != nil {
		return report.Missing("ffmpeg fetch failed: "+last.Error(), "blazium-toolchain interdvd setup")
	}
	return report.Missing("interdvd setup needs ffmpeg and ffprobe", "blazium-toolchain interdvd setup")
}

// RunTool spawns the cached ffmpeg or ffprobe binary. Exit 3 if setup was not run.
func (t *Tool) RunTool(ctx context.Context, name string, args []string, opts platforms.CommonOptions) error {
	env := t.discover(opts.Prefix)
	var bin string
	switch strings.ToLower(name) {
	case "ffmpeg":
		bin = env["FFMPEG"]
	case "ffprobe":
		bin = env["FFPROBE"]
	default:
		return report.Usage("interdvd " + name)
	}
	if bin == "" || !fileExists(bin) {
		return report.Missing("interdvd ffmpeg/ffprobe not ready", "blazium-toolchain interdvd setup")
	}
	return t.runner().Run(ctx, bin, args, opts.Stdout, opts.Stderr)
}
