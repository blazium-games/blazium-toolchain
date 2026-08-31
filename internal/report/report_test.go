package report

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/blazium-games/blazium-toolchain/internal/platforms"
)

func TestLineAlignsVerb(t *testing.T) {
	var buf bytes.Buffer
	Line(&buf, Fetching, "libdragon toolchain")
	got := buf.String()
	if got != "fetching   libdragon toolchain\n" {
		t.Fatalf("got %q", got)
	}
}

func TestWriteErrorTicketBlock(t *testing.T) {
	err := Missing("n64 compile needs N64_INST (mips64-elf-gcc, n64.mk, libdragon.a)", "blazium-toolchain n64 setup --profile compile")
	if !errors.Is(err, platforms.ErrMissingTool) {
		t.Fatalf("unwrap: %v", err)
	}
	var buf bytes.Buffer
	WriteError(&buf, err, Context{
		Command: "n64 build --out hello.z64",
		Prefix:  `C:\cache`,
		Version: "0.1.0",
	})
	got := buf.String()
	for _, want := range []string{
		"error      missing-tool",
		"n64 compile needs N64_INST",
		"fix        blazium-toolchain n64 setup --profile compile",
		"---",
		"command    blazium-toolchain n64 build --out hello.z64",
		"host       " + runtime.GOOS + "/" + runtime.GOARCH,
		"version    blazium-toolchain 0.1.0",
		"prefix     C:\\cache",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
}

func TestWriteErrorJSON(t *testing.T) {
	err := Usage("rom requires --out FILE.z64")
	var buf bytes.Buffer
	WriteError(&buf, err, Context{Command: "n64 rom", Version: "0.1.0", JSON: true})
	var rec Record
	if e := json.Unmarshal(buf.Bytes(), &rec); e != nil {
		t.Fatal(e)
	}
	if rec.Kind != "usage" || rec.Message != "rom requires --out FILE.z64" {
		t.Fatalf("%+v", rec)
	}
	if rec.Command != "n64 rom" || rec.Fix != "blazium-toolchain --help" {
		t.Fatalf("%+v", rec)
	}
}

func TestExitCode(t *testing.T) {
	if ExitCode(nil) != ExitOK {
		t.Fatal("nil")
	}
	if ExitCode(Usage("x")) != ExitUsage {
		t.Fatal("usage")
	}
	if ExitCode(Missing("x", "")) != ExitTool {
		t.Fatal("missing")
	}
	if ExitCode(Planned("x", "")) != ExitPlanned {
		t.Fatal("planned")
	}
	if ExitCode(Fail("x", "")) != ExitFail {
		t.Fatal("fail")
	}
}

func TestStripPlainSentinel(t *testing.T) {
	rec := Classify(fmt.Errorf("%w: cmake", platforms.ErrMissingTool), Context{})
	if rec.Kind != "missing-tool" || rec.Message != "cmake" {
		t.Fatalf("%+v", rec)
	}
}
