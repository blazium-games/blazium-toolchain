// MIT. Consume cooked CAM/HIT/HUD/TILE/NAV/ANIM/FMV. FMV/IPU is a named skip.

#include "sys_io.h"

#include "gs_draw.h"
#include "pack_io.h"
#include "pad_io.h"
#include "sfx_io.h"

#include <math.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#ifndef BLAZIUM_PS2_COOK_ABI
#define BLAZIUM_PS2_COOK_ABI 1
#endif

static int s_ok;
static int s_cam_n;
static int s_cam_i;
static float s_cam_x[8];
static float s_cam_y[8];
static float s_cam_z[8];
static float s_cam_pitch[8];
static float s_cam_yaw[8];
static float s_cam_roll[8];
static float s_cam_fov[8];
static float s_shake_amp;
static float s_shake_left;
static unsigned s_rng;
static int s_tw_on[8];
static float s_tw_from[8];
static float s_tw_to[8];
static float s_tw_t[8];
static float s_tw_dur[8];
static int s_tw_kind[8];
static int s_tw_last;
static int s_tm_on[8];
static float s_tm_left[8];
static int s_tm_last;
static int s_hit_n;
static float s_hit_mn[64][3];
static float s_hit_mx[64][3];
static unsigned char s_hit_kit[64];
static float s_px;
static float s_py;
static float s_pz;
static int s_on_floor;
static int s_on_wall;
static int s_on_ceil;
static int s_player_ok;
static int s_hud_n;
static unsigned char s_hud_kind[16];
static char s_hud_text[16][65];
static int s_hud_x[16];
static int s_hud_y[16];
static int s_hud_w[16];
static int s_hud_h[16];
static int s_hud_r[16];
static int s_hud_g[16];
static int s_hud_b[16];
static int s_hud_focus;
static int s_btn_n;
static int s_btn_slot[16];
static float s_say_left;
static int s_say_done;
static int s_tile_n;
static short s_tile_x[64];
static short s_tile_y[64];
static unsigned char s_tile_flags[64];
static unsigned long long s_over_prev;
static unsigned long long s_over_now;
static int s_over_entered;
static float s_ray_x;
static float s_ray_y;
static float s_ray_z;
static float s_fade_left;
static float s_fade_dur;
static int s_fade_r;
static int s_fade_g;
static int s_fade_b;
static int s_fade_pack;
static int s_fade_phase;
static int s_nav_n;
static int s_anim_n;
static float s_anim_len;
static int s_anim_node;
static int s_anim_keys;
static float s_ak_t[64];
static float s_ak_px[64];
static float s_ak_py[64];
static float s_ak_pz[64];
static float s_ak_rx[64];
static float s_ak_ry[64];
static float s_ak_rz[64];
static float s_anim_t;
static float s_anim_speed;
static int s_fmv_n;
static int s_hp = 3;
static int s_frame;
static int s_nav_i;
static float s_nav_x;
static float s_nav_z;
static char s_fmv_err[80];

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

static FILE *open_sys(const char *host, const char *iso_bs, const char *iso)
{
	FILE *f = fopen(host, "rb");
	if (f) {
		return f;
	}
	f = fopen(iso_bs, "rb");
	if (f) {
		return f;
	}
	return fopen(iso, "rb");
}

static unsigned char *read_sys(const char *host, const char *iso_bs, const char *iso, unsigned *sz)
{
	FILE *f = open_sys(host, iso_bs, iso);
	if (!f) {
		return 0;
	}
	if (fseek(f, 0, SEEK_END) != 0) {
		fclose(f);
		return 0;
	}
	long n = ftell(f);
	if (n < 8) {
		fclose(f);
		return 0;
	}
	rewind(f);
	unsigned char *buf = (unsigned char *)malloc((size_t)n);
	if (!buf) {
		fclose(f);
		return 0;
	}
	if (fread(buf, 1, (size_t)n, f) != (size_t)n) {
		free(buf);
		fclose(f);
		return 0;
	}
	fclose(f);
	*sz = (unsigned)n;
	return buf;
}

static int mag4(const unsigned char *b, char a, char c, char d, char e)
{
	return b[0] == (unsigned char)a && b[1] == (unsigned char)c && b[2] == (unsigned char)d && b[3] == (unsigned char)e;
}

static void apply_cam(const unsigned char *b, unsigned sz)
{
	if (!b || sz < 8 || !mag4(b, 'C', 'A', 'M', ' ') || ru16(b + 4) != BLAZIUM_PS2_COOK_ABI) {
		return;
	}
	s_cam_n = (int)ru16(b + 6);
	if (s_cam_n > 8) {
		s_cam_n = 8;
	}
	if (s_cam_n < 1 || sz < 8 + 32) {
		return;
	}
	for (int i = 0; i < s_cam_n; i++) {
		if (8 + (unsigned)(i + 1) * 32 > sz) {
			s_cam_n = i;
			break;
		}
		const unsigned char *r = b + 8 + (unsigned)i * 32;
		s_cam_x[i] = rf32(r + 4);
		s_cam_y[i] = rf32(r + 8);
		s_cam_z[i] = rf32(r + 12);
		s_cam_pitch[i] = rf32(r + 16);
		s_cam_yaw[i] = rf32(r + 20);
		s_cam_roll[i] = rf32(r + 24);
		s_cam_fov[i] = rf32(r + 28);
	}
	s_cam_i = 0;
	gs_draw_set_camera(s_cam_x[0], s_cam_y[0], s_cam_z[0], s_cam_pitch[0], s_cam_yaw[0], s_cam_roll[0], s_cam_fov[0]);
}

static void apply_hit(const unsigned char *b, unsigned sz)
{
	if (!b || sz < 8 || !mag4(b, 'H', 'I', 'T', ' ') || ru16(b + 4) != BLAZIUM_PS2_COOK_ABI) {
		return;
	}
	s_hit_n = (int)ru16(b + 6);
	if (s_hit_n > 64) {
		s_hit_n = 64;
	}
	const unsigned rec = 28;
	for (int i = 0; i < s_hit_n; i++) {
		if (8 + (unsigned)(i + 1) * rec > sz) {
			s_hit_n = i;
			break;
		}
		const unsigned char *r = b + 8 + (unsigned)i * rec;
		s_hit_kit[i] = r[2];
		s_hit_mn[i][0] = rf32(r + 4);
		s_hit_mn[i][1] = rf32(r + 8);
		s_hit_mn[i][2] = rf32(r + 12);
		s_hit_mx[i][0] = rf32(r + 16);
		s_hit_mx[i][1] = rf32(r + 20);
		s_hit_mx[i][2] = rf32(r + 24);
	}
}

static void redraw_hud(void)
{
	s_btn_n = 0;
	for (int i = 0; i < s_hud_n && i < 16; i++) {
		int r = s_hud_r[i];
		int g = s_hud_g[i];
		int b = s_hud_b[i];
		if (s_hud_kind[i] == 3) {
			s_btn_slot[s_btn_n++] = i;
			if (s_hud_focus == i) {
				r = 255;
				g = 220;
				b = 40;
			}
		}
		gs_draw_hud_quad(i, s_hud_x[i], s_hud_y[i], s_hud_w[i], s_hud_h[i], r, g, b);
	}
}

static void apply_hud(const unsigned char *b, unsigned sz)
{
	if (!b || sz < 8 || !mag4(b, 'H', 'U', 'D', ' ') || ru16(b + 4) != BLAZIUM_PS2_COOK_ABI) {
		return;
	}
	s_hud_n = (int)ru16(b + 6);
	const unsigned rec = 76;
	if (8 + (unsigned)s_hud_n * rec > sz) {
		s_hud_n = 0;
		return;
	}
	if (s_hud_n > 16) {
		s_hud_n = 16;
	}
	s_btn_n = 0;
	s_hud_focus = -1;
	for (int i = 0; i < s_hud_n; i++) {
		const unsigned char *r = b + 8 + (unsigned)i * rec;
		s_hud_x[i] = (int)(short)ru16(r);
		s_hud_y[i] = (int)(short)ru16(r + 2);
		s_hud_w[i] = (int)(short)ru16(r + 4);
		s_hud_h[i] = (int)(short)ru16(r + 6);
		s_hud_kind[i] = r[8];
		s_hud_r[i] = r[9];
		s_hud_g[i] = r[10];
		s_hud_b[i] = r[11];
		memcpy(s_hud_text[i], r + 12, 64);
		s_hud_text[i][64] = 0;
		if (s_hud_kind[i] == 3 && s_hud_focus < 0) {
			s_hud_focus = i;
		}
	}
	redraw_hud();
}

static void apply_tile(const unsigned char *b, unsigned sz)
{
	if (!b || sz < 10 || !mag4(b, 'T', 'I', 'L', 'E') || ru16(b + 4) != BLAZIUM_PS2_COOK_ABI) {
		return;
	}
	s_tile_n = (int)ru16(b + 6);
	if (s_tile_n > 64) {
		s_tile_n = 64;
	}
	const unsigned rec = 6;
	for (int i = 0; i < s_tile_n; i++) {
		if (10 + (unsigned)(i + 1) * rec > sz) {
			s_tile_n = i;
			break;
		}
		const unsigned char *r = b + 10 + (unsigned)i * rec;
		s_tile_x[i] = (short)ru16(r);
		s_tile_y[i] = (short)ru16(r + 2);
		s_tile_flags[i] = r[4];
	}
}

static void apply_nav(const unsigned char *b, unsigned sz)
{
	if (!b || sz < 10 || !mag4(b, 'N', 'A', 'V', ' ') || ru16(b + 4) != BLAZIUM_PS2_COOK_ABI) {
		return;
	}
	s_nav_n = (int)ru16(b + 6);
	if (s_nav_n > 0 && sz >= 10 + 16) {
		s_nav_x = rf32(b + 10);
		s_nav_z = rf32(b + 18);
	}
}

static void apply_anim_pose(void)
{
	if (s_anim_keys < 1) {
		return;
	}
	float t = s_anim_t;
	if (t < 0.0f) {
		t = 0.0f;
	}
	if (s_anim_len > 0.0f && t > s_anim_len) {
		t = s_anim_len;
	}
	int a = 0;
	int b = 0;
	for (int i = 0; i < s_anim_keys; i++) {
		if (s_ak_t[i] <= t) {
			a = i;
		}
		if (s_ak_t[i] >= t) {
			b = i;
			break;
		}
		b = i;
	}
	float u = 0.0f;
	const float dt = s_ak_t[b] - s_ak_t[a];
	if (dt > 0.0001f) {
		u = (t - s_ak_t[a]) / dt;
	}
	const float px = s_ak_px[a] + (s_ak_px[b] - s_ak_px[a]) * u;
	const float py = s_ak_py[a] + (s_ak_py[b] - s_ak_py[a]) * u;
	const float pz = s_ak_pz[a] + (s_ak_pz[b] - s_ak_pz[a]) * u;
	gs_draw_set_node_ofs(s_anim_node, px, py, pz);
	(void)s_ak_rx[a];
	(void)s_ak_ry[a];
	(void)s_ak_rz[a];
}

static void apply_anim(const unsigned char *b, unsigned sz)
{
	if (!b || sz < 8 || !mag4(b, 'A', 'N', 'I', 'M') || ru16(b + 4) != BLAZIUM_PS2_COOK_ABI) {
		return;
	}
	s_anim_n = (int)ru16(b + 6);
	s_anim_keys = 0;
	s_anim_node = 0;
	s_anim_len = 1.0f;
	s_anim_t = 0.0f;
	s_anim_speed = 1.0f;
	if (s_anim_n < 1 || sz < 8 + 36) {
		return;
	}
	const unsigned char *clip = b + 8;
	s_anim_len = rf32(clip + 32);
	s_anim_keys = (int)ru16(clip + 36);
	s_anim_node = (int)(short)ru16(clip + 38);
	if (s_anim_keys > 64) {
		s_anim_keys = 64;
	}
	unsigned off = 40;
	for (int k = 0; k < s_anim_keys; k++) {
		if (off + 28 > sz) {
			s_anim_keys = k;
			break;
		}
		s_ak_t[k] = rf32(clip + off);
		s_ak_px[k] = rf32(clip + off + 4);
		s_ak_py[k] = rf32(clip + off + 8);
		s_ak_pz[k] = rf32(clip + off + 12);
		s_ak_rx[k] = rf32(clip + off + 16);
		s_ak_ry[k] = rf32(clip + off + 20);
		s_ak_rz[k] = rf32(clip + off + 24);
		off += 28;
	}
	apply_anim_pose();
}

/* Pack 0 CAM00 HUD00 NAV00 plus CAM%02d HIT%02d HUD%02d TILE%02d NAV%02d ANIM%02d. */
static void load_sys_named(int pack, const char *stem,
		void (*apply)(const unsigned char *, unsigned))
{
	char host[40];
	char iso_bs[48];
	char iso[48];
	sprintf(host, "host:%s%02d.bin", stem, pack);
	sprintf(iso_bs, "cdrom0:\\%s%02d.BIN;1", stem, pack);
	sprintf(iso, "cdrom0:%s%02d.BIN;1", stem, pack);
	unsigned sz = 0;
	unsigned char *buf = read_sys(host, iso_bs, iso, &sz);
	if (buf) {
		apply(buf, sz);
		free(buf);
		s_ok = 1;
	}
}

int sys_io_load_pack(int pack)
{
	if (pack < 0) {
		pack = 0;
	}
	s_cam_n = s_hit_n = s_hud_n = s_tile_n = s_nav_n = s_anim_n = 0;
	s_anim_keys = 0;
	s_btn_n = 0;
	s_hud_focus = -1;
	load_sys_named(pack, "CAM", apply_cam);
	load_sys_named(pack, "HIT", apply_hit);
	load_sys_named(pack, "HUD", apply_hud);
	load_sys_named(pack, "TILE", apply_tile);
	load_sys_named(pack, "NAV", apply_nav);
	load_sys_named(pack, "ANIM", apply_anim);
	return s_ok;
}

int sys_io_init(void)
{
	s_ok = 0;
	s_cam_n = s_cam_i = s_hit_n = s_hud_n = s_tile_n = s_nav_n = s_anim_n = s_fmv_n = 0;
	s_shake_amp = 0.0f;
	s_shake_left = 0.0f;
	s_rng = 1;
	s_tw_last = -1;
	s_tm_last = -1;
	for (int i = 0; i < 8; i++) {
		s_tw_on[i] = 0;
		s_tm_on[i] = 0;
		s_tm_left[i] = 0.0f;
	}
	s_hp = 3;
	s_frame = 0;
	s_nav_i = 0;
	s_hit_n = 0;
	s_tile_n = 0;
	s_on_floor = s_on_wall = s_on_ceil = 0;
	s_player_ok = 0;
	s_px = s_py = s_pz = 0.0f;
	s_over_prev = s_over_now = 0;
	s_over_entered = 0;
	s_ray_x = s_ray_y = s_ray_z = 0.0f;
	s_fade_left = s_fade_dur = 0.0f;
	s_fade_r = s_fade_g = s_fade_b = 0;
	s_fade_pack = -1;
	s_fade_phase = 0;
	s_say_left = 0.0f;
	s_say_done = 1;
	s_anim_speed = 1.0f;
	gs_draw_set_fade(0, 0, 0, 0);
	memcpy(s_fmv_err, "FMV/IPU not shipped on PS2 ABI 1 (no license-clean decoder)", 59);
	s_fmv_err[59] = 0;
	sys_io_load_pack(0);
	gs_draw_look_point(&s_px, &s_py, &s_pz);
	s_player_ok = 1;
	unsigned sz = 0;
	unsigned char *fmv = read_sys("host:FMV00.bin", "cdrom0:\\FMV00.BIN;1", "cdrom0:FMV00.BIN;1", &sz);
	if (fmv) {
		if (mag4(fmv, 'F', 'M', 'V', ' ') && ru16(fmv + 4) == BLAZIUM_PS2_COOK_ABI) {
			s_fmv_n = (int)ru16(fmv + 6);
		}
		free(fmv);
		s_ok = 1;
	}
	return s_ok;
}

int sys_io_loaded(void) { return s_ok; }
int sys_io_cam_count(void) { return s_cam_n; }
int sys_io_hit_count(void) { return s_hit_n; }
int sys_io_hud_count(void) { return s_hud_n; }
int sys_io_tile_count(void) { return s_tile_n; }
int sys_io_nav_count(void) { return s_nav_n; }
int sys_io_anim_count(void) { return s_anim_n; }
int sys_io_fmv_count(void) { return s_fmv_n; }
const char *sys_io_fmv_error(void) { return s_fmv_err; }

void sys_io_say(const char *line)
{
	if (s_hud_n < 1) {
		s_hud_n = 1;
		s_hud_x[0] = 8;
		s_hud_y[0] = 8;
		s_hud_w[0] = 96;
		s_hud_h[0] = 16;
		s_hud_r[0] = 255;
		s_hud_g[0] = 255;
		s_hud_b[0] = 80;
		s_hud_kind[0] = 1;
	}
	if (line && line[0]) {
		unsigned i = 0;
		while (line[i] && i < 64) {
			s_hud_text[0][i] = line[i];
			i++;
		}
		s_hud_text[0][i] = 0;
	} else if (s_hud_text[0][0] == 0) {
		memcpy(s_hud_text[0], "PS2", 4);
	}
	s_say_left = 0.35f;
	s_say_done = 0;
	redraw_hud();
}

int sys_io_say_done(void)
{
	return s_say_done;
}

void sys_io_set_hud_text(int slot, const char *text)
{
	if (slot < 0 || slot >= 16) {
		slot = 0;
	}
	if (slot >= s_hud_n) {
		s_hud_n = slot + 1;
	}
	unsigned i = 0;
	if (text) {
		while (text[i] && i < 64) {
			s_hud_text[slot][i] = text[i];
			i++;
		}
	}
	s_hud_text[slot][i] = 0;
	redraw_hud();
}

void sys_io_nav_follow(float speed, float delta)
{
	if (s_nav_n < 2) {
		return;
	}
	gs_draw_nudge(speed * delta, 0.0f);
	(void)s_nav_x;
	(void)s_nav_z;
}

int sys_io_nav_next(int from, int to)
{
	(void)to;
	int n = from + 1;
	if (n >= s_nav_n) {
		n = 0;
	}
	s_nav_i = n;
	return n;
}

int sys_io_set_hp(int hp)
{
	s_hp = hp;
	return s_hp;
}

int sys_io_get_hp(void)
{
	return s_hp;
}

void sys_io_set_frame(int i)
{
	s_frame = i;
	if (s_anim_keys > 0) {
		int k = i;
		if (k < 0) {
			k = 0;
		}
		if (k >= s_anim_keys) {
			k = s_anim_keys - 1;
		}
		s_anim_t = s_ak_t[k];
		apply_anim_pose();
	}
}

void sys_io_seek_anim(float sec)
{
	s_anim_t = sec;
	apply_anim_pose();
}

void sys_io_set_anim_speed(float s)
{
	s_anim_speed = s;
}

int sys_io_play_fmv(void)
{
	return 0;
}

void sys_io_tick(float delta)
{
	if (s_anim_n > 0) {
		s_frame++;
		s_anim_t += delta * s_anim_speed;
		if (s_anim_len > 0.0f && s_anim_t > s_anim_len) {
			s_anim_t = s_anim_len;
		}
		apply_anim_pose();
	}
	if (s_say_left > 0.0f) {
		s_say_left -= delta;
		if (s_say_left <= 0.0f) {
			s_say_left = 0.0f;
			s_say_done = 1;
		}
	}
	if (s_btn_n > 0) {
		int idx = 0;
		for (int i = 0; i < s_btn_n; i++) {
			if (s_btn_slot[i] == s_hud_focus) {
				idx = i;
				break;
			}
		}
		if (pad_io_just_pressed(0) || pad_io_just_pressed(2)) {
			idx--;
			if (idx < 0) {
				idx = s_btn_n - 1;
			}
			s_hud_focus = s_btn_slot[idx];
			redraw_hud();
		}
		if (pad_io_just_pressed(1) || pad_io_just_pressed(3)) {
			idx++;
			if (idx >= s_btn_n) {
				idx = 0;
			}
			s_hud_focus = s_btn_slot[idx];
			redraw_hud();
		}
		if (pad_io_just_pressed(4)) {
			sfx_io_play();
		}
	}
	if (s_shake_left <= 0.0f) {
		gs_draw_shake(0.0f, 0.0f, 0.0f);
	} else {
		s_shake_left -= delta * 1000.0f;
		if (s_shake_left <= 0.0f) {
			s_shake_left = 0.0f;
			gs_draw_shake(0.0f, 0.0f, 0.0f);
		} else {
			s_rng = s_rng * 1103515245u + 12345u;
			const float nx = ((float)((s_rng >> 16) & 0x7fff) / 16384.0f) - 1.0f;
			s_rng = s_rng * 1103515245u + 12345u;
			const float ny = ((float)((s_rng >> 16) & 0x7fff) / 16384.0f) - 1.0f;
			s_rng = s_rng * 1103515245u + 12345u;
			const float nz = ((float)((s_rng >> 16) & 0x7fff) / 16384.0f) - 1.0f;
			gs_draw_shake(nx * s_shake_amp, ny * s_shake_amp, nz * s_shake_amp);
		}
	}
	for (int i = 0; i < 8; i++) {
		if (s_tw_on[i]) {
			s_tw_t[i] += delta;
			float u = (s_tw_dur[i] <= 0.0f) ? 1.0f : (s_tw_t[i] / s_tw_dur[i]);
			if (u >= 1.0f) {
				u = 1.0f;
				s_tw_on[i] = 0;
			}
			const float v = s_tw_from[i] + (s_tw_to[i] - s_tw_from[i]) * u;
			if (s_tw_kind[i] == 1) {
				sfx_io_music_set_vol(v);
			}
		}
		if (s_tm_on[i]) {
			s_tm_left[i] -= delta;
			if (s_tm_left[i] <= 0.0f) {
				s_tm_left[i] = 0.0f;
				s_tm_on[i] = 0;
			}
		}
	}
	if (s_fade_phase == 1) {
		s_fade_left -= delta;
		if (s_fade_left <= 0.0f || s_fade_dur <= 0.0f) {
			s_fade_left = 0.0f;
			gs_draw_set_fade(128, s_fade_r, s_fade_g, s_fade_b);
			if (s_fade_pack >= 0) {
				pack_io_swap(s_fade_pack);
				s_fade_pack = -1;
			}
			s_fade_phase = 2;
			s_fade_left = s_fade_dur > 0.0f ? s_fade_dur : 0.01f;
		} else {
			const float u = 1.0f - (s_fade_left / s_fade_dur);
			gs_draw_set_fade((int)(u * 128.0f), s_fade_r, s_fade_g, s_fade_b);
		}
	} else if (s_fade_phase == 2 || s_fade_left > 0.0f) {
		s_fade_left -= delta;
		if (s_fade_left <= 0.0f || s_fade_dur <= 0.0f) {
			s_fade_left = 0.0f;
			s_fade_phase = 0;
			gs_draw_set_fade(0, s_fade_r, s_fade_g, s_fade_b);
		} else {
			const float u = s_fade_left / s_fade_dur;
			gs_draw_set_fade((int)(u * 128.0f), s_fade_r, s_fade_g, s_fade_b);
		}
	}
}

void sys_io_look(float yaw, float pitch, float roll)
{
	gs_draw_look(yaw, pitch, roll);
}

void sys_io_look_stick(float delta)
{
	float sx = 0.0f;
	float sy = 0.0f;
	pad_io_stick(1, &sx, &sy);
	if (sx == 0.0f && sy == 0.0f) {
		pad_io_stick(0, &sx, &sy);
		sx *= 0.35f;
		sy *= 0.35f;
	}
	gs_draw_look_delta(sx * 1.6f * delta, -sy * 1.2f * delta);
}

void sys_io_orbit(float yaw, float pitch, float dist)
{
	gs_draw_orbit_sph(yaw, pitch, dist);
}

void sys_io_attach(float ox, float oy, float oz)
{
	gs_draw_attach_offset(ox, oy, oz);
}

void sys_io_shake(float amp, float ms)
{
	if (amp < 0.0f) {
		amp = 0.0f;
	}
	s_shake_amp = amp;
	s_shake_left = ms;
	if (ms <= 0.0f || amp <= 0.0f) {
		s_shake_left = 0.0f;
		gs_draw_shake(0.0f, 0.0f, 0.0f);
	}
}

static void apply_cam_i(int i)
{
	if (i < 0 || i >= s_cam_n) {
		return;
	}
	s_cam_i = i;
	gs_draw_set_camera(s_cam_x[i], s_cam_y[i], s_cam_z[i], s_cam_pitch[i], s_cam_yaw[i], s_cam_roll[i], s_cam_fov[i]);
}

void sys_io_next_cam(void)
{
	if (s_cam_n < 1) {
		return;
	}
	apply_cam_i((s_cam_i + 1) % s_cam_n);
}

void sys_io_prev_cam(void)
{
	if (s_cam_n < 1) {
		return;
	}
	int i = s_cam_i - 1;
	if (i < 0) {
		i = s_cam_n - 1;
	}
	apply_cam_i(i);
}

int sys_io_cam_index(void)
{
	return s_cam_i;
}

int sys_io_tween_start(float from, float to, float sec, int kind)
{
	int slot = -1;
	for (int i = 0; i < 8; i++) {
		if (!s_tw_on[i]) {
			slot = i;
			break;
		}
	}
	if (slot < 0) {
		return -1;
	}
	s_tw_on[slot] = 1;
	s_tw_from[slot] = from;
	s_tw_to[slot] = to;
	s_tw_t[slot] = 0.0f;
	s_tw_dur[slot] = sec < 0.0f ? 0.0f : sec;
	s_tw_kind[slot] = kind;
	s_tw_last = slot;
	if (s_tw_dur[slot] <= 0.0f) {
		s_tw_on[slot] = 0;
		if (kind == 1) {
			sfx_io_music_set_vol(to);
		}
	}
	return slot;
}

void sys_io_kill_tweens(void)
{
	for (int i = 0; i < 8; i++) {
		s_tw_on[i] = 0;
	}
}

int sys_io_tween_count(void)
{
	int n = 0;
	for (int i = 0; i < 8; i++) {
		if (s_tw_on[i]) {
			n++;
		}
	}
	return n;
}

int sys_io_tween_done(int id)
{
	if (id < 0) {
		id = s_tw_last;
	}
	if (id < 0 || id >= 8) {
		return 1;
	}
	return s_tw_on[id] ? 0 : 1;
}

int sys_io_timer_start(float sec)
{
	int slot = -1;
	for (int i = 0; i < 8; i++) {
		if (!s_tm_on[i]) {
			slot = i;
			break;
		}
	}
	if (slot < 0) {
		return -1;
	}
	s_tm_on[slot] = 1;
	s_tm_left[slot] = sec < 0.0f ? 0.0f : sec;
	s_tm_last = slot;
	if (s_tm_left[slot] <= 0.0f) {
		s_tm_on[slot] = 0;
	}
	return slot;
}

int sys_io_timer_done(int id)
{
	if (id < 0) {
		id = s_tm_last;
	}
	if (id < 0 || id >= 8) {
		return 1;
	}
	return s_tm_on[id] ? 0 : 1;
}

static int hit_solid(int i)
{
	const unsigned char k = s_hit_kit[i];
	return k != 1 && k != 4 && k != 6 && k != 7 && k != 10;
}

static int aabb_overlap(float px, float py, float pz, int i)
{
	const float hx = 0.35f;
	const float hy = 0.85f;
	const float hz = 0.35f;
	const float mn0 = px - hx;
	const float mx0 = px + hx;
	const float mn1 = py;
	const float mx1 = py + hy;
	const float mn2 = pz - hz;
	const float mx2 = pz + hz;
	if (mx0 < s_hit_mn[i][0] || mn0 > s_hit_mx[i][0]) {
		return 0;
	}
	if (mx1 < s_hit_mn[i][1] || mn1 > s_hit_mx[i][1]) {
		return 0;
	}
	if (mx2 < s_hit_mn[i][2] || mn2 > s_hit_mx[i][2]) {
		return 0;
	}
	return 1;
}

static int hits_at(float px, float py, float pz)
{
	for (int i = 0; i < s_hit_n; i++) {
		if (hit_solid(i) && aabb_overlap(px, py, pz, i)) {
			return 1;
		}
	}
	return 0;
}

static int enters_hit(float ox, float oy, float oz, float nx, float ny, float nz)
{
	for (int i = 0; i < s_hit_n; i++) {
		if (!hit_solid(i)) {
			continue;
		}
		if (!aabb_overlap(ox, oy, oz, i) && aabb_overlap(nx, ny, nz, i)) {
			return 1;
		}
	}
	return 0;
}

void sys_io_slide(float vx, float vy, float vz, float delta)
{
	if (!s_player_ok) {
		gs_draw_look_point(&s_px, &s_py, &s_pz);
		s_player_ok = 1;
	}
	s_on_floor = s_on_wall = s_on_ceil = 0;
	const float dx = vx * delta;
	const float dy = vy * delta;
	const float dz = vz * delta;
	float nx = s_px;
	float ny = s_py;
	float nz = s_pz;
	if (dx != 0.0f) {
		if (enters_hit(s_px, s_py, s_pz, s_px + dx, s_py, s_pz)) {
			s_on_wall = 1;
		} else {
			nx += dx;
		}
	}
	if (dz != 0.0f) {
		if (enters_hit(nx, s_py, s_pz, nx, s_py, s_pz + dz)) {
			s_on_wall = 1;
		} else {
			nz += dz;
		}
	}
	if (dy != 0.0f) {
		if (enters_hit(nx, s_py, nz, nx, s_py + dy, nz)) {
			if (dy < 0.0f) {
				s_on_floor = 1;
			} else {
				s_on_ceil = 1;
			}
		} else {
			ny += dy;
		}
	}
	if (hits_at(nx, ny - 0.08f, nz)) {
		s_on_floor = 1;
	}
	gs_draw_nudge3(nx - s_px, ny - s_py, nz - s_pz);
	s_px = nx;
	s_py = ny;
	s_pz = nz;
}

int sys_io_on_floor(void)
{
	return s_on_floor;
}

int sys_io_on_wall(void)
{
	return s_on_wall;
}

int sys_io_on_ceiling(void)
{
	return s_on_ceil;
}

int sys_io_hit_hazard(float x, float z)
{
	for (int i = 0; i < s_hit_n; i++) {
		if (s_hit_kit[i] == 5 && aabb_overlap(x, s_py, z, i)) {
			return 1;
		}
	}
	return 0;
}

static void ensure_player(void)
{
	if (!s_player_ok) {
		gs_draw_look_point(&s_px, &s_py, &s_pz);
		s_player_ok = 1;
	}
}

void sys_io_overlap_refresh(void)
{
	ensure_player();
	unsigned long long now = 0;
	for (int i = 0; i < s_hit_n && i < 64; i++) {
		if (aabb_overlap(s_px, s_py, s_pz, i)) {
			now |= (1ull << i);
		}
	}
	s_over_entered = (now & ~s_over_prev) != 0;
	s_over_now = now;
	s_over_prev = now;
}

int sys_io_overlaps(void)
{
	return s_over_now != 0;
}

int sys_io_overlaps_entered(void)
{
	return s_over_entered;
}

int sys_io_has_overlapping(void)
{
	return s_over_now != 0;
}

int sys_io_hitbox_kind(void)
{
	for (int i = 0; i < s_hit_n && i < 64; i++) {
		if (s_over_now & (1ull << i)) {
			return (int)s_hit_kit[i];
		}
	}
	return 0;
}

static int ray_aabb(float ox, float oy, float oz, float dx, float dy, float dz, float dist, int i, float *t_hit)
{
	float tmin = 0.0f;
	float tmax = dist;
	const float o[3] = { ox, oy, oz };
	const float d[3] = { dx, dy, dz };
	for (int a = 0; a < 3; a++) {
		const float mn = s_hit_mn[i][a];
		const float mx = s_hit_mx[i][a];
		if (d[a] > -0.00001f && d[a] < 0.00001f) {
			if (o[a] < mn || o[a] > mx) {
				return 0;
			}
			continue;
		}
		const float inv = 1.0f / d[a];
		float t0 = (mn - o[a]) * inv;
		float t1 = (mx - o[a]) * inv;
		if (t0 > t1) {
			const float tmp = t0;
			t0 = t1;
			t1 = tmp;
		}
		if (t0 > tmin) {
			tmin = t0;
		}
		if (t1 < tmax) {
			tmax = t1;
		}
		if (tmin > tmax) {
			return 0;
		}
	}
	*t_hit = tmin;
	return 1;
}

int sys_io_raycast(float ox, float oy, float oz, float dx, float dy, float dz, float dist, int mask)
{
	(void)mask;
	if (dist <= 0.0f) {
		return 0;
	}
	float best = dist;
	int hit = 0;
	for (int i = 0; i < s_hit_n; i++) {
		float t = 0.0f;
		if (ray_aabb(ox, oy, oz, dx, dy, dz, dist, i, &t) && t >= 0.0f && t < best) {
			best = t;
			hit = 1;
		}
	}
	if (hit) {
		s_ray_x = ox + dx * best;
		s_ray_y = oy + dy * best;
		s_ray_z = oz + dz * best;
	}
	return hit;
}

void sys_io_ray_point(float *x, float *y, float *z)
{
	if (x) {
		*x = s_ray_x;
	}
	if (y) {
		*y = s_ray_y;
	}
	if (z) {
		*z = s_ray_z;
	}
}

int sys_io_tile_at(float x, float y)
{
	const short cx = (short)floorf(x);
	const short cy = (short)floorf(y);
	for (int i = 0; i < s_tile_n; i++) {
		if (s_tile_x[i] == cx && s_tile_y[i] == cy) {
			return (int)s_tile_flags[i];
		}
	}
	return 0;
}

int sys_io_tile_solid_at(float x, float y)
{
	return sys_io_tile_at(x, y) != 0;
}

void sys_io_set_fade(float alpha, float r, float g, float b)
{
	s_fade_left = 0.0f;
	s_fade_dur = 0.0f;
	s_fade_r = (int)r;
	s_fade_g = (int)g;
	s_fade_b = (int)b;
	int a = (int)(alpha * 128.0f);
	if (a < 0) {
		a = 0;
	}
	if (a > 128) {
		a = 128;
	}
	gs_draw_set_fade(a, s_fade_r, s_fade_g, s_fade_b);
}

void sys_io_scene_fade(float sec)
{
	s_fade_r = 0;
	s_fade_g = 0;
	s_fade_b = 0;
	if (sec <= 0.0f) {
		s_fade_left = 0.0f;
		s_fade_dur = 0.0f;
		s_fade_phase = 0;
		if (s_fade_pack >= 0) {
			pack_io_swap(s_fade_pack);
			s_fade_pack = -1;
		}
		gs_draw_set_fade(0, 0, 0, 0);
		return;
	}
	s_fade_dur = sec;
	s_fade_left = sec;
	s_fade_phase = 1;
	gs_draw_set_fade(0, 0, 0, 0);
}

void sys_io_set_fade_pack(int pack)
{
	s_fade_pack = pack;
}
