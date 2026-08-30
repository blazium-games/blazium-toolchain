// MIT. Optional VU1 transform of cooked MESH ABI 1. CPU GIF is the fallback.

#ifndef BLAZIUM_PS2_VU1_DRAW_H
#define BLAZIUM_PS2_VU1_DRAW_H

#include <draw.h>
#include <tamtypes.h>

#ifdef __cplusplus
extern "C" {
#endif

int vu1_draw_init(const unsigned char *mesh, unsigned mesh_sz);
int vu1_draw_ready(void);
void vu1_draw_scene(framebuffer_t *frame, zbuffer_t *z);

#ifdef __cplusplus
}
#endif

#endif
