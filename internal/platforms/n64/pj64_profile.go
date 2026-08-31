package n64

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/blazium-games/blazium-toolchain/internal/settings"
)

const (
	pj64SkipPlugin = "project64  video plugin missing (need Parallel-RDP or Angrylion, not Jabo)"
	pj64SkipInit   = "project64  video plugin failed to init"
)

func pj64UnknownRDRAM() string {
	v := strings.TrimSpace(settings.Current().N64.PJ64.UnknownRDRAM)
	if v == "" {
		return "8388608"
	}
	return v
}

func pj64ViRefresh() string {
	v := strings.TrimSpace(settings.Current().N64.PJ64.ViRefresh)
	if v == "" {
		return "1500"
	}
	return v
}

func pj64GfxDir(exe string) string {
	return filepath.Join(filepath.Dir(exe), "GFX")
}

func pj64CfgPath(exe string) string {
	return filepath.Join(filepath.Dir(exe), "Config", "Project64.cfg")
}

func pj64GfxRel(name string) string {
	return `GFX\` + name
}

func pj64ForbiddenGfx(name string) bool {
	low := strings.ToLower(name)
	return strings.Contains(low, "jabo") || strings.Contains(low, "direct3d8")
}

func pickPj64Gfx(gfxDir string) string {
	ents, err := os.ReadDir(gfxDir)
	if err != nil {
		return ""
	}
	var parallel, angrylion string
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		low := strings.ToLower(name)
		if !strings.HasSuffix(low, ".dll") {
			continue
		}
		if pj64ForbiddenGfx(name) {
			continue
		}
		if strings.Contains(low, "parallel") && parallel == "" {
			parallel = name
		}
		if strings.Contains(low, "angrylion") && angrylion == "" {
			angrylion = name
		}
	}
	if parallel != "" {
		return parallel
	}
	return angrylion
}

func writePj64TestProfile(cfgPath, gfxRel string) error {
	if gfxRel == "" || pj64ForbiddenGfx(gfxRel) {
		return fmt.Errorf("refusing empty or Jabo graphics dll")
	}
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		return err
	}
	existing := ""
	if raw, err := os.ReadFile(cfgPath); err == nil {
		existing = string(raw)
	}
	merged := mergePj64Ini(existing, map[string]map[string]string{
		"Plugin": {
			"Graphics Dll":         gfxRel,
			"Graphics Dll Default": gfxRel,
		},
		"Defaults": {
			"Unknown RDRAM Size": pj64UnknownRDRAM(),
			"Fixed Audio":        "1",
			"Audio-Sync Audio":   "1",
			"ViRefresh":          pj64ViRefresh(),
		},
	})
	return os.WriteFile(cfgPath, []byte(merged), 0o644)
}

func mergePj64Ini(src string, updates map[string]map[string]string) string {
	order := []string{}
	sections := map[string][]string{}
	cur := ""
	for _, line := range strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "[") && strings.HasSuffix(trim, "]") {
			cur = trim[1 : len(trim)-1]
			if _, ok := sections[cur]; !ok {
				order = append(order, cur)
				sections[cur] = nil
			}
			continue
		}
		if cur == "" {
			continue
		}
		sections[cur] = append(sections[cur], line)
	}
	for name, keys := range updates {
		if _, ok := sections[name]; !ok {
			order = append(order, name)
			sections[name] = nil
		}
		kept := []string{}
		seen := map[string]bool{}
		for _, line := range sections[name] {
			key := iniKey(line)
			if key != "" {
				if _, ok := keys[key]; ok {
					continue
				}
			}
			kept = append(kept, line)
		}
		var keyOrder []string
		switch name {
		case "Plugin":
			keyOrder = []string{"Graphics Dll", "Graphics Dll Default"}
		case "Defaults":
			keyOrder = []string{"Unknown RDRAM Size", "Fixed Audio", "Audio-Sync Audio", "ViRefresh"}
		default:
			for k := range keys {
				keyOrder = append(keyOrder, k)
			}
		}
		for _, k := range keyOrder {
			if v, ok := keys[k]; ok {
				kept = append(kept, k+"="+v)
				seen[k] = true
			}
		}
		_ = seen
		sections[name] = kept
	}
	var b strings.Builder
	for i, name := range order {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString("[")
		b.WriteString(name)
		b.WriteString("]\n")
		for _, line := range sections[name] {
			if strings.TrimSpace(line) == "" {
				continue
			}
			b.WriteString(line)
			if !strings.HasSuffix(line, "\n") {
				b.WriteByte('\n')
			}
		}
	}
	return b.String()
}

func iniKey(line string) string {
	s := strings.TrimSpace(line)
	if s == "" || strings.HasPrefix(s, ";") || strings.HasPrefix(s, "#") {
		return ""
	}
	i := strings.IndexByte(s, '=')
	if i <= 0 {
		return ""
	}
	return strings.TrimSpace(s[:i])
}

func ensurePj64TestProfile(exe string) (string, error) {
	dll := pickPj64Gfx(pj64GfxDir(exe))
	if dll == "" {
		return "", fmt.Errorf("no Parallel-RDP or Angrylion plugin")
	}
	rel := pj64GfxRel(dll)
	if err := writePj64TestProfile(pj64CfgPath(exe), rel); err != nil {
		return "", err
	}
	return rel, nil
}

func pj64PluginInitFail(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "exited before smoke timeout")
}
