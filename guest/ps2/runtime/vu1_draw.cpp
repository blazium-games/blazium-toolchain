// MIT. Optional VU1 path using AFL packet2 + draw_3D.vsm. No zbyszek.raw. No gsKit.

#include "vu1_draw.h"

#include "gs_draw.h"

#include <string.h>

#ifndef BLAZIUM_PS2_COOK_ABI
#define BLAZIUM_PS2_COOK_ABI 1
#endif

#define VU1_BATCH 48

#ifndef BLAZIUM_PS2_HAS_VU1
int vu1_draw_init(const unsigned char *mesh, unsigned mesh_sz)
{
	(void)mesh;
	(void)mesh_sz;
	return 0;
}

int vu1_draw_ready(void)
{
	return 0;
}

void vu1_draw_scene(framebuffer_t *frame, zbuffer_t *z)
{
	gs_draw_scene(frame, z);
}
#else

#include <dma.h>
#include <graph.h>
#include <gs_psm.h>
#include <kernel.h>
#include <malloc.h>
#include <packet.h>
#include <packet2.h>
#include <packet2_utils.h>

extern u32 VU1Draw3D_CodeStart __attribute__((section(".vudata")));
extern u32 VU1Draw3D_CodeEnd __attribute__((section(".vudata")));

struct CookVert {
	float x, y, z;
	unsigned char r, g, b, a;
	float u, v;
	unsigned short node;
	unsigned short tex;
};

static int s_ready;
static const unsigned char *s_verts;
static const unsigned char *s_indices;
static unsigned s_vert_count;
static unsigned s_index_count;
static packet2_t *s_vif[2];
static packet2_t *s_hdr;
static int s_ctx;
static float (*s_pos)[4];
static float (*s_st)[4];

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
	s_vert_count = ru32(blob + 8);
	s_index_count = ru32(blob + 12);
	const unsigned vert_bytes = s_vert_count * 28;
	if (16 + vert_bytes + s_index_count * 4 > sz) {
		return 0;
	}
	s_verts = blob + 16;
	s_indices = blob + 16 + vert_bytes;
	return s_vert_count > 0 && s_index_count >= 3;
}

static void vu1_upload_program(void)
{
	u32 packet_size = packet2_utils_get_packet_size_for_program(&VU1Draw3D_CodeStart, &VU1Draw3D_CodeEnd) + 1;
	packet2_t *p = packet2_create(packet_size, P2_TYPE_NORMAL, P2_MODE_CHAIN, 1);
	packet2_vif_add_micro_program(p, 0, &VU1Draw3D_CodeStart, &VU1Draw3D_CodeEnd);
	packet2_utils_vu_add_end_tag(p);
	dma_channel_send_packet2(p, DMA_CHANNEL_VIF1, 1);
	dma_channel_wait(DMA_CHANNEL_VIF1, 0);
	packet2_free(p);
}

static void vu1_set_double_buffer(void)
{
	packet2_t *p = packet2_create(1, P2_TYPE_NORMAL, P2_MODE_CHAIN, 1);
	packet2_utils_vu_add_double_buffer(p, 8, 496);
	packet2_utils_vu_add_end_tag(p);
	dma_channel_send_packet2(p, DMA_CHANNEL_VIF1, 1);
	dma_channel_wait(DMA_CHANNEL_VIF1, 0);
	packet2_free(p);
}

int vu1_draw_init(const unsigned char *mesh, unsigned mesh_sz)
{
	s_ready = 0;
	if (!parse_mesh(mesh, mesh_sz)) {
		return 0;
	}
	dma_channel_initialize(DMA_CHANNEL_VIF1, NULL, 0);
	dma_channel_fast_waits(DMA_CHANNEL_VIF1);
	vu1_upload_program();
	vu1_set_double_buffer();
	s_hdr = packet2_create(24, P2_TYPE_NORMAL, P2_MODE_CHAIN, 1);
	s_vif[0] = packet2_create(200, P2_TYPE_NORMAL, P2_MODE_CHAIN, 1);
	s_vif[1] = packet2_create(200, P2_TYPE_NORMAL, P2_MODE_CHAIN, 1);
	s_pos = (float (*)[4])memalign(128, sizeof(float) * 4 * VU1_BATCH);
	s_st = (float (*)[4])memalign(128, sizeof(float) * 4 * VU1_BATCH);
	if (!s_hdr || !s_vif[0] || !s_vif[1] || !s_pos || !s_st) {
		return 0;
	}
	s_ctx = 0;
	s_ready = 1;
	return 1;
}

int vu1_draw_ready(void)
{
	return s_ready;
}

static void fill_header(texbuffer_t *tex, int nverts)
{
	lod_t lod;
	clutbuffer_t clut;
	prim_t prim;
	memset(&lod, 0, sizeof(lod));
	memset(&clut, 0, sizeof(clut));
	memset(&prim, 0, sizeof(prim));
	lod.calculation = LOD_USE_K;
	lod.mag_filter = LOD_MAG_NEAREST;
	lod.min_filter = LOD_MIN_NEAREST;
	clut.storage_mode = CLUT_STORAGE_MODE1;
	clut.load_method = CLUT_NO_LOAD;
	prim.type = PRIM_TRIANGLE;
	prim.shading = PRIM_SHADE_GOURAUD;
	prim.mapping = DRAW_ENABLE;
	prim.mapping_type = PRIM_MAP_ST;
	prim.colorfix = PRIM_UNFIXED;
	packet2_reset(s_hdr, 0);
	packet2_add_float(s_hdr, 2048.0f);
	packet2_add_float(s_hdr, 2048.0f);
	packet2_add_float(s_hdr, ((float)0xFFFFFF) / 32.0f);
	packet2_add_s32(s_hdr, nverts);
	packet2_utils_gif_add_set(s_hdr, 1);
	packet2_utils_gs_add_lod(s_hdr, &lod);
	packet2_utils_gs_add_texbuff_clut(s_hdr, tex, &clut);
	packet2_utils_gs_add_prim_giftag(s_hdr, &prim, nverts, DRAW_STQ2_REGLIST, 3, 0);
	int i;
	for (i = 0; i < 4; i++) {
		packet2_add_u32(s_hdr, 128);
	}
}

static void kick_batch(float *mvp, texbuffer_t *tex, int nverts)
{
	if (nverts < 3) {
		return;
	}
	fill_header(tex, nverts);
	packet2_t *vif = s_vif[s_ctx];
	packet2_reset(vif, 0);
	packet2_utils_vu_add_unpack_data(vif, 0, mvp, 8, 0);
	u32 added = 0;
	packet2_utils_vu_add_unpack_data(vif, added, s_hdr->base, packet2_get_qw_count(s_hdr), 1);
	added += packet2_get_qw_count(s_hdr);
	packet2_utils_vu_add_unpack_data(vif, added, s_pos, (u32)nverts, 1);
	added += (u32)nverts;
	packet2_utils_vu_add_unpack_data(vif, added, s_st, (u32)nverts, 1);
	packet2_utils_vu_add_start_program(vif, 0);
	packet2_utils_vu_add_end_tag(vif);
	dma_channel_wait(DMA_CHANNEL_VIF1, 0);
	dma_channel_send_packet2(vif, DMA_CHANNEL_VIF1, 1);
	s_ctx ^= 1;
}

void vu1_draw_scene(framebuffer_t *frame, zbuffer_t *z)
{
	if (!s_ready || !frame) {
		gs_draw_scene(frame, z);
		return;
	}
	packet_t *clear = packet_init(32, PACKET_NORMAL);
	qword_t *q = clear->data;
	q = draw_framebuffer(q, 0, frame);
	q = draw_disable_tests(q, 0, z);
	q = draw_clear(q, 0, 2048.0f - 320.0f, 2048.0f - 224.0f, (float)frame->width, (float)frame->height, 32, 64, 160);
	q = draw_enable_tests(q, 0, z);
	q = draw_finish(q);
	FlushCache(0);
	dma_wait_fast();
	dma_channel_send_normal(DMA_CHANNEL_GIF, clear->data, q - clear->data, 0, 0);
	dma_wait_fast();
	packet_free(clear);

	float mvp[16] __attribute__((aligned(16)));
	gs_draw_fill_mvp(mvp, frame->width, frame->height);

	texbuffer_t tex;
	memset(&tex, 0, sizeof(tex));
	int vram = 0, tw = 64, th = 64, psm = GS_PSM_16;
	if (!gs_draw_tex_info(0, &vram, &tw, &th, &psm)) {
		gs_draw_scene(frame, z);
		return;
	}
	tex.width = tw;
	tex.psm = psm;
	tex.address = vram;
	tex.info.width = draw_log2((int)tw);
	tex.info.height = draw_log2((int)th);
	tex.info.components = TEXTURE_COMPONENTS_RGB;
	tex.info.function = TEXTURE_FUNCTION_DECAL;

	int n = 0;
	unsigned i;
	for (i = 0; i + 2 < s_index_count; i += 3) {
		unsigned ia = ru32(s_indices + i * 4);
		unsigned ib = ru32(s_indices + (i + 1) * 4);
		unsigned ic = ru32(s_indices + (i + 2) * 4);
		if (ia >= s_vert_count || ib >= s_vert_count || ic >= s_vert_count) {
			continue;
		}
		CookVert va, vb, vc;
		read_vert(s_verts + ia * 28, &va);
		read_vert(s_verts + ib * 28, &vb);
		read_vert(s_verts + ic * 28, &vc);
		const CookVert *vs[3] = {&va, &vb, &vc};
		int k;
		for (k = 0; k < 3; k++) {
			s_pos[n][0] = vs[k]->x;
			s_pos[n][1] = vs[k]->y;
			s_pos[n][2] = vs[k]->z;
			s_pos[n][3] = 1.0f;
			s_st[n][0] = vs[k]->u;
			s_st[n][1] = vs[k]->v;
			s_st[n][2] = 0.0f;
			s_st[n][3] = 1.0f;
			n++;
		}
		if (n + 3 > VU1_BATCH) {
			kick_batch(mvp, &tex, n);
			n = 0;
		}
	}
	if (n >= 3) {
		kick_batch(mvp, &tex, n);
	}
	dma_channel_wait(DMA_CHANNEL_VIF1, 0);
	draw_wait_finish();
}

#endif
