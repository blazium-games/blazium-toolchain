// MIT. rdpq first (v1). Gold rdpq_triangle cube when MESH ABI 1 is embedded.
// Two-color fill rects when MESH is absent or rejected. No Tiny3D / GL.

#include "rdpq_draw.h"

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

#ifdef BLAZIUM_N64_HAS_MESH
static uint16_t ru16le(const unsigned char *p)
{
	return (uint16_t)p[0] | ((uint16_t)p[1] << 8);
}

static uint32_t ru32le(const unsigned char *p)
{
	return (uint32_t)p[0] | ((uint32_t)p[1] << 8) | ((uint32_t)p[2] << 16) | ((uint32_t)p[3] << 24);
}

static float rf32le(const unsigned char *p)
{
	uint32_t bits = ru32le(p);
	float v;
	memcpy(&v, &bits, sizeof(v));
	return v;
}

static int mesh_header_ok(const unsigned char *m, unsigned sz, uint32_t *tri_count)
{
	if (!m || sz < 12) {
		return 0;
	}
	if (m[0] == 0x4D && m[1] == 0x5A) {
		return 0;
	}
	if (m[0] == 'T' && m[1] == 'I' && m[2] == 'M' && m[3] == ' ') {
		return 0;
	}
	if (m[0] == 'G' && m[1] == 'T' && m[2] == 'E' && m[3] == 'X') {
		return 0;
	}
	if (m[0] != 'M' || m[1] != 'E' || m[2] != 'S' || m[3] != 'H') {
		return 0;
	}
	if (ru16le(m + 4) != (uint16_t)BLAZIUM_N64_COOK_ABI) {
		return 0;
	}
	const uint32_t n = ru32le(m + 8);
	if (n < 1 || n > 16384) {
		return 0;
	}
	if (sz < 12u + n * 36u) {
		return 0;
	}
	*tri_count = n;
	return 1;
}

static int project_vert(float wx, float wy, float wz, float *sx, float *sy)
{
	const float eye_x = 0.0f;
	const float eye_y = 2.0f;
	const float eye_z = 6.0f;
	const float fx = 0.0f;
	const float fy = -0.316227766f;
	const float fz = -0.948683298f;
	const float rx = 1.0f;
	const float ry = 0.0f;
	const float rz = 0.0f;
	const float ux = 0.0f;
	const float uy = 0.948683298f;
	const float uz = -0.316227766f;
	const float dx = wx - eye_x;
	const float dy = wy - eye_y;
	const float dz = wz - eye_z;
	const float cx = dx * rx + dy * ry + dz * rz;
	const float cy = dx * ux + dy * uy + dz * uz;
	const float cz = dx * fx + dy * fy + dz * fz;
	if (cz < 0.15f) {
		return 0;
	}
	const float f = 1.92098213f;
	const float aspect = 320.0f / 240.0f;
	*sx = 160.0f + (cx * f / aspect / cz) * 160.0f;
	*sy = 120.0f - (cy * f / cz) * 120.0f;
	return 1;
}

static int draw_cooked_mesh(void)
{
	const unsigned char *m = cooked_mesh;
	const unsigned sz = (unsigned)(cooked_mesh_end - cooked_mesh);
	uint32_t tris = 0;
	if (!mesh_header_ok(m, sz, &tris)) {
		return 0;
	}
	rdpq_set_mode_standard();
	rdpq_mode_combiner(RDPQ_COMBINER_FLAT);
	rdpq_set_prim_color(RGBA32(200, 140, 40, 255));
	const unsigned char *v = m + 12;
	for (uint32_t i = 0; i < tris; i++) {
		float p[3][2];
		int ok = 1;
		for (int k = 0; k < 3; k++) {
			const float wx = rf32le(v + (unsigned)k * 12u);
			const float wy = rf32le(v + (unsigned)k * 12u + 4u);
			const float wz = rf32le(v + (unsigned)k * 12u + 8u);
			if (!project_vert(wx, wy, wz, &p[k][0], &p[k][1])) {
				ok = 0;
				break;
			}
		}
		v += 36;
		if (!ok) {
			continue;
		}
		const float cross = (p[1][0] - p[0][0]) * (p[2][1] - p[0][1]) - (p[2][0] - p[0][0]) * (p[1][1] - p[0][1]);
		if (cross >= 0.0f) {
			continue;
		}
		rdpq_triangle(&TRIFMT_FILL, p[0], p[1], p[2]);
	}
	return 1;
}
#endif

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

#ifdef BLAZIUM_N64_HAS_MESH
	if (!draw_cooked_mesh()) {
		rdpq_set_mode_fill(RGBA32(40, 160, 90, 255));
		rdpq_fill_rectangle(96, 64, 160, 176);
		rdpq_set_mode_fill(RGBA32(200, 140, 40, 255));
		rdpq_fill_rectangle(160, 64, 224, 176);
	}
#else
	rdpq_set_mode_fill(RGBA32(40, 160, 90, 255));
	rdpq_fill_rectangle(96, 64, 160, 176);
	rdpq_set_mode_fill(RGBA32(200, 140, 40, 255));
	rdpq_fill_rectangle(160, 64, 224, 176);
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
