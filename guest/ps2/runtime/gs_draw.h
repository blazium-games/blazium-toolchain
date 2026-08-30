// MIT. CPU-side GS draw of cooked MESH + GTEX (ABI 1). No VU1. No Godot.
// Positions in MESH are Godot Y-up; this unit maps them to GS (X right, Y down).

#ifndef BLAZIUM_PS2_GS_DRAW_H
#define BLAZIUM_PS2_GS_DRAW_H

#include <draw.h>
#include <tamtypes.h>

#ifdef __cplusplus
extern "C" {
#endif

// Returns 1 if MESH ABI 1 parsed. node/gtex may be NULL.
int gs_draw_init(const unsigned char *mesh, unsigned mesh_sz,
		const unsigned char *gtex, unsigned gtex_sz,
		const unsigned char *node, unsigned node_sz);

int gs_draw_ready(void);

// Digital / stick: yaw around look-at, dolly along view distance.
void gs_draw_orbit(float yaw, float dolly);

// Script-driven Y-up world spin (rotate_y). Independent of pad orbit.
void gs_draw_set_world_yaw(float yaw);

// Clear + textured tris + slot-0 HUD stamp. Sends GIF packets itself.
void gs_draw_scene(framebuffer_t *frame, zbuffer_t *z);

#ifdef __cplusplus
}
#endif

#endif
