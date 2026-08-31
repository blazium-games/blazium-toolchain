package settings

import (
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
)

var envKeyOK = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

// CheckURL accepts https URLs with a host and no userinfo. Empty is ok.
func CheckURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("bad url %s: %w", raw, err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("url must be https: %s", raw)
	}
	if u.Host == "" {
		return fmt.Errorf("url missing host: %s", raw)
	}
	if u.User != nil {
		return fmt.Errorf("url must not include credentials: %s", u.Host)
	}
	return nil
}

// CheckFetchURL is CheckURL plus loopback http (httptest, local mirrors).
func CheckFetchURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("empty download url")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("bad url %s: %w", raw, err)
	}
	switch u.Scheme {
	case "https":
	case "http":
		if !loopbackHost(u.Hostname()) {
			return fmt.Errorf("url must be https: %s", raw)
		}
	default:
		return fmt.Errorf("url scheme not allowed: %s", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("url missing host: %s", raw)
	}
	if u.User != nil {
		return fmt.Errorf("url must not include credentials: %s", u.Host)
	}
	return nil
}

func loopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// Validate checks resolved settings. Call after merge so defaults fill empties.
func Validate(s Settings) error {
	for _, pair := range []struct {
		name, val string
	}{
		{"smoke_timeout", s.SmokeTimeout},
		{"download_timeout", s.DownloadTimeout},
		{"json_timeout", s.JSONTimeout},
	} {
		if err := checkDuration(pair.name, pair.val); err != nil {
			return err
		}
	}
	if strings.ContainsAny(s.UserAgent, "\r\n\x00") {
		return fmt.Errorf("user_agent contains a newline or null")
	}
	if s.DownloadMaxBytes < 0 {
		return fmt.Errorf("download_max_bytes must be positive")
	}
	for k, v := range s.Env {
		if !envKeyOK.MatchString(k) {
			return fmt.Errorf("env key %s is not a POSIX name", k)
		}
		if strings.ContainsAny(v, "\r\n\x00") {
			return fmt.Errorf("env.%s contains a newline or null", k)
		}
	}
	if err := checkOneOf("ps1.profile", s.PS1.Profile, "compile", "dev", "iso"); err != nil {
		return err
	}
	if err := checkOneOf("ps2.profile", s.PS2.Profile, "compile", "dev", "iso"); err != nil {
		return err
	}
	if err := checkOneOf("n64.profile", s.N64.Profile, "compile", "dev", "rom"); err != nil {
		return err
	}
	if err := checkOneOf("n64.display", s.N64.Display, "320", "640"); err != nil {
		return err
	}
	if err := checkOneOf("n64.rdram", s.N64.Rdram, "8", "4"); err != nil {
		return err
	}
	if err := checkOneOf("n64.emu", s.N64.Emu, "ares", "project64", "both"); err != nil {
		return err
	}
	if err := checkOneOf("ps2.cnf_vmode", strings.ToUpper(s.PS2.CNFVMode), "NTSC", "PAL"); err != nil {
		return err
	}
	if name := filepath.Base(s.PS2.DefaultELF); name != s.PS2.DefaultELF || strings.ContainsAny(s.PS2.DefaultELF, `/\`) {
		return fmt.Errorf("ps2.default_elf must be a file name, not a path")
	}
	for _, u := range []string{
		s.PS1.Fetch.DistribRoot,
		s.PS1.Fetch.PCSXCatalog,
		s.PS1.Fetch.PCSXInfoBase,
		s.PS1.Fetch.PCSXPinnedZip,
		s.PS2.Fetch.Windows,
		s.PS2.Fetch.Linux,
		s.PS2.Fetch.IconvZip,
		s.N64.Fetch.ToolchainBase,
		s.N64.Fetch.Libdragon,
		s.N64.Fetch.Tiny3D,
		s.N64.Fetch.AresWindows,
	} {
		if err := CheckURL(u); err != nil {
			return err
		}
	}
	for _, u := range s.PS2.Fetch.Msys32 {
		if err := CheckURL(u); err != nil {
			return err
		}
	}
	return nil
}

func checkDuration(name, raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return fmt.Errorf("%s is not a duration: %s", name, raw)
	}
	return nil
}

func checkOneOf(name, val string, allow ...string) error {
	val = strings.TrimSpace(val)
	if val == "" {
		return nil
	}
	for _, a := range allow {
		if strings.EqualFold(val, a) {
			return nil
		}
	}
	return fmt.Errorf("%s must be %s (got %s)", name, strings.Join(allow, ", "), val)
}

// CleanVolume keeps A-Z, 0-9, underscore; max 32. Empty stays empty.
func CleanVolume(s string) string {
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
	return strings.Trim(b.String(), "_")
}
