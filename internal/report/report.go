// Package report formats CLI process lines and errors so they can be
// pasted into a GitHub issue without extra context.
package report

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"runtime"
	"sort"
	"strings"

	"github.com/blazium-games/blazium-toolchain/internal/platforms"
)

// Process verbs. Keep these stable — they are the only stdout kinds.
const (
	Fetching   = "fetching"
	Reusing    = "reusing"
	Installing = "installing"
	Building   = "building"
	Staged     = "staged"
	Skip       = "skip"
	Wrote      = "wrote"
	Ready      = "ready"
	Note       = "note"
	Prefix     = "prefix"
)

// Exit codes match internal/app.
const (
	ExitOK      = 0
	ExitUsage   = 1
	ExitPlanned = 2
	ExitTool    = 3
	ExitFail    = 4
)

// Context is attached to every error so a pasted block is self-contained.
type Context struct {
	Command  string
	Prefix   string
	Version  string
	Settings string
	JSON     bool
}

// Record is the ticket/JSON shape of a failure.
type Record struct {
	Kind    string `json:"error"`
	Message string `json:"message"`
	Fix     string `json:"fix,omitempty"`
	Command string `json:"command,omitempty"`
	Prefix   string `json:"prefix,omitempty"`
	Version  string `json:"version,omitempty"`
	Settings string `json:"settings,omitempty"`
	GOOS     string `json:"os"`
	GOARCH   string `json:"arch"`
}

type coded struct {
	kind error
	what string
	fix  string
}

func (c *coded) Error() string {
	if c.what == "" {
		return c.kind.Error()
	}
	return c.kind.Error() + ": " + c.what
}

func (c *coded) Unwrap() error { return c.kind }

// Wrap attaches a human what/fix to a sentinel. Error() stays unwrap-friendly.
func Wrap(kind error, what, fix string) error {
	if kind == nil {
		kind = errFail
	}
	return &coded{kind: kind, what: strings.TrimSpace(what), fix: strings.TrimSpace(fix)}
}

var errFail = errors.New("fail")

func Usage(what string) error { return Wrap(platforms.ErrUsage, what, "blazium-toolchain --help") }
func Missing(what, fix string) error {
	return Wrap(platforms.ErrMissingTool, what, fix)
}
func Offline(what, fix string) error { return Wrap(platforms.ErrOffline, what, fix) }
func Planned(what, fix string) error { return Wrap(platforms.ErrPlanned, what, fix) }
func Fail(what, fix string) error    { return Wrap(errFail, what, fix) }

// Line writes one process line: "verb       detail".
func Line(w io.Writer, verb, msg string) {
	if w == nil {
		return
	}
	verb = strings.TrimSpace(verb)
	msg = strings.TrimSpace(msg)
	if verb == "" {
		return
	}
	if msg == "" {
		fmt.Fprintln(w, verb)
		return
	}
	fmt.Fprintf(w, "%-10s %s\n", verb, msg)
}

// Linef is Line with fmt.Sprintf.
func Linef(w io.Writer, verb, format string, args ...any) {
	Line(w, verb, fmt.Sprintf(format, args...))
}

// Setup writes the ready/prefix/env block used by every platform setup.
func Setup(w io.Writer, plat, profile, prefix string, env map[string]string) {
	Linef(w, Ready, "%s %s", plat, profile)
	Line(w, Prefix, prefix)
	Env(w, env)
}

// Env writes KEY=value lines (setup/env product). Empty values are omitted.
func Env(w io.Writer, env map[string]string) {
	if w == nil || len(env) == 0 {
		return
	}
	keys := make([]string, 0, len(env))
	for k, v := range env {
		if v != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(w, "%s=%s\n", k, env[k])
	}
}

// Classify builds a Record from err. Safe to call with a nil Context.
func Classify(err error, ctx Context) Record {
	rec := Record{
		Kind:    kindLabel(err),
		Message: stripKindPrefix(err),
		Command: strings.TrimSpace(ctx.Command),
		Prefix:   strings.TrimSpace(ctx.Prefix),
		Version:  strings.TrimSpace(ctx.Version),
		Settings: strings.TrimSpace(ctx.Settings),
		GOOS:     runtime.GOOS,
		GOARCH:  runtime.GOARCH,
	}
	var c *coded
	if errors.As(err, &c) {
		rec.Message = c.what
		rec.Fix = c.fix
	}
	if rec.Message == "" {
		rec.Message = err.Error()
	}
	return rec
}

// WriteError prints a ticket-ready block (or JSON when ctx.JSON).
func WriteError(w io.Writer, err error, ctx Context) {
	if err == nil || w == nil {
		return
	}
	rec := Classify(err, ctx)
	if ctx.JSON {
		_ = json.NewEncoder(w).Encode(rec)
		return
	}
	fmt.Fprintf(w, "error      %s\n", rec.Kind)
	for _, line := range splitMsg(rec.Message) {
		fmt.Fprintf(w, "           %s\n", line)
	}
	if rec.Fix != "" {
		fmt.Fprintf(w, "fix        %s\n", rec.Fix)
	}
	fmt.Fprintln(w, "---")
	if rec.Command != "" {
		fmt.Fprintf(w, "command    blazium-toolchain %s\n", rec.Command)
	}
	fmt.Fprintf(w, "host       %s/%s\n", rec.GOOS, rec.GOARCH)
	if rec.Version != "" {
		fmt.Fprintf(w, "version    blazium-toolchain %s\n", rec.Version)
	}
	if rec.Prefix != "" {
		fmt.Fprintf(w, "prefix     %s\n", rec.Prefix)
	}
	if rec.Settings != "" {
		fmt.Fprintf(w, "settings   %s\n", rec.Settings)
	}
}

// ExitCode maps sentinels to the CLI exit codes.
func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	switch {
	case errors.Is(err, platforms.ErrUsage), errors.Is(err, platforms.ErrUnknownPlatform):
		return ExitUsage
	case errors.Is(err, platforms.ErrPlanned):
		return ExitPlanned
	case errors.Is(err, platforms.ErrMissingTool), errors.Is(err, platforms.ErrOffline):
		return ExitTool
	default:
		return ExitFail
	}
}

func kindLabel(err error) string {
	switch {
	case errors.Is(err, platforms.ErrUsage):
		return "usage"
	case errors.Is(err, platforms.ErrMissingTool):
		return "missing-tool"
	case errors.Is(err, platforms.ErrOffline):
		return "offline"
	case errors.Is(err, platforms.ErrPlanned):
		return "not-implemented"
	case errors.Is(err, platforms.ErrUnknownPlatform):
		return "unknown-platform"
	default:
		return "fail"
	}
}

func stripKindPrefix(err error) string {
	s := err.Error()
	for _, kind := range []error{
		platforms.ErrUsage,
		platforms.ErrMissingTool,
		platforms.ErrOffline,
		platforms.ErrPlanned,
		platforms.ErrUnknownPlatform,
		errFail,
	} {
		p := kind.Error() + ": "
		if strings.HasPrefix(s, p) {
			return strings.TrimPrefix(s, p)
		}
	}
	return s
}

func splitMsg(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "\n")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
