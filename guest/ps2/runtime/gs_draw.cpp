// MIT. Parse ABI 1 MESH/GTEX, upload GS textures, CPU-project Y-up verts, GIF tris + HUD.

#include "gs_draw.h"

#include <dma.h>
#include <graph.h>
#include <gs_psm.h>
#include <kernel.h>
#include <malloc.h>
#include <math.h>
#include <packet.h>
#include <string.h>

#ifndef BLAZIUM_PS2_COOK_ABI
#define BLAZIUM_PS2_COOK_ABI 1
#endif

#define GS_MAX_TEX 64
#define GS_BATCH_TRIS 48
#define GS_PACKET_QWORDS 1024
#define GS_HUD_PX 64.0f

enum {
	COOK_PSM_T4 = 0,
	COOK_PSM_T8 = 1,
	COOK_PSM_CT16 = 2
};

struct CookVert {
	float x, y, z;
	unsigned char r, g, b, a;
	float u, v;
	unsigned short node;
	unsigned short tex;
};

struct UploadedTex {
	int ready;
	int hw_psm;
	int width;
	int height;
	int vram;
	int clut_vram;
	int clut_count;
	void *aligned;
	void *clut_aligned;
};

struct Mat4 {
	float m[16];
};

static int g_ready;
static const unsigned char *g_mesh;
static unsigned g_mesh_sz;
static unsigned g_vert_count;
static unsigned g_index_count;
static const unsigned char *g_verts;
static const unsigned char *g_indices;
static UploadedTex g_tex[GS_MAX_TEX];
static int g_tex_count;
static float g_cam_x = 0.0f;
static float g_cam_y = 3.0f;
static float g_cam_z = 8.0f;
static float g_world_yaw = 0.0f;

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

static int cook_psm_to_gs(int cook)
{
	if (cook == COOK_PSM_T4) {
		return GS_PSM_4;
	}
	if (cook == COOK_PSM_T8) {
		return GS_PSM_8;
	}
	return GS_PSM_16;
}

static void read_vert(const unsigned char *p, CookVert *v)
{
	v->x = rf32(p);
	v->y = rf32(p + 4);
	v->z = rf32(p + 8);
	v->r = p[12];
	v->g = p[13];
	v->b = p[14];
	v->a = p[15];
	v->u = rf32(p + 16);
	v->v = rf32(p + 20);
	v->node = (unsigned short)ru16(p + 24);
	v->tex = (unsigned short)ru16(p + 26);
}

static int parse_mesh(const unsigned char *blob, unsigned sz)
{
	if (!blob || sz < 16) {
		return 0;
	}
	if (blob[0] != 'M' || blob[1] != 'E' || blob[2] != 'S' || blob[3] != 'H') {
		return 0;
	}
	if (ru16(blob + 4) != BLAZIUM_PS2_COOK_ABI) {
		return 0;
	}
	g_vert_count = ru32(blob + 8);
	g_index_count = ru32(blob + 12);
	const unsigned vert_bytes = g_vert_count * 28;
	if (16 + vert_bytes + g_index_count * 4 > sz) {
		return 0;
	}
	g_verts = blob + 16;
	g_indices = blob + 16 + vert_bytes;
	g_mesh = blob;
	g_mesh_sz = sz;
	return g_vert_count > 0;
}

static void parse_camera(const unsigned char *blob, unsigned sz)
{
	g_cam_x = 0.0f;
	g_cam_y = 3.0f;
	g_cam_z = 8.0f;
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
		if (n[4] == 3) {
			g_cam_x = rf32(n + 6);
			g_cam_y = rf32(n + 10);
			g_cam_z = rf32(n + 14);
			return;
		}
	}
}

static void *dup_align(const void *src, unsigned n)
{
	void *p = memalign(64, n < 64 ? 64 : n);
	if (!p) {
		return 0;
	}
	memset(p, 0, n < 64 ? 64 : n);
	if (src && n) {
		memcpy(p, src, n);
	}
	return p;
}

static int upload_one(UploadedTex *t, const unsigned char *data, unsigned data_n, const unsigned char *clut, unsigned clut_n)
{
	t->aligned = dup_align(data, data_n);
	if (!t->aligned) {
		return 0;
	}
	t->vram = graph_vram_allocate(t->width, t->height, t->hw_psm, GRAPH_ALIGN_BLOCK);
	if (t->vram < 0) {
		return 0;
	}
	if (clut_n && clut) {
		t->clut_aligned = dup_align(clut, clut_n * 2);
		t->clut_vram = graph_vram_allocate(t->clut_count >= 256 ? 256 : 16, 1, GS_PSM_16, GRAPH_ALIGN_BLOCK);
		if (t->clut_vram < 0) {
			t->clut_count = 0;
		}
	}
	packet_t *packet = packet_init(64, PACKET_NORMAL);
	qword_t *q = packet->data;
	q = draw_texture_transfer(q, t->aligned, t->width, t->height, t->hw_psm, t->vram, t->width);
	if (t->clut_count && t->clut_aligned && t->clut_vram >= 0) {
		const int cw = t->clut_count >= 256 ? 256 : 16;
		q = draw_texture_transfer(q, t->clut_aligned, cw, 1, GS_PSM_16, t->clut_vram, cw);
	}
	q = draw_texture_flush(q);
	FlushCache(0);
	dma_channel_send_chain(DMA_CHANNEL_GIF, packet->data, q - packet->data, 0, 0);
	dma_wait_fast();
	packet_free(packet);
	t->ready = 1;
	return 1;
}

static void parse_upload_gtex(const unsigned char *blob, unsigned sz)
{
	g_tex_count = 0;
	memset(g_tex, 0, sizeof(g_tex));
	if (!blob || sz < 8) {
		return;
	}
	if (blob[0] != 'G' || blob[1] != 'T' || blob[2] != 'E' || blob[3] != 'X') {
		return;
	}
	if (ru16(blob + 4) != BLAZIUM_PS2_COOK_ABI) {
		return;
	}
	const unsigned count = ru16(blob + 6);
	unsigned off = 8;
	for (unsigned i = 0; i < count && g_tex_count < GS_MAX_TEX; i++) {
		if (off + 12 > sz) {
			break;
		}
		UploadedTex *t = &g_tex[g_tex_count];
		t->width = (int)ru16(blob + off);
		t->height = (int)ru16(blob + off + 2);
		t->hw_psm = cook_psm_to_gs((int)blob[off + 4]);
		t->clut_count = (int)ru16(blob + off + 6);
		const unsigned data_n = ru32(blob + off + 8);
		off += 12;
		if (off + data_n + (unsigned)t->clut_count * 2 > sz) {
			break;
		}
		const unsigned char *data = blob + off;
		off += data_n;
		const unsigned char *clut = blob + off;
		off += (unsigned)t->clut_count * 2;
		if (t->width <= 0 || t->height <= 0) {
			continue;
		}
		if (upload_one(t, data, data_n, clut, (unsigned)t->clut_count)) {
			g_tex_count++;
		}
	}
}

static qword_t *bind_tex(qword_t *q, int slot)
{
	if (slot < 0 || slot >= g_tex_count || !g_tex[slot].ready) {
		slot = 0;
	}
	if (slot >= g_tex_count || !g_tex[slot].ready) {
		return q;
	}
	UploadedTex *t = &g_tex[slot];
	texbuffer_t texbuf;
	clutbuffer_t clut;
	lod_t lod;
	memset(&texbuf, 0, sizeof(texbuf));
	memset(&clut, 0, sizeof(clut));
	memset(&lod, 0, sizeof(lod));
	texbuf.address = (unsigned)t->vram;
	texbuf.width = (unsigned)t->width;
	texbuf.psm = (unsigned)t->hw_psm;
	texbuf.info.width = draw_log2((unsigned)t->width);
	texbuf.info.height = draw_log2((unsigned)t->height);
	texbuf.info.components = TEXTURE_COMPONENTS_RGBA;
	texbuf.info.function = TEXTURE_FUNCTION_MODULATE;
	lod.calculation = LOD_USE_K;
	lod.mag_filter = LOD_MAG_NEAREST;
	lod.min_filter = LOD_MIN_NEAREST;
	if (t->clut_count && t->clut_vram >= 0) {
		clut.storage_mode = CLUT_STORAGE_MODE1;
		clut.start = 0;
		clut.psm = GS_PSM_16;
		clut.load_method = CLUT_LOAD;
		clut.address = (unsigned)t->clut_vram;
	} else {
		clut.load_method = CLUT_NO_LOAD;
	}
	q = draw_texture_sampling(q, 0, &lod);
	q = draw_texturebuffer(q, 0, &texbuf, &clut);
	return q;
}

static void mat_ident(Mat4 *o)
{
	memset(o->m, 0, sizeof(o->m));
	o->m[0] = o->m[5] = o->m[10] = o->m[15] = 1.0f;
}

static void mat_mul(const Mat4 *a, const Mat4 *b, Mat4 *o)
{
	Mat4 t;
	for (int c = 0; c < 4; c++) {
		for (int r = 0; r < 4; r++) {
			t.m[c * 4 + r] =
					a->m[0 * 4 + r] * b->m[c * 4 + 0] +
					a->m[1 * 4 + r] * b->m[c * 4 + 1] +
					a->m[2 * 4 + r] * b->m[c * 4 + 2] +
					a->m[3 * 4 + r] * b->m[c * 4 + 3];
		}
	}
	*o = t;
}

static void look_at(Mat4 *o, float ex, float ey, float ez, float tx, float ty, float tz)
{
	float zx = ex - tx, zy = ey - ty, zz = ez - tz;
	float zl = sqrtf(zx * zx + zy * zy + zz * zz);
	if (zl < 1e-5f) {
		zl = 1.0f;
	}
	zx /= zl;
	zy /= zl;
	zz /= zl;
	float xx = 1.0f * zz - 0.0f * zy;
	float xy = 0.0f * zx - 0.0f * zz;
	float xz = 0.0f * zy - 1.0f * zx;
	float xl = sqrtf(xx * xx + xy * xy + xz * xz);
	if (xl < 1e-5f) {
		xx = 1.0f;
		xy = 0.0f;
		xz = 0.0f;
		xl = 1.0f;
	}
	xx /= xl;
	xy /= xl;
	xz /= xl;
	const float yx = zy * xz - zz * xy;
	const float yy = zz * xx - zx * xz;
	const float yz = zx * xy - zy * xx;
	mat_ident(o);
	o->m[0] = xx;
	o->m[4] = xy;
	o->m[8] = xz;
	o->m[12] = -(xx * ex + xy * ey + xz * ez);
	o->m[1] = yx;
	o->m[5] = yy;
	o->m[9] = yz;
	o->m[13] = -(yx * ex + yy * ey + yz * ez);
	o->m[2] = zx;
	o->m[6] = zy;
	o->m[10] = zz;
	o->m[14] = -(zx * ex + zy * ey + zz * ez);
}

static void perspective(Mat4 *o, float fov_deg, float aspect, float zn, float zf)
{
	const float f = 1.0f / tanf(fov_deg * 0.5f * 3.14159265f / 180.0f);
	mat_ident(o);
	o->m[0] = f / aspect;
	o->m[5] = f;
	o->m[10] = (zf + zn) / (zn - zf);
	o->m[11] = -1.0f;
	o->m[14] = (2.0f * zf * zn) / (zn - zf);
	o->m[15] = 0.0f;
}

static void xform(const Mat4 *m, float x, float y, float z, float *ox, float *oy, float *oz, float *ow)
{
	*ox = m->m[0] * x + m->m[4] * y + m->m[8] * z + m->m[12];
	*oy = m->m[1] * x + m->m[5] * y + m->m[9] * z + m->m[13];
	*oz = m->m[2] * x + m->m[6] * y + m->m[10] * z + m->m[14];
	*ow = m->m[3] * x + m->m[7] * y + m->m[11] * z + m->m[15];
}

static int project_vert(const Mat4 *mvp, const CookVert *v, vertex_f_t *clip, color_f_t *col, texel_f_t *st)
{
	float x, y, z, w;
	xform(mvp, v->x, v->y, v->z, &x, &y, &z, &w);
	if (w <= 0.08f) {
		return 0;
	}
	clip->x = x / w;
	clip->y = y / w;
	clip->z = z / w;
	clip->w = w;
	col->r = (float)v->r / 255.0f;
	col->g = (float)v->g / 255.0f;
	col->b = (float)v->b / 255.0f;
	col->a = (float)v->a / 255.0f;
	st->s = v->u;
	st->t = v->v;
	st->r = 0.0f;
	st->q = 1.0f;
	return 1;
}

void gs_draw_fb_origin(int width, int height, float *ox, float *oy)
{
	const int w = width > 0 ? width : 640;
	const int h = height > 0 ? height : 448;
	if (ox) {
		*ox = 2048.0f - (float)(w / 2);
	}
	if (oy) {
		*oy = 2048.0f - (float)(h / 2);
	}
}

static qword_t *emit_hud(qword_t *q, int width, int height)
{
	if (g_tex_count < 1 || !g_tex[0].ready) {
		return q;
	}
	q = bind_tex(q, 0);
	texrect_t rect;
	memset(&rect, 0, sizeof(rect));
	float ox = 0.0f;
	float oy = 0.0f;
	gs_draw_fb_origin(width, height, &ox, &oy);
	rect.v0.x = ox + 8.0f;
	rect.v0.y = oy + 8.0f;
	rect.v0.z = 1;
	rect.v1.x = rect.v0.x + GS_HUD_PX;
	rect.v1.y = rect.v0.y + GS_HUD_PX;
	rect.t0.u = 0.0f;
	rect.t0.v = 0.0f;
	rect.t1.u = (float)g_tex[0].width;
	rect.t1.v = (float)g_tex[0].height;
	rect.color.r = 0x80;
	rect.color.g = 0x80;
	rect.color.b = 0x80;
	rect.color.a = 0x80;
	rect.color.q = 1.0f;
	return draw_rect_textured(q, 0, &rect);
}

int gs_draw_init(const unsigned char *mesh, unsigned mesh_sz,
		const unsigned char *gtex, unsigned gtex_sz,
		const unsigned char *node, unsigned node_sz)
{
	g_ready = 0;
	g_tex_count = 0;
	parse_camera(node, node_sz);
	parse_upload_gtex(gtex, gtex_sz);
	if (!parse_mesh(mesh, mesh_sz)) {
		return 0;
	}
	g_ready = 1;
	return 1;
}

int gs_draw_ready(void)
{
	return g_ready;
}

void gs_draw_set_world_yaw(float yaw)
{
	g_world_yaw = yaw;
}

void gs_draw_camera(float *x, float *y, float *z)
{
	if (x) {
		*x = g_cam_x;
	}
	if (y) {
		*y = g_cam_y;
	}
	if (z) {
		*z = g_cam_z;
	}
}

float gs_draw_world_yaw(void)
{
	return g_world_yaw;
}

int gs_draw_tex_info(int i, int *vram, int *w, int *h, int *psm)
{
	if (i < 0 || i >= g_tex_count || !g_tex[i].ready) {
		return 0;
	}
	if (vram) {
		*vram = g_tex[i].vram;
	}
	if (w) {
		*w = g_tex[i].width;
	}
	if (h) {
		*h = g_tex[i].height;
	}
	if (psm) {
		*psm = g_tex[i].hw_psm;
	}
	return 1;
}

static void yaw_y(Mat4 *o, float yaw)
{
	const float c = cosf(yaw);
	const float s = sinf(yaw);
	mat_ident(o);
	o->m[0] = c;
	o->m[8] = s;
	o->m[2] = -s;
	o->m[10] = c;
}

void gs_draw_fill_mvp(float out[16], int width, int height)
{
	if (!out) {
		return;
	}
	Mat4 world, view, proj, tmp, mvp;
	yaw_y(&world, g_world_yaw);
	look_at(&view, g_cam_x, g_cam_y, g_cam_z, 0.0f, 1.5f, 0.0f);
	perspective(&proj, 55.0f, (float)width / (float)(height ? height : 1), 0.25f, 400.0f);
	mat_mul(&view, &world, &tmp);
	mat_mul(&proj, &tmp, &mvp);
	memcpy(out, mvp.m, sizeof(mvp.m));
}

void gs_draw_orbit(float yaw, float dolly)
{
	const float tx = 0.0f;
	const float ty = 1.5f;
	const float tz = 0.0f;
	float dx = g_cam_x - tx;
	float dz = g_cam_z - tz;
	float dist = sqrtf(dx * dx + dz * dz);
	if (dist < 0.25f) {
		dist = 0.25f;
	}
	float az = atan2f(dx, dz);
	az += yaw;
	dist += dolly;
	if (dist < 1.0f) {
		dist = 1.0f;
	}
	if (dist > 80.0f) {
		dist = 80.0f;
	}
	g_cam_x = tx + sinf(az) * dist;
	g_cam_z = tz + cosf(az) * dist;
	(void)ty;
}

static void send_packet(packet_t *packet, qword_t *q)
{
	if (!packet || !q) {
		return;
	}
	FlushCache(0);
	dma_wait_fast();
	dma_channel_send_normal(DMA_CHANNEL_GIF, packet->data, q - packet->data, 0, 0);
	dma_wait_fast();
}

static qword_t *end_prim(qword_t *q, int *in_prim)
{
	if (!*in_prim) {
		return q;
	}
	u64 *dw = (u64 *)q;
	if (((u32)dw) & 8) {
		*dw++ = 0;
	}
	q = draw_prim_end((qword_t *)dw, 3, DRAW_STQ_REGLIST);
	*in_prim = 0;
	return q;
}

void gs_draw_scene(framebuffer_t *frame, zbuffer_t *z)
{
	if (!frame) {
		return;
	}
	packet_t *packet = packet_init(GS_PACKET_QWORDS, PACKET_NORMAL);
	qword_t *q = packet->data;
	qword_t *limit = packet->data + (GS_PACKET_QWORDS - 80);

	float ox = 0.0f;
	float oy = 0.0f;
	gs_draw_fb_origin(frame->width, frame->height, &ox, &oy);
	q = draw_framebuffer(q, 0, frame);
	q = draw_disable_tests(q, 0, z);
	q = draw_clear(q, 0, ox, oy, (float)frame->width, (float)frame->height, 32, 64, 160);
	q = draw_enable_tests(q, 0, z);
	if (!g_ready) {
		q = emit_hud(q, frame->width, frame->height);
		q = draw_finish(q);
		send_packet(packet, q);
		packet_free(packet);
		return;
	}

	Mat4 world, view, proj, tmp, mvp;
	yaw_y(&world, g_world_yaw);
	look_at(&view, g_cam_x, g_cam_y, g_cam_z, 0.0f, 1.5f, 0.0f);
	perspective(&proj, 55.0f, (float)frame->width / (float)frame->height, 0.25f, 400.0f);
	mat_mul(&view, &world, &tmp);
	mat_mul(&proj, &tmp, &mvp);

	prim_t prim;
	color_t color;
	memset(&prim, 0, sizeof(prim));
	memset(&color, 0, sizeof(color));
	prim.type = PRIM_TRIANGLE;
	prim.shading = PRIM_SHADE_GOURAUD;
	prim.mapping = g_tex_count > 0 ? DRAW_ENABLE : DRAW_DISABLE;
	prim.mapping_type = PRIM_MAP_ST;
	prim.colorfix = PRIM_UNFIXED;
	color.r = 0x80;
	color.g = 0x80;
	color.b = 0x80;
	color.a = 0x80;
	color.q = 1.0f;

	vertex_f_t clip[3];
	color_f_t cols[3];
	texel_f_t sts[3];
	xyz_t xyz[3];
	color_t rgba[3];
	texel_t st[3];

	int bound = -1;
	int in_prim = 0;
	unsigned emitted = 0;

	const unsigned ntri = g_index_count / 3;
	for (unsigned t = 0; t < ntri; t++) {
		const unsigned i0 = ru32(g_indices + t * 12);
		const unsigned i1 = ru32(g_indices + t * 12 + 4);
		const unsigned i2 = ru32(g_indices + t * 12 + 8);
		if (i0 >= g_vert_count || i1 >= g_vert_count || i2 >= g_vert_count) {
			continue;
		}
		CookVert v0, v1, v2;
		read_vert(g_verts + i0 * 28, &v0);
		read_vert(g_verts + i1 * 28, &v1);
		read_vert(g_verts + i2 * 28, &v2);
		if (!project_vert(&mvp, &v0, &clip[0], &cols[0], &sts[0]) ||
				!project_vert(&mvp, &v1, &clip[1], &cols[1], &sts[1]) ||
				!project_vert(&mvp, &v2, &clip[2], &cols[2], &sts[2])) {
			continue;
		}
		const int slot = (int)v0.tex;
		if (q >= limit) {
			q = end_prim(q, &in_prim);
			q = draw_finish(q);
			send_packet(packet, q);
			q = packet->data;
			bound = -1;
			emitted = 0;
		}
		if (slot != bound || emitted >= GS_BATCH_TRIS) {
			q = end_prim(q, &in_prim);
			emitted = 0;
			if (slot != bound) {
				q = bind_tex(q, slot);
				bound = slot;
			}
			q = draw_prim_start(q, 0, &prim, &color);
			in_prim = 1;
		}
		draw_convert_xyz(xyz, 2048.0f, 2048.0f, 16, 3, clip);
		draw_convert_rgbq(rgba, 3, clip, cols, 0x80);
		draw_convert_st(st, 3, clip, sts);
		u64 *dw = (u64 *)q;
		for (int k = 0; k < 3; k++) {
			*dw++ = rgba[k].rgbaq;
			*dw++ = st[k].uv;
			*dw++ = xyz[k].xyz;
		}
		q = (qword_t *)dw;
		emitted++;
	}
	q = end_prim(q, &in_prim);
	q = emit_hud(q, frame->width, frame->height);
	q = draw_finish(q);
	send_packet(packet, q);
	packet_free(packet);
}
