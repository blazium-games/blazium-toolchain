package n64

import "github.com/blazium-games/blazium-toolchain/internal/execx"

func lookPath(name string) (string, error) {
	return execx.Host{}.LookPath(name)
}
