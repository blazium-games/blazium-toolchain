package interdvd

import (
	"strings"
	"unicode"

	"github.com/blazium-games/blazium-toolchain/internal/iso"
)

func extraFromSpec(s string) iso.Extra {
	s = strings.TrimSpace(s)
	if s == "" {
		return iso.Extra{}
	}
	i := strings.LastIndex(s, ":")
	if i <= 0 {
		return iso.Extra{Host: s}
	}
	if i == 1 && len(s) > 2 && (s[2] == '\\' || s[2] == '/') && unicode.IsLetter(rune(s[0])) {
		return iso.Extra{Host: s}
	}
	return iso.Extra{Host: s[:i], Disc: strings.TrimPrefix(strings.ReplaceAll(s[i+1:], "\\", "/"), "/")}
}
