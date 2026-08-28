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
