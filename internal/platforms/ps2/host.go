package ps2

import (
	"fmt"
	"runtime"

	"github.com/blazium-games/blazium-toolchain/internal/platforms"
)

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
	return fmt.Errorf("%w: ps2 is Windows/Linux only (this host is %s)", platforms.ErrPlanned, runtime.GOOS)
}
