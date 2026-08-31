// MIT. Parse CAMN/HITN/ANIM/SPRN/TILN. Drive rdpq camera, AABB hits, TRS clips.

#include "sys_io.h"

#include "dfs_io.h"
#include "pack_io.h"
#include "rdpq_draw.h"
#include "script_vm.h"

#include <math.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#ifndef BLAZIUM_N64_COOK_ABI
#define BLAZIUM_N64_COOK_ABI 1
#endif

enum { kMaxCams = 8, kMaxHits = 64, kMaxAnims = 16, kMaxSprites = 32, kMaxTiles = 256, kMaxAnimKeys = 64 };

static int s_paused;
static int s_hud_just_accept;
static int s_hud_kind;
static char s_hud_act[32];
static float s_ak_t;
static float s_fade;
static int s_cam;
static int s_default_cam;
static int s_kit_player;
static float s_spawn[3];
static int s_entered_kit;
static int s_attach;
static int s_anim_playing;
static int s_anim_id;
static float s_anim_t;

struct CamRow {
	int node_id;
	unsigned flags;
	float pos[3];
	float euler[3];
	float fov;
};

struct HitRow {
	int node_id;
	unsigned char kit;
	float mn[3];
	float mx[3];
	unsigned char flags;
	unsigned char kind;
	unsigned char layer;
	unsigned short frame_lo;
	unsigned short frame_hi;
};

struct AnimKey {
	float t;
	float pos[3];
	float rot[3];
};

struct AnimRow {
	char name[32];
	float length;
	int node_id;
	int nkeys;
	AnimKey keys[kMaxAnimKeys];
};

struct SprtRow {
	int node_id;
	int16_t x, y, w, h, z;
};

struct TileRow {
	int16_t tx, ty;
	unsigned char solid;
};

static CamRow s_cams[kMaxCams];
static int s_cam_n;
static HitRow s_hits[kMaxHits];
static int s_hit_n;
static unsigned char *s_hit_blob;
static unsigned s_hit_blob_sz;
static AnimRow s_anims[kMaxAnims];
static int s_anim_n;
static unsigned char *s_anim_blob;
static unsigned s_anim_blob_sz;
static SprtRow s_sprts[kMaxSprites];
static int s_sprt_n;
static unsigned char *s_sprt_blob;
static unsigned s_sprt_blob_sz;
static TileRow s_tiles[kMaxTiles];
static int s_tile_n;
static unsigned char *s_tile_blob;
static unsigned s_tile_blob_sz;

static uint16_t ru16le(const unsigned char *p)
{
	return (uint16_t)p[0] | ((uint16_t)p[1] << 8);
}

static float rf32le(const unsigned char *p)
{
	unsigned bits = (unsigned)p[0] | ((unsigned)p[1] << 8) | ((unsigned)p[2] << 16) | ((unsigned)p[3] << 24);
	float f;
	memcpy(&f, &bits, sizeof(f));
	return f;
}

static int load_rom_bytes(const char *fmt, int pack, unsigned char **out, unsigned *out_sz)
{
	char path[64];
	snprintf(path, sizeof(path), fmt, pack);
	FILE *f = dfs_io_fopen(path);
	if (!f) {
		return -1;
	}
	if (fseek(f, 0, SEEK_END) != 0) {
		fclose(f);
		return -1;
	}
	const long sz = ftell(f);
	if (sz < 8 || sz > 2 * 1024 * 1024) {
		fclose(f);
		return -1;
	}
	if (fseek(f, 0, SEEK_SET) != 0) {
		fclose(f);
		return -1;
	}
	if (!pack_io_can_fit((int)sz)) {
		fclose(f);
		return -1;
	}
	unsigned char *buf = (unsigned char *)malloc((size_t)sz);
	if (!buf) {
		fclose(f);
		return -1;
	}
	const size_t n = fread(buf, 1, (size_t)sz, f);
	fclose(f);
	if (n != (size_t)sz) {
		free(buf);
		return -1;
	}
	*out = buf;
	*out_sz = (unsigned)sz;
	return 0;
}

static void apply_cam_row(int id)
{
	if (id < 0 || id >= s_cam_n) {
		return;
	}
	const CamRow *c = &s_cams[id];
	rdpq_draw_set_camera(c->pos[0], c->pos[1], c->pos[2], c->euler[1], c->euler[0], c->fov);
	rdpq_draw_set_ortho((c->flags & 1) ? 1 : 0);
	s_cam = id;
}

static void parse_camn(const unsigned char *p, unsigned sz)
{
	s_cam_n = 0;
	if (!p || sz < 12 || p[0] != 'C' || p[1] != 'A' || p[2] != 'M' || p[3] != 'N') {
		return;
	}
	if (ru16le(p + 4) != (uint16_t)BLAZIUM_N64_COOK_ABI) {
		return;
	}
	const unsigned n = ru16le(p + 6);
	s_default_cam = (int)ru16le(p + 8);
	if (n > (unsigned)kMaxCams || 12u + n * 32u > sz) {
		return;
	}
	for (unsigned i = 0; i < n; i++) {
		const unsigned char *r = p + 12 + i * 32;
		CamRow *c = &s_cams[s_cam_n];
		c->node_id = (int)(int16_t)ru16le(r);
		c->flags = ru16le(r + 2);
		c->pos[0] = rf32le(r + 4);
		c->pos[1] = rf32le(r + 8);
		c->pos[2] = rf32le(r + 12);
		c->euler[0] = rf32le(r + 16);
		c->euler[1] = rf32le(r + 20);
		c->euler[2] = rf32le(r + 24);
		c->fov = rf32le(r + 28);
		s_cam_n++;
	}
	if (s_default_cam < 0 || s_default_cam >= s_cam_n) {
		s_default_cam = 0;
	}
	apply_cam_row(s_default_cam);
}

static void parse_hitn(const unsigned char *p, unsigned sz)
{
	s_hit_n = 0;
	if (!p || sz < 8 || p[0] != 'H' || p[1] != 'I' || p[2] != 'T' || p[3] != 'N') {
		return;
	}
	if (ru16le(p + 4) != (uint16_t)BLAZIUM_N64_COOK_ABI) {
		return;
	}
	const unsigned n = ru16le(p + 6);
	if (n > (unsigned)kMaxHits) {
		return;
	}
	unsigned off = 8;
	for (unsigned i = 0; i < n && off + 35 <= sz; i++) {
		const unsigned char *r = p + off;
		HitRow *h = &s_hits[s_hit_n];
		h->node_id = (int)(int16_t)ru16le(r);
		h->kit = r[2];
		h->mn[0] = rf32le(r + 4);
		h->mn[1] = rf32le(r + 8);
		h->mn[2] = rf32le(r + 12);
		h->mx[0] = rf32le(r + 16);
		h->mx[1] = rf32le(r + 20);
		h->mx[2] = rf32le(r + 24);
		h->flags = r[28];
		h->kind = r[29];
		h->layer = r[30];
		h->frame_lo = ru16le(r + 31);
		h->frame_hi = ru16le(r + 33);
		s_hit_n++;
		off += 35;
	}
}

static void parse_anim(const unsigned char *p, unsigned sz)
{
	s_anim_n = 0;
	if (!p || sz < 8 || p[0] != 'A' || p[1] != 'N' || p[2] != 'I' || p[3] != 'M') {
		return;
	}
	if (ru16le(p + 4) != (uint16_t)BLAZIUM_N64_COOK_ABI) {
		return;
	}
	const unsigned n = ru16le(p + 6);
	unsigned off = 8;
	for (unsigned i = 0; i < n && i < (unsigned)kMaxAnims && off + 40 <= sz; i++) {
		AnimRow *a = &s_anims[s_anim_n];
		memcpy(a->name, p + off, 32);
		a->name[31] = 0;
		a->length = rf32le(p + off + 32);
		a->nkeys = (int)ru16le(p + off + 36);
		a->node_id = (int)(int16_t)ru16le(p + off + 38);
		off += 40;
		if (a->nkeys > kMaxAnimKeys) {
			a->nkeys = kMaxAnimKeys;
		}
		int k;
		for (k = 0; k < a->nkeys && off + 28 <= sz; k++) {
			a->keys[k].t = rf32le(p + off);
			a->keys[k].pos[0] = rf32le(p + off + 4);
			a->keys[k].pos[1] = rf32le(p + off + 8);
			a->keys[k].pos[2] = rf32le(p + off + 12);
			a->keys[k].rot[0] = rf32le(p + off + 16);
			a->keys[k].rot[1] = rf32le(p + off + 20);
			a->keys[k].rot[2] = rf32le(p + off + 24);
			off += 28;
		}
		a->nkeys = k;
		s_anim_n++;
	}
}

static void parse_sprn(const unsigned char *p, unsigned sz)
{
	s_sprt_n = 0;
	rdpq_draw_clear_sprites();
	if (!p || sz < 8 || p[0] != 'S' || p[1] != 'P' || p[2] != 'R' || p[3] != 'N') {
		return;
	}
	if (ru16le(p + 4) != (uint16_t)BLAZIUM_N64_COOK_ABI) {
		return;
	}
	const unsigned n = ru16le(p + 6);
	if (n > (unsigned)kMaxSprites || 8u + n * 16u > sz) {
		return;
	}
	for (unsigned i = 0; i < n; i++) {
		const unsigned char *r = p + 8 + i * 16;
		SprtRow *s = &s_sprts[s_sprt_n];
		s->node_id = (int)(int16_t)ru16le(r);
		s->x = (int16_t)ru16le(r + 4);
		s->y = (int16_t)ru16le(r + 6);
		s->w = (int16_t)ru16le(r + 8);
		s->h = (int16_t)ru16le(r + 10);
		s->z = (int16_t)ru16le(r + 14);
		rdpq_draw_sprite(s_sprt_n, s->x, s->y, s->w, s->h);
		s_sprt_n++;
	}
}

static void parse_tiln(const unsigned char *p, unsigned sz)
{
	s_tile_n = 0;
	if (!p || sz < 8 || p[0] != 'T' || p[1] != 'I' || p[2] != 'L' || p[3] != 'N') {
		return;
	}
	if (ru16le(p + 4) != (uint16_t)BLAZIUM_N64_COOK_ABI) {
		return;
	}
	const unsigned n = ru16le(p + 6);
	if (n > (unsigned)kMaxTiles || 8u + n * 6u > sz) {
		return;
	}
	for (unsigned i = 0; i < n; i++) {
		const unsigned char *r = p + 8 + i * 6;
		s_tiles[s_tile_n].tx = (int16_t)ru16le(r);
		s_tiles[s_tile_n].ty = (int16_t)ru16le(r + 2);
		s_tiles[s_tile_n].solid = r[4];
		s_tile_n++;
	}
}

static void free_layer(unsigned char **p, unsigned *sz)
{
	free(*p);
	*p = NULL;
	*sz = 0;
}

static void load_sidecars(int pack)
{
	unsigned char *cam = NULL;
	unsigned cam_sz = 0;
	if (load_rom_bytes("rom://CAM%02d.bin", pack, &cam, &cam_sz) == 0) {
		parse_camn(cam, cam_sz);
		free(cam);
	} else if (pack <= 0) {
		FILE *f = dfs_io_fopen("rom://CAM00.bin");
		if (f) {
			fclose(f);
		}
	}
	free_layer(&s_hit_blob, &s_hit_blob_sz);
	if (load_rom_bytes("rom://HIT%02d.bin", pack, &s_hit_blob, &s_hit_blob_sz) == 0) {
		parse_hitn(s_hit_blob, s_hit_blob_sz);
	} else {
		s_hit_n = 0;
	}
	free_layer(&s_anim_blob, &s_anim_blob_sz);
	if (load_rom_bytes("rom://ANIM%02d.bin", pack, &s_anim_blob, &s_anim_blob_sz) == 0) {
		parse_anim(s_anim_blob, s_anim_blob_sz);
	} else {
		s_anim_n = 0;
	}
	free_layer(&s_sprt_blob, &s_sprt_blob_sz);
	if (load_rom_bytes("rom://SPRN%02d.bin", pack, &s_sprt_blob, &s_sprt_blob_sz) == 0) {
		parse_sprn(s_sprt_blob, s_sprt_blob_sz);
	} else {
		s_sprt_n = 0;
		rdpq_draw_clear_sprites();
	}
	free_layer(&s_tile_blob, &s_tile_blob_sz);
	if (load_rom_bytes("rom://TILN%02d.bin", pack, &s_tile_blob, &s_tile_blob_sz) == 0) {
		parse_tiln(s_tile_blob, s_tile_blob_sz);
	} else {
		s_tile_n = 0;
	}
}

static int aabb_hit(const HitRow *h, float x, float y, float z)
{
	if (x < h->mn[0] || x > h->mx[0] || y < h->mn[1] || y > h->mx[1] || z < h->mn[2] || z > h->mx[2]) {
		return 0;
	}
	const int fr = (int)(s_anim_t * 30.0f);
	if (fr < (int)h->frame_lo || fr > (int)h->frame_hi) {
		return 0;
	}
	return 1;
}

static void kit_enter(const HitRow *h)
{
	s_entered_kit = (int)h->kit;
	if (h->kit == 6) {
		rdpq_draw_talk("Talk");
	}
	if (h->kit == 7) {
		(void)pack_io_instantiate(1);
	}
}

void sys_io_init(void)
{
	s_kit_player = 1;
	s_cam = 0;
	s_default_cam = 0;
	load_sidecars(0);
	if (s_cam_n == 0) {
		rdpq_draw_set_camera(0.0f, 2.0f, 6.0f, 0.0f, 0.321750554f, 55.0f);
	}
}

void sys_io_tick(float delta)
{
	if (s_anim_playing && s_anim_id >= 0 && s_anim_id < s_anim_n) {
		s_anim_t += delta;
		if (s_anims[s_anim_id].length > 0.0f && s_anim_t > s_anims[s_anim_id].length) {
			s_anim_t = fmodf(s_anim_t, s_anims[s_anim_id].length);
		}
		sys_io_seek_anim(s_anim_t);
	}
	if (s_attach) {
		rdpq_draw_set_camera(s_spawn[0], s_spawn[1] + 2.0f, s_spawn[2] + 6.0f, 0.0f, 0.321750554f, 55.0f);
	}
	(void)s_paused;
	(void)s_hud_just_accept;
	(void)s_hud_kind;
	(void)s_hud_act[0];
}

void sys_io_load_pack(int pack)
{
	if (pack_io_swap(pack) != 0) {
		return;
	}
	if (pack <= 0) {
		rdpq_draw_rebind_embedded();
		script_vm_rebind_node(NULL, 0);
		load_sidecars(0);
		return;
	}
	unsigned mesh_sz = 0;
	unsigned ntex_sz = 0;
	unsigned node_sz = 0;
	const unsigned char *mesh = pack_io_cur_mesh(&mesh_sz);
	const unsigned char *ntex = pack_io_cur_ntex(&ntex_sz);
	const unsigned char *node = pack_io_cur_node(&node_sz);
	if (mesh && mesh_sz >= 12) {
		rdpq_draw_rebind(mesh, mesh_sz, ntex, ntex_sz);
	}
	script_vm_rebind_node(node, node_sz);
	load_sidecars(pack);
}

int sys_io_overlap_refresh(void)
{
	return sys_io_overlaps(s_spawn[0], s_spawn[1], s_spawn[2]);
}

int sys_io_entered_kit(void)
{
	return s_entered_kit;
}

void sys_io_spawn_ofs(float x, float y, float z)
{
	s_spawn[0] = x;
	s_spawn[1] = y;
	s_spawn[2] = z;
	(void)rdpq_draw_primary_mesh_node();
}

void sys_io_slide(float *x, float *y, float *z, float dx, float dy, float dz)
{
	float nx = x ? *x + dx : dx;
	float ny = y ? *y + dy : dy;
	float nz = z ? *z + dz : dz;
	for (int i = 0; i < s_hit_n; i++) {
		if (!(s_hits[i].flags & 1)) {
			continue;
		}
		if (aabb_hit(&s_hits[i], nx, ny, nz)) {
			return;
		}
	}
	if (x) {
		*x = nx;
	}
	if (y) {
		*y = ny;
	}
	if (z) {
		*z = nz;
	}
}

int sys_io_overlaps(float x, float y, float z)
{
	int n = 0;
	s_entered_kit = 0;
	for (int i = 0; i < s_hit_n; i++) {
		if (aabb_hit(&s_hits[i], x, y, z)) {
			kit_enter(&s_hits[i]);
			n++;
		}
	}
	return n;
}

int sys_io_raycast(float x, float y, float z, float dx, float dy, float dz)
{
	const float len = sqrtf(dx * dx + dy * dy + dz * dz);
	if (len < 0.0001f) {
		return 0;
	}
	const float inv = 1.0f / len;
	float px = x;
	float py = y;
	float pz = z;
	for (int s = 0; s < 16; s++) {
		px += dx * inv * 0.25f;
		py += dy * inv * 0.25f;
		pz += dz * inv * 0.25f;
		for (int i = 0; i < s_hit_n; i++) {
			if (aabb_hit(&s_hits[i], px, py, pz)) {
				return i + 1;
			}
		}
	}
	return 0;
}

int sys_io_tile_solid_at(int tx, int ty)
{
	for (int i = 0; i < s_tile_n; i++) {
		if (s_tiles[i].tx == (int16_t)tx && s_tiles[i].ty == (int16_t)ty) {
			return s_tiles[i].solid ? 1 : 0;
		}
	}
	return 0;
}

void sys_io_set_fade(float a)
{
	s_fade = a;
	rdpq_draw_set_fade(a);
}

void sys_io_scene_fade(int pack, float t)
{
	(void)t;
	sys_io_set_fade(1.0f);
	sys_io_load_pack(pack);
	sys_io_set_fade(0.0f);
}

void sys_io_set_fade_pack(int pack)
{
	pack_io_swap(pack);
	sys_io_load_pack(pack);
}

void sys_io_set_cam(int id)
{
	apply_cam_row(id);
}

void sys_io_set_default_cam(int id)
{
	s_default_cam = id;
	apply_cam_row(id);
}

void sys_io_look(float yaw, float pitch)
{
	rdpq_draw_look(yaw, pitch);
}

void sys_io_next_cam(void)
{
	if (s_cam_n <= 0) {
		s_cam++;
		return;
	}
	s_cam = (s_cam + 1) % s_cam_n;
	apply_cam_row(s_cam);
}

void sys_io_attach_camera(int on)
{
	s_attach = on;
}

void sys_io_tween_start(int id, float dur)
{
	(void)id;
	(void)dur;
}

void sys_io_timer_start(int id, float dur)
{
	(void)id;
	(void)dur;
}

int sys_io_say_done(void)
{
	return 1;
}

void sys_io_seek_anim(float t)
{
	s_ak_t = t;
	if (s_anim_id < 0 || s_anim_id >= s_anim_n) {
		return;
	}
	const AnimRow *a = &s_anims[s_anim_id];
	if (a->nkeys <= 0) {
		return;
	}
	int i = 0;
	while (i + 1 < a->nkeys && a->keys[i + 1].t <= t) {
		i++;
	}
	float x = a->keys[i].pos[0];
	float y = a->keys[i].pos[1];
	float z = a->keys[i].pos[2];
	float yaw = a->keys[i].rot[1];
	if (i + 1 < a->nkeys) {
		const float dt = a->keys[i + 1].t - a->keys[i].t;
		const float u = dt > 0.0001f ? (t - a->keys[i].t) / dt : 0.0f;
		x += (a->keys[i + 1].pos[0] - x) * u;
		y += (a->keys[i + 1].pos[1] - y) * u;
		z += (a->keys[i + 1].pos[2] - z) * u;
		yaw += (a->keys[i + 1].rot[1] - yaw) * u;
	}
	rdpq_draw_set_anim_ofs(x, y, z, yaw);
}

void sys_io_play_anim(int id)
{
	if (id < 0 || id >= s_anim_n) {
		id = 0;
	}
	s_anim_id = id;
	s_anim_playing = 1;
	s_anim_t = 0.0f;
	sys_io_seek_anim(0.0f);
}

void sys_io_stop_anim(void)
{
	s_anim_playing = 0;
}

int sys_io_load_anims(int pack)
{
	free_layer(&s_anim_blob, &s_anim_blob_sz);
	if (load_rom_bytes("rom://ANIM%02d.bin", pack, &s_anim_blob, &s_anim_blob_sz) != 0) {
		s_anim_n = 0;
		return 0;
	}
	parse_anim(s_anim_blob, s_anim_blob_sz);
	return s_anim_n > 0;
}

void sys_io_unload_anims(void)
{
	free_layer(&s_anim_blob, &s_anim_blob_sz);
	s_anim_n = 0;
	s_anim_playing = 0;
	rdpq_draw_set_anim_ofs(0, 0, 0, 0);
}

int sys_io_load_sprites(int pack)
{
	free_layer(&s_sprt_blob, &s_sprt_blob_sz);
	if (load_rom_bytes("rom://SPRN%02d.bin", pack, &s_sprt_blob, &s_sprt_blob_sz) != 0) {
		s_sprt_n = 0;
		return 0;
	}
	parse_sprn(s_sprt_blob, s_sprt_blob_sz);
	return s_sprt_n > 0;
}

void sys_io_unload_sprites(void)
{
	free_layer(&s_sprt_blob, &s_sprt_blob_sz);
	s_sprt_n = 0;
	rdpq_draw_clear_sprites();
}

int sys_io_load_hits(int pack)
{
	free_layer(&s_hit_blob, &s_hit_blob_sz);
	if (load_rom_bytes("rom://HIT%02d.bin", pack, &s_hit_blob, &s_hit_blob_sz) != 0) {
		s_hit_n = 0;
		return 0;
	}
	parse_hitn(s_hit_blob, s_hit_blob_sz);
	return s_hit_n > 0;
}

void sys_io_unload_hits(void)
{
	free_layer(&s_hit_blob, &s_hit_blob_sz);
	s_hit_n = 0;
}

void sys_io_move_planar(float dx, float dz)
{
	sys_io_slide(&s_spawn[0], &s_spawn[1], &s_spawn[2], dx, 0, dz);
}

void sys_io_move_6dof(float dx, float dy, float dz)
{
	sys_io_slide(&s_spawn[0], &s_spawn[1], &s_spawn[2], dx, dy, dz);
}

void sys_io_hurt(int amount)
{
	(void)amount;
}

void sys_io_path_follow(int path)
{
	(void)path;
}

int sys_io_kit_player(void)
{
	return s_kit_player;
}
