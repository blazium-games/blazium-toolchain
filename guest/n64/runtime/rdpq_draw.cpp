// MIT. rdpq first (v1). Gold two-color fallback when MESH/NTEX are absent.
// P8 mesh path is not linked here.

#include "rdpq_draw.h"

#include <libdragon.h>

#ifndef BLAZIUM_N64_COOK_ABI
#define BLAZIUM_N64_COOK_ABI 1
#endif

#if defined(__has_include)
#if __has_include("cook_flags.h")
#include "cook_flags.h"
#endif
#endif

#ifdef BLAZIUM_N64_HAS_MESH
extern "C" {
extern const unsigned char cooked_mesh[];
extern const unsigned char cooked_mesh_end[];
}
#endif
#ifdef BLAZIUM_N64_HAS_NTEX
extern "C" {
extern const unsigned char cooked_ntex[];
extern const unsigned char cooked_ntex_end[];
}
#endif

static float g_fov = 55.0f;
static float g_fade;
static int g_ly_n;
static int g_ortho;

void rdpq_draw_init(void)
{
	rdpq_init();
	rdpq_text_register_font(1, rdpq_font_load_builtin(FONT_BUILTIN_DEBUG_VAR));
}

void rdpq_draw_begin(void)
{
}

void rdpq_draw_end(void)
{
}

void rdpq_draw_set_camera(float x, float y, float z, float yaw, float pitch, float fov)
{
	(void)x;
	(void)y;
	(void)z;
	(void)yaw;
	(void)pitch;
	if (fov > 1.0f) {
		g_fov = fov;
	}
}

void rdpq_draw_set_ortho(int on)
{
	g_ortho = on;
}

void rdpq_draw_look(float yaw, float pitch)
{
	(void)yaw;
	(void)pitch;
}

void rdpq_draw_orbit_sph(float az, float el, float rad)
{
	(void)az;
	(void)el;
	(void)rad;
}

void rdpq_draw_shake(float amp)
{
	(void)amp;
}

void rdpq_draw_set_fade(float a)
{
	g_fade = a;
}

void rdpq_draw_overlay(void)
{
	if (g_fade <= 0.0f) {
		return;
	}
	int a = (int)(g_fade * 180.0f);
	if (a > 255) {
		a = 255;
	}
	rdpq_set_mode_standard();
	rdpq_mode_blender(RDPQ_BLENDER_MULTIPLY);
	rdpq_set_prim_color(RGBA32(0, 0, 0, (uint8_t)a));
	rdpq_fill_rectangle(0, 0, 320, 240);
}

int rdpq_draw_primary_mesh_node(void)
{
	return 0;
}

void rdpq_draw_layer_add(int id)
{
	(void)id;
	g_ly_n++;
}

void rdpq_draw_frame(void)
{
	surface_t *disp = display_get();
	rdpq_attach(disp, NULL);
	rdpq_clear(RGBA32(8, 12, 28, 255));

	rdpq_set_mode_fill(RGBA32(40, 160, 90, 255));
	rdpq_fill_rectangle(96, 64, 160, 176);
	rdpq_set_mode_fill(RGBA32(200, 140, 40, 255));
	rdpq_fill_rectangle(160, 64, 224, 176);

#ifdef BLAZIUM_N64_HAS_MESH
	(void)cooked_mesh;
	(void)cooked_mesh_end;
#endif
#ifdef BLAZIUM_N64_HAS_NTEX
	(void)cooked_ntex;
	(void)cooked_ntex_end;
#endif
	(void)g_fov;
	(void)g_ortho;
	(void)g_ly_n;

	rdpq_text_printf(NULL, 1, 16, 16, "Blazium N64 ABI %d", BLAZIUM_N64_COOK_ABI);
	rdpq_draw_overlay();
	rdpq_detach_show();
}
