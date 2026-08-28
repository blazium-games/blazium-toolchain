package platforms

import (
	"context"
	"io"
)

// Status is the public lifecycle of a console target.
type Status string

const (
	StatusSupported Status = "supported"
	StatusPlanned   Status = "planned"
)

// Info is listed by `blazium-toolchain list` and the 3rd-party downloader.
type Info struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Status      Status   `json:"status"`
	Commands    []string `json:"commands"`
	Description string   `json:"description"`
}

// EnvMap is toolchain paths the editor or a parent installer can consume.
type EnvMap map[string]string

// Platform is one console family. Only PS1 is implemented; others return ErrPlanned.
type Platform interface {
	Info() Info
	Setup(ctx context.Context, opts SetupOptions) error
	Env(opts CommonOptions) (EnvMap, error)
	Status(opts CommonOptions) (map[string]any, error)
	Build(ctx context.Context, opts BuildOptions) error
	Run(ctx context.Context, opts RunOptions) error
	ISO(ctx context.Context, opts ISOOptions) error
}

// CommonOptions apply to most commands.
type CommonOptions struct {
	Prefix string
	JSON   bool
	Stdout io.Writer
	Stderr io.Writer
}

// SetupOptions selects a download/layout profile.
type SetupOptions struct {
	CommonOptions
	Profile string
	Offline bool
}

// BuildOptions compile a guest with the platform SDK (spawn only).
type BuildOptions struct {
	CommonOptions
	Src string
	Out string
}

// RunOptions boot the emulator.
type RunOptions struct {
	CommonOptions
	Exe string
	ISO string
}

// ISOOptions spawn an ISO packer (GPL tools must stay external).
type ISOOptions struct {
	CommonOptions
	XML string
	Out string
}
