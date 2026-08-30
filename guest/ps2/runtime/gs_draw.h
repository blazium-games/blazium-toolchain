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

// Input.get_vector / Node3D.translate: slide the look-at target on XZ.
void gs_draw_nudge(float dx, float dz);
void gs_draw_nudge3(float dx, float dy, float dz);
void gs_draw_look_point(float *x, float *y, float *z);
void gs_draw_set_eye(float x, float y, float z);
// Camera3D: Godot Y-up YXZ euler, look along local -Z. fov_deg is the cooked FOV.
void gs_draw_set_camera(float x, float y, float z, float pitch, float yaw, float roll, float fov_deg);
void gs_draw_look(float yaw, float pitch, float roll);
void gs_draw_look_delta(float dyaw, float dpitch);
void gs_draw_orbit_sph(float yaw, float pitch, float dist);
void gs_draw_attach_offset(float x, float y, float z);
void gs_draw_shake(float x, float y, float z);
void gs_draw_hud_quad(int slot, int x, int y, int w, int h, int r, int g, int b);
void gs_draw_set_fade(int a, int r, int g, int b);
void gs_draw_set_node_ofs(int node, float x, float y, float z);
void gs_draw_apply_node(int node, float *x, float *y, float *z);
void gs_draw_overlay(framebuffer_t *frame, zbuffer_t *z);

// Clear + textured tris + slot-0 HUD stamp. Sends GIF packets itself.
void gs_draw_scene(framebuffer_t *frame, zbuffer_t *z);

void gs_draw_camera(float *x, float *y, float *z);
float gs_draw_world_yaw(void);
void gs_draw_fill_mvp(float out[16], int width, int height);
int gs_draw_tex_info(int i, int *vram, int *w, int *h, int *psm);

// GS origin for a framebuffer (2048 - w/2, 2048 - h/2).
void gs_draw_fb_origin(int width, int height, float *ox, float *oy);

#ifdef __cplusplus
}
#endif

#endif
