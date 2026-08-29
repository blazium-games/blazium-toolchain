package iso

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode"
)

var reservedRoots = map[string]bool{
	"VIDEO_TS": true,
	"AUDIO_TS": true,
}

func parseExtraSpec(s string) Extra {
	s = strings.TrimSpace(s)
	if s == "" {
		return Extra{}
	}
	host, disc := splitHostDisc(s)
	return Extra{Host: host, Disc: disc}
}

func splitHostDisc(s string) (host, disc string) {
	i := strings.LastIndex(s, ":")
	if i <= 0 {
		return s, ""
	}
	// Windows drive: C:\foo or C:/foo with no dest.
	if i == 1 && len(s) > 2 && (s[2] == '\\' || s[2] == '/') && unicode.IsLetter(rune(s[0])) {
		return s, ""
	}
	host, dest := s[:i], s[i+1:]
	dest = strings.TrimPrefix(filepath.ToSlash(dest), "/")
	if dest == "" {
		return host, ""
	}
	return host, dest
}

func cleanDiscPath(p string) (string, error) {
	p = strings.TrimSpace(filepath.ToSlash(p))
	p = strings.TrimPrefix(p, "/")
	if p == "" {
		return "", fmt.Errorf("iso: empty disc path")
	}
	parts := strings.Split(p, "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			return "", fmt.Errorf("iso: disc path %q escapes root", p)
		}
		out = append(out, part)
	}
	if len(out) == 0 {
		return "", fmt.Errorf("iso: empty disc path")
	}
	return path.Join(out...), nil
}

func reservedDisc(disc string) error {
	top := strings.ToUpper(strings.Split(disc, "/")[0])
	if reservedRoots[top] {
		return fmt.Errorf("iso: extras cannot overwrite %s", top)
	}
	return nil
}

func defaultDiscName(host string) string {
	return filepath.Base(filepath.Clean(host))
}

func collectExtras(m DiscMeta) ([]Extra, error) {
	seen := map[string]bool{}
	var out []Extra
	add := func(e Extra) error {
		if e.Host == "" {
			return fmt.Errorf("iso: extra host path is empty")
		}
		st, err := os.Stat(e.Host)
		if err != nil {
			return fmt.Errorf("iso: extra %s: %w", e.Host, err)
		}
		disc := e.Disc
		if disc == "" {
			disc = defaultDiscName(e.Host)
		}
		disc, err = cleanDiscPath(disc)
		if err != nil {
			return err
		}
		if err := reservedDisc(disc); err != nil {
			return err
		}
		key := strings.ToUpper(disc)
		if seen[key] {
			return fmt.Errorf("iso: duplicate disc path %s", disc)
		}
		seen[key] = true
		e.Disc = disc
		e.Recursive = e.Recursive || m.Recursive
		if st.IsDir() && !e.Recursive {
			e.Recursive = false
		}
		out = append(out, e)
		return nil
	}
	if m.ExtrasDir != "" {
		entries, err := os.ReadDir(m.ExtrasDir)
		if err != nil {
			return nil, fmt.Errorf("iso: extras_dir: %w", err)
		}
		for _, ent := range entries {
			if err := add(Extra{Host: filepath.Join(m.ExtrasDir, ent.Name()), Disc: ent.Name(), Recursive: m.Recursive}); err != nil {
				return nil, err
			}
		}
	}
	for _, e := range m.Extras {
		if err := add(e); err != nil {
			return nil, err
		}
	}
	return out, nil
}
