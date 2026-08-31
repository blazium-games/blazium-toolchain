// MIT. Parse CAMN/HITN/ANIM/SPRN/TILN. Drive rdpq camera, AABB hits, TRS clips.

#include "sys_io.h"

#include "dfs_io.h"
#include "pack_io.h"
#include "pad_io.h"
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

enum {
	kMaxCams = 8,
	kMaxHits = 64,
	kMaxAnims = 16,
	kMaxSprites = 32,
	kMaxTiles = 256,
	kMaxAnimKeys = 64,
	kMaxParts = 8,
	kMaxPaths = 8,
	kMaxPathPts = 32,
	kMaxWays = 32,
	kMaxNavEdges = 96,
	kMaxNavmVerts = 64,
	kMaxNavmTris = 32,
	kMaxVehls = 4,
	kMaxTweens = 8,
	kMaxLiveParts = 32
};

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
	uint8_t flip;
	uint8_t frame;
};

struct PartRow {
	int node_id;
	unsigned char mode;
	float rate;
	float life;
	float speed;
	unsigned color;
	float u0, v0, u1, v1;
};

struct PathRow {
	int node_id;
	int npts;
	float pts[kMaxPathPts][3];
};

struct WayRow {
	int node_id;
	float pos[3];
};

struct NavEdge {
	unsigned short a;
	unsigned short b;
};

struct NavmVert {
	float pos[3];
};

struct NavmTri {
	unsigned short a, b, c;
};

struct VehlRow {
	int node_id;
	unsigned char mode;
	float pos[3];
	float yaw;
	float speed;
	float turn;
};

struct TweenSlot {
	int used;
	int id;
	float dur;
	float t;
	float from[3];
	float to[3];
};

struct LivePart {
	int used;
	int mode;
	float x, y, z;
	float px, py, pz;
	float vx, vy, vz;
	float age;
	float life;
	float frame;
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
static PartRow s_parts[kMaxParts];
static int s_part_n;
static unsigned char *s_part_blob;
static unsigned s_part_blob_sz;
static PathRow s_paths[kMaxPaths];
static int s_path_n;
static unsigned char *s_path_blob;
static unsigned s_path_blob_sz;
static WayRow s_ways[kMaxWays];
static int s_way_n;
static unsigned char *s_way_blob;
static unsigned s_way_blob_sz;
static NavEdge s_nav_edges[kMaxNavEdges];
static int s_nav_edge_n;
static unsigned char *s_navn_blob;
static unsigned s_navn_blob_sz;
static NavmVert s_navm_verts[kMaxNavmVerts];
static int s_navm_vn;
static NavmTri s_navm_tris[kMaxNavmTris];
static int s_navm_tn;
static unsigned char *s_navm_blob;
static unsigned s_navm_blob_sz;
static VehlRow s_vehls[kMaxVehls];
static int s_vehl_n;
static unsigned char *s_vehl_blob;
static unsigned s_vehl_blob_sz;
static TweenSlot s_tweens[kMaxTweens];
static LivePart s_live[kMaxLiveParts];
static float s_emit_acc[kMaxParts];
static int s_follow_id = -1;
static int s_path_id;
static float s_path_t;
static int s_nav_cur;
static int s_drive_id = -1;
static float s_pitch;
static float s_roll;
static int s_parts_playing;

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
		s->flip = r[12];
		s->frame = r[13];
		s->z = (int16_t)ru16le(r + 14);
		rdpq_draw_sprite_ex(s_sprt_n, s->x, s->y, s->w, s->h, s->flip, s->frame);
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
		rdpq_draw_tile_cell(s_tiles[s_tile_n].tx, s_tiles[s_tile_n].ty);
		s_tile_n++;
	}
}

static void parse_prtn(const unsigned char *p, unsigned sz)
{
	s_part_n = 0;
	if (!p || sz < 8 || p[0] != 'P' || p[1] != 'R' || p[2] != 'T' || p[3] != 'N') {
		return;
	}
	if (ru16le(p + 4) != (uint16_t)BLAZIUM_N64_COOK_ABI) {
		return;
	}
	const unsigned n = ru16le(p + 6);
	if (n > (unsigned)kMaxParts || 8u + n * 36u > sz) {
		return;
	}
	for (unsigned i = 0; i < n; i++) {
		const unsigned char *r = p + 8 + i * 36;
		PartRow *e = &s_parts[s_part_n];
		e->node_id = (int)(int16_t)ru16le(r);
		e->mode = r[2]; /* 0 billboard 1 flipbook 2 trail 3 point 4 stream */
		e->rate = rf32le(r + 4);
		e->life = rf32le(r + 8);
		e->speed = rf32le(r + 12);
		e->color = (unsigned)r[16] | ((unsigned)r[17] << 8) | ((unsigned)r[18] << 16) | ((unsigned)r[19] << 24);
		e->u0 = rf32le(r + 20);
		e->v0 = rf32le(r + 24);
		e->u1 = rf32le(r + 28);
		e->v1 = rf32le(r + 32);
		s_part_n++;
	}
}

static void parse_pthn(const unsigned char *p, unsigned sz)
{
	s_path_n = 0;
	if (!p || sz < 8 || p[0] != 'P' || p[1] != 'T' || p[2] != 'H' || p[3] != 'N') {
		return;
	}
	if (ru16le(p + 4) != (uint16_t)BLAZIUM_N64_COOK_ABI) {
		return;
	}
	const unsigned n = ru16le(p + 6);
	unsigned off = 8;
	for (unsigned i = 0; i < n && i < (unsigned)kMaxPaths && off + 4 <= sz; i++) {
		PathRow *path = &s_paths[s_path_n];
		path->node_id = (int)(int16_t)ru16le(p + off);
		path->npts = (int)ru16le(p + off + 2);
		off += 4;
		if (path->npts > kMaxPathPts) {
			path->npts = kMaxPathPts;
		}
		int k;
		for (k = 0; k < path->npts && off + 12 <= sz; k++) {
			path->pts[k][0] = rf32le(p + off);
			path->pts[k][1] = rf32le(p + off + 4);
			path->pts[k][2] = rf32le(p + off + 8);
			off += 12;
		}
		path->npts = k;
		s_path_n++;
	}
}

static void parse_wayn(const unsigned char *p, unsigned sz)
{
	s_way_n = 0;
	if (!p || sz < 8 || p[0] != 'W' || p[1] != 'A' || p[2] != 'Y' || p[3] != 'N') {
		return;
	}
	if (ru16le(p + 4) != (uint16_t)BLAZIUM_N64_COOK_ABI) {
		return;
	}
	const unsigned n = ru16le(p + 6);
	if (n > (unsigned)kMaxWays || 8u + n * 16u > sz) {
		return;
	}
	for (unsigned i = 0; i < n; i++) {
		const unsigned char *r = p + 8 + i * 16;
		s_ways[s_way_n].node_id = (int)(int16_t)ru16le(r);
		s_ways[s_way_n].pos[0] = rf32le(r + 4);
		s_ways[s_way_n].pos[1] = rf32le(r + 8);
		s_ways[s_way_n].pos[2] = rf32le(r + 12);
		s_way_n++;
	}
}

static void parse_navn(const unsigned char *p, unsigned sz)
{
	s_nav_edge_n = 0;
	if (!p || sz < 8 || p[0] != 'N' || p[1] != 'A' || p[2] != 'V' || p[3] != 'N') {
		return;
	}
	if (ru16le(p + 4) != (uint16_t)BLAZIUM_N64_COOK_ABI) {
		return;
	}
	const unsigned n = ru16le(p + 6);
	if (n > (unsigned)kMaxNavEdges || 8u + n * 4u > sz) {
		return;
	}
	for (unsigned i = 0; i < n; i++) {
		s_nav_edges[s_nav_edge_n].a = ru16le(p + 8 + i * 4);
		s_nav_edges[s_nav_edge_n].b = ru16le(p + 10 + i * 4);
		s_nav_edge_n++;
	}
}

static void parse_navm(const unsigned char *p, unsigned sz)
{
	s_navm_vn = 0;
	s_navm_tn = 0;
	if (!p || sz < 12 || p[0] != 'N' || p[1] != 'A' || p[2] != 'V' || p[3] != 'M') {
		return;
	}
	if (ru16le(p + 4) != (uint16_t)BLAZIUM_N64_COOK_ABI) {
		return;
	}
	const unsigned nv = ru16le(p + 6);
	const unsigned nt = ru16le(p + 8);
	if (nv > (unsigned)kMaxNavmVerts || nt > (unsigned)kMaxNavmTris) {
		return;
	}
	unsigned off = 12;
	for (unsigned i = 0; i < nv && off + 12 <= sz; i++) {
		s_navm_verts[s_navm_vn].pos[0] = rf32le(p + off);
		s_navm_verts[s_navm_vn].pos[1] = rf32le(p + off + 4);
		s_navm_verts[s_navm_vn].pos[2] = rf32le(p + off + 8);
		s_navm_vn++;
		off += 12;
	}
	for (unsigned i = 0; i < nt && off + 6 <= sz; i++) {
		s_navm_tris[s_navm_tn].a = ru16le(p + off);
		s_navm_tris[s_navm_tn].b = ru16le(p + off + 2);
		s_navm_tris[s_navm_tn].c = ru16le(p + off + 4);
		s_navm_tn++;
		off += 6;
	}
}

static void parse_vehn(const unsigned char *p, unsigned sz)
{
	s_vehl_n = 0;
	if (!p || sz < 8 || p[0] != 'V' || p[1] != 'E' || p[2] != 'H' || p[3] != 'N') {
		return;
	}
	if (ru16le(p + 4) != (uint16_t)BLAZIUM_N64_COOK_ABI) {
		return;
	}
	const unsigned n = ru16le(p + 6);
	if (n > (unsigned)kMaxVehls || 8u + n * 28u > sz) {
		return;
	}
	for (unsigned i = 0; i < n; i++) {
		const unsigned char *r = p + 8 + i * 28;
		VehlRow *v = &s_vehls[s_vehl_n];
		v->node_id = (int)(int16_t)ru16le(r);
		v->mode = r[2];
		v->pos[0] = rf32le(r + 4);
		v->pos[1] = rf32le(r + 8);
		v->pos[2] = rf32le(r + 12);
		v->yaw = rf32le(r + 16);
		v->speed = rf32le(r + 20);
		v->turn = rf32le(r + 24);
		s_vehl_n++;
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
		rdpq_draw_clear_tiles();
		parse_tiln(s_tile_blob, s_tile_blob_sz);
	} else {
		s_tile_n = 0;
		rdpq_draw_clear_tiles();
	}
	free_layer(&s_part_blob, &s_part_blob_sz);
	if (load_rom_bytes("rom://PRTN%02d.bin", pack, &s_part_blob, &s_part_blob_sz) == 0) {
		parse_prtn(s_part_blob, s_part_blob_sz);
	} else {
		s_part_n = 0;
	}
	free_layer(&s_path_blob, &s_path_blob_sz);
	if (load_rom_bytes("rom://PTHN%02d.bin", pack, &s_path_blob, &s_path_blob_sz) == 0) {
		parse_pthn(s_path_blob, s_path_blob_sz);
	} else {
		s_path_n = 0;
	}
	free_layer(&s_way_blob, &s_way_blob_sz);
	if (load_rom_bytes("rom://WAYN%02d.bin", pack, &s_way_blob, &s_way_blob_sz) == 0) {
		parse_wayn(s_way_blob, s_way_blob_sz);
	} else {
		s_way_n = 0;
	}
	free_layer(&s_navn_blob, &s_navn_blob_sz);
	if (load_rom_bytes("rom://NAVN%02d.bin", pack, &s_navn_blob, &s_navn_blob_sz) == 0) {
		parse_navn(s_navn_blob, s_navn_blob_sz);
	} else {
		s_nav_edge_n = 0;
	}
	free_layer(&s_navm_blob, &s_navm_blob_sz);
	if (load_rom_bytes("rom://NAVM%02d.bin", pack, &s_navm_blob, &s_navm_blob_sz) == 0) {
		parse_navm(s_navm_blob, s_navm_blob_sz);
	} else {
		s_navm_vn = 0;
		s_navm_tn = 0;
	}
	free_layer(&s_vehl_blob, &s_vehl_blob_sz);
	if (load_rom_bytes("rom://VEHN%02d.bin", pack, &s_vehl_blob, &s_vehl_blob_sz) == 0) {
		parse_vehn(s_vehl_blob, s_vehl_blob_sz);
	} else {
		s_vehl_n = 0;
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
		rdpq_draw_set_camera(s_spawn[0], s_spawn[1] + 2.0f, s_spawn[2] + 6.0f, 0.0f, 0.321750554f + s_pitch, 55.0f);
	}
	if (s_follow_id >= 0) {
		rdpq_draw_set_camera(s_spawn[0], s_spawn[1] + 1.5f, s_spawn[2] + 4.0f, 0.0f, s_pitch, 55.0f);
	}
	for (int i = 0; i < kMaxTweens; i++) {
		if (!s_tweens[i].used) {
			continue;
		}
		s_tweens[i].t += delta;
		float u = s_tweens[i].dur > 0.0001f ? s_tweens[i].t / s_tweens[i].dur : 1.0f;
		if (u > 1.0f) {
			u = 1.0f;
			s_tweens[i].used = 0;
		}
		s_spawn[0] = s_tweens[i].from[0] + (s_tweens[i].to[0] - s_tweens[i].from[0]) * u;
		s_spawn[1] = s_tweens[i].from[1] + (s_tweens[i].to[1] - s_tweens[i].from[1]) * u;
		s_spawn[2] = s_tweens[i].from[2] + (s_tweens[i].to[2] - s_tweens[i].from[2]) * u;
	}
	if (s_drive_id >= 0 && s_drive_id < s_vehl_n) {
		float sx = 0;
		float sy = 0;
		pad_io_stick(&sx, &sy);
		VehlRow *v = &s_vehls[s_drive_id];
		v->yaw += sx * v->turn * delta;
		const float spd = (pad_io_held(0) ? v->speed : v->speed * 0.5f) * sy;
		if (v->mode == 2) {
			sys_io_slide(&v->pos[0], &v->pos[1], &v->pos[2], sinf(v->yaw) * spd * delta, s_pitch * spd * delta, cosf(v->yaw) * spd * delta);
		} else {
			const float gy = v->mode == 1 ? 0.0f : -0.4f * delta;
			sys_io_slide(&v->pos[0], &v->pos[1], &v->pos[2], sinf(v->yaw) * spd * delta, gy, cosf(v->yaw) * spd * delta);
		}
		s_spawn[0] = v->pos[0];
		s_spawn[1] = v->pos[1];
		s_spawn[2] = v->pos[2];
	}
	if (s_parts_playing) {
		rdpq_draw_clear_parts();
		for (int e = 0; e < s_part_n; e++) {
			s_emit_acc[e] += delta * (s_parts[e].rate > 0.0f ? s_parts[e].rate : 4.0f);
			while (s_emit_acc[e] >= 1.0f) {
				s_emit_acc[e] -= 1.0f;
				int slot = -1;
				for (int i = 0; i < kMaxLiveParts; i++) {
					if (!s_live[i].used) {
						slot = i;
						break;
					}
				}
				if (slot < 0) {
					break;
				}
				s_live[slot].used = 1;
				s_live[slot].mode = s_parts[e].mode;
				s_live[slot].x = s_spawn[0];
				s_live[slot].y = s_spawn[1] + 0.5f;
				s_live[slot].z = s_spawn[2];
				s_live[slot].px = s_live[slot].x;
				s_live[slot].py = s_live[slot].y;
				s_live[slot].pz = s_live[slot].z;
				s_live[slot].vx = ((float)((slot * 17) % 7) - 3.0f) * 0.15f * s_parts[e].speed;
				s_live[slot].vy = s_parts[e].speed * 0.4f;
				s_live[slot].vz = ((float)((slot * 13) % 7) - 3.0f) * 0.15f * s_parts[e].speed;
				s_live[slot].age = 0.0f;
				s_live[slot].life = s_parts[e].life > 0.05f ? s_parts[e].life : 0.6f;
				s_live[slot].frame = 0.0f;
			}
		}
		int drawn = 0;
		for (int i = 0; i < kMaxLiveParts; i++) {
			if (!s_live[i].used) {
				continue;
			}
			s_live[i].age += delta;
			if (s_live[i].age >= s_live[i].life && s_live[i].mode != 4) {
				s_live[i].used = 0;
				continue;
			}
			if (s_live[i].mode == 4 && s_live[i].age >= s_live[i].life) {
				s_live[i].age = 0.0f;
			}
			s_live[i].px = s_live[i].x;
			s_live[i].py = s_live[i].y;
			s_live[i].pz = s_live[i].z;
			s_live[i].x += s_live[i].vx * delta;
			s_live[i].y += s_live[i].vy * delta;
			s_live[i].z += s_live[i].vz * delta;
			s_live[i].frame += delta * 8.0f;
			float w = 4.0f;
			float h = 4.0f;
			if (s_live[i].mode == 2) {
				w = 8.0f;
				h = 2.0f;
			} else if (s_live[i].mode == 3) {
				w = 1.0f;
				h = 1.0f;
			} else if (s_live[i].mode == 1) {
				w = 6.0f;
				h = 6.0f;
			}
			float sx = s_live[i].x * 16.0f + 160.0f;
			float sy = 120.0f - s_live[i].y * 16.0f;
			(void)s_live[i].z;
			rdpq_draw_part_quad(drawn, sx, sy, w, h);
			drawn++;
			if (drawn >= kMaxLiveParts) {
				break;
			}
		}
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
	int slot = -1;
	for (int i = 0; i < kMaxTweens; i++) {
		if (!s_tweens[i].used) {
			slot = i;
			break;
		}
	}
	if (slot < 0) {
		slot = 0;
	}
	s_tweens[slot].used = 1;
	s_tweens[slot].id = id;
	s_tweens[slot].dur = dur > 0.01f ? dur : 0.5f;
	s_tweens[slot].t = 0.0f;
	s_tweens[slot].from[0] = s_spawn[0];
	s_tweens[slot].from[1] = s_spawn[1];
	s_tweens[slot].from[2] = s_spawn[2];
	s_tweens[slot].to[0] = s_spawn[0] + 1.0f;
	s_tweens[slot].to[1] = s_spawn[1];
	s_tweens[slot].to[2] = s_spawn[2];
}

void sys_io_tween_kill(void)
{
	for (int i = 0; i < kMaxTweens; i++) {
		s_tweens[i].used = 0;
	}
}

int sys_io_tween_done(int id)
{
	for (int i = 0; i < kMaxTweens; i++) {
		if (s_tweens[i].used && s_tweens[i].id == id) {
			return 0;
		}
	}
	return 1;
}

void sys_io_follow_node(int id)
{
	s_follow_id = id;
	s_attach = 1;
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
	float sx = 0;
	float sy = 0;
	pad_io_stick(&sx, &sy);
	s_pitch += dy * 0.05f;
	s_roll += dx * 0.05f;
	const float cy = cosf(s_pitch);
	sys_io_slide(&s_spawn[0], &s_spawn[1], &s_spawn[2], (dx + sx) * cy, dy + s_pitch * 0.1f, dz + sy);
}

void sys_io_hurt(int amount)
{
	(void)amount;
}

void sys_io_path_follow(int path)
{
	if (path < 0 || path >= s_path_n) {
		path = 0;
	}
	if (s_path_n <= 0 || s_paths[path].npts < 2) {
		return;
	}
	s_path_id = path;
	s_path_t += 0.02f;
	if (s_path_t > 1.0f) {
		s_path_t = 0.0f;
	}
	const PathRow *p = &s_paths[path];
	const float ft = s_path_t * (float)(p->npts - 1);
	int i0 = (int)ft;
	if (i0 < 0) {
		i0 = 0;
	}
	if (i0 >= p->npts - 1) {
		i0 = p->npts - 2;
	}
	const float u = ft - (float)i0;
	s_spawn[0] = p->pts[i0][0] + (p->pts[i0 + 1][0] - p->pts[i0][0]) * u;
	s_spawn[1] = p->pts[i0][1] + (p->pts[i0 + 1][1] - p->pts[i0][1]) * u;
	s_spawn[2] = p->pts[i0][2] + (p->pts[i0 + 1][2] - p->pts[i0][2]) * u;
	sys_io_spawn_ofs(s_spawn[0], s_spawn[1], s_spawn[2]);
}

void sys_io_nav_follow(int id)
{
	if (s_way_n <= 0) {
		return;
	}
	int cur = s_nav_cur;
	if (cur < 0 || cur >= s_way_n) {
		cur = 0;
	}
	int next = -1;
	for (int i = 0; i < s_nav_edge_n; i++) {
		if ((int)s_nav_edges[i].a == cur) {
			next = (int)s_nav_edges[i].b;
			break;
		}
	}
	if (next < 0) {
		next = (cur + 1) % s_way_n;
	}
	(void)id;
	s_nav_cur = next;
	s_spawn[0] = s_ways[next].pos[0];
	s_spawn[1] = s_ways[next].pos[1];
	s_spawn[2] = s_ways[next].pos[2];
}

int sys_io_navmesh_next(void)
{
	if (s_navm_tn <= 0 || s_navm_vn <= 0) {
		return -1;
	}
	int best = 0;
	float best_d = 1.0e9f;
	for (int i = 0; i < s_navm_tn; i++) {
		const NavmTri *t = &s_navm_tris[i];
		if (t->a >= (unsigned)s_navm_vn || t->b >= (unsigned)s_navm_vn || t->c >= (unsigned)s_navm_vn) {
			continue;
		}
		const float cx = (s_navm_verts[t->a].pos[0] + s_navm_verts[t->b].pos[0] + s_navm_verts[t->c].pos[0]) / 3.0f;
		const float cz = (s_navm_verts[t->a].pos[2] + s_navm_verts[t->b].pos[2] + s_navm_verts[t->c].pos[2]) / 3.0f;
		const float dx = cx - s_spawn[0];
		const float dz = cz - s_spawn[2];
		const float d = dx * dx + dz * dz;
		if (d < best_d && d > 0.01f) {
			best_d = d;
			best = i;
		}
	}
	const NavmTri *t = &s_navm_tris[best];
	s_spawn[0] = (s_navm_verts[t->a].pos[0] + s_navm_verts[t->b].pos[0] + s_navm_verts[t->c].pos[0]) / 3.0f;
	s_spawn[1] = (s_navm_verts[t->a].pos[1] + s_navm_verts[t->b].pos[1] + s_navm_verts[t->c].pos[1]) / 3.0f;
	s_spawn[2] = (s_navm_verts[t->a].pos[2] + s_navm_verts[t->b].pos[2] + s_navm_verts[t->c].pos[2]) / 3.0f;
	return best;
}

int sys_io_load_particles(int pack)
{
	free_layer(&s_part_blob, &s_part_blob_sz);
	if (load_rom_bytes("rom://PRTN%02d.bin", pack, &s_part_blob, &s_part_blob_sz) != 0) {
		s_part_n = 0;
		return 0;
	}
	parse_prtn(s_part_blob, s_part_blob_sz);
	s_parts_playing = 1;
	return s_part_n > 0;
}

void sys_io_unload_particles(void)
{
	free_layer(&s_part_blob, &s_part_blob_sz);
	s_part_n = 0;
	s_parts_playing = 0;
	for (int i = 0; i < kMaxLiveParts; i++) {
		s_live[i].used = 0;
	}
	rdpq_draw_clear_parts();
}

void sys_io_play_particles(int id)
{
	(void)id;
	s_parts_playing = 1;
}

void sys_io_drive_vehicle(int id)
{
	if (id < 0 || id >= s_vehl_n) {
		id = 0;
	}
	s_drive_id = id;
}

int sys_io_kit_player(void)
{
	return s_kit_player;
}
