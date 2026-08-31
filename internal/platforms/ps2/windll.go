package ps2

import (
	"context"
	"debug/pe"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/blazium-games/blazium-toolchain/internal/fetch"
	"github.com/blazium-games/blazium-toolchain/internal/report"
	"github.com/blazium-games/blazium-toolchain/internal/settings"
)

var windowsHostDLLs = []string{
	"libwinpthread-1.dll",
	"libiconv-2.dll",
	"libgmp-10.dll",
	"libisl-23.dll",
	"libmpc-3.dll",
	"libmpfr-6.dll",
	"libzstd.dll",
	"libgcc_s_dw2-1.dll",
}

func gnuwin32IconvZip() string {
	return settings.Current().PS2.Fetch.IconvZip
}

func msys32RuntimeZst() []string {
	return settings.Current().PS2.Fetch.Msys32
}

func peMachine(path string) (uint16, error) {
	f, err := pe.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return f.FileHeader.Machine, nil
}

func ensureWindowsHostDLLs(gccPath string, log io.Writer) error {
	if runtime.GOOS != "windows" || gccPath == "" || !fileExists(gccPath) {
		return nil
	}
	want, err := peMachine(gccPath)
	if err != nil {
		if log != nil {
			report.Line(log, report.Skip, "could not read EE gcc PE header; host DLLs not staged")
		}
		return nil
	}
	dir := filepath.Dir(gccPath)
	stageDirs := hostDLLStageDirs(gccPath)
	var missing []string
	for _, name := range windowsHostDLLs {
		if stagedMatching(dir, name, want) {
			if err := copyFileToDirs(filepath.Join(dir, name), stageDirs); err != nil {
				return err
			}
			continue
		}
		src := findHostDLL(name, want)
		if src == "" && name == "libiconv-2.dll" {
			src = fetchIconvDLL(dir, want, log)
		}
		if src == "" {
			missing = append(missing, name)
			continue
		}
		if err := copyFileToDirs(src, stageDirs); err != nil {
			return err
		}
		if log != nil {
			report.Line(log, report.Staged, name)
		}
	}
	if want == 0x14c {
		runtimeDir := filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(dir))), "downloads", "mingw32-runtime")
		if err := fetchMsys32Runtime(dir, want, log); err != nil && log != nil {
			report.Line(log, report.Skip, "msys32 runtime: "+err.Error())
		}
		var still []string
		for _, name := range windowsHostDLLs {
			src := findDLLInTree(runtimeDir, name, want)
			if src == "" {
				if !stagedMatching(dir, name, want) {
					still = append(still, name)
				}
				continue
			}
			if err := copyFileToDirs(src, stageDirs); err != nil {
				return err
			}
			if log != nil {
				report.Line(log, report.Staged, name)
			}
		}
		missing = still
	}
	if len(missing) == 0 {
		return nil
	}
	arch := "i686"
	if want == 0x8664 {
		arch = "x86_64"
	}
	return report.Missing(fmt.Sprintf("EE gcc is %s and needs %s beside %s (official Windows tarball omits these MinGW DLLs)", arch, strings.Join(missing, ", "), dir), "")
}

func findHostDLL(name string, want uint16) string {
	var roots []string
	if exe, err := os.Executable(); err == nil {
		roots = append(roots, filepath.Dir(exe))
	}
	if pathEnv := os.Getenv("PATH"); pathEnv != "" {
		roots = append(roots, filepath.SplitList(pathEnv)...)
	}
	for _, root := range roots {
		if root == "" {
			continue
		}
		cand := filepath.Join(root, name)
		if !fileExists(cand) {
			continue
		}
		got, err := peMachine(cand)
		if err == nil && got == want {
			return cand
		}
	}
	return ""
}

func fetchIconvDLL(gccBinDir string, want uint16, log io.Writer) string {
	if want != 0x14c {
		return ""
	}
	cacheDir := filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(gccBinDir))), "downloads", "mingw-iconv")
	if log != nil {
		report.Line(log, report.Fetching, "32-bit libiconv for EE gcc")
	}
	if err := (fetch.HTTP{}).FetchZip(context.Background(), gnuwin32IconvZip(), "", cacheDir, log); err != nil {
		return ""
	}
	var found string
	_ = filepath.WalkDir(cacheDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		base := strings.ToLower(d.Name())
		if base != "libiconv-2.dll" && base != "libiconv2.dll" {
			return nil
		}
		got, err := peMachine(path)
		if err == nil && got == want {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func stagedMatching(dir, name string, want uint16) bool {
	dest := filepath.Join(dir, name)
	if !fileExists(dest) {
		return false
	}
	got, err := peMachine(dest)
	if err != nil || got != want {
		_ = os.Remove(dest)
		return false
	}
	return true
}

func hostDLLStageDirs(gccPath string) []string {
	dirs := []string{filepath.Dir(gccPath)}
	root := filepath.Dir(filepath.Dir(gccPath))
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		n := strings.ToLower(d.Name())
		if n == "cc1.exe" || n == "cc1plus.exe" || n == "cc1" || n == "cc1plus" {
			dirs = append(dirs, filepath.Dir(path))
		}
		return nil
	})
	return dirs
}

func copyFileToDirs(src string, dirs []string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	base := filepath.Base(src)
	if strings.EqualFold(base, "libiconv2.dll") {
		base = "libiconv-2.dll"
	}
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, base), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func fetchMsys32Runtime(gccBinDir string, want uint16, log io.Writer) error {
	if want != 0x14c {
		return nil
	}
	cacheDir := filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(gccBinDir))), "downloads", "mingw32-runtime")
	for _, url := range msys32RuntimeZst() {
		if log != nil {
			report.Line(log, report.Fetching, filepath.Base(url))
		}
		if err := (fetch.HTTP{}).FetchZip(context.Background(), url, "", cacheDir, log); err != nil {
			return err
		}
	}
	return nil
}

func findDLLInTree(root, name string, want uint16) string {
	if !dirExists(root) {
		return ""
	}
	var found string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.EqualFold(d.Name(), name) {
			return nil
		}
		got, err := peMachine(path)
		if err == nil && got == want {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	return found
}
