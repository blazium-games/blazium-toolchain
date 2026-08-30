package guest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallWritesRuntime(t *testing.T) {
	dir := t.TempDir()
	if err := Install(dir); err != nil {
		t.Fatal(err)
	}
	if CookABI != 1 {
		t.Fatalf("CookABI %d", CookABI)
	}
	for _, name := range RuntimeNames {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
}

func TestEmbeddedRuntimeComplete(t *testing.T) {
	for _, name := range RuntimeNames {
		if _, err := files.ReadFile("runtime/" + name); err != nil {
			t.Fatalf("embed missing %s: %v", name, err)
		}
	}
}

func TestRuntimeNamesHasVU1(t *testing.T) {
	var cpp, hdr, vsm bool
	for _, name := range RuntimeNames {
		switch name {
		case "vu1_draw.cpp":
			cpp = true
		case "vu1_draw.h":
			hdr = true
		case "draw_3D.vsm":
			vsm = true
		}
	}
	if !cpp || !hdr || !vsm {
		t.Fatalf("RuntimeNames missing VU1 files: %v", RuntimeNames)
	}
}

func TestGuestPack1AndDisp(t *testing.T) {
	pack, err := files.ReadFile("runtime/pack_io.cpp")
	if err != nil {
		t.Fatal(err)
	}
	src := string(pack)
	if (!strings.Contains(src, "MESH01") && !strings.Contains(src, "MESH%02d")) || !strings.Contains(src, "pack_io_swap") {
		t.Fatal("pack_io.cpp must load MESH%02d / MESH01 and expose pack_io_swap")
	}
	if !strings.Contains(src, "STREAM") {
		t.Fatal("pack_io.cpp must read STREAM.bin for extra packs")
	}
	main, err := files.ReadFile("runtime/main.cpp")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(main), "DISP.bin") {
		t.Fatal("main.cpp must read DISP.bin for region/width")
	}
	sfx, err := files.ReadFile("runtime/sfx_io.cpp")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(sfx), "sfx_io_audible") || !strings.Contains(string(sfx), "rSdInit") || !strings.Contains(string(sfx), "sce_SDR_DEV") {
		t.Fatal("sfx_io.cpp must use sdrdrv RPC (rSdInit / sce_SDR_DEV) and expose sfx_io_audible")
	}
	if strings.Contains(string(sfx), "#include <libsd.h>") || strings.Contains(string(sfx), "#include <libsdr.h>") || strings.Contains(string(sfx), "sceSdInit(") || strings.Contains(string(sfx), "sceSdRemote(") {
		t.Fatal("sfx_io.cpp must not include libsd.h/libsdr.h or call sceSdInit/sceSdRemote")
	}
	if !strings.Contains(string(sfx), "sdrdrv") && !strings.Contains(string(sfx), "SDRDRV") {
		t.Fatal("sfx_io.cpp must load sdrdrv IRX for the libsdr RPC server")
	}
	if !strings.Contains(string(sfx), "MUSIC00") || !strings.Contains(string(sfx), "sfx_io_music_play") {
		t.Fatal("sfx_io.cpp must load MUSIC00 and expose sfx_io_music_play")
	}
	if !strings.Contains(string(sfx), "ADPCM_LOOP") || !strings.Contains(string(sfx), "0x10000") {
		t.Fatal("sfx_io.cpp must loop music VAG on SPU addr 0x10000")
	}
}

func TestGuestInputAndUserIO(t *testing.T) {
	pad, err := files.ReadFile("runtime/pad_io.h")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(pad), "pad_io_pressed") || !strings.Contains(string(pad), "pad_io_stick") {
		t.Fatal("pad_io.h must expose Input action/stick queries")
	}
	pack, err := files.ReadFile("runtime/pack_io.h")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(pack), "pack_io_user_save") || !strings.Contains(string(pack), "pack_io_user_error") {
		t.Fatal("pack_io.h must map user:// with a named error")
	}
	vm, err := files.ReadFile("runtime/script_vm.cpp")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(vm), "OP_INPUT_TRANSLATE") || !strings.Contains(string(vm), "OP_USER_SAVE") {
		t.Fatal("script_vm must interpret Input translate and user:// save")
	}
	if !strings.Contains(string(vm), "OP_KIT_TICK") || !strings.Contains(string(vm), "Checkpoint") {
		t.Fatal("script_vm must interpret kit tick and Checkpoint")
	}
	if !strings.Contains(string(vm), "OP_SAY") || !strings.Contains(string(vm), "OP_PLAY_FMV") {
		t.Fatal("script_vm must interpret say and play_fmv")
	}
	if !strings.Contains(string(vm), "OP_PLAY_MUSIC") || !strings.Contains(string(vm), "OP_MUSIC_VOL") {
		t.Fatal("script_vm must interpret load_music / set_music_volume")
	}
	if !strings.Contains(string(vm), "OP_LOOK_STICK") || !strings.Contains(string(vm), "OP_SHAKE_CAMERA") {
		t.Fatal("script_vm must interpret look_camera / shake_camera")
	}
	if !strings.Contains(string(vm), "OP_TWEEN") || !strings.Contains(string(vm), "OP_TIMER") {
		t.Fatal("script_vm must interpret create_tween / create_timer")
	}
}

func TestGuestCameraLookAndVu1Fallback(t *testing.T) {
	gs, err := files.ReadFile("runtime/gs_draw.cpp")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(gs), "gs_draw_set_camera") || !strings.Contains(string(gs), "euler_yxz_negz") {
		t.Fatal("gs_draw.cpp must apply Camera3D look-at from YXZ euler")
	}
	if !strings.Contains(string(gs), "gs_draw_look") || !strings.Contains(string(gs), "gs_draw_orbit_sph") || !strings.Contains(string(gs), "gs_draw_shake") {
		t.Fatal("gs_draw.cpp must expose look/orbit/shake")
	}
	if !strings.Contains(string(gs), "g_fov") {
		t.Fatal("gs_draw.cpp must use cooked FOV instead of a hardcoded 55")
	}
	sys, err := files.ReadFile("runtime/sys_io.cpp")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(sys), "gs_draw_set_camera") {
		t.Fatal("sys_io.cpp must feed CAM00 euler+fov into gs_draw_set_camera")
	}
	if !strings.Contains(string(sys), "sys_io_look") || !strings.Contains(string(sys), "sys_io_next_cam") {
		t.Fatal("sys_io.cpp must expose look_camera / next_camera")
	}
	if !strings.Contains(string(sys), "sys_io_tween_start") || !strings.Contains(string(sys), "sys_io_timer_start") {
		t.Fatal("sys_io.cpp must expose linear tweens and timers")
	}
	cmake, err := files.ReadFile("runtime/CMakeLists.txt")
	if err != nil {
		t.Fatal(err)
	}
	cm := string(cmake)
	if !strings.Contains(cm, "CPU GIF fallback") || !strings.Contains(cm, "BLAZIUM_DVP_AS") {
		t.Fatal("CMakeLists must enable VU1 when dvp-as exists and name the CPU GIF fallback")
	}
	if strings.Contains(cm, "-lgskit") || strings.Contains(strings.ToLower(cm), "libgskit") {
		t.Fatal("CMakeLists must not link gsKit")
	}
}

func TestGuestSysIO(t *testing.T) {
	sys, err := files.ReadFile("runtime/sys_io.cpp")
	if err != nil {
		t.Fatal(err)
	}
	src := string(sys)
	if !strings.Contains(src, "CAM00") || !strings.Contains(src, "HUD00") || !strings.Contains(src, "NAV00") {
		t.Fatal("sys_io.cpp must load CAM/HUD/NAV sidecars")
	}
	if !strings.Contains(src, "FMV/IPU") && !strings.Contains(src, "license-clean") {
		t.Fatal("sys_io.cpp must name FMV/IPU skip")
	}
}

func TestRuntimeNamesHasScriptVM(t *testing.T) {
	var vm, hdr bool
	for _, name := range RuntimeNames {
		if name == "script_vm.cpp" {
			vm = true
		}
		if name == "script_vm.h" {
			hdr = true
		}
	}
	if !vm || !hdr {
		t.Fatalf("RuntimeNames missing script_vm: %v", RuntimeNames)
	}
}

func TestOverlayReplacesAndAdds(t *testing.T) {
	dir := t.TempDir()
	if err := Install(dir); err != nil {
		t.Fatal(err)
	}
	over := t.TempDir()
	if err := os.MkdirAll(filepath.Join(over, "extra"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(over, "main.cpp"), []byte("// overlay main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(over, "extra", "hooks.cpp"), []byte("// extra hook\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Overlay(dir, over); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "main.cpp"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "// overlay main\n" {
		t.Fatalf("main.cpp not replaced: %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "extra", "hooks.cpp")); err != nil {
		t.Fatal(err)
	}
}
