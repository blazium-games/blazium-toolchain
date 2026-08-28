package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

const (
	VendorDir   = "Blazium"
	ProductDir  = "blazium-toolchain"
	StateFile   = "state.json"
)

// DefaultPrefix is %LOCALAPPDATA%\Blazium\blazium-toolchain on Windows,
// ~/.local/share/blazium-toolchain elsewhere.
func DefaultPrefix() string {
	if p := os.Getenv("BLAZIUM_TOOLCHAIN_PREFIX"); p != "" {
		return p
	}
	if runtime.GOOS == "windows" {
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			base = os.TempDir()
		}
		return filepath.Join(base, VendorDir, ProductDir)
	}
	home, _ := os.UserHomeDir()
	if home == "" {
		return filepath.Join(os.TempDir(), ProductDir)
	}
	return filepath.Join(home, ".local", "share", ProductDir)
}

// PlatformDir is <prefix>/<platformID>/.
func PlatformDir(prefix, platformID string) string {
	if prefix == "" {
		prefix = DefaultPrefix()
	}
	return filepath.Join(prefix, platformID)
}

// State is persisted after setup so env/status need no network.
type State struct {
	Platform string            `json:"platform"`
	Profile  string            `json:"profile"`
	Env      map[string]string `json:"env"`
	Notes    []string          `json:"notes,omitempty"`
}

func StatePath(prefix, platformID string) string {
	return filepath.Join(PlatformDir(prefix, platformID), StateFile)
}

func WriteState(prefix, platformID string, st State) error {
	dir := PlatformDir(prefix, platformID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(StatePath(prefix, platformID), append(data, '\n'), 0o644)
}

func ReadState(prefix, platformID string) (State, error) {
	var st State
	data, err := os.ReadFile(StatePath(prefix, platformID))
	if err != nil {
		return st, err
	}
	err = json.Unmarshal(data, &st)
	return st, err
}
