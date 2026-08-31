package ps1

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/blazium-games/blazium-toolchain/internal/cache"
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

func findVendorFile(prefix string, rels ...string) string {
	var expanded []string
	for _, rel := range rels {
		dir, name := filepath.Split(rel)
		for _, n := range hostNames(name) {
			expanded = append(expanded, filepath.Join(dir, n))
		}
	}
	for _, root := range vendorRoots(prefix) {
		for _, rel := range expanded {
			p := filepath.Join(root, rel)
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				if abs, err := filepath.Abs(p); err == nil {
					return abs
				}
				return p
			}
		}
	}
	return ""
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
		want[nameKey(n)] = true
	}
	var found string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if want[nameKey(d.Name())] {
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

func findLibpsn00b(roots ...string) string {
	var fallback string
	for _, root := range roots {
		if root == "" {
			continue
		}
		if st, err := os.Stat(root); err != nil || !st.IsDir() {
			continue
		}
		var preferred string
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil || !d.IsDir() || d.Name() != "libpsn00b" {
				return nil
			}
			if filepath.Base(filepath.Dir(path)) == "lib" {
				preferred = path
				return filepath.SkipAll
			}
			if fallback == "" {
				fallback = path
			}
			return nil
		})
		if preferred != "" {
			if abs, err := filepath.Abs(preferred); err == nil {
				return abs
			}
			return preferred
		}
	}
	if fallback == "" {
		return ""
	}
	if abs, err := filepath.Abs(fallback); err == nil {
		return abs
	}
	return fallback
}

func walkDirNamed(root string, names ...string) string {
	if root == "" {
		return ""
	}
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		return ""
	}
	want := make(map[string]bool, len(names))
	for _, n := range names {
		want[n] = true
	}
	var found string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		if want[d.Name()] {
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

func nameKey(n string) string {
	if runtime.GOOS == "windows" {
		return strings.ToLower(n)
	}
	return n
}

func findVendorDir(prefix string, rels ...string) string {
	for _, root := range vendorRoots(prefix) {
		for _, rel := range rels {
			p := filepath.Join(root, rel)
			if st, err := os.Stat(p); err == nil && st.IsDir() {
				if abs, err := filepath.Abs(p); err == nil {
					return abs
				}
				return p
			}
		}
	}
	return ""
}
