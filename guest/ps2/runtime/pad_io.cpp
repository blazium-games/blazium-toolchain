// MIT. Fail-soft pad for PCSX2. BUILD_FOR_PCSX2 on IOP reset. Prefer rom0: modules.

#include "pad_io.h"

#include <kernel.h>
#include <loadfile.h>
#include <sifrpc.h>
#include <string.h>
#include <tamtypes.h>

#ifndef BUILD_FOR_PCSX2
#define BUILD_FOR_PCSX2 1
#endif

#if defined(__has_include)
#if __has_include("irx_flags.h")
#include "irx_flags.h"
#endif
#if __has_include(<libpad.h>)
#include <libpad.h>
#define BLAZIUM_PS2_HAS_LIBPAD 1
#endif
#if __has_include(<iopcontrol.h>)
#include <iopcontrol.h>
#define BLAZIUM_PS2_HAS_IOPCONTROL 1
#endif
#if __has_include(<sbv_patches.h>)
#include <sbv_patches.h>
#define BLAZIUM_PS2_HAS_SBV 1
#endif
#endif

#ifdef BLAZIUM_PS2_HAS_IRX_FILEXIO
extern "C" {
extern const unsigned char irx_fileXio[];
extern const unsigned int size_irx_fileXio;
}
#endif
#ifdef BLAZIUM_PS2_HAS_IRX_IOMANX
extern "C" {
extern const unsigned char irx_iomanX[];
extern const unsigned int size_irx_iomanX;
}
#endif

#ifdef BLAZIUM_PS2_HAS_LIBPAD
static char s_pad_buf[256] __attribute__((aligned(64)));
static int s_ready;
static int s_prev_cross;

static int load_rom_or_buf(const char *rom, const unsigned char *buf, unsigned sz)
{
	int ret = SifLoadModule(rom, 0, NULL);
	if (ret >= 0) {
		return 1;
	}
	if (buf && sz) {
		ret = SifExecModuleBuffer((void *)buf, (int)sz, 0, NULL, NULL);
		return ret >= 0;
	}
	return 0;
}

int pad_io_init(void)
{
	s_ready = 0;
	s_prev_cross = 0;
	SifInitRpc(0);
#ifdef BLAZIUM_PS2_HAS_IOPCONTROL
#if BUILD_FOR_PCSX2
	while (!SifIopReset("", 0)) {
	}
#else
	while (!SifIopReset("rom0:UDNL rom0:EELOADCNF", 0)) {
	}
#endif
	while (!SifIopSync()) {
	}
	SifInitRpc(0);
#endif
#ifdef BLAZIUM_PS2_HAS_SBV
	sbv_patch_enable_lmb();
	sbv_patch_disable_prefix_check();
#endif
#ifdef BLAZIUM_PS2_HAS_IRX_IOMANX
	load_rom_or_buf("rom0:IOMANX", irx_iomanX, size_irx_iomanX);
#else
	SifLoadModule("rom0:IOMANX", 0, NULL);
#endif
#ifdef BLAZIUM_PS2_HAS_IRX_FILEXIO
	load_rom_or_buf("host:fileXio.irx", irx_fileXio, size_irx_fileXio);
#endif
	SifLoadModule("rom0:SIO2MAN", 0, NULL);
	if (SifLoadModule("rom0:PADMAN", 0, NULL) < 0) {
		return 0;
	}
	padInit(0);
	if (padPortOpen(0, 0, s_pad_buf) == 0) {
		return 0;
	}
	s_ready = 1;
	return 1;
}

void pad_io_poll(float *yaw, float *dolly, int *cross_down)
{
	if (yaw) {
		*yaw = 0.0f;
	}
	if (dolly) {
		*dolly = 0.0f;
	}
	if (cross_down) {
		*cross_down = 0;
	}
	if (!s_ready) {
		return;
	}
	struct padButtonStatus buttons;
	int state = padGetState(0, 0);
	if (state != PAD_STATE_STABLE && state != PAD_STATE_FINDCTP1) {
		return;
	}
	if (padRead(0, 0, &buttons) == 0) {
		return;
	}
	const unsigned short btns = ~buttons.btns;
	if (yaw) {
		if (btns & PAD_LEFT) {
			*yaw -= 0.04f;
		}
		if (btns & PAD_RIGHT) {
			*yaw += 0.04f;
		}
		*yaw += ((int)buttons.rjoy_h - 128) * 0.00035f;
	}
	if (dolly) {
		if (btns & PAD_UP) {
			*dolly -= 0.12f;
		}
		if (btns & PAD_DOWN) {
			*dolly += 0.12f;
		}
		*dolly += ((int)buttons.ljoy_v - 128) * 0.0012f;
	}
	const int cross = (btns & PAD_CROSS) ? 1 : 0;
	if (cross_down) {
		*cross_down = (cross && !s_prev_cross) ? 1 : 0;
	}
	s_prev_cross = cross;
}
#else
int pad_io_init(void)
{
	return 0;
}

void pad_io_poll(float *yaw, float *dolly, int *cross_down)
{
	if (yaw) {
		*yaw = 0.0f;
	}
	if (dolly) {
		*dolly = 0.0f;
	}
	if (cross_down) {
		*cross_down = 0;
	}
}
#endif
