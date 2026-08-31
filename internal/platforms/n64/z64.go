package n64

import (
	"fmt"
	"os"

	"github.com/blazium-games/blazium-toolchain/internal/report"
	"github.com/blazium-games/blazium-toolchain/internal/settings"
)

// z64Magic is the big-endian N64 ROM identifier (0x80371240).
var z64Magic = []byte{0x80, 0x37, 0x12, 0x40}

func verifyZ64(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(raw) < 64 {
		return fmt.Errorf("z64 too small: %s (%d bytes)", path, len(raw))
	}
	if raw[0] == 'M' && raw[1] == 'Z' {
		return fmt.Errorf("%s is a Windows PE, not a .z64", path)
	}
	if len(raw) >= 4 && raw[0] == 0x7f && raw[1] == 'E' && raw[2] == 'L' && raw[3] == 'F' {
		return fmt.Errorf("%s is an ELF; product must be a big-endian .z64", path)
	}
	if raw[0] != z64Magic[0] || raw[1] != z64Magic[1] || raw[2] != z64Magic[2] || raw[3] != z64Magic[3] {
		return fmt.Errorf("%s is not a big-endian .z64 (got %02x %02x %02x %02x)", path, raw[0], raw[1], raw[2], raw[3])
	}
	if max := settings.Current().N64.CartMax; max > 0 && int64(len(raw)) > max {
		return report.Fail(fmt.Sprintf("%s exceeds cart_max (%d > %d bytes)", path, len(raw), max), "raise n64.cart_max in blazium-toolchain.yml or shrink the ROM")
	}
	return nil
}

func isZ64(path string) bool {
	return verifyZ64(path) == nil
}
