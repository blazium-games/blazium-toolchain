package ps2

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/blazium-games/blazium-toolchain/internal/cache"
	"github.com/blazium-games/blazium-toolchain/internal/settings"
)

func vendorRoots(prefix string) []string {
	return cache.VendorRoots(prefix, ID)
}

func hostNames(base string) []string {
	if runtime.GOOS == "windows" {
		return []string{base, base + ".exe"}
	}
	return []string{base}
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
	if abs, err := filepath.Abs(found); err == nil {
		return abs
	}
	return found
}

func fileExists(p string) bool {
	if p == "" {
		return false
	}
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func dirExists(p string) bool {
	if p == "" {
		return false
	}
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func absOr(p string) string {
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}

func lookFile(name string) string {
	p, err := lookPath(name)
	if err != nil {
		return ""
	}
	return p
}

func siblingPS2Stuff() []string {
	var out []string
	seen := map[string]bool{}
	add := func(p string) {
		if p == "" {
			return
		}
		if abs, err := filepath.Abs(p); err == nil {
			p = abs
		}
		if seen[p] || !dirExists(p) {
			return
		}
		seen[p] = true
		out = append(out, p)
	}
	roots := settings.Current().PS2.SiblingRoots
	if len(roots) == 0 {
		roots = []string{"ps2_stuff"}
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		for _, name := range roots {
			add(filepath.Join(dir, "..", name))
			add(filepath.Join(dir, name))
		}
	}
	if wd, err := os.Getwd(); err == nil {
		for _, name := range roots {
			add(filepath.Join(wd, name))
			add(filepath.Join(wd, "..", name))
		}
	}
	return out
}

func findSDKRoot(root string) string {
	if root == "" {
		return ""
	}
	if dirExists(filepath.Join(root, "ee", "include")) && dirExists(filepath.Join(root, "samples")) {
		return absOr(root)
	}
	if hit := filepath.Join(root, "ps2sdk"); dirExists(filepath.Join(hit, "ee", "include")) {
		return absOr(hit)
	}
	var found string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		if d.Name() == "ps2sdk" && dirExists(filepath.Join(path, "ee", "include")) {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if found != "" {
		return absOr(found)
	}
	return ""
}
