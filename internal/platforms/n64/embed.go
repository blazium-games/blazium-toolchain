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
		{flag: "inp", src: opts.Inp, bin: "INP600.bin", symbol: "cooked_inp", define: "BLAZIUM_N64_HAS_INP"},
		{flag: "script", src: opts.Script, bin: "SCRP00.bin", symbol: "cooked_script", define: "BLAZIUM_N64_HAS_SCRIPT"},
		{flag: "gdbc", src: opts.Gdbc, bin: "GDBC00.bin", symbol: "cooked_gdbc", define: "BLAZIUM_N64_HAS_GDBC"},
		{flag: "luau", src: opts.Luau, bin: "LUAU00.bin", symbol: "cooked_luau", define: "BLAZIUM_N64_HAS_LUAU"},
	}
}

func hasCookAudio(opts platforms.BuildOptions) bool {
	return strings.TrimSpace(opts.Sfx) != "" || strings.TrimSpace(opts.Music) != ""
}

func hasCookSlices(opts platforms.BuildOptions) bool {
	for _, s := range cookSlices(opts) {
		if strings.TrimSpace(s.src) != "" {
			return true
		}
	}
	return hasCookAudio(opts)
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

	var hdr strings.Builder
	hdr.WriteString("#pragma once\n")
	var mk strings.Builder
	if len(present) > 0 {
		var asm strings.Builder
		asm.WriteString("\t.section .rodata\n")
		for _, s := range present {
			// Start/end labels only. A .word size after .incbin lands in GP small-data
			// and mips64-elf ld fails (R_MIPS_GPREL16) once NODE/MESH is non-empty.
			asm.WriteString("\t.align 4\n")
			asm.WriteString("\t.global " + s.symbol + "\n")
			asm.WriteString("\t.global " + s.symbol + "_end\n")
			asm.WriteString(s.symbol + ":\n")
			asm.WriteString("\t.incbin \"" + s.bin + "\"\n")
			asm.WriteString("\t.align 4\n")
			asm.WriteString(s.symbol + "_end:\n")
		}
		if err := os.WriteFile(filepath.Join(dest, "cook_embed.S"), []byte(asm.String()), 0o644); err != nil {
			return err
		}
		for _, s := range present {
			hdr.WriteString("#define " + s.define + " 1\n")
		}
		mk.WriteString("OBJS += $(BUILD_DIR)/cook_embed.o\n")
		var flags []string
		for _, s := range present {
			flags = append(flags, "-D"+s.define+"=1")
		}
		if len(flags) > 0 {
			mk.WriteString("CXXFLAGS += " + strings.Join(flags, " ") + "\n")
		}
	}
	if err := stageCookAudio(dest, opts, &hdr, &mk); err != nil {
		return err
	}
	if hdr.Len() > len("#pragma once\n") {
		if err := os.WriteFile(filepath.Join(dest, "cook_flags.h"), []byte(hdr.String()), 0o644); err != nil {
			return err
		}
	}
	if mk.Len() > 0 {
		if err := os.WriteFile(filepath.Join(dest, "cook.mk"), []byte(mk.String()), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func stageCookAudio(dest string, opts platforms.BuildOptions, hdr, mk *strings.Builder) error {
	type wavAsset struct {
		src, name, define string
	}
	assets := []wavAsset{
		{src: opts.Sfx, name: "SFX00.wav", define: "BLAZIUM_N64_HAS_SFX"},
		{src: opts.Music, name: "MUSIC00.wav", define: "BLAZIUM_N64_HAS_MUSIC"},
	}
	copied := 0
	if err := os.MkdirAll(filepath.Join(dest, "assets"), 0o755); err != nil {
		return err
	}
	for _, a := range assets {
		if strings.TrimSpace(a.src) == "" {
			continue
		}
		raw, err := os.ReadFile(a.src)
		if err != nil {
			return fmt.Errorf("cook audio %s: %w", a.name, err)
		}
		if len(raw) >= 2 && raw[0] == 0x4D && raw[1] == 0x5A {
			return fmt.Errorf("cook audio %s: PE/MZ refused", a.name)
		}
		if len(raw) < 12 || string(raw[0:4]) != "RIFF" || string(raw[8:12]) != "WAVE" {
			return fmt.Errorf("cook audio %s: need RIFF WAVE (not VAG/wav64)", a.name)
		}
		if err := os.WriteFile(filepath.Join(dest, "assets", a.name), raw, 0o644); err != nil {
			return err
		}
		wav64Name := a.name[:len(a.name)-4] + ".wav64"
		loop := a.name == "MUSIC00.wav"
		if err := writeWav64File(filepath.Join(dest, "assets", a.name), filepath.Join(dest, "filesystem", wav64Name), loop); err != nil {
			return fmt.Errorf("wav64 %s: %w", wav64Name, err)
		}
		hdr.WriteString("#define " + a.define + " 1\n")
		mk.WriteString("CXXFLAGS += -D" + a.define + "=1\n")
		mk.WriteString("$(BUILD_DIR)/$(ROMNAME).dfs: filesystem/" + wav64Name + "\n")
		copied++
	}
	if copied == 0 {
		return nil
	}
	mk.WriteString("$(ROMNAME).z64: $(BUILD_DIR)/$(ROMNAME).dfs\n")
	return nil
}
