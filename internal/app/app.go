package app

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/blazium-games/blazium-toolchain/internal/cache"
	"github.com/blazium-games/blazium-toolchain/internal/embedfs"
	"github.com/blazium-games/blazium-toolchain/internal/iso"
	"github.com/blazium-games/blazium-toolchain/internal/platforms"
	"github.com/blazium-games/blazium-toolchain/internal/platforms/future"
	"github.com/blazium-games/blazium-toolchain/internal/platforms/interdvd"
	"github.com/blazium-games/blazium-toolchain/internal/platforms/ps1"
)

// Version and License come from the embedded manifest.json. CI may override
// Version with -ldflags "-X github.com/blazium-games/blazium-toolchain/internal/app.Version=vX.Y.Z".
var (
	Version = embedfs.MustManifest().Version
	License = embedfs.MustManifest().License
)

const (
	ExitOK      = 0
	ExitUsage   = 1
	ExitPlanned = 2
	ExitTool    = 3
	ExitFail    = 4
)

func init() {
	platforms.Register(ps1.New())
	platforms.Register(interdvd.New())
	future.Register()
}

// Run is the CLI entry. argv is os.Args[1:].
func Run(ctx context.Context, argv []string, stdout, stderr io.Writer) int {
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}

	fs := flag.NewFlagSet("blazium-toolchain", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "machine-readable output (for the installer / editor)")
	prefix := fs.String("prefix", "", "toolchain cache root (default: $BLAZIUM_TOOLCHAIN_PREFIX or OS cache)")
	if err := fs.Parse(argv); err != nil {
		return ExitUsage
	}
	args := fs.Args()
	if *prefix == "" {
		*prefix = cache.DefaultPrefix()
	}

	if len(args) == 0 {
		usage(stderr)
		return ExitUsage
	}

	switch args[0] {
	case "help", "-h", "--help":
		usage(stdout)
		return ExitOK
	case "version":
		return cmdVersion(*jsonOut, stdout)
	case "list":
		return cmdList(*jsonOut, stdout)
	}

	// Platform-first: blazium-toolchain ps1 setup
	platID, rest := args[0], args[1:]
	p, err := platforms.Lookup(platID)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		if errors.Is(err, platforms.ErrPlanned) {
			return ExitPlanned
		}
		if errors.Is(err, platforms.ErrUnknownPlatform) {
			usage(stderr)
			return ExitUsage
		}
		return ExitFail
	}
	if len(rest) == 0 {
		fmt.Fprintf(stderr, "usage: blazium-toolchain %s <setup|env|status|build|run|iso|fmv|meta>\n", platID)
		return ExitUsage
	}
	return dispatch(ctx, p, rest, common(*prefix, *jsonOut, stdout, stderr), stdout, stderr)
}

func common(prefix string, jsonOut bool, stdout, stderr io.Writer) platforms.CommonOptions {
	return platforms.CommonOptions{Prefix: prefix, JSON: jsonOut, Stdout: stdout, Stderr: stderr}
}

func cmdVersion(jsonOut bool, w io.Writer) int {
	if jsonOut {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"name":    "blazium-toolchain",
			"version": Version,
			"license": License,
		})
		return ExitOK
	}
	fmt.Fprintf(w, "blazium-toolchain %s\n", Version)
	return ExitOK
}

func cmdList(jsonOut bool, w io.Writer) int {
	list := platforms.List()
	if jsonOut {
		_ = json.NewEncoder(w).Encode(list)
		return ExitOK
	}
	for _, info := range list {
		fmt.Fprintf(w, "%-6s %-12s %s\n", info.ID, info.Status, info.Name)
	}
	return ExitOK
}

func dispatch(ctx context.Context, p platforms.Platform, args []string, base platforms.CommonOptions, stdout, stderr io.Writer) int {
	cmd := args[0]
	rest := args[1:]
	var err error
	switch cmd {
	case "setup":
		err = runSetup(ctx, p, rest, base)
	case "env":
		err = runEnv(p, base, stdout)
	case "status":
		err = runStatus(p, base, stdout)
	case "build":
		err = runBuild(ctx, p, rest, base)
	case "run":
		err = runRun(ctx, p, rest, base)
	case "iso":
		err = runISO(ctx, p, rest, base)
	case "fmv":
		err = runFMV(p, rest, base, stdout)
	case "meta":
		err = runMeta(p, rest, base, stdout)
	default:
		fmt.Fprintf(stderr, "unknown command %q for platform %s\n", cmd, p.Info().ID)
		return ExitUsage
	}
	return mapErr(err, stderr)
}

func runSetup(ctx context.Context, p platforms.Platform, args []string, base platforms.CommonOptions) error {
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	fs.SetOutput(base.Stderr)
	profile := fs.String("profile", "compile", "compile|dev|iso")
	offline := fs.Bool("offline", false, "do not fetch; only discover local tools")
	if err := fs.Parse(args); err != nil {
		return platforms.ErrUsage
	}
	return p.Setup(ctx, platforms.SetupOptions{CommonOptions: base, Profile: *profile, Offline: *offline})
}

func runEnv(p platforms.Platform, base platforms.CommonOptions, stdout io.Writer) error {
	env, err := p.Env(base)
	if err != nil {
		return err
	}
	if base.JSON {
		return json.NewEncoder(stdout).Encode(env)
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(stdout, "%s=%s\n", k, env[k])
	}
	return nil
}

func runStatus(p platforms.Platform, base platforms.CommonOptions, stdout io.Writer) error {
	st, err := p.Status(base)
	if err != nil {
		return err
	}
	return json.NewEncoder(stdout).Encode(st)
}

func runBuild(ctx context.Context, p platforms.Platform, args []string, base platforms.CommonOptions) error {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(base.Stderr)
	src := fs.String("src", "", "guest CMake source dir")
	out := fs.String("out", "", "output PS-X EXE path")
	sample := fs.String("sample", "", "official SDK sample: template or gte (when --src is empty)")
	tim := fs.String("tim", "", "optional cooked TIM to embed in the guest")
	mesh := fs.String("mesh", "", "optional cooked SVECTOR mesh to embed in the guest")
	vag := fs.String("vag", "", "optional cooked VAG to embed in the guest")
	sprite := fs.String("sprite", "", "optional cooked SPRITE table to embed in the guest")
	script := fs.String("script", "", "optional cooked SCRIPT.IR to embed in the guest")
	gdbc := fs.String("gdbc", "", "optional cooked SCRIPT.GD.BC to embed in the guest")
	luau := fs.String("luau", "", "optional cooked SCRIPT.LU.BC to embed in the guest")
	str := fs.String("str", "", "optional cooked FMV00.STR to embed in the guest")
	xa := fs.String("xa", "", "optional cooked FMV00.XA to embed in the guest")
	node := fs.String("node", "", "optional cooked NODE00.bin to embed in the guest")
	hud := fs.String("hud", "", "optional cooked HUD00.bin to embed in the guest")
	tile := fs.String("tile", "", "optional cooked TILE00.bin to embed in the guest")
	scene := fs.String("scene", "", "optional cooked SCENE00.bin to embed in the guest")
	anim := fs.String("anim", "", "optional cooked ANIM00.bin to embed in the guest")
	cam := fs.String("cam", "", "optional cooked CAM00.bin to embed in the guest")
	hit := fs.String("hit", "", "optional cooked HIT00.bin to embed in the guest")
	nav := fs.String("nav", "", "optional cooked NAV00.bin to embed in the guest")
	pathTbl := fs.String("path", "", "optional cooked PATH00.bin to embed in the guest")
	way := fs.String("way", "", "optional cooked WAY00.bin to embed in the guest")
	if err := fs.Parse(args); err != nil {
		return platforms.ErrUsage
	}
	return p.Build(ctx, platforms.BuildOptions{CommonOptions: base, Src: *src, Out: *out, Sample: *sample, Tim: *tim, Mesh: *mesh, Vag: *vag, Sprite: *sprite, Script: *script, Gdbc: *gdbc, Luau: *luau, Str: *str, Xa: *xa, Node: *node, Hud: *hud, Tile: *tile, Scene: *scene, Anim: *anim, Cam: *cam, Hit: *hit, Nav: *nav, Path: *pathTbl, Way: *way})
}

func runRun(ctx context.Context, p platforms.Platform, args []string, base platforms.CommonOptions) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(base.Stderr)
	iso := fs.String("iso", "", "optional cue/bin path")
	timeout := fs.Duration("timeout", 120*time.Second, "stop the emulator after this duration (smoke)")
	pcdrv := fs.String("pcdrv", "", "host directory for pcsx-redux -pcdrvbase (default: EXE dir)")
	ui := fs.Bool("ui", false, "show the pcsx-redux window (default is -no-ui smoke)")
	if err := fs.Parse(args); err != nil {
		return platforms.ErrUsage
	}
	exe := ""
	if fs.NArg() > 0 {
		exe = fs.Arg(0)
	}
	return p.Run(ctx, platforms.RunOptions{CommonOptions: base, Exe: exe, ISO: *iso, Timeout: *timeout, Pcdrv: *pcdrv, UI: *ui})
}

func runFMV(p platforms.Platform, args []string, base platforms.CommonOptions, stdout io.Writer) error {
	_ = p
	_ = args
	if base.JSON {
		return json.NewEncoder(stdout).Encode(map[string]any{
			"encoder":   "blazium-mit",
			"guest":     "DecDCTin + DecDCTout (MDEC RLE; no psxpress encode)",
			"spawn":     false,
			"in_editor": true,
			"note":      "Editor encodes STR/XA with MIT writers; guest DecDCTin only.",
		})
	}
	fmt.Fprintln(stdout, "ps1 fmv: in-editor MIT STR/XA encode (encoder=blazium-mit). Guest DecDCTin only.")
	fmt.Fprintln(stdout, "No psxpress encode spawn. DuckStation/Sony BIOS/hardware are out of scope.")
	return nil
}

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func runISO(ctx context.Context, p platforms.Platform, args []string, base platforms.CommonOptions) error {
	fs := flag.NewFlagSet("iso", flag.ContinueOnError)
	fs.SetOutput(base.Stderr)
	xml := fs.String("xml", "", "mkpsxiso project xml")
	dir := fs.String("dir", "", "VIDEO_TS parent directory")
	out := fs.String("out", "", "output path")
	volume := fs.String("volume", "", "ISO9660 volume label")
	title := fs.String("title", "", "movie / disc title")
	publisher := fs.String("publisher", "", "publisher identifier")
	preparer := fs.String("preparer", "", "data preparer")
	application := fs.String("application", "", "application identifier")
	system := fs.String("system", "", "system identifier")
	provider := fs.String("provider", "", "UDF provider")
	copyright := fs.String("copyright", "", "copyright text")
	copyrightFile := fs.String("copyright-file", "", "copyright file to pack")
	license := fs.String("license", "", "license text")
	licenseFile := fs.String("license-file", "", "license file to pack")
	readme := fs.String("readme", "", "readme text")
	readmeFile := fs.String("readme-file", "", "readme file to pack")
	credits := fs.String("credits", "", "credits text")
	creditsFile := fs.String("credits-file", "", "credits file to pack")
	author := fs.String("author", "", "author")
	studio := fs.String("studio", "", "studio")
	website := fs.String("website", "", "website")
	contact := fs.String("contact", "", "contact")
	version := fs.String("version", "", "version")
	catalog := fs.String("catalog", "", "catalog / SKU")
	description := fs.String("description", "", "synopsis")
	abstract := fs.String("abstract", "", "abstract text")
	abstractFile := fs.String("abstract-file", "", "abstract file")
	bibliographic := fs.String("bibliographic", "", "bibliographic text")
	biblioFile := fs.String("biblio-file", "", "bibliographic file")
	created := fs.String("created", "", "RFC3339 volume time")
	disc := fs.Int("disc", 0, "volume sequence number")
	discs := fs.Int("discs", 0, "volume set size")
	region := fs.Int("region-mask", 0, "DVD region mask (metadata)")
	parental := fs.Int("parental-level", 0, "parental level (metadata)")
	menuLang := fs.String("menu-language", "", "menu language (metadata)")
	audioLang := fs.String("audio-language", "", "audio language (metadata)")
	subLang := fs.String("subtitle-language", "", "subtitle language (metadata)")
	autorunLabel := fs.String("autorun-label", "", "AUTORUN.INF label")
	autorunOpen := fs.String("autorun-open", "", "AUTORUN.INF open")
	autorunIcon := fs.String("autorun-icon", "", "AUTORUN.INF icon")
	extrasDir := fs.String("extras-dir", "", "merge children onto disc root")
	recursive := fs.Bool("recursive", false, "walk nested extra folders")
	meta := fs.String("meta", "", "load disc.interdvd.json")
	writeMeta := fs.String("write-meta", "", "save resolved metadata JSON")
	var extras stringList
	fs.Var(&extras, "extra", "HOST or HOST:DISC (repeatable)")
	if err := fs.Parse(args); err != nil {
		return platforms.ErrUsage
	}
	return p.ISO(ctx, platforms.ISOOptions{
		CommonOptions: base, XML: *xml, Dir: *dir, Out: *out, VolumeID: *volume,
		Extra: extras, ExtrasDir: *extrasDir, Recursive: *recursive,
		Title: *title, Publisher: *publisher, Preparer: *preparer, Application: *application,
		System: *system, Provider: *provider,
		Copyright: *copyright, CopyrightFile: *copyrightFile,
		License: *license, LicenseFile: *licenseFile,
		Readme: *readme, ReadmeFile: *readmeFile,
		Credits: *credits, CreditsFile: *creditsFile,
		Author: *author, Studio: *studio, Website: *website, Contact: *contact,
		Version: *version, Catalog: *catalog, Description: *description,
		Abstract: *abstract, AbstractFile: *abstractFile,
		Bibliographic: *bibliographic, BiblioFile: *biblioFile,
		Created: *created, Disc: *disc, Discs: *discs,
		RegionMask: *region, ParentalLevel: *parental,
		MenuLanguage: *menuLang, AudioLanguage: *audioLang, SubtitleLanguage: *subLang,
		AutorunLabel: *autorunLabel, AutorunOpen: *autorunOpen, AutorunIcon: *autorunIcon,
		Meta: *meta, WriteMeta: *writeMeta,
	})
}

func runMeta(p platforms.Platform, args []string, base platforms.CommonOptions, stdout io.Writer) error {
	if p.Info().ID != interdvd.ID {
		return fmt.Errorf("%w: meta is an interdvd command", platforms.ErrUsage)
	}
	if len(args) == 0 {
		return fmt.Errorf("%w: interdvd meta <init|validate>", platforms.ErrUsage)
	}
	switch args[0] {
	case "init":
		fs := flag.NewFlagSet("meta init", flag.ContinueOnError)
		fs.SetOutput(base.Stderr)
		out := fs.String("out", "", "metadata JSON path")
		if err := fs.Parse(args[1:]); err != nil || *out == "" {
			return fmt.Errorf("%w: interdvd meta init --out FILE", platforms.ErrUsage)
		}
		if err := iso.WriteMeta(*out, iso.InitMeta()); err != nil {
			return err
		}
		if base.JSON {
			return json.NewEncoder(stdout).Encode(map[string]string{"out": *out, "schema": iso.SchemaV1})
		}
		fmt.Fprintf(stdout, "wrote %s\n", *out)
		return nil
	case "validate":
		fs := flag.NewFlagSet("meta validate", flag.ContinueOnError)
		fs.SetOutput(base.Stderr)
		meta := fs.String("meta", "", "metadata JSON path")
		if err := fs.Parse(args[1:]); err != nil || *meta == "" {
			return fmt.Errorf("%w: interdvd meta validate --meta FILE", platforms.ErrUsage)
		}
		m, err := iso.LoadMeta(*meta)
		if err != nil {
			return err
		}
		if err := iso.ValidateMeta(m); err != nil {
			return err
		}
		resolved, err := m.Resolve()
		if err != nil {
			return err
		}
		if base.JSON {
			return json.NewEncoder(stdout).Encode(resolved)
		}
		fmt.Fprintln(stdout, "ok")
		return nil
	default:
		return fmt.Errorf("%w: interdvd meta <init|validate>", platforms.ErrUsage)
	}
}

func mapErr(err error, stderr io.Writer) int {
	if err == nil {
		return ExitOK
	}
	fmt.Fprintln(stderr, err.Error())
	switch {
	case errors.Is(err, platforms.ErrUsage):
		return ExitUsage
	case errors.Is(err, platforms.ErrPlanned):
		return ExitPlanned
	case errors.Is(err, platforms.ErrMissingTool), errors.Is(err, platforms.ErrOffline):
		return ExitTool
	default:
		return ExitFail
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, strings.TrimSpace(`
blazium-toolchain — official Blazium console toolchain manager

Usage:
  blazium-toolchain [--json] [--prefix DIR] version
  blazium-toolchain [--json] [--prefix DIR] list
  blazium-toolchain [--json] [--prefix DIR] <platform> <command> [flags]

Platforms:
  ps1       supported (PlayStation 1)
  interdvd  supported (Interactive DVD ISO9660+UDF)
  ps2       planned
  ps3       planned
  ps4       planned

PS1 host tools: Windows and Linux only.

PS1 commands:
  setup [--profile compile|dev|iso] [--offline]
  env
  status
  build --out FILE [--src DIR | --sample template|gte] [--tim|--mesh|--vag|--sprite|--script|--gdbc|--luau|--str|--xa|--node|--hud|--tile|--scene|--anim|--cam|--hit|--nav|--path|--way]
  run [--iso CUE] [--timeout 120s] [--ui] [GAME.EXE]
  iso --xml FILE [--out PATH]
  fmv

Interactive DVD commands:
  iso --dir DIR --out FILE [--meta FILE] [--write-meta FILE] [--volume ID] [--title TEXT]
      [--copyright TEXT] [--license TEXT] [--extra HOST[:DISC]] [--extras-dir DIR] [--recursive]
  meta init --out FILE
  meta validate --meta FILE

License: GPL-3.0-or-later (this repo may contain GCC, PSn00bSDK, mkpsxiso, pcsx-redux).
The 3rd-party installer should invoke this binary (not the Blazium editor).
`)+"\n")
}
