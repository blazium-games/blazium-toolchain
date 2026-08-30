// MIT. libdragon joypad. Rumble is P8-only (no-op in v1).

#include "pad_io.h"

#include <libdragon.h>

static float s_dead = 0.2f;
static float s_sx;
static float s_sy;
static int s_held;

enum {
	ACT_ACCEPT = 0,
	ACT_CANCEL = 1,
	ACT_PAUSE = 2,
	ACT_LEFT = 3,
	ACT_RIGHT = 4,
	ACT_UP = 5,
	ACT_DOWN = 6
};

void pad_io_init(void)
{
	joypad_init();
}

void pad_io_poll(void)
{
	joypad_poll();
	joypad_buttons_t b = joypad_get_buttons_pressed(JOYPAD_PORT_1);
	joypad_inputs_t in = joypad_get_inputs(JOYPAD_PORT_1);
	s_held = 0;
	if (b.a) {
		s_held |= 1 << ACT_ACCEPT;
	}
	if (b.b) {
		s_held |= 1 << ACT_CANCEL;
	}
	if (b.start) {
		s_held |= 1 << ACT_PAUSE;
	}
	if (b.d_left) {
		s_held |= 1 << ACT_LEFT;
	}
	if (b.d_right) {
		s_held |= 1 << ACT_RIGHT;
	}
	if (b.d_up) {
		s_held |= 1 << ACT_UP;
	}
	if (b.d_down) {
		s_held |= 1 << ACT_DOWN;
	}
	s_sx = (float)in.stick_x / 80.0f;
	s_sy = (float)in.stick_y / 80.0f;
	if (s_sx > -s_dead && s_sx < s_dead) {
		s_sx = 0.0f;
	}
	if (s_sy > -s_dead && s_sy < s_dead) {
		s_sy = 0.0f;
	}
}

int pad_io_pressed(int action)
{
	if (action < 0 || action > 6) {
		return 0;
	}
	return (s_held & (1 << action)) ? 1 : 0;
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
	(void)on;
}

void pad_io_stop_rumble(void)
{
}
