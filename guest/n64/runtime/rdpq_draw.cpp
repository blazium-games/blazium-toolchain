// MIT. rdpq first (v1). Gold fill cube when MESH is embedded; TRIFMT_TEX when NTEX is valid.
// Two-color fill rects when MESH is absent or rejected. No Tiny3D / GL / mksprite.

#include "rdpq_draw.h"

#include <libdragon.h>
#include <math.h>
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
static float g_look_yaw;
static float g_look_pitch;
static float g_orbit_az;
static float g_orbit_el;
static float g_orbit_rad;
static float g_shake;
static float g_eye_x;
static float g_eye_y = 2.0f;
static float g_eye_z = 6.0f;
static float g_cam_yaw;
static float g_cam_pitch = 0.321750554f;
static float g_anim_x;
static float g_anim_y;
static float g_anim_z;
static float g_anim_yaw;
static int g_ly_n;
static int g_ortho;
static int g_talk;
static char g_talk_txt[48];
static int g_spr_n;
static int16_t g_spr_x[32];
static int16_t g_spr_y[32];
static int16_t g_spr_w[32];
static int16_t g_spr_h[32];
static uint8_t g_spr_flip[32];
static uint8_t g_spr_frame[32];
static int g_tile_n;
static int16_t g_tile_x[256];
static int16_t g_tile_y[256];
static int g_part_n;
static float g_part_x[32];
static float g_part_y[32];
static float g_part_w[32];
static float g_part_h[32];
static const unsigned char *s_draw_mesh;
static unsigned s_draw_mesh_sz;
static const unsigned char *s_draw_ntex;
static unsigned s_draw_ntex_sz;
static const unsigned char *s_ly_mesh[16];
static unsigned s_ly_mesh_sz[16];
static int s_ly_mesh_n;

#if defined(BLAZIUM_N64_HAS_MESH) || defined(BLAZIUM_N64_HAS_NTEX)
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
#endif

#ifdef BLAZIUM_N64_HAS_NTEX
static uint16_t s_texels[2048];
static int s_ntex_w;
static int s_ntex_h;
static int s_ntex_ready;

static int ntex_header_ok(const unsigned char *t, unsigned sz, int *w, int *h)
{
	if (!t || sz < 12) {
		return 0;
	}
	if (t[0] == 0x4D && t[1] == 0x5A) {
		return 0;
	}
	if (t[0] == 'T' && t[1] == 'I' && t[2] == 'M' && t[3] == ' ') {
		return 0;
	}
	if (t[0] == 'G' && t[1] == 'T' && t[2] == 'E' && t[3] == 'X') {
		return 0;
	}
	if (t[0] != 'N' || t[1] != 'T' || t[2] != 'E' || t[3] != 'X') {
		return 0;
	}
	if (ru16le(t + 4) != (uint16_t)BLAZIUM_N64_COOK_ABI) {
		return 0;
	}
	const int tw = (int)ru16le(t + 6);
	const int th = (int)ru16le(t + 8);
	const int fmt = (int)ru16le(t + 10);
	if (tw < 1 || th < 1 || fmt != 1) {
		return 0;
	}
	if (tw * th * 2 > 4096) {
		return 0;
	}
	if (sz < 12u + (unsigned)tw * (unsigned)th * 2u) {
		return 0;
	}
	*w = tw;
	*h = th;
	return 1;
}

static int upload_cooked_ntex(int *w, int *h)
{
	if (s_ntex_ready < 0) {
		return 0;
	}
	if (s_ntex_ready == 0) {
		int tw = 0;
		int th = 0;
		const unsigned char *t = s_draw_ntex;
		unsigned sz = s_draw_ntex_sz;
		if (!t) {
			t = cooked_ntex;
			sz = (unsigned)(cooked_ntex_end - cooked_ntex);
		}
		if (!ntex_header_ok(t, sz, &tw, &th) || tw * th > 2048) {
			s_ntex_ready = -1;
			return 0;
		}
		const unsigned char *px = t + 12;
		const int n = tw * th;
		for (int i = 0; i < n; i++) {
			s_texels[i] = ru16le(px + (unsigned)i * 2u);
		}
		s_ntex_w = tw;
		s_ntex_h = th;
		s_ntex_ready = 1;
	}
	surface_t surf = surface_make_linear(s_texels, FMT_RGBA16, (uint16_t)s_ntex_w, (uint16_t)s_ntex_h);
	rdpq_tex_upload(TILE0, &surf, NULL);
	*w = s_ntex_w;
	*h = s_ntex_h;
	return 1;
}
#endif

#ifdef BLAZIUM_N64_HAS_MESH
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

static int project_vert(float wx, float wy, float wz, float *sx, float *sy, float *out_cz)
{
	const float dw = (float)display_get_width();
	const float dh = (float)display_get_height();
	const float hw = dw * 0.5f;
	const float hh = dh * 0.5f;
	if (g_ortho) {
		*sx = wx - g_eye_x + hw;
		*sy = wy - g_eye_y + hh;
		*out_cz = 1.0f;
		return 1;
	}
	const float cyaw = cosf(g_look_yaw);
	const float syaw = sinf(g_look_yaw);
	const float rwx = wx * cyaw + wz * syaw;
	const float rwz = -wx * syaw + wz * cyaw;
	wx = rwx + g_anim_x;
	wy = wy + g_anim_y;
	wz = rwz + g_anim_z;
	float eye_x = g_eye_x;
	float eye_y = g_eye_y;
	float eye_z = g_eye_z;
	if (g_orbit_rad > 0.01f) {
		const float cp = cosf(g_orbit_el);
		eye_x += sinf(g_orbit_az) * cp * g_orbit_rad;
		eye_y += sinf(g_orbit_el) * g_orbit_rad;
		eye_z += cosf(g_orbit_az) * cp * g_orbit_rad;
	}
	if (g_shake > 0.0f) {
		eye_x += g_shake * 0.05f;
		eye_y += g_shake * 0.03f;
	}
	const float yaw = g_cam_yaw + g_anim_yaw;
	const float pitch = g_cam_pitch + g_look_pitch;
	const float cp = cosf(pitch);
	const float sp = sinf(pitch);
	const float c_yaw = cosf(yaw);
	const float s_yaw = sinf(yaw);
	const float fx = s_yaw * cp;
	const float fy = -sp;
	const float fz = -c_yaw * cp;
	const float rx = c_yaw;
	const float ry = 0.0f;
	const float rz = s_yaw;
	const float ux = fy * rz - fz * ry;
	const float uy = fz * rx - fx * rz;
	const float uz = fx * ry - fy * rx;
	const float dx = wx - eye_x;
	const float dy = wy - eye_y;
	const float dz = wz - eye_z;
	const float cx = dx * rx + dy * ry + dz * rz;
	const float vy = dx * ux + dy * uy + dz * uz;
	const float cz = dx * fx + dy * fy + dz * fz;
	if (cz < 0.15f) {
		return 0;
	}
	const float f = 1.92098213f;
	const float aspect = dw / dh;
	*sx = hw + (cx * f / aspect / cz) * hw;
	*sy = hh - (vy * f / cz) * hh;
	*out_cz = cz;
	return 1;
}

static void box_uv(float wx, float wy, float wz, float nx, float ny, float nz, float tw, float th, float *s, float *t)
{
	const float ax = nx < 0.0f ? -nx : nx;
	const float ay = ny < 0.0f ? -ny : ny;
	const float az = nz < 0.0f ? -nz : nz;
	float u;
	float v;
	if (ax >= ay && ax >= az) {
		u = wy + 0.5f;
		v = wz + 0.5f;
	} else if (ay >= ax && ay >= az) {
		u = wx + 0.5f;
		v = wz + 0.5f;
	} else {
		u = wx + 0.5f;
		v = wy + 0.5f;
	}
	*s = u * tw;
	*t = v * th;
}

static int draw_mesh_bytes(const unsigned char *m, unsigned sz)
{
	uint32_t tris = 0;
	if (!mesh_header_ok(m, sz, &tris)) {
		return 0;
	}
	int tw = 0;
	int th = 0;
	int tex = 0;
#ifdef BLAZIUM_N64_HAS_NTEX
	tex = upload_cooked_ntex(&tw, &th);
#endif
	rdpq_set_mode_standard();
	if (tex) {
		rdpq_mode_combiner(RDPQ_COMBINER_TEX);
	} else {
		rdpq_mode_combiner(RDPQ_COMBINER_FLAT);
		rdpq_set_prim_color(RGBA32(200, 140, 40, 255));
	}
	const unsigned char *vp = m + 12;
	for (uint32_t i = 0; i < tris; i++) {
		float wx[3];
		float wy[3];
		float wz[3];
		float p[3][2];
		float cz[3];
		int ok = 1;
		for (int k = 0; k < 3; k++) {
			wx[k] = rf32le(vp + (unsigned)k * 12u);
			wy[k] = rf32le(vp + (unsigned)k * 12u + 4u);
			wz[k] = rf32le(vp + (unsigned)k * 12u + 8u);
			if (!project_vert(wx[k], wy[k], wz[k], &p[k][0], &p[k][1], &cz[k])) {
				ok = 0;
				break;
			}
		}
		vp += 36;
		if (!ok) {
			continue;
		}
		const float cross = (p[1][0] - p[0][0]) * (p[2][1] - p[0][1]) - (p[2][0] - p[0][0]) * (p[1][1] - p[0][1]);
		if (cross >= 0.0f) {
			continue;
		}
		if (tex) {
			const float e1x = wx[1] - wx[0];
			const float e1y = wy[1] - wy[0];
			const float e1z = wz[1] - wz[0];
			const float e2x = wx[2] - wx[0];
			const float e2y = wy[2] - wy[0];
			const float e2z = wz[2] - wz[0];
			const float nx = e1y * e2z - e1z * e2y;
			const float ny = e1z * e2x - e1x * e2z;
			const float nz = e1x * e2y - e1y * e2x;
			float tv[3][5];
			for (int k = 0; k < 3; k++) {
				tv[k][0] = p[k][0];
				tv[k][1] = p[k][1];
				box_uv(wx[k], wy[k], wz[k], nx, ny, nz, (float)tw, (float)th, &tv[k][2], &tv[k][3]);
				tv[k][4] = 1.0f / cz[k];
			}
			rdpq_triangle(&TRIFMT_TEX, tv[0], tv[1], tv[2]);
		} else {
			rdpq_triangle(&TRIFMT_FILL, p[0], p[1], p[2]);
		}
	}
	return 1;
}

static int draw_cooked_mesh(void)
{
	const unsigned char *m = s_draw_mesh ? s_draw_mesh : cooked_mesh;
	const unsigned sz = s_draw_mesh ? s_draw_mesh_sz : (unsigned)(cooked_mesh_end - cooked_mesh);
	if (!draw_mesh_bytes(m, sz)) {
		return 0;
	}
	for (int i = 0; i < s_ly_mesh_n; i++) {
		(void)draw_mesh_bytes(s_ly_mesh[i], s_ly_mesh_sz[i]);
	}
	return 1;
}
#endif

void rdpq_draw_init(void)
{
	rdpq_init();
	rdpq_text_register_font(1, rdpq_font_load_builtin(FONT_BUILTIN_DEBUG_VAR));
	rdpq_draw_rebind_embedded();
}

void rdpq_draw_begin(void)
{
}

void rdpq_draw_end(void)
{
}

void rdpq_draw_set_camera(float x, float y, float z, float yaw, float pitch, float fov)
{
	g_eye_x = x;
	g_eye_y = y;
	g_eye_z = z;
	g_cam_yaw = yaw;
	g_cam_pitch = pitch;
	if (fov > 1.0f) {
		g_fov = fov;
	}
}

void rdpq_draw_set_anim_ofs(float x, float y, float z, float yaw)
{
	g_anim_x = x;
	g_anim_y = y;
	g_anim_z = z;
	g_anim_yaw = yaw;
}

void rdpq_draw_sprite(int i, int16_t x, int16_t y, int16_t w, int16_t h)
{
	rdpq_draw_sprite_ex(i, x, y, w, h, 0, 0);
}

void rdpq_draw_sprite_ex(int i, int16_t x, int16_t y, int16_t w, int16_t h, uint8_t flip, uint8_t frame)
{
	if (i < 0 || i >= 32) {
		return;
	}
	g_spr_x[i] = x;
	g_spr_y[i] = y;
	g_spr_w[i] = w;
	g_spr_h[i] = h;
	g_spr_flip[i] = flip;
	g_spr_frame[i] = frame;
	if (i + 1 > g_spr_n) {
		g_spr_n = i + 1;
	}
}

void rdpq_draw_tile_cell(int16_t tx, int16_t ty)
{
	if (g_tile_n >= 256) {
		return;
	}
	g_tile_x[g_tile_n] = tx;
	g_tile_y[g_tile_n] = ty;
	g_tile_n++;
}

void rdpq_draw_clear_tiles(void)
{
	g_tile_n = 0;
}

void rdpq_draw_part_quad(int i, float x, float y, float w, float h)
{
	if (i < 0 || i >= 32) {
		return;
	}
	g_part_x[i] = x;
	g_part_y[i] = y;
	g_part_w[i] = w;
	g_part_h[i] = h;
	if (i + 1 > g_part_n) {
		g_part_n = i + 1;
	}
}

void rdpq_draw_clear_parts(void)
{
	g_part_n = 0;
}

void rdpq_draw_talk(const char *txt)
{
	g_talk = txt && txt[0] ? 1 : 0;
	g_talk_txt[0] = 0;
	if (txt) {
		unsigned n = 0;
		while (txt[n] && n < sizeof(g_talk_txt) - 1) {
			g_talk_txt[n] = txt[n];
			n++;
		}
		g_talk_txt[n] = 0;
	}
}

void rdpq_draw_clear_sprites(void)
{
	g_spr_n = 0;
}

void rdpq_draw_set_ortho(int on)
{
	g_ortho = on;
}

void rdpq_draw_look(float yaw, float pitch)
{
	g_look_yaw += yaw;
	g_look_pitch += pitch;
}

void rdpq_draw_orbit_sph(float az, float el, float rad)
{
	g_orbit_az = az;
	g_orbit_el = el;
	g_orbit_rad = rad;
}

void rdpq_draw_shake(float amp)
{
	g_shake = amp;
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
	rdpq_fill_rectangle(0, 0, display_get_width(), display_get_height());
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

/* Rebind current MESH/NTEX pointers (rom://MESH%02d.bin after pack swap). */
void rdpq_draw_rebind(const unsigned char *mesh, unsigned mesh_sz, const unsigned char *ntex, unsigned ntex_sz)
{
	s_draw_mesh = mesh;
	s_draw_mesh_sz = mesh_sz;
	s_draw_ntex = ntex;
	s_draw_ntex_sz = ntex_sz;
#ifdef BLAZIUM_N64_HAS_NTEX
	s_ntex_ready = 0;
#endif
}

void rdpq_draw_rebind_embedded(void)
{
#ifdef BLAZIUM_N64_HAS_MESH
	s_draw_mesh = cooked_mesh;
	s_draw_mesh_sz = (unsigned)(cooked_mesh_end - cooked_mesh);
#else
	s_draw_mesh = NULL;
	s_draw_mesh_sz = 0;
#endif
#ifdef BLAZIUM_N64_HAS_NTEX
	s_draw_ntex = cooked_ntex;
	s_draw_ntex_sz = (unsigned)(cooked_ntex_end - cooked_ntex);
	s_ntex_ready = 0;
#else
	s_draw_ntex = NULL;
	s_draw_ntex_sz = 0;
#endif
	rdpq_draw_clear_layers();
}

int rdpq_draw_layer_add_mesh(const unsigned char *mesh, unsigned mesh_sz)
{
	if (!mesh || mesh_sz < 12 || s_ly_mesh_n >= 16) {
		return 0;
	}
	s_ly_mesh[s_ly_mesh_n] = mesh;
	s_ly_mesh_sz[s_ly_mesh_n] = mesh_sz;
	s_ly_mesh_n++;
	g_ly_n++;
	return 1;
}

void rdpq_draw_clear_layers(void)
{
	s_ly_mesh_n = 0;
	g_ly_n = 0;
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
	(void)g_fov;
	(void)g_ly_n;
	if (g_shake > 0.0f) {
		g_shake *= 0.85f;
		if (g_shake < 0.01f) {
			g_shake = 0.0f;
		}
	}

	rdpq_text_printf(NULL, 1, 16, 16, "Blazium N64 ABI %d", BLAZIUM_N64_COOK_ABI);
	rdpq_set_mode_fill(RGBA32(80, 90, 70, 255));
	for (int i = 0; i < g_tile_n; i++) {
		const int x0 = (int)g_tile_x[i] * 16;
		const int y0 = (int)g_tile_y[i] * 16;
		rdpq_fill_rectangle(x0, y0, x0 + 16, y0 + 16);
	}
#ifdef BLAZIUM_N64_HAS_NTEX
	{
		int tw = 0;
		int th = 0;
		const int tex = upload_cooked_ntex(&tw, &th);
		if (tex) {
			rdpq_set_mode_standard();
			rdpq_mode_combiner(RDPQ_COMBINER_TEX);
			for (int i = 0; i < g_spr_n; i++) {
				const float x0 = (float)g_spr_x[i];
				const float y0 = (float)g_spr_y[i];
				const float x1 = x0 + (float)g_spr_w[i];
				const float y1 = y0 + (float)g_spr_h[i];
				float u0 = (float)g_spr_frame[i] * 8.0f;
				float u1 = u0 + (float)tw;
				if (g_spr_flip[i] & 1) {
					const float t = u0;
					u0 = u1;
					u1 = t;
				}
				float tv0[5] = { x0, y0, u0, 0.0f, 1.0f };
				float tv1[5] = { x1, y0, u1, 0.0f, 1.0f };
				float tv2[5] = { x1, y1, u1, (float)th, 1.0f };
				float tv3[5] = { x0, y1, u0, (float)th, 1.0f };
				rdpq_triangle(&TRIFMT_TEX, tv0, tv1, tv2);
				rdpq_triangle(&TRIFMT_TEX, tv0, tv2, tv3);
			}
		} else {
			rdpq_set_mode_fill(RGBA32(220, 220, 240, 255));
			for (int i = 0; i < g_spr_n; i++) {
				const int x0 = (int)g_spr_x[i];
				const int y0 = (int)g_spr_y[i];
				rdpq_fill_rectangle(x0, y0, x0 + (int)g_spr_w[i], y0 + (int)g_spr_h[i]);
			}
		}
	}
#else
	rdpq_set_mode_fill(RGBA32(220, 220, 240, 255));
	for (int i = 0; i < g_spr_n; i++) {
		const int x0 = (int)g_spr_x[i];
		const int y0 = (int)g_spr_y[i];
		rdpq_fill_rectangle(x0, y0, x0 + (int)g_spr_w[i], y0 + (int)g_spr_h[i]);
	}
#endif
	rdpq_set_mode_fill(RGBA32(240, 200, 80, 255));
	for (int i = 0; i < g_part_n; i++) {
		const int x0 = (int)g_part_x[i];
		const int y0 = (int)g_part_y[i];
		rdpq_fill_rectangle(x0, y0, x0 + (int)g_part_w[i], y0 + (int)g_part_h[i]);
	}
	if (g_talk) {
		rdpq_text_printf(NULL, 1, 24, 200, "%s", g_talk_txt);
	}
	rdpq_draw_overlay();
	rdpq_detach_show();
}
