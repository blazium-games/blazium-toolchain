package execx

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Runner runs host tools. Tests substitute a fake.
type Runner interface {
	LookPath(name string) (string, error)
	Run(ctx context.Context, name string, args []string, stdout, stderr io.Writer) error
}

type Host struct{}

func (Host) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func (Host) Run(ctx context.Context, name string, args []string, stdout, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

// LookPrefersEnv returns envPath if set and exists, else LookPath.
func LookPrefersEnv(r Runner, envKey, command string) (string, error) {
	if v := os.Getenv(envKey); v != "" {
		if st, err := os.Stat(v); err == nil && !st.IsDir() {
			return v, nil
		}
		// env may be an install root
		if command != "" {
			cand := filepath.Join(v, "bin", command)
			if st, err := os.Stat(cand); err == nil && !st.IsDir() {
				return cand, nil
			}
			cand = filepath.Join(v, command)
			if st, err := os.Stat(cand); err == nil && !st.IsDir() {
				return cand, nil
			}
		}
	}
	if r == nil {
		r = Host{}
	}
	return r.LookPath(command)
}

func Quote(parts ...string) string {
	return strings.Join(parts, " ")
}

func Need(r Runner, name string) error {
	_, err := r.LookPath(name)
	if err != nil {
		return fmt.Errorf("%w: %s", err, name)
	}
	return nil
}
