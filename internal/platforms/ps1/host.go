package ps1

import (
	"fmt"
	"runtime"

	"github.com/blazium-games/blazium-toolchain/internal/platforms"
)

// hostOK reports whether goos can run PS1 host tools (gcc fetch, cmake, emu).
func hostOK(goos string) bool {
	return goos == "windows" || goos == "linux"
}

// HostSupported is true on Windows and Linux only.
func HostSupported() bool {
	return hostOK(runtime.GOOS)
}

func requireHost() error {
	if HostSupported() {
		return nil
	}
	return fmt.Errorf("%w: ps1 is Windows/Linux only (this host is %s)", platforms.ErrPlanned, runtime.GOOS)
}
