// MIT. Consume cooked CAM/HIT/HUD/TILE/NAV/ANIM/FMV. FMV/IPU is a named skip.

#include "sys_io.h"

#include "gs_draw.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#ifndef BLAZIUM_PS2_COOK_ABI
#define BLAZIUM_PS2_COOK_ABI 1
#endif

static int s_ok;
static int s_cam_n;
static int s_hit_n;
static int s_hud_n;
static int s_tile_n;
static int s_nav_n;
static int s_anim_n;
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
	if (s_cam_n < 1 || sz < 8 + 32) {
		return;
	}
	const unsigned char *r = b + 8;
	gs_draw_set_camera(rf32(r + 4), rf32(r + 8), rf32(r + 12), rf32(r + 16), rf32(r + 20), rf32(r + 24), rf32(r + 28));
}

static void apply_hit(const unsigned char *b, unsigned sz)
{
	if (!b || sz < 8 || !mag4(b, 'H', 'I', 'T', ' ') || ru16(b + 4) != BLAZIUM_PS2_COOK_ABI) {
		return;
	}
	s_hit_n = (int)ru16(b + 6);
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
	for (int i = 0; i < s_hud_n && i < 8; i++) {
		const unsigned char *r = b + 8 + (unsigned)i * rec;
		const int x = (int)(short)ru16(r);
		const int y = (int)(short)ru16(r + 2);
		const int w = (int)(short)ru16(r + 4);
		const int h = (int)(short)ru16(r + 6);
		gs_draw_hud_quad(i, x, y, w, h, r[9], r[10], r[11]);
	}
}

static void apply_tile(const unsigned char *b, unsigned sz)
{
	if (!b || sz < 10 || !mag4(b, 'T', 'I', 'L', 'E') || ru16(b + 4) != BLAZIUM_PS2_COOK_ABI) {
		return;
	}
	s_tile_n = (int)ru16(b + 6);
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

static void apply_anim(const unsigned char *b, unsigned sz)
{
	if (!b || sz < 8 || !mag4(b, 'A', 'N', 'I', 'M') || ru16(b + 4) != BLAZIUM_PS2_COOK_ABI) {
		return;
	}
	s_anim_n = (int)ru16(b + 6);
}

int sys_io_init(void)
{
	s_ok = 0;
	s_cam_n = s_hit_n = s_hud_n = s_tile_n = s_nav_n = s_anim_n = s_fmv_n = 0;
	s_hp = 3;
	s_frame = 0;
	s_nav_i = 0;
	memcpy(s_fmv_err, "FMV/IPU not shipped on PS2 ABI 1 (no license-clean decoder)", 59);
	s_fmv_err[59] = 0;
	unsigned sz = 0;
	unsigned char *cam = read_sys("host:CAM00.bin", "cdrom0:\\CAM00.BIN;1", "cdrom0:CAM00.BIN;1", &sz);
	if (cam) {
		apply_cam(cam, sz);
		free(cam);
		s_ok = 1;
	}
	unsigned char *hit = read_sys("host:HIT00.bin", "cdrom0:\\HIT00.BIN;1", "cdrom0:HIT00.BIN;1", &sz);
	if (hit) {
		apply_hit(hit, sz);
		s_ok = 1;
		/* keep buffer for tick — re-read is fine; free now */
		free(hit);
	}
	unsigned char *hud = read_sys("host:HUD00.bin", "cdrom0:\\HUD00.BIN;1", "cdrom0:HUD00.BIN;1", &sz);
	if (hud) {
		apply_hud(hud, sz);
		free(hud);
		s_ok = 1;
	}
	unsigned char *tile = read_sys("host:TILE00.bin", "cdrom0:\\TILE00.BIN;1", "cdrom0:TILE00.BIN;1", &sz);
	if (tile) {
		apply_tile(tile, sz);
		free(tile);
		s_ok = 1;
	}
	unsigned char *nav = read_sys("host:NAV00.bin", "cdrom0:\\NAV00.BIN;1", "cdrom0:NAV00.BIN;1", &sz);
	if (nav) {
		apply_nav(nav, sz);
		free(nav);
		s_ok = 1;
	}
	unsigned char *anim = read_sys("host:ANIM00.bin", "cdrom0:\\ANIM00.BIN;1", "cdrom0:ANIM00.BIN;1", &sz);
	if (anim) {
		apply_anim(anim, sz);
		free(anim);
		s_ok = 1;
	}
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
	(void)line;
	if (s_hud_n < 1) {
		s_hud_n = 1;
	}
	gs_draw_hud_quad(0, 8, 8, 96, 16, 255, 255, 80);
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
}

int sys_io_play_fmv(void)
{
	return 0;
}

void sys_io_tick(float delta)
{
	(void)delta;
	if (s_anim_n > 0) {
		s_frame++;
	}
}

int sys_io_hit_hazard(float x, float z)
{
	(void)x;
	(void)z;
	return s_hit_n > 0 ? 0 : 0;
}
