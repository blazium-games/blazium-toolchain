// Package safepath keeps extracted and overlay files inside a root directory.
package safepath

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Join returns root/rel only when the result stays under root.
func Join(root, rel string) (string, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	rel = filepath.Clean(rel)
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes %s", root)
	}
	target := filepath.Join(root, rel)
	if err := Under(root, target); err != nil {
		return "", err
	}
	return target, nil
}

// Under reports whether candidate is inside root (after Abs).
func Under(root, candidate string) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	candidate, err = filepath.Abs(candidate)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return fmt.Errorf("path escapes %s", root)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path escapes %s", root)
	}
	return nil
}

// ResolveUnder follows symlinks on candidate and requires the result to stay under root.
func ResolveUnder(root, candidate string) error {
	if err := Under(root, candidate); err != nil {
		return err
	}
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return Under(root, resolved)
}
