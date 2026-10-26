package ps3

// Windows: ps3dev is a Linux toolchain, so the build runs in WSL. The guest
// is staged in the cache on the Windows side as usual and built where it is
// (/mnt/c/...); ps3dev lives in the WSL distribution (PS3DEV_WSL, ~/ps3dev
// by default), where setup fetches and unpacks the pinned release with the
// distribution's own curl and tar.

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/blazium-games/blazium-toolchain/internal/platforms"
	"github.com/blazium-games/blazium-toolchain/internal/report"
)

func wslExe() string {
	if root := os.Getenv("SystemRoot"); root != "" {
		p := filepath.Join(root, "System32", "wsl.exe")
		if fileExists(p) {
			return p
		}
	}
	return "wsl.exe"
}

// wslPath: C:\Users\me\x -> /mnt/c/Users/me/x
func wslPath(win string) (string, error) {
	abs := win
	if a, err := filepath.Abs(win); err == nil {
		abs = a
	}
	abs = strings.ReplaceAll(abs, `\`, "/")
	if len(abs) >= 2 && abs[1] == ':' && unicode.IsLetter(rune(abs[0])) {
		return "/mnt/" + strings.ToLower(abs[:1]) + abs[2:], nil
	}
	if strings.HasPrefix(abs, "/") {
		return abs, nil
	}
	return "", fmt.Errorf("WSL cannot reach %s (not on a drive letter)", win)
}

// shellQuote for bash: in single quotes, each ' closed, escaped and reopened
func shellQuote(s string) string {
	if s != "" && strings.IndexFunc(s, func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("/._-=+:,~", r))
	}) < 0 && !strings.HasPrefix(s, "~") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// wslDir is PS3DEV_WSL as bash sees it: "~/x" -> "$HOME/x" (quoted)
func wslDir(p string) string {
	if strings.HasPrefix(p, "~/") {
		return `"$HOME"/` + shellQuote(p[2:])
	}
	return shellQuote(p)
}

func (t *Tool) wslRun(ctx context.Context, script string, stdout, stderr io.Writer) error {
	return t.runner().Run(ctx, wslExe(), []string{"-e", "bash", "-c", script}, writerOr(stdout), writerOr(stderr))
}

// wslReady: ps3dev's PPU compiler and make are there in WSL.
func (t *Tool) wslReady(ctx context.Context, env map[string]string) bool {
	d := wslDir(env["PS3DEV_WSL"])
	return t.wslRun(ctx, "test -x "+d+"/ppu/bin/ppu-g++ && test -f "+d+"/ppu_rules && command -v make >/dev/null", nil, nil) == nil
}

func (t *Tool) setupWSL(ctx context.Context, opts platforms.SetupOptions, env map[string]string, log io.Writer) error {
	if err := t.wslRun(ctx, "true", nil, nil); err != nil {
		return report.Missing("WSL (the PS3 build runs ps3dev, a Linux toolchain, in it)",
			"install WSL with Ubuntu (wsl --install), then run ps3 setup again")
	}
	if t.wslReady(ctx, env) {
		env["WSL_READY"] = "1"
		return nil
	}
	if opts.Offline {
		return report.Missing("ps3dev in WSL ("+env["PS3DEV_WSL"]+")", "run ps3 setup without --offline, or set PS3DEV_WSL")
	}
	d := wslDir(env["PS3DEV_WSL"])
	script := strings.Join([]string{
		"set -e",
		"command -v curl >/dev/null || { echo 'WSL has no curl: sudo apt install curl make' >&2; exit 1; }",
		"tmp=$(mktemp -d)",
		"curl -fL --retry 3 -o \"$tmp/ps3dev.tar.gz\" " + shellQuote(ps3devURL),
		"echo " + shellQuote(ps3devSHA256+"  ") + "\"$tmp/ps3dev.tar.gz\" | sha256sum -c -",
		"mkdir -p " + d,
		"tar -xzf \"$tmp/ps3dev.tar.gz\" --strip-components=1 -C " + d,
		"rm -rf \"$tmp\"",
	}, "\n")
	if log != nil {
		report.Line(log, report.Fetching, ps3devURL+" (in WSL, to "+env["PS3DEV_WSL"]+")")
	}
	if err := t.wslRun(ctx, script, log, opts.Stderr); err != nil {
		return report.Fail("ps3dev did not install in WSL: "+err.Error(), "check WSL's network, or install ps3dev there and set PS3DEV_WSL")
	}
	if !t.wslReady(ctx, env) {
		return report.Missing("make, or ps3dev's libraries, in WSL", "in WSL: sudo apt install make libelf1 libgmp10 libssl3")
	}
	env["WSL_READY"] = "1"
	return nil
}
