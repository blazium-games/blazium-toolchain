// Package guest embeds the MIT Blazium PS1 CMake stub shipped with this CLI.
// Game developers never need the editor source tree; ps1 setup installs these
// files into the toolchain prefix and ps1 build uses them when --src is omitted.
package guest

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// CookABI is the cooked blob layout version. Bump together with the editor cooker
// and guest/ps1/runtime/CMakeLists.txt (BLAZIUM_PS1_COOK_ABI).
const CookABI = 20

// RuntimeNames are the files Install writes from the embed.
var RuntimeNames = []string{
	"CMakeLists.txt",
	"main.cpp",
	"script_vm.cpp",
	"script_vm.h",
	"fmv_play.cpp",
	"fmv_play.h",
}

//go:embed runtime
var files embed.FS

// Install writes the guest stub into destDir (prefix/ps1/guest/runtime).
func Install(destDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	return fs.WalkDir(files, "runtime", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "runtime" {
			return nil
		}
		rel := strings.TrimPrefix(path, "runtime/")
		target := filepath.Join(destDir, filepath.FromSlash(rel))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, readErr := files.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
