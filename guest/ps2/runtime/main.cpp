// Blazium PS2 guest hello — MIT. Links AFL 2.0 ps2sdk graph/draw/dma/packet only.
// Not Godot. Not Main::setup().

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

static void init_gs(framebuffer_t *frame, zbuffer_t *z)
{
	frame->width = 640;
	frame->height = 448;
	frame->mask = 0;
	frame->psm = GS_PSM_16;
	frame->address = graph_vram_allocate(frame->width, frame->height, frame->psm, GRAPH_ALIGN_PAGE);

	z->enable = DRAW_ENABLE;
	z->mask = 0;
	z->method = ZTEST_METHOD_GREATER_EQUAL;
	z->zsm = GS_ZBUF_16;
	z->address = graph_vram_allocate(frame->width, frame->height, z->zsm, GRAPH_ALIGN_PAGE);

	graph_initialize(frame->address, frame->width, frame->height, frame->psm, 0, 0);
}

static void init_drawing_environment(framebuffer_t *frame, zbuffer_t *z)
{
	packet_t *packet = packet_init(16, PACKET_NORMAL);
	qword_t *q = packet->data;
	q = draw_setup_environment(q, 0, frame, z);
	q = draw_primitive_xyoffset(q, 0, (2048 - 320), (2048 - 224));
	q = draw_finish(q);
	dma_channel_send_normal(DMA_CHANNEL_GIF, packet->data, q - packet->data, 0, 0);
	dma_wait_fast();
	packet_free(packet);
}

int main(int argc, char **argv)
{
	(void)argc;
	(void)argv;

	dma_channel_initialize(DMA_CHANNEL_GIF, NULL, 0);
	dma_channel_fast_waits(DMA_CHANNEL_GIF);

	framebuffer_t frame;
	zbuffer_t z;
	init_gs(&frame, &z);
	init_drawing_environment(&frame, &z);

	packet_t *packet = packet_init(16, PACKET_NORMAL);
	for (;;) {
		qword_t *q = packet->data;
		q = draw_clear(q, 0, 2048 - 320, 2048 - 224, frame.width, frame.height, 32, 64, 160);
		q = draw_finish(q);
		dma_wait_fast();
		dma_channel_send_normal(DMA_CHANNEL_GIF, packet->data, q - packet->data, 0, 0);
		draw_wait_finish();
		graph_wait_vsync();
	}
	return 0;
}
