#ifndef BLAZIUM_N64_SYS_IO_H
#define BLAZIUM_N64_SYS_IO_H

#ifdef __cplusplus
extern "C" {
#endif

void sys_io_init(void);
void sys_io_tick(float delta);
void sys_io_load_pack(int pack);
int sys_io_overlap_refresh(void);
int sys_io_entered_kit(void);
void sys_io_spawn_ofs(float x, float y, float z);
void sys_io_slide(float *x, float *y, float *z, float dx, float dy, float dz);
int sys_io_overlaps(float x, float y, float z);
int sys_io_raycast(float x, float y, float z, float dx, float dy, float dz);
int sys_io_tile_solid_at(int tx, int ty);
void sys_io_set_fade(float a);
void sys_io_scene_fade(int pack, float t);
void sys_io_set_fade_pack(int pack);
void sys_io_set_cam(int id);
void sys_io_set_default_cam(int id);
void sys_io_look(float yaw, float pitch);
void sys_io_next_cam(void);
void sys_io_tween_start(int id, float dur);
void sys_io_timer_start(int id, float dur);
int sys_io_say_done(void);
void sys_io_seek_anim(float t);
void sys_io_move_planar(float dx, float dz);
void sys_io_move_6dof(float dx, float dy, float dz);
void sys_io_hurt(int amount);
void sys_io_path_follow(int path);
int sys_io_kit_player(void);

#ifdef __cplusplus
}
#endif

#endif
