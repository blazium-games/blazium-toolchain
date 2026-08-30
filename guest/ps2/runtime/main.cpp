// Blazium PS2 guest — MIT. Links AFL 2.0 ps2sdk graph/draw/dma/packet only.
// P4: double 16-bit frame + Z, CPU GIF textured MESH/GTEX. Not Godot. Not VU1.

#include "gs_draw.h"

#include <dma.h>
#include <draw.h>
#include <graph.h>
#include <gs_psm.h>
#include <kernel.h>
#include <packet.h>
#include <tamtypes.h>

#define BLAZIUM_PS2_COOK_ABI_VALUE 1

#ifndef BLAZIUM_PS2_COOK_ABI
#define BLAZIUM_PS2_COOK_ABI BLAZIUM_PS2_COOK_ABI_VALUE
#endif

#if defined(__has_include)
#if __has_include("cook_flags.h")
#include "cook_flags.h"
#endif
#endif

#ifdef BLAZIUM_PS2_HAS_NODE
extern "C" {
extern const unsigned char cooked_node[];
extern const unsigned int size_cooked_node;
}
#endif
#ifdef BLAZIUM_PS2_HAS_MESH
extern "C" {
extern const unsigned char cooked_mesh[];
extern const unsigned int size_cooked_mesh;
}
#endif
#ifdef BLAZIUM_PS2_HAS_GTEX
extern "C" {
extern const unsigned char cooked_gtex[];
extern const unsigned int size_cooked_gtex;
}
#endif

static void init_gs(framebuffer_t *frames, zbuffer_t *z)
{
	for (int i = 0; i < 2; i++) {
		frames[i].width = 640;
		frames[i].height = 448;
		frames[i].mask = 0;
		frames[i].psm = GS_PSM_16;
		frames[i].address = graph_vram_allocate(frames[i].width, frames[i].height, frames[i].psm, GRAPH_ALIGN_PAGE);
	}

	z->enable = DRAW_ENABLE;
	z->mask = 0;
	z->method = ZTEST_METHOD_GREATER_EQUAL;
	z->zsm = GS_ZBUF_16;
	z->address = graph_vram_allocate(frames[0].width, frames[0].height, z->zsm, GRAPH_ALIGN_PAGE);

	graph_initialize(frames[0].address, frames[0].width, frames[0].height, frames[0].psm, 0, 0);
}

static void init_drawing_environment(framebuffer_t *frame, zbuffer_t *z)
{
	packet_t *packet = packet_init(16, PACKET_NORMAL);
	qword_t *q = packet->data;
	q = draw_setup_environment(q, 0, frame, z);
	q = draw_primitive_xyoffset(q, 0, (2048 - 320), (2048 - 224));
	q = draw_finish(q);
	FlushCache(0);
	dma_channel_send_normal(DMA_CHANNEL_GIF, packet->data, q - packet->data, 0, 0);
	dma_wait_fast();
	packet_free(packet);
}

static void clear_loop(framebuffer_t *frame)
{
	packet_t *packet = packet_init(16, PACKET_NORMAL);
	for (;;) {
		qword_t *q = packet->data;
		q = draw_clear(q, 0, 2048 - 320, 2048 - 224, frame->width, frame->height, 32, 64, 160);
		q = draw_finish(q);
		FlushCache(0);
		dma_wait_fast();
		dma_channel_send_normal(DMA_CHANNEL_GIF, packet->data, q - packet->data, 0, 0);
		draw_wait_finish();
		graph_wait_vsync();
	}
}

int main(int argc, char **argv)
{
	(void)argc;
	(void)argv;

	dma_channel_initialize(DMA_CHANNEL_GIF, NULL, 0);
	dma_channel_fast_waits(DMA_CHANNEL_GIF);

	framebuffer_t frames[2];
	zbuffer_t z;
	init_gs(frames, &z);
	init_drawing_environment(&frames[0], &z);

	const unsigned char *mesh = 0;
	unsigned mesh_sz = 0;
	const unsigned char *gtex = 0;
	unsigned gtex_sz = 0;
	const unsigned char *node = 0;
	unsigned node_sz = 0;
#ifdef BLAZIUM_PS2_HAS_MESH
	mesh = cooked_mesh;
	mesh_sz = size_cooked_mesh;
#endif
#ifdef BLAZIUM_PS2_HAS_GTEX
	gtex = cooked_gtex;
	gtex_sz = size_cooked_gtex;
#endif
#ifdef BLAZIUM_PS2_HAS_NODE
	node = cooked_node;
	node_sz = size_cooked_node;
#endif

	if (!mesh || !gs_draw_init(mesh, mesh_sz, gtex, gtex_sz, node, node_sz) || !gs_draw_ready()) {
		clear_loop(&frames[0]);
		return 0;
	}

	int context = 0;
	for (;;) {
		gs_draw_scene(&frames[context], &z);
		draw_wait_finish();
		graph_wait_vsync();
		graph_set_framebuffer_filtered(frames[context].address, frames[context].width, frames[context].psm, 0, 0);
		context ^= 1;
	}
	return 0;
}
