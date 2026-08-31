// Blazium N64 guest — MIT. Links Unlicense libdragon only (rdpq / joypad / mixer / dfs).
// Product is a big-endian .z64. Intermediate .elf is GDB-only.

#include "dfs_io.h"
#include "guest_hooks.h"
#include "pack_io.h"
#include "pad_io.h"
#include "rdpq_draw.h"
#include "script_vm.h"
#include "sfx_io.h"
#include "sys_io.h"

#include <libdragon.h>

#define BLAZIUM_N64_COOK_ABI_VALUE 1

#ifndef BLAZIUM_N64_COOK_ABI
#define BLAZIUM_N64_COOK_ABI BLAZIUM_N64_COOK_ABI_VALUE
#endif

#if defined(__has_include)
#if __has_include("cook_flags.h")
#include "cook_flags.h"
#endif
#endif

#ifdef BLAZIUM_N64_HAS_NODE
extern "C" {
extern const unsigned char cooked_node[];
extern const unsigned char cooked_node_end[];
}
#endif
#ifdef BLAZIUM_N64_HAS_SCRIPT
extern "C" {
extern const unsigned char cooked_script[];
extern const unsigned char cooked_script_end[];
}
#endif

int main(void)
{
#ifdef BLAZIUM_N64_DISPLAY_640
	display_init(RESOLUTION_640x480, DEPTH_16_BPP, 2, GAMMA_NONE, FILTERS_RESAMPLE);
#else
	display_init(RESOLUTION_320x240, DEPTH_16_BPP, 2, GAMMA_NONE, FILTERS_RESAMPLE);
#endif
	rdpq_draw_init();
	joypad_init();
	pad_io_init();
	dfs_io_init();
	pack_io_init();
	sfx_io_init();
	sys_io_init();

	const unsigned char *scrp = NULL;
	unsigned scrp_sz = 0;
	const unsigned char *node = NULL;
	unsigned node_sz = 0;
#ifdef BLAZIUM_N64_HAS_SCRIPT
	scrp = cooked_script;
	scrp_sz = (unsigned)(cooked_script_end - cooked_script);
#endif
#ifdef BLAZIUM_N64_HAS_NODE
	node = cooked_node;
	node_sz = (unsigned)(cooked_node_end - cooked_node);
#endif
	script_vm_init(scrp, scrp_sz, node, node_sz);
	if (user_init) {
		user_init();
	}

	while (1) {
		pad_io_poll();
		if (user_pad) {
			user_pad();
		}
		script_vm_process(1.0f / 60.0f);
		if (user_tick) {
			user_tick(1.0f / 60.0f);
		}
		rdpq_draw_frame();
		sfx_io_tick();
	}
}
