package n64

import (
	"fmt"
	"os"
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
	return nil
}

func isZ64(path string) bool {
	return verifyZ64(path) == nil
}
