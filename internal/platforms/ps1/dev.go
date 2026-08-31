package ps1

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/blazium-games/blazium-toolchain/internal/cache"
	"github.com/blazium-games/blazium-toolchain/internal/report"
)

func needsDev(profile string) bool {
	return profile == "dev" || profile == "iso"
}

func destReady(env map[string]string) bool {
	return compileReady(env) && fileExists(env["OPENBIOS"]) && fileExists(env["PCSX_EXE"])
}

func profileReady(profile string, env map[string]string) bool {
	if needsDev(profile) {
		return destReady(env)
	}
	return compileReady(env)
}

func (t *Tool) ensureDev(ctx context.Context, prefix string, log io.Writer) error {
	env, _ := t.discover(prefix)
	if destReady(env) {
		return nil
	}
	plat := cache.PlatformDir(prefix, ID)
	dest := filepath.Join(plat, "pcsx-redux")
	if !pcsxPresent(dest) && !fileExists(env["PCSX_EXE"]) {
		if runtime.GOOS != "windows" && t.CLIURL == "" {
			return report.Missing("pcsx-redux AppDistrib is Windows-only", "install pcsx-redux on PATH or set PCSX_EXE; put OpenBIOS in prefix/ps1/openbios/openbios.bin")
		}
		url, err := t.pcsxZipURL(ctx)
		if err != nil {
			return err
		}
		report.Linef(log, report.Fetching, "pcsx-redux-cli  %s", url)
		if err := t.fetcher().FetchZip(ctx, url, "", dest, log); err != nil {
			return report.Missing("pcsx-redux-cli fetch failed: "+err.Error(), "check network or set PCSX_EXE")
		}
	}
	if err := t.installOpenBIOS(prefix, log); err != nil {
		return err
	}
	return nil
}

func pcsxPresent(dest string) bool {
	return findPCSX(dest) != ""
}

func findPCSX(roots ...string) string {
	names := append(hostNames("pcsx-redux"), hostNames("pcsx-redux-cli")...)
	for _, root := range roots {
		if root == "" {
			continue
		}
		if st, err := os.Stat(root); err == nil && !st.IsDir() {
			return absOr(root)
		}
		if p := walkNamed(root, names...); p != "" {
			return p
		}
	}
	return ""
}

func findOpenBIOSFile(roots ...string) string {
	for _, root := range roots {
		if root == "" {
			continue
		}
		if st, err := os.Stat(root); err == nil && !st.IsDir() && nameKey(filepath.Base(root)) == nameKey("openbios.bin") {
			return absOr(root)
		}
		if p := walkNamed(root, "openbios.bin"); p != "" {
			return p
		}
	}
	return ""
}

func (t *Tool) installOpenBIOS(prefix string, log io.Writer) error {
	canon := filepath.Join(cache.PlatformDir(prefix, ID), "openbios", "openbios.bin")
	if fileExists(canon) {
		return nil
	}
	plat := cache.PlatformDir(prefix, ID)
	src := findOpenBIOSFile(filepath.Join(plat, "openbios"), filepath.Join(plat, "pcsx-redux"), plat)
	if src == "" {
		return report.Missing("openbios.bin not in the prefix or pcsx-redux zip", "run setup without --offline or set OPENBIOS")
	}
	if sameFile(src, canon) {
		return nil
	}
	report.Line(log, report.Installing, "OpenBIOS")
	if err := copyFile(src, canon); err != nil {
		return err
	}
	return nil
}

func copyFile(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func sameFile(a, b string) bool {
	aa, errA := filepath.Abs(a)
	bb, errB := filepath.Abs(b)
	return errA == nil && errB == nil && filepath.Clean(aa) == filepath.Clean(bb)
}

func absOr(p string) string {
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}
