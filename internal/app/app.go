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

	"github.com/blazium-games/blazium-toolchain/internal/cache"
	"github.com/blazium-games/blazium-toolchain/internal/platforms"
	"github.com/blazium-games/blazium-toolchain/internal/platforms/future"
	"github.com/blazium-games/blazium-toolchain/internal/platforms/ps1"
)

const Version = "0.1.0"

const (
	ExitOK      = 0
	ExitUsage   = 1
	ExitPlanned = 2
	ExitTool    = 3
	ExitFail    = 4
)

func init() {
	platforms.Register(ps1.New())
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
		fmt.Fprintf(stderr, "usage: blazium-toolchain %s <setup|env|status|build|run|iso>\n", platID)
		return ExitUsage
	}
	return dispatch(ctx, p, rest, common(*prefix, *jsonOut, stdout, stderr), stdout, stderr)
}

func common(prefix string, jsonOut bool, stdout, stderr io.Writer) platforms.CommonOptions {
	return platforms.CommonOptions{Prefix: prefix, JSON: jsonOut, Stdout: stdout, Stderr: stderr}
}

func cmdVersion(jsonOut bool, w io.Writer) int {
	if jsonOut {
		_ = json.NewEncoder(w).Encode(map[string]string{"name": "blazium-toolchain", "version": Version})
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
	if err := fs.Parse(args); err != nil {
		return platforms.ErrUsage
	}
	return p.Build(ctx, platforms.BuildOptions{CommonOptions: base, Src: *src, Out: *out})
}

func runRun(ctx context.Context, p platforms.Platform, args []string, base platforms.CommonOptions) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(base.Stderr)
	iso := fs.String("iso", "", "optional cue/bin path")
	if err := fs.Parse(args); err != nil {
		return platforms.ErrUsage
	}
	exe := ""
	if fs.NArg() > 0 {
		exe = fs.Arg(0)
	}
	return p.Run(ctx, platforms.RunOptions{CommonOptions: base, Exe: exe, ISO: *iso})
}

func runISO(ctx context.Context, p platforms.Platform, args []string, base platforms.CommonOptions) error {
	fs := flag.NewFlagSet("iso", flag.ContinueOnError)
	fs.SetOutput(base.Stderr)
	xml := fs.String("xml", "", "mkpsxiso project xml")
	out := fs.String("out", "", "output path")
	if err := fs.Parse(args); err != nil {
		return platforms.ErrUsage
	}
	return p.ISO(ctx, platforms.ISOOptions{CommonOptions: base, XML: *xml, Out: *out})
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
  ps1     supported (PlayStation 1)
  ps2     planned
  ps3     planned
  ps4     planned

PS1 commands:
  setup [--profile compile|dev|iso] [--offline]
  env
  status
  build --src DIR --out FILE
  run [--iso CUE] [GAME.EXE]
  iso --xml FILE [--out PATH]

The 3rd-party installer should invoke this binary (not the Blazium editor).
`) + "\n")
}
