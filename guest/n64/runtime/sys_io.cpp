// MIT. CAM00 / HUD00 / NAV00 / ANIM / HIT / TILE / SPRT / PART / NAVM sidecars.

#include "sys_io.h"

#include "dfs_io.h"
#include "pack_io.h"
#include "rdpq_draw.h"

#include <stdio.h>
#include <string.h>

static int s_paused;
static int s_hud_just_accept;
static int s_hud_kind;
static char s_hud_act[32];
static float s_ak_t;
static float s_fade;
static int s_cam;
static int s_kit_player;
static float s_spawn[3];
static int s_entered_kit;

static void try_open(const char *name)
{
	FILE *f = dfs_io_fopen(name);
	if (f) {
		fclose(f);
	}
}

static void apply_sprt(void)
{
	try_open("SPRT.bin");
}

static void apply_part(void)
{
	try_open("PART.bin");
}

static void apply_navm(void)
{
	try_open("NAVM.bin");
}

void sys_io_init(void)
{
	try_open("CAM00.bin");
	try_open("HUD00.bin");
	try_open("NAV00.bin");
	try_open("HIT00.bin");
	try_open("TILE00.bin");
	try_open("ANIM00.bin");
	apply_sprt();
	apply_part();
	apply_navm();
	s_kit_player = 1;
}

void sys_io_tick(float delta)
{
	(void)delta;
	rdpq_draw_set_camera(0, 0, 0, 0, 0, 55.0f);
}

void sys_io_load_pack(int pack)
{
	char name[32];
	snprintf(name, sizeof(name), "CAM%02d.bin", pack);
	try_open(name);
	snprintf(name, sizeof(name), "HUD%02d.bin", pack);
	try_open(name);
	snprintf(name, sizeof(name), "NAV%02d.bin", pack);
	try_open(name);
	try_open("SPRT.bin");
	try_open("PART.bin");
	try_open("NAVM.bin");
	pack_io_swap(pack);
}

int sys_io_overlap_refresh(void)
{
	return 0;
}

int sys_io_entered_kit(void)
{
	return s_entered_kit;
}

void sys_io_spawn_ofs(float x, float y, float z)
{
	s_spawn[0] = x;
	s_spawn[1] = y;
	s_spawn[2] = z;
	(void)rdpq_draw_primary_mesh_node();
}

void sys_io_slide(float *x, float *y, float *z, float dx, float dy, float dz)
{
	if (x) {
		*x += dx;
	}
	if (y) {
		*y += dy;
	}
	if (z) {
		*z += dz;
	}
}

int sys_io_overlaps(float x, float y, float z)
{
	(void)x;
	(void)y;
	(void)z;
	return 0;
}

int sys_io_raycast(float x, float y, float z, float dx, float dy, float dz)
{
	(void)x;
	(void)y;
	(void)z;
	(void)dx;
	(void)dy;
	(void)dz;
	return 0;
}

int sys_io_tile_solid_at(int tx, int ty)
{
	(void)tx;
	(void)ty;
	return 0;
}

void sys_io_set_fade(float a)
{
	s_fade = a;
	rdpq_draw_set_fade(a);
}

void sys_io_scene_fade(int pack, float t)
{
	(void)t;
	sys_io_set_fade(1.0f);
	sys_io_load_pack(pack);
	sys_io_set_fade(0.0f);
}

void sys_io_set_fade_pack(int pack)
{
	pack_io_swap(pack);
	sys_io_load_pack(pack);
}

void sys_io_set_cam(int id)
{
	s_cam = id;
}

void sys_io_set_default_cam(int id)
{
	s_cam = id;
}

void sys_io_look(float yaw, float pitch)
{
	rdpq_draw_look(yaw, pitch);
}

void sys_io_next_cam(void)
{
	s_cam++;
}

void sys_io_tween_start(int id, float dur)
{
	(void)id;
	(void)dur;
}

void sys_io_timer_start(int id, float dur)
{
	(void)id;
	(void)dur;
}

int sys_io_say_done(void)
{
	(void)s_paused;
	(void)s_hud_just_accept;
	(void)s_hud_kind;
	(void)s_hud_act[0];
	return 1;
}

void sys_io_seek_anim(float t)
{
	s_ak_t = t;
}

void sys_io_move_planar(float dx, float dz)
{
	(void)dx;
	(void)dz;
}

void sys_io_move_6dof(float dx, float dy, float dz)
{
	(void)dx;
	(void)dy;
	(void)dz;
}

void sys_io_hurt(int amount)
{
	(void)amount;
}

void sys_io_path_follow(int path)
{
	(void)path;
}

int sys_io_kit_player(void)
{
	return s_kit_player;
}
