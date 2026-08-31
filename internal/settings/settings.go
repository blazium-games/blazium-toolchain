// Package settings loads project YAML (URLs, timeouts, volumes, tool paths).
// Hardcoded defaults live in Defaults(); a project file overlays them.
package settings

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	EnvSettings        = "BLAZIUM_TOOLCHAIN_SETTINGS"
	FileName           = "blazium-toolchain.yml"
	DotFileName        = ".blazium-toolchain.yml"
	MaxFileBytes       = 1 << 20
	DefaultDownloadMax = 2 << 30
)

// Settings is the project overlay. Empty fields keep Defaults().
type Settings struct {
	Prefix          string            `yaml:"prefix,omitempty" json:"prefix,omitempty"`
	SmokeTimeout    string            `yaml:"smoke_timeout,omitempty" json:"smoke_timeout,omitempty"`
	DownloadTimeout string            `yaml:"download_timeout,omitempty" json:"download_timeout,omitempty"`
	JSONTimeout      string            `yaml:"json_timeout,omitempty" json:"json_timeout,omitempty"`
	UserAgent        string            `yaml:"user_agent,omitempty" json:"user_agent,omitempty"`
	DownloadMaxBytes int64             `yaml:"download_max_bytes,omitempty" json:"download_max_bytes,omitempty"`
	Env              map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	PS1             PS1               `yaml:"ps1,omitempty" json:"ps1,omitempty"`
	PS2             PS2               `yaml:"ps2,omitempty" json:"ps2,omitempty"`
	N64             N64               `yaml:"n64,omitempty" json:"n64,omitempty"`
	InterDVD        InterDVD          `yaml:"interdvd,omitempty" json:"interdvd,omitempty"`
}

type PS1 struct {
	Profile string   `yaml:"profile,omitempty" json:"profile,omitempty"`
	Overlay string   `yaml:"overlay,omitempty" json:"overlay,omitempty"`
	Fetch   PS1Fetch `yaml:"fetch,omitempty" json:"fetch,omitempty"`
}

type PS1Fetch struct {
	PCSXCatalog   string `yaml:"pcsx_catalog,omitempty" json:"pcsx_catalog,omitempty"`
	PCSXInfoBase  string `yaml:"pcsx_info_base,omitempty" json:"pcsx_info_base,omitempty"`
	PCSXPinnedZip string `yaml:"pcsx_pinned_zip,omitempty" json:"pcsx_pinned_zip,omitempty"`
	DistribRoot   string `yaml:"distrib_root,omitempty" json:"distrib_root,omitempty"`
}

type PS2 struct {
	Profile    string   `yaml:"profile,omitempty" json:"profile,omitempty"`
	Overlay    string   `yaml:"overlay,omitempty" json:"overlay,omitempty"`
	ISOVolume  string   `yaml:"iso_volume,omitempty" json:"iso_volume,omitempty"`
	ISOApp     string   `yaml:"iso_application,omitempty" json:"iso_application,omitempty"`
	ISOSystem  string   `yaml:"iso_system,omitempty" json:"iso_system,omitempty"`
	CNFVer     string   `yaml:"cnf_ver,omitempty" json:"cnf_ver,omitempty"`
	CNFVMode   string   `yaml:"cnf_vmode,omitempty" json:"cnf_vmode,omitempty"`
	DefaultELF string   `yaml:"default_elf,omitempty" json:"default_elf,omitempty"`
	BIOSPrefer string   `yaml:"bios_prefer,omitempty" json:"bios_prefer,omitempty"`
	PCSX2Exe   string   `yaml:"pcsx2_exe,omitempty" json:"pcsx2_exe,omitempty"`
	BIOSDir    string   `yaml:"bios_dir,omitempty" json:"bios_dir,omitempty"`
	ElfTextMax      int64    `yaml:"elf_text_max,omitempty" json:"elf_text_max,omitempty"`
	ElfFileMax      int64    `yaml:"elf_file_max,omitempty" json:"elf_file_max,omitempty"`
	MkdirPath       []string `yaml:"mkdir_path,omitempty" json:"mkdir_path,omitempty"`
	SiblingRoots    []string `yaml:"sibling_roots,omitempty" json:"sibling_roots,omitempty"`
	BIOSSearchNames []string `yaml:"bios_search_names,omitempty" json:"bios_search_names,omitempty"`
	Fetch           PS2Fetch `yaml:"fetch,omitempty" json:"fetch,omitempty"`
}

type PS2Fetch struct {
	Windows  string   `yaml:"windows,omitempty" json:"windows,omitempty"`
	Linux    string   `yaml:"linux,omitempty" json:"linux,omitempty"`
	IconvZip string   `yaml:"iconv_zip,omitempty" json:"iconv_zip,omitempty"`
	Msys32   []string `yaml:"msys32,omitempty" json:"msys32,omitempty"`
}

type N64 struct {
	Profile      string   `yaml:"profile,omitempty" json:"profile,omitempty"`
	Overlay      string   `yaml:"overlay,omitempty" json:"overlay,omitempty"`
	Display      string   `yaml:"display,omitempty" json:"display,omitempty"`
	Rdram        string   `yaml:"rdram,omitempty" json:"rdram,omitempty"`
	Rumble       bool     `yaml:"rumble,omitempty" json:"rumble,omitempty"`
	Emu          string   `yaml:"emu,omitempty" json:"emu,omitempty"`
	RomTitle     string   `yaml:"rom_title,omitempty" json:"rom_title,omitempty"`
	AresExe      string   `yaml:"ares_exe,omitempty" json:"ares_exe,omitempty"`
	PJ64Exe      string   `yaml:"project64_exe,omitempty" json:"project64_exe,omitempty"`
	CartMax      int64    `yaml:"cart_max,omitempty" json:"cart_max,omitempty"`
	MakePath     []string `yaml:"make_path,omitempty" json:"make_path,omitempty"`
	HostMingw    []string `yaml:"host_mingw_path,omitempty" json:"host_mingw_path,omitempty"`
	BashPath     []string `yaml:"bash_path,omitempty" json:"bash_path,omitempty"`
	SiblingRoots []string `yaml:"sibling_roots,omitempty" json:"sibling_roots,omitempty"`
	PJ64         N64PJ64  `yaml:"pj64,omitempty" json:"pj64,omitempty"`
	Fetch        N64Fetch `yaml:"fetch,omitempty" json:"fetch,omitempty"`
}

type N64PJ64 struct {
	UnknownRDRAM string `yaml:"unknown_rdram,omitempty" json:"unknown_rdram,omitempty"`
	ViRefresh    string `yaml:"vi_refresh,omitempty" json:"vi_refresh,omitempty"`
}

type N64Fetch struct {
	ToolchainBase     string `yaml:"toolchain_base,omitempty" json:"toolchain_base,omitempty"`
	ToolchainWindows  string `yaml:"toolchain_windows,omitempty" json:"toolchain_windows,omitempty"`
	ToolchainLinux    string `yaml:"toolchain_linux,omitempty" json:"toolchain_linux,omitempty"`
	ToolchainLinuxARM string `yaml:"toolchain_linux_arm,omitempty" json:"toolchain_linux_arm,omitempty"`
	Libdragon         string `yaml:"libdragon,omitempty" json:"libdragon,omitempty"`
	Tiny3D            string `yaml:"tiny3d,omitempty" json:"tiny3d,omitempty"`
	AresWindows       string `yaml:"ares_windows,omitempty" json:"ares_windows,omitempty"`
}

type InterDVD struct {
	Volume           string `yaml:"volume,omitempty" json:"volume,omitempty"`
	Preparer         string `yaml:"preparer,omitempty" json:"preparer,omitempty"`
	Application      string `yaml:"application,omitempty" json:"application,omitempty"`
	Provider         string `yaml:"provider,omitempty" json:"provider,omitempty"`
	System           string `yaml:"system,omitempty" json:"system,omitempty"`
	MenuLanguage     string `yaml:"menu_language,omitempty" json:"menu_language,omitempty"`
	AudioLanguage    string `yaml:"audio_language,omitempty" json:"audio_language,omitempty"`
	SubtitleLanguage string `yaml:"subtitle_language,omitempty" json:"subtitle_language,omitempty"`
}

var (
	mu      sync.RWMutex
	current Settings
	loaded  string
	ready   bool
)

func init() {
	current = Defaults()
	ready = true
}

// Defaults are the built-in values used when no project file is present.
func Defaults() Settings {
	return Settings{
		SmokeTimeout:     "120s",
		DownloadTimeout:  "15m",
		JSONTimeout:      "60s",
		UserAgent:        "blazium-toolchain/0.1.0",
		DownloadMaxBytes: DefaultDownloadMax,
		PS1: PS1{
			Profile: "compile",
			Fetch: PS1Fetch{
				DistribRoot:   "https://distrib.app",
				PCSXCatalog:   "https://distrib.app/storage/manifests/pcsx-redux/dev-win-cli-x64/manifest.json",
				PCSXInfoBase:  "https://distrib.app/storage/manifests/pcsx-redux/dev-win-cli-x64/",
				PCSXPinnedZip: "https://distrib.app/storage/assets/265/a79/ea7/640c8fce00aca4a92d8638443a0533597780c73adddabc747cb3a50/pcsx-redux-nightly-25100.20260827.8-x64-cli.zip",
			},
		},
		PS2: PS2{
			Profile:    "compile",
			ISOVolume:  "BLAZIUM2",
			ISOApp:     "BLAZIUM PS2",
			ISOSystem:  "PLAYSTATION",
			CNFVer:     "1.00",
			CNFVMode:   "NTSC",
			DefaultELF: "GAME.ELF",
			BIOSPrefer: "39001",
			ElfTextMax: 512 * 1024,
			ElfFileMax: 2 * 1024 * 1024,
			MkdirPath: []string{
				`C:\Program Files\Git\usr\bin\mkdir.exe`,
				`C:\Program Files (x86)\Git\usr\bin\mkdir.exe`,
			},
			SiblingRoots:    []string{"ps2_stuff"},
			BIOSSearchNames: []string{"ps2 bios usa", "bios"},
			Fetch: PS2Fetch{
				Windows:  "https://github.com/ps2dev/ps2dev/releases/download/latest/ps2dev-windows-latest.tar.gz",
				Linux:    "https://github.com/ps2dev/ps2dev/releases/download/latest/ps2dev-ubuntu-latest.tar.gz",
				IconvZip: "https://downloads.sourceforge.net/project/gnuwin32/libiconv/1.9.2-1/libiconv-1.9.2-1-bin.zip",
				Msys32: []string{
					"https://mirror.msys2.org/mingw/mingw32/mingw-w64-i686-gmp-6.3.0-2-any.pkg.tar.zst",
					"https://mirror.msys2.org/mingw/mingw32/mingw-w64-i686-isl-0.28-1-any.pkg.tar.zst",
					"https://mirror.msys2.org/mingw/mingw32/mingw-w64-i686-mpc-1.4.1-1-any.pkg.tar.zst",
					"https://mirror.msys2.org/mingw/mingw32/mingw-w64-i686-mpfr-4.2.2-3-any.pkg.tar.zst",
					"https://mirror.msys2.org/mingw/mingw32/mingw-w64-i686-zstd-1.5.7-2-any.pkg.tar.zst",
					"https://mirror.msys2.org/mingw/mingw32/mingw-w64-i686-gcc-libs-15.1.0-1-any.pkg.tar.zst",
					"https://mirror.msys2.org/mingw/mingw32/mingw-w64-i686-libiconv-1.19-1-any.pkg.tar.zst",
					"https://mirror.msys2.org/mingw/mingw32/mingw-w64-i686-libwinpthread-14.0.0.r302.gd7f3c5201-1-any.pkg.tar.zst",
				},
			},
		},
		N64: N64{
			Profile:  "compile",
			Display:  "320",
			Rdram:    "8",
			Emu:      "both",
			RomTitle: "Blazium N64",
			CartMax:  64 * 1024 * 1024,
			MakePath: []string{
				`C:\Program Files\Git\usr\bin`,
				`C:\Program Files\Git\bin`,
				`C:\msys64\usr\bin`,
				`C:\msys64\ucrt64\bin`,
			},
			HostMingw: []string{
				`C:\Strawberry\c\bin`,
				`C:\msys64\ucrt64\bin`,
				`C:\msys64\mingw64\bin`,
				`C:\msys64\usr\bin`,
			},
			BashPath: []string{
				`C:\Program Files\Git\bin\bash.exe`,
				`C:\Program Files\Git\usr\bin\bash.exe`,
				`C:\msys64\usr\bin\bash.exe`,
				`C:\msys64\ucrt64\bin\bash.exe`,
			},
			SiblingRoots: []string{"n64_stuff"},
			PJ64: N64PJ64{
				UnknownRDRAM: "8388608",
				ViRefresh:    "1500",
			},
			Fetch: N64Fetch{
				ToolchainBase:     "https://github.com/DragonMinded/libdragon/releases/download/toolchain-continuous-prerelease/",
				ToolchainWindows:  "gcc-toolchain-mips64-win64.zip",
				ToolchainLinux:    "gcc-toolchain-mips64-x86_64.deb",
				ToolchainLinuxARM: "gcc-toolchain-mips64-aarch64.deb",
				Libdragon:         "https://github.com/DragonMinded/libdragon/archive/refs/heads/preview.zip",
				Tiny3D:            "https://github.com/HailToDodongo/tiny3d/archive/refs/heads/master.zip",
				AresWindows:       "https://github.com/ares-emulator/ares/releases/download/v148/ares-windows-x64.zip",
			},
		},
		InterDVD: InterDVD{
			Volume:           "BLAZIUM_DVD",
			Preparer:         "BLAZIUM TOOLCHAIN",
			Application:      "BLAZIUM INTER-DVD",
			Provider:         "BLAZIUM INTER-DVD",
			System:           "DVD-VIDEO",
			MenuLanguage:     "en",
			AudioLanguage:    "en",
			SubtitleLanguage: "en",
		},
	}
}

// Current is the resolved project settings (defaults + optional YAML).
func Current() Settings {
	mu.RLock()
	defer mu.RUnlock()
	if !ready {
		return Defaults()
	}
	return current
}

// LoadedPath is the YAML that was applied, or empty when using built-in defaults.
func LoadedPath() string {
	mu.RLock()
	defer mu.RUnlock()
	return loaded
}

// Reset restores built-in defaults (tests).
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	current = Defaults()
	loaded = ""
	ready = true
}

// Set replaces the resolved settings (tests).
func Set(s Settings) {
	mu.Lock()
	defer mu.Unlock()
	current = merge(Defaults(), s)
	ready = true
}

// LoadFile overlays path on Defaults and makes it Current.
func LoadFile(path string) error {
	st, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("cannot read settings %s: %w", path, err)
	}
	if st.IsDir() {
		return fmt.Errorf("settings path is a directory: %s", path)
	}
	if st.Size() > MaxFileBytes {
		return fmt.Errorf("settings file is %d bytes (max %d): %s", st.Size(), MaxFileBytes, path)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("cannot read settings %s: %w", path, err)
	}
	var over Settings
	if err := yaml.Unmarshal(raw, &over); err != nil {
		return fmt.Errorf("parse settings %s: %w", path, err)
	}
	merged := merge(Defaults(), over)
	if err := Validate(merged); err != nil {
		return err
	}
	mu.Lock()
	defer mu.Unlock()
	current = merged
	loaded = path
	ready = true
	return nil
}

// LoadFromCWD finds a project YAML (or uses explicit) and loads it.
func LoadFromCWD(explicit string) error {
	path := Find(explicit)
	if path == "" {
		Reset()
		return nil
	}
	return LoadFile(path)
}

// Find returns the settings path: explicit, $BLAZIUM_TOOLCHAIN_SETTINGS, or a walk from cwd.
func Find(explicit string) string {
	if p := strings.TrimSpace(explicit); p != "" {
		return p
	}
	if p := strings.TrimSpace(os.Getenv(EnvSettings)); p != "" {
		return p
	}
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	for i := 0; i < 8 && wd != ""; i++ {
		for _, name := range []string{FileName, DotFileName} {
			cand := filepath.Join(wd, name)
			if st, err := os.Stat(cand); err == nil && !st.IsDir() {
				return cand
			}
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}
	return ""
}

// ApplyEnv sets process env from the file when the variable is not already set.
func ApplyEnv(s Settings) {
	put := func(key, val string) {
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if key == "" || val == "" || os.Getenv(key) != "" {
			return
		}
		_ = os.Setenv(key, val)
	}
	for k, v := range s.Env {
		put(k, v)
	}
	put("BLAZIUM_TOOLCHAIN_PREFIX", s.Prefix)
	put("BLAZIUM_PS1_OVERLAY", s.PS1.Overlay)
	put("BLAZIUM_PS2_OVERLAY", s.PS2.Overlay)
	put("BLAZIUM_N64_OVERLAY", s.N64.Overlay)
	put("PCSX2_EXE", s.PS2.PCSX2Exe)
	put("PS2_BIOS_DIR", s.PS2.BIOSDir)
	put("ARES_EXE", s.N64.AresExe)
	put("PROJECT64_EXE", s.N64.PJ64Exe)
}

func (s Settings) Smoke() time.Duration { return parseDur(s.SmokeTimeout, 120*time.Second) }
func (s Settings) Download() time.Duration {
	return parseDur(s.DownloadTimeout, 15*time.Minute)
}
func (s Settings) JSON() time.Duration { return parseDur(s.JSONTimeout, 60*time.Second) }

func (s Settings) DownloadMax() int64 {
	if s.DownloadMaxBytes > 0 {
		return s.DownloadMaxBytes
	}
	return DefaultDownloadMax
}

func (s Settings) ProfileFor(plat string) string {
	switch plat {
	case "ps1":
		return or(s.PS1.Profile, "compile")
	case "ps2":
		return or(s.PS2.Profile, "compile")
	case "n64":
		return or(s.N64.Profile, "compile")
	default:
		return "compile"
	}
}

// Marshal writes resolved settings as YAML (for `blazium-toolchain settings`).
func Marshal(s Settings) ([]byte, error) {
	return yaml.Marshal(s)
}

func parseDur(s string, fallback time.Duration) time.Duration {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}

func or(v, fallback string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return fallback
	}
	return v
}

func merge(base, over Settings) Settings {
	out := base
	if over.Prefix != "" {
		out.Prefix = over.Prefix
	}
	if over.SmokeTimeout != "" {
		out.SmokeTimeout = over.SmokeTimeout
	}
	if over.DownloadTimeout != "" {
		out.DownloadTimeout = over.DownloadTimeout
	}
	if over.JSONTimeout != "" {
		out.JSONTimeout = over.JSONTimeout
	}
	if over.UserAgent != "" {
		out.UserAgent = over.UserAgent
	}
	if over.DownloadMaxBytes > 0 {
		out.DownloadMaxBytes = over.DownloadMaxBytes
	}
	if len(over.Env) > 0 {
		if out.Env == nil {
			out.Env = map[string]string{}
		}
		for k, v := range over.Env {
			out.Env[k] = v
		}
	}
	out.PS1 = mergePS1(base.PS1, over.PS1)
	out.PS2 = mergePS2(base.PS2, over.PS2)
	out.N64 = mergeN64(base.N64, over.N64)
	out.InterDVD = mergeDVD(base.InterDVD, over.InterDVD)
	return out
}

func mergePS1(b, o PS1) PS1 {
	if o.Profile != "" {
		b.Profile = o.Profile
	}
	if o.Overlay != "" {
		b.Overlay = o.Overlay
	}
	b.Fetch.DistribRoot = or(o.Fetch.DistribRoot, b.Fetch.DistribRoot)
	b.Fetch.PCSXCatalog = or(o.Fetch.PCSXCatalog, b.Fetch.PCSXCatalog)
	b.Fetch.PCSXInfoBase = or(o.Fetch.PCSXInfoBase, b.Fetch.PCSXInfoBase)
	b.Fetch.PCSXPinnedZip = or(o.Fetch.PCSXPinnedZip, b.Fetch.PCSXPinnedZip)
	return b
}

func mergePS2(b, o PS2) PS2 {
	if o.Profile != "" {
		b.Profile = o.Profile
	}
	if o.Overlay != "" {
		b.Overlay = o.Overlay
	}
	b.ISOVolume = or(o.ISOVolume, b.ISOVolume)
	b.ISOApp = or(o.ISOApp, b.ISOApp)
	b.ISOSystem = or(o.ISOSystem, b.ISOSystem)
	b.CNFVer = or(o.CNFVer, b.CNFVer)
	b.CNFVMode = or(o.CNFVMode, b.CNFVMode)
	b.DefaultELF = or(o.DefaultELF, b.DefaultELF)
	b.BIOSPrefer = or(o.BIOSPrefer, b.BIOSPrefer)
	b.PCSX2Exe = or(o.PCSX2Exe, b.PCSX2Exe)
	b.BIOSDir = or(o.BIOSDir, b.BIOSDir)
	if o.ElfTextMax > 0 {
		b.ElfTextMax = o.ElfTextMax
	}
	if o.ElfFileMax > 0 {
		b.ElfFileMax = o.ElfFileMax
	}
	if len(o.MkdirPath) > 0 {
		b.MkdirPath = append([]string(nil), o.MkdirPath...)
	}
	if len(o.SiblingRoots) > 0 {
		b.SiblingRoots = append([]string(nil), o.SiblingRoots...)
	}
	if len(o.BIOSSearchNames) > 0 {
		b.BIOSSearchNames = append([]string(nil), o.BIOSSearchNames...)
	}
	b.Fetch.Windows = or(o.Fetch.Windows, b.Fetch.Windows)
	b.Fetch.Linux = or(o.Fetch.Linux, b.Fetch.Linux)
	b.Fetch.IconvZip = or(o.Fetch.IconvZip, b.Fetch.IconvZip)
	if len(o.Fetch.Msys32) > 0 {
		b.Fetch.Msys32 = append([]string(nil), o.Fetch.Msys32...)
	}
	return b
}

func mergeN64(b, o N64) N64 {
	if o.Profile != "" {
		b.Profile = o.Profile
	}
	if o.Overlay != "" {
		b.Overlay = o.Overlay
	}
	b.Display = or(o.Display, b.Display)
	b.Rdram = or(o.Rdram, b.Rdram)
	b.Emu = or(o.Emu, b.Emu)
	b.RomTitle = or(o.RomTitle, b.RomTitle)
	b.AresExe = or(o.AresExe, b.AresExe)
	b.PJ64Exe = or(o.PJ64Exe, b.PJ64Exe)
	if o.Rumble {
		b.Rumble = true
	}
	if o.CartMax > 0 {
		b.CartMax = o.CartMax
	}
	if len(o.MakePath) > 0 {
		b.MakePath = append([]string(nil), o.MakePath...)
	}
	if len(o.HostMingw) > 0 {
		b.HostMingw = append([]string(nil), o.HostMingw...)
	}
	if len(o.BashPath) > 0 {
		b.BashPath = append([]string(nil), o.BashPath...)
	}
	if len(o.SiblingRoots) > 0 {
		b.SiblingRoots = append([]string(nil), o.SiblingRoots...)
	}
	b.PJ64.UnknownRDRAM = or(o.PJ64.UnknownRDRAM, b.PJ64.UnknownRDRAM)
	b.PJ64.ViRefresh = or(o.PJ64.ViRefresh, b.PJ64.ViRefresh)
	b.Fetch.ToolchainBase = or(o.Fetch.ToolchainBase, b.Fetch.ToolchainBase)
	b.Fetch.ToolchainWindows = or(o.Fetch.ToolchainWindows, b.Fetch.ToolchainWindows)
	b.Fetch.ToolchainLinux = or(o.Fetch.ToolchainLinux, b.Fetch.ToolchainLinux)
	b.Fetch.ToolchainLinuxARM = or(o.Fetch.ToolchainLinuxARM, b.Fetch.ToolchainLinuxARM)
	b.Fetch.Libdragon = or(o.Fetch.Libdragon, b.Fetch.Libdragon)
	b.Fetch.Tiny3D = or(o.Fetch.Tiny3D, b.Fetch.Tiny3D)
	b.Fetch.AresWindows = or(o.Fetch.AresWindows, b.Fetch.AresWindows)
	return b
}

func mergeDVD(b, o InterDVD) InterDVD {
	b.Volume = or(o.Volume, b.Volume)
	b.Preparer = or(o.Preparer, b.Preparer)
	b.Application = or(o.Application, b.Application)
	b.Provider = or(o.Provider, b.Provider)
	b.System = or(o.System, b.System)
	b.MenuLanguage = or(o.MenuLanguage, b.MenuLanguage)
	b.AudioLanguage = or(o.AudioLanguage, b.AudioLanguage)
	b.SubtitleLanguage = or(o.SubtitleLanguage, b.SubtitleLanguage)
	return b
}
