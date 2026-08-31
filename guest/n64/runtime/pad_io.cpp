// MIT. libdragon joypad. INP6 ABI 1 maps InputMap actions. Rumble is P8-only.

#include "pad_io.h"

#include <libdragon.h>
#include <string.h>

#ifndef BLAZIUM_N64_COOK_ABI
#define BLAZIUM_N64_COOK_ABI 1
#endif

#if defined(__has_include)
#if __has_include("cook_flags.h")
#include "cook_flags.h"
#endif
#endif

#ifdef BLAZIUM_N64_HAS_INP
extern "C" {
extern const unsigned char cooked_inp[];
extern const unsigned char cooked_inp_end[];
}
#endif

static float s_dead = 0.2f;
static float s_sx;
static float s_sy;
static unsigned s_held;
static unsigned s_down;

enum {
	ACT_ACCEPT = 0,
	ACT_CANCEL = 1,
	ACT_PAUSE = 2,
	ACT_LEFT = 3,
	ACT_RIGHT = 4,
	ACT_UP = 5,
	ACT_DOWN = 6
};

enum {
	BTN_A = 1 << 0,
	BTN_B = 1 << 1,
	BTN_Z = 1 << 2,
	BTN_START = 1 << 3,
	BTN_L = 1 << 4,
	BTN_R = 1 << 5,
	BTN_DLEFT = 1 << 6,
	BTN_DRIGHT = 1 << 7,
	BTN_DUP = 1 << 8,
	BTN_DDOWN = 1 << 9,
	BTN_CLEFT = 1 << 10,
	BTN_CRIGHT = 1 << 11,
	BTN_CUP = 1 << 12,
	BTN_CDOWN = 1 << 13
};

#ifdef BLAZIUM_N64_HAS_INP
static uint16_t ru16le(const unsigned char *p)
{
	return (uint16_t)p[0] | ((uint16_t)p[1] << 8);
}

struct InpRow {
	uint8_t id;
	uint16_t buttons;
	uint8_t stick;
};

static InpRow s_rows[32];
static int s_row_n;
static int s_inp_ok;

static int load_inp6(void)
{
	if (s_inp_ok != 0) {
		return s_inp_ok > 0;
	}
	const unsigned char *p = cooked_inp;
	const unsigned sz = (unsigned)(cooked_inp_end - cooked_inp);
	if (!p || sz < 8 || p[0] != 'I' || p[1] != 'N' || p[2] != 'P' || p[3] != '6') {
		s_inp_ok = -1;
		return 0;
	}
	if (p[0] == 0x4D && p[1] == 0x5A) {
		s_inp_ok = -1;
		return 0;
	}
	if (ru16le(p + 4) != (uint16_t)BLAZIUM_N64_COOK_ABI) {
		s_inp_ok = -1;
		return 0;
	}
	const unsigned n = ru16le(p + 6);
	if (n < 1 || n > 32 || sz < 8u + n * 32u) {
		s_inp_ok = -1;
		return 0;
	}
	for (unsigned i = 0; i < n; i++) {
		const unsigned char *row = p + 8 + i * 32;
		s_rows[i].id = row[0];
		s_rows[i].buttons = ru16le(row + 1);
		s_rows[i].stick = row[3];
	}
	s_row_n = (int)n;
	s_inp_ok = 1;
	return 1;
}

static unsigned joy_mask(joypad_buttons_t b)
{
	unsigned m = 0;
	if (b.a) {
		m |= BTN_A;
	}
	if (b.b) {
		m |= BTN_B;
	}
	if (b.z) {
		m |= BTN_Z;
	}
	if (b.start) {
		m |= BTN_START;
	}
	if (b.l) {
		m |= BTN_L;
	}
	if (b.r) {
		m |= BTN_R;
	}
	if (b.d_left) {
		m |= BTN_DLEFT;
	}
	if (b.d_right) {
		m |= BTN_DRIGHT;
	}
	if (b.d_up) {
		m |= BTN_DUP;
	}
	if (b.d_down) {
		m |= BTN_DDOWN;
	}
	if (b.c_left) {
		m |= BTN_CLEFT;
	}
	if (b.c_right) {
		m |= BTN_CRIGHT;
	}
	if (b.c_up) {
		m |= BTN_CUP;
	}
	if (b.c_down) {
		m |= BTN_CDOWN;
	}
	return m;
}
#endif

void pad_io_init(void)
{
	joypad_init();
#ifdef BLAZIUM_N64_HAS_INP
	(void)load_inp6();
#endif
}

void pad_io_poll(void)
{
	joypad_poll();
	joypad_buttons_t b = joypad_get_buttons_pressed(JOYPAD_PORT_1);
	joypad_inputs_t in = joypad_get_inputs(JOYPAD_PORT_1);
	s_sx = (float)in.stick_x / 80.0f;
	s_sy = (float)in.stick_y / 80.0f;
	if (s_sx > -s_dead && s_sx < s_dead) {
		s_sx = 0.0f;
	}
	if (s_sy > -s_dead && s_sy < s_dead) {
		s_sy = 0.0f;
	}
	s_held = 0;
#ifdef BLAZIUM_N64_HAS_INP
	if (load_inp6()) {
		const unsigned now = joy_mask(b);
		for (int i = 0; i < s_row_n; i++) {
			const InpRow *r = &s_rows[i];
			int fire = 0;
			if (r->buttons && (now & r->buttons)) {
				fire = 1;
			}
			if (r->stick == 1) {
				if ((r->buttons & BTN_DLEFT) && s_sx < -s_dead) {
					fire = 1;
				}
				if ((r->buttons & BTN_DRIGHT) && s_sx > s_dead) {
					fire = 1;
				}
			}
			if (r->stick == 2) {
				if ((r->buttons & BTN_DUP) && s_sy > s_dead) {
					fire = 1;
				}
				if ((r->buttons & BTN_DDOWN) && s_sy < -s_dead) {
					fire = 1;
				}
			}
			if (fire && r->id < 32) {
				s_held |= 1u << r->id;
			}
		}
		s_down = s_held;
		{
			const unsigned heldm = joy_mask(joypad_get_buttons_held(JOYPAD_PORT_1));
			for (int i = 0; i < s_row_n; i++) {
				const InpRow *r = &s_rows[i];
				if (r->buttons && (heldm & r->buttons) && r->id < 32) {
					s_down |= 1u << r->id;
				}
			}
		}
		return;
	}
#endif
	if (b.a) {
		s_held |= 1u << ACT_ACCEPT;
	}
	if (b.b) {
		s_held |= 1u << ACT_CANCEL;
	}
	if (b.start) {
		s_held |= 1u << ACT_PAUSE;
	}
	if (b.d_left) {
		s_held |= 1u << ACT_LEFT;
	}
	if (b.d_right) {
		s_held |= 1u << ACT_RIGHT;
	}
	if (b.d_up) {
		s_held |= 1u << ACT_UP;
	}
	if (b.d_down) {
		s_held |= 1u << ACT_DOWN;
	}
	if (b.z || b.l || b.r || b.c_left || b.c_right || b.c_up || b.c_down) {
		/* Full N64 pad: Z/L/R + C-buttons stay readable in the fallback path. */
	}
	s_down = s_held;
	{
		joypad_buttons_t held = joypad_get_buttons_held(JOYPAD_PORT_1);
		if (held.z) {
			s_down |= 1u << 11;
		}
		if (held.l) {
			s_down |= 1u << 12;
		}
		if (held.r) {
			s_down |= 1u << 13;
		}
		if (held.c_left) {
			s_down |= 1u << 7;
		}
		if (held.c_right) {
			s_down |= 1u << 8;
		}
		if (held.c_up) {
			s_down |= 1u << 9;
		}
		if (held.c_down) {
			s_down |= 1u << 10;
		}
	}
}

int pad_io_held(int action)
{
	if (action < 0 || action > 31) {
		return 0;
	}
	return (s_down & (1u << action)) ? 1 : 0;
}

int pad_io_pressed(int action)
{
	if (action < 0 || action > 31) {
		return 0;
	}
#ifndef BLAZIUM_N64_HAS_INP
	if (action > 6) {
		return 0;
	}
#endif
	return (s_held & (1u << action)) ? 1 : 0;
}

void pad_io_stick(float *x, float *y)
{
	if (x) {
		*x = s_sx;
	}
	if (y) {
		*y = s_sy;
	}
}

void pad_io_set_deadzone(float dz)
{
	if (dz < 0.0f) {
		dz = 0.0f;
	}
	if (dz > 0.9f) {
		dz = 0.9f;
	}
	s_dead = dz;
}

void pad_io_set_rumble(int on)
{
#ifdef BLAZIUM_N64_RUMBLE
	if (!joypad_get_rumble_supported(JOYPAD_PORT_1)) {
		return;
	}
	joypad_set_rumble_active(JOYPAD_PORT_1, on != 0);
#else
	(void)on;
#endif
}

void pad_io_stop_rumble(void)
{
#ifdef BLAZIUM_N64_RUMBLE
	pad_io_set_rumble(0);
#endif
}
