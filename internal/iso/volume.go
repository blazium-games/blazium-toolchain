package iso

import (
	"strings"
	"unicode"
)

const (
	DefaultVolume     = "BLAZIUM_DVD"
	DefaultPreparer   = "BLAZIUM TOOLCHAIN"
	DefaultApplication = "BLAZIUM INTER-DVD"
	DefaultProvider   = "BLAZIUM INTER-DVD"
	DefaultSystem     = "DVD-VIDEO"
	SchemaV1          = "blazium.interdvd.meta/v1"
	SectorSize        = 2048
)

// SanitizeVolumeID returns an ISO9660-style volume label: A–Z, 0–9, _, max 32.
// Empty input becomes BLAZIUM_DVD.
func SanitizeVolumeID(s string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		switch {
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
		case unicode.IsSpace(r) || r == '-' || r == '.':
			if b.Len() > 0 {
				b.WriteByte('_')
			}
		}
		if b.Len() >= 32 {
			break
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return DefaultVolume
	}
	return out
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
