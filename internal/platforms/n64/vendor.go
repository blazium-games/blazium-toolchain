package n64

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func vendorRoots(prefix string) []string {
	var roots []string
	add := func(p string) {
		if p != "" && !forbiddenUltra(p) {
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
	return roots
}

func hostNames(base string) []string {
	if runtime.GOOS == "windows" {
		return []string{base, base + ".exe"}
	}
	return []string{base}
}

func walkNamed(root string, names ...string) string {
	if root == "" || forbiddenUltra(root) {
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
		if forbiddenUltra(path) {
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
	if p == "" || forbiddenUltra(p) {
		return false
	}
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func dirExists(p string) bool {
	if p == "" || forbiddenUltra(p) {
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
	if forbiddenUltra(p) {
		return ""
	}
	return p
}

func forbiddenUltra(p string) bool {
	n := strings.ToLower(filepath.Clean(p))
	slash := filepath.ToSlash(n)
	return strings.Contains(slash, "/ultra/") || strings.HasSuffix(slash, "/ultra")
}

func siblingN64Stuff() []string {
	var out []string
	seen := map[string]bool{}
	add := func(p string) {
		if p == "" || forbiddenUltra(p) {
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
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		add(filepath.Join(dir, "..", "n64_stuff"))
		add(filepath.Join(dir, "n64_stuff"))
		add(filepath.Join(dir, "..", "..", "n64_stuff"))
	}
	if wd, err := os.Getwd(); err == nil {
		add(filepath.Join(wd, "n64_stuff"))
		add(filepath.Join(wd, "..", "n64_stuff"))
		add(filepath.Join(wd, "..", "..", "n64_stuff"))
	}
	return out
}

func lookBash() string {
	for _, p := range []string{
		`C:\Program Files\Git\bin\bash.exe`,
		`C:\Program Files\Git\usr\bin\bash.exe`,
		`C:\msys64\usr\bin\bash.exe`,
		`C:\msys64\ucrt64\bin\bash.exe`,
	} {
		if fileExists(p) {
			return p
		}
	}
	p := lookFile("bash")
	if p != "" && !strings.Contains(strings.ToLower(p), `system32\bash`) && !strings.Contains(strings.ToLower(p), `/system32/bash`) {
		return p
	}
	return ""
}

func siblingLibdragon() string {
	for _, stuff := range siblingN64Stuff() {
		lib := filepath.Join(stuff, "libdragon")
		if fileExists(filepath.Join(lib, "n64.mk")) {
			return absOr(lib)
		}
	}
	return ""
}
