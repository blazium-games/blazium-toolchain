// MIT. Interpret cooked SCRP ABI 1 (rotate_y + kit name bind). No Godot/Luau VM.

#include "script_vm.h"

#include "gs_draw.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#ifndef BLAZIUM_PS2_COOK_ABI
#define BLAZIUM_PS2_COOK_ABI 1
#endif

enum {
	OP_END = 0,
	OP_LOADK_F = 1,
	OP_LOAD_DELTA = 2,
	OP_MUL = 3,
	OP_CALL_ROTATE_Y = 4,
	OP_READY = 5,
	OP_PROCESS = 6,
	OP_HAS_FEATURE_PS2 = 7
};

static const unsigned char *s_tape;
static unsigned s_tape_n;
static unsigned s_process_off;
static int s_ready;
static int s_kit_spawn;
static int s_kit_player;
static int s_kit_portal;
static float s_yaw;

static unsigned ru16(const unsigned char *p)
{
	return (unsigned)p[0] | ((unsigned)p[1] << 8);
}

static unsigned ru32(const unsigned char *p)
{
	return (unsigned)p[0] | ((unsigned)p[1] << 8) | ((unsigned)p[2] << 16) | ((unsigned)p[3] << 24);
}

static float rf32(const unsigned char *p)
{
	unsigned u = ru32(p);
	float f;
	memcpy(&f, &u, 4);
	return f;
}

static int name_is(const char *n, const char *want)
{
	if (!n || !want) {
		return 0;
	}
	return strcmp(n, want) == 0;
}

static void bind_nodes(const unsigned char *blob, unsigned sz)
{
	if (!blob || sz < 8) {
		return;
	}
	if (blob[0] != 'N' || blob[1] != 'O' || blob[2] != 'D' || blob[3] != 'E') {
		return;
	}
	if (ru16(blob + 4) != BLAZIUM_PS2_COOK_ABI) {
		return;
	}
	const unsigned count = ru16(blob + 6);
	const unsigned rec = 74;
	if (8 + count * rec > sz) {
		return;
	}
	for (unsigned i = 0; i < count; i++) {
		const unsigned char *n = blob + 8 + i * rec;
		char name[33];
		memcpy(name, n + 42, 32);
		name[32] = 0;
		if (name_is(name, "Spawn")) {
			s_kit_spawn = 1;
		} else if (name_is(name, "Player")) {
			s_kit_player = 1;
		} else if (name_is(name, "Portal")) {
			s_kit_portal = 1;
		}
	}
}

static void try_bind_node01(void)
{
	static const char *paths[] = {
		"host:NODE01.bin",
		"cdrom0:\\NODE01.BIN;1",
		"cdrom0:NODE01.BIN;1",
		NULL
	};
	for (int i = 0; paths[i]; i++) {
		FILE *f = fopen(paths[i], "rb");
		if (!f) {
			continue;
		}
		if (fseek(f, 0, SEEK_END) != 0) {
			fclose(f);
			continue;
		}
		long n = ftell(f);
		if (n < 8) {
			fclose(f);
			continue;
		}
		rewind(f);
		unsigned char *buf = (unsigned char *)malloc((size_t)n);
		if (!buf) {
			fclose(f);
			continue;
		}
		if (fread(buf, 1, (size_t)n, f) == (size_t)n) {
			bind_nodes(buf, (unsigned)n);
		}
		free(buf);
		fclose(f);
		return;
	}
}

static void run_range(unsigned from, unsigned to, float delta)
{
	float stack[8];
	int sp = 0;
	unsigned i = from;
	while (i < to && i < s_tape_n) {
		const unsigned char op = s_tape[i++];
		if (op == OP_END) {
			break;
		}
		if (op == OP_READY) {
			continue;
		}
		if (op == OP_PROCESS) {
			break;
		}
		if (op == OP_LOADK_F) {
			if (i + 4 > s_tape_n || sp >= 8) {
				return;
			}
			stack[sp++] = rf32(s_tape + i);
			i += 4;
			continue;
		}
		if (op == OP_LOAD_DELTA) {
			if (sp >= 8) {
				return;
			}
			stack[sp++] = delta;
			continue;
		}
		if (op == OP_MUL) {
			if (sp < 2) {
				return;
			}
			const float b = stack[--sp];
			const float a = stack[--sp];
			stack[sp++] = a * b;
			continue;
		}
		if (op == OP_CALL_ROTATE_Y) {
			if (sp < 1) {
				return;
			}
			s_yaw += stack[--sp];
			gs_draw_set_world_yaw(s_yaw);
			continue;
		}
		if (op == OP_HAS_FEATURE_PS2) {
			if (sp < 8) {
				stack[sp++] = 1.0f;
			}
			continue;
		}
	}
}

int script_vm_init(const unsigned char *scrp, unsigned scrp_sz,
		const unsigned char *node, unsigned node_sz)
{
	s_ready = 0;
	s_tape = 0;
	s_tape_n = 0;
	s_process_off = 0;
	s_kit_spawn = 0;
	s_kit_player = 0;
	s_kit_portal = 0;
	s_yaw = 0.0f;
	bind_nodes(node, node_sz);
	try_bind_node01();
	if (!scrp || scrp_sz < 12) {
		return 0;
	}
	if (scrp[0] != 'S' || scrp[1] != 'C' || scrp[2] != 'R' || scrp[3] != 'P') {
		return 0;
	}
	if (ru16(scrp + 4) != BLAZIUM_PS2_COOK_ABI) {
		return 0;
	}
	const unsigned tape_n = ru32(scrp + 8);
	if (12 + tape_n > scrp_sz) {
		return 0;
	}
	s_tape = scrp + 12;
	s_tape_n = tape_n;
	s_process_off = 0;
	for (unsigned i = 0; i < s_tape_n; i++) {
		if (s_tape[i] == OP_PROCESS) {
			s_process_off = i + 1;
			break;
		}
		if (s_tape[i] == OP_LOADK_F) {
			i += 4;
		}
	}
	run_range(0, s_tape_n, 0.0f);
	s_ready = 1;
	return 1;
}

void script_vm_process(float delta)
{
	if (!s_ready || !s_tape) {
		return;
	}
	run_range(s_process_off, s_tape_n, delta);
}

int script_vm_ready(void)
{
	return s_ready;
}

int script_vm_kit_spawn(void)
{
	return s_kit_spawn;
}

int script_vm_kit_player(void)
{
	return s_kit_player;
}

int script_vm_kit_portal(void)
{
	return s_kit_portal;
}
