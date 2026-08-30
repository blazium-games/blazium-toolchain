// MIT. Load CAM/HIT/HUD/TILE/NAV/ANIM/FMV ABI 1 sidecars from host: then cdrom0:.

#ifndef BLAZIUM_PS2_SYS_IO_H
#define BLAZIUM_PS2_SYS_IO_H

#ifdef __cplusplus
extern "C" {
#endif

int sys_io_init(void);
int sys_io_loaded(void);
int sys_io_cam_count(void);
int sys_io_hit_count(void);
int sys_io_hud_count(void);
int sys_io_tile_count(void);
int sys_io_nav_count(void);
int sys_io_anim_count(void);
int sys_io_fmv_count(void);
const char *sys_io_fmv_error(void);

void sys_io_say(const char *line);
void sys_io_nav_follow(float speed, float delta);
int sys_io_nav_next(int from, int to);
int sys_io_set_hp(int hp);
int sys_io_get_hp(void);
void sys_io_set_frame(int i);
int sys_io_play_fmv(void);
void sys_io_tick(float delta);
int sys_io_hit_hazard(float x, float z);
void sys_io_look(float yaw, float pitch, float roll);
void sys_io_look_stick(float delta);
void sys_io_orbit(float yaw, float pitch, float dist);
void sys_io_attach(float ox, float oy, float oz);
void sys_io_shake(float amp, float ms);
void sys_io_next_cam(void);
void sys_io_prev_cam(void);
int sys_io_cam_index(void);

#ifdef __cplusplus
}
#endif

#endif
