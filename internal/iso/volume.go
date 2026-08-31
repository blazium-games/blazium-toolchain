package iso

import (
	"strings"

	"github.com/blazium-games/blazium-toolchain/internal/settings"
)

const (
	SchemaV1   = "blazium.interdvd.meta/v1"
	SectorSize = 2048
)

// SanitizeVolumeID returns an ISO9660-style volume label: A-Z, 0-9, _, max 32.
// Empty or junk input returns "".
func SanitizeVolumeID(s string) string {
	return settings.CleanVolume(s)
}

// VolumeOr sanitizes s, then fallback, then "DISC".
func VolumeOr(s, fallback string) string {
	if out := SanitizeVolumeID(s); out != "" {
		return out
	}
	if out := SanitizeVolumeID(fallback); out != "" {
		return out
	}
	return "DISC"
}

func padUpper(s string, n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = ' '
	}
	u := strings.ToUpper(s)
	if len(u) > n {
		u = u[:n]
	}
	copy(b, u)
	return b
}

func padSpace(s string, n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = ' '
	}
	if len(s) > n {
		s = s[:n]
	}
	copy(b, s)
	return b
}
