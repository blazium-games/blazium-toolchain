#ifndef BLAZIUM_N64_RDPQ_DRAW_H
#define BLAZIUM_N64_RDPQ_DRAW_H

#ifdef __cplusplus
extern "C" {
#endif

void rdpq_draw_init(void);
void rdpq_draw_begin(void);
void rdpq_draw_end(void);
void rdpq_draw_frame(void);
void rdpq_draw_set_camera(float x, float y, float z, float yaw, float pitch, float fov);
void rdpq_draw_set_ortho(int on);
void rdpq_draw_look(float yaw, float pitch);
void rdpq_draw_orbit_sph(float az, float el, float rad);
void rdpq_draw_shake(float amp);
void rdpq_draw_set_fade(float a);
void rdpq_draw_overlay(void);
int rdpq_draw_primary_mesh_node(void);
void rdpq_draw_layer_add(int id);
void rdpq_draw_rebind(const unsigned char *mesh, unsigned mesh_sz, const unsigned char *ntex, unsigned ntex_sz);
void rdpq_draw_rebind_embedded(void);
int rdpq_draw_layer_add_mesh(const unsigned char *mesh, unsigned mesh_sz);
void rdpq_draw_clear_layers(void);

#ifdef __cplusplus
}
#endif

#endif
