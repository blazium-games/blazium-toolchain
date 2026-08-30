package n64

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/blazium-games/blazium-toolchain/internal/platforms"
)

type cookSlice struct {
	flag   string
	src    string
	bin    string
	symbol string
	define string
}

func cookSlices(opts platforms.BuildOptions) []cookSlice {
	return []cookSlice{
		{flag: "node", src: opts.Node, bin: "NODE00.bin", symbol: "cooked_node", define: "BLAZIUM_N64_HAS_NODE"},
		{flag: "mesh", src: opts.Mesh, bin: "MESH00.bin", symbol: "cooked_mesh", define: "BLAZIUM_N64_HAS_MESH"},
		{flag: "ntex", src: opts.Ntex, bin: "NTEX00.bin", symbol: "cooked_ntex", define: "BLAZIUM_N64_HAS_NTEX"},
		{flag: "script", src: opts.Script, bin: "SCRP00.bin", symbol: "cooked_script", define: "BLAZIUM_N64_HAS_SCRIPT"},
		{flag: "gdbc", src: opts.Gdbc, bin: "GDBC00.bin", symbol: "cooked_gdbc", define: "BLAZIUM_N64_HAS_GDBC"},
		{flag: "luau", src: opts.Luau, bin: "LUAU00.bin", symbol: "cooked_luau", define: "BLAZIUM_N64_HAS_LUAU"},
	}
}

func hasCookSlices(opts platforms.BuildOptions) bool {
	for _, s := range cookSlices(opts) {
		if strings.TrimSpace(s.src) != "" {
			return true
		}
	}
	return false
}

func stageCookEmbed(dest string, opts platforms.BuildOptions) error {
	slices := cookSlices(opts)
	var present []cookSlice
	for _, s := range slices {
		if strings.TrimSpace(s.src) == "" {
			continue
		}
		raw, err := os.ReadFile(s.src)
		if err != nil {
			return fmt.Errorf("cook %s: %w", s.flag, err)
		}
		if err := os.WriteFile(filepath.Join(dest, s.bin), raw, 0o644); err != nil {
			return err
		}
		present = append(present, s)
	}
	if len(present) == 0 {
		return nil
	}

	var asm strings.Builder
	asm.WriteString("\t.section .rodata\n")
	for _, s := range present {
		asm.WriteString("\t.align 4\n")
		asm.WriteString("\t.global " + s.symbol + "\n")
		asm.WriteString("\t.global size_" + s.symbol + "\n")
		asm.WriteString(s.symbol + ":\n")
		asm.WriteString("\t.incbin \"" + s.bin + "\"\n")
		asm.WriteString(s.symbol + "_end:\n")
		asm.WriteString("\t.align 4\n")
		asm.WriteString("size_" + s.symbol + ":\n")
		asm.WriteString("\t.word " + s.symbol + "_end - " + s.symbol + "\n")
	}
	if err := os.WriteFile(filepath.Join(dest, "cook_embed.S"), []byte(asm.String()), 0o644); err != nil {
		return err
	}

	var hdr strings.Builder
	hdr.WriteString("#pragma once\n")
	for _, s := range present {
		hdr.WriteString("#define " + s.define + " 1\n")
	}
	if err := os.WriteFile(filepath.Join(dest, "cook_flags.h"), []byte(hdr.String()), 0o644); err != nil {
		return err
	}

	var mk strings.Builder
	mk.WriteString("OBJS += $(BUILD_DIR)/cook_embed.o\n")
	var flags []string
	for _, s := range present {
		flags = append(flags, "-D"+s.define+"=1")
	}
	if len(flags) > 0 {
		mk.WriteString("CXXFLAGS += " + strings.Join(flags, " ") + "\n")
	}
	return os.WriteFile(filepath.Join(dest, "cook.mk"), []byte(mk.String()), 0o644)
}
