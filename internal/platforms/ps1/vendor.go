package ps1

import (
	"os"
	"path/filepath"
	"runtime"
)

// vendorRoots are directories this GPL project may use to contain toolchain bits.
// Order: cache prefix, then a third_party tree next to the binary or cwd.
func vendorRoots(prefix string) []string {
	var roots []string
	add := func(p string) {
		if p != "" {
			roots = append(roots, p)
		}
	}
	if prefix != "" {
		add(filepath.Join(prefix, ID))
		add(filepath.Join(prefix, "third_party", ID))
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		add(filepath.Join(dir, "third_party", ID))
	}
	if wd, err := os.Getwd(); err == nil {
		add(filepath.Join(wd, "third_party", ID))
		add(filepath.Join(wd, "..", "third_party", ID))
		add(filepath.Join(wd, "..", "..", "third_party", ID))
	}
	return roots
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
		want[n] = true
	}
	var found string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
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

func siblingSDKRoots() []string {
	wd, err := os.Getwd()
	if err != nil {
		return nil
	}
	return []string{
		filepath.Join(wd, "PSn00bSDK"),
		filepath.Join(wd, "..", "PSn00bSDK"),
		filepath.Join(wd, "..", "..", "PSn00bSDK"),
	}
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
