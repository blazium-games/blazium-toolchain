// MIT. Interpret cooked SCRP ABI 1 (rotate_y + kit name/id bind). No Godot/Luau VM.

#include "script_vm.h"

#include "pack_io.h"
#include "pad_io.h"
#include "rdpq_draw.h"
#include "sfx_io.h"
#include "sys_io.h"

#include <stdio.h>
#include <string.h>

#ifndef BLAZIUM_N64_COOK_ABI
#define BLAZIUM_N64_COOK_ABI 1
#endif

enum {
	OP_END = 0,
	OP_LOADK_F = 1,
	OP_LOAD_DELTA = 2,
	OP_MUL = 3,
	OP_CALL_ROTATE_Y = 4,
	OP_READY = 5,
	OP_PROCESS = 6,
	OP_HAS_FEATURE_N64 = 7,
	OP_SWAP_PACK = 8,
	OP_PLAY_SFX = 9,
	OP_SET_RUMBLE = 10,
	OP_STOP_RUMBLE = 11,
	OP_INPUT_TRANSLATE = 12,
	OP_JUST_ACCEPT_SFX = 13,
	OP_USER_SAVE = 14,
	OP_USER_LOAD = 15,
	OP_MEMCARD_PRESENT = 16,
	OP_KIT_TICK = 17,
	OP_SAY = 18,
	OP_NAV_FOLLOW = 19,
	OP_SAVE_SLOT = 20,
	OP_PLAY_FMV = 21,
	OP_SYS_TICK = 22,
	OP_PLAY_MUSIC = 23,
	OP_STOP_MUSIC = 24,
	OP_MUSIC_VOL = 25,
	OP_LOOK_STICK = 26,
	OP_LOOK_CAMERA = 27,
	OP_ORBIT_CAMERA = 28,
	OP_SHAKE_CAMERA = 29,
	OP_NEXT_CAMERA = 30,
	OP_ATTACH_CAMERA = 31,
	OP_TWEEN = 32,
	OP_TIMER = 33,
	OP_KILL_TWEENS = 34,
	OP_SLIDE = 35,
	OP_OVERLAP = 36,
	OP_RAYCAST = 37,
	OP_TILE_AT = 38,
	OP_SET_FADE = 39,
	OP_SCENE_FADE = 40,
	OP_CHANGE_SCENE = 41,
	OP_INSTANTIATE = 42,
	OP_UNLOAD_PACK = 43,
	OP_SAY_TEXT = 44,
	OP_SEEK_ANIM = 45,
	OP_JMP = 46,
	OP_JZ = 47,
	OP_CALL_NATIVE = 48,
	OP_LOADK_S = 49,
	OP_LOAD_SCENE = 50
};

enum {
	NAT_RDRAM_USED = 51,
	NAT_RDRAM_FREE = 52,
	NAT_RDRAM_LIMIT = 53,
	NAT_TMEM_USED = 54,
	NAT_TMEM_FREE = 55,
	NAT_TMEM_LIMIT = 56,
	NAT_PACK_COST_RDRAM = 57,
	NAT_CAN_LOAD = 59,
	NAT_CAN_INSTANTIATE = 60,
	NAT_SET_CHECKPOINT = 61,
	NAT_RESPAWN = 62,
	NAT_SET_CAM = 63,
	NAT_LOAD_SPRITES = 64,
	NAT_MOVE_PLANAR = 65,
	NAT_HURT = 66,
	NAT_POKE = 67,
	NAT_GET_PRESSURE = 68,
	NAT_MOVE_6DOF = 69,
	NAT_LOAD_PARTICLES = 70,
	NAT_PATH_FOLLOW = 71
};

static const unsigned char *s_code;
static unsigned s_len;
static int s_ready;
static int s_kit_player = 1;
static int s_kit_follow;
static float s_grav_vy;
static float s_px, s_py, s_pz;
static float s_ck_x, s_ck_y, s_ck_z;
static float s_stack[8];
static int s_sp;

static float popf(void)
{
	if (s_sp <= 0) {
		return 0.0f;
	}
	return s_stack[--s_sp];
}

static void pushf(float v)
{
	if (s_sp < 8) {
		s_stack[s_sp++] = v;
	}
}

static void do_checkpoint(void)
{
	/* Kit name Checkpoint — same bind as PS1/PS2 scene kit. */
	s_ck_x = s_px;
	s_ck_y = s_py;
	s_ck_z = s_pz;
}

static void do_native(int nat)
{
	switch (nat) {
	case NAT_SET_CHECKPOINT:
		do_checkpoint();
		break;
	case NAT_RESPAWN:
		s_px = s_ck_x;
		s_py = s_ck_y;
		s_pz = s_ck_z;
		sys_io_spawn_ofs(s_px, s_py, s_pz);
		break;
	case NAT_SET_CAM:
		sys_io_set_cam((int)popf());
		break;
	case NAT_LOAD_SPRITES:
		break;
	case NAT_MOVE_PLANAR:
		sys_io_move_planar(popf(), popf());
		break;
	case NAT_HURT:
		sys_io_hurt((int)popf());
		break;
	case NAT_POKE:
		pack_io_poke((unsigned)popf(), (unsigned char)popf());
		break;
	case NAT_GET_PRESSURE:
		pushf((float)pack_io_peek(0));
		break;
	case NAT_MOVE_6DOF:
		sys_io_move_6dof(popf(), popf(), popf());
		break;
	case NAT_LOAD_PARTICLES:
		break;
	case NAT_PATH_FOLLOW:
		sys_io_path_follow((int)popf());
		break;
	case NAT_CAN_INSTANTIATE:
		pushf((float)pack_io_can_fit(64));
		break;
	default:
		break;
	}
}

int script_vm_init(const unsigned char *scrp, unsigned scrp_sz,
		const unsigned char *node, unsigned node_sz)
{
	s_code = scrp;
	s_len = scrp_sz;
	s_ready = 0;
	s_sp = 0;
	(void)node;
	(void)node_sz;
	(void)pack_io_find_path("res://STREAM");
	return 0;
}

void script_vm_process(float delta)
{
	if (!s_code || s_len == 0) {
		sys_io_tick(delta);
		return;
	}
	unsigned i = 0;
	while (i < s_len) {
		unsigned op = s_code[i++];
		switch (op) {
		case OP_END:
			return;
		case OP_LOADK_F:
			if (i + 4 <= s_len) {
				float v;
				memcpy(&v, s_code + i, 4);
				i += 4;
				pushf(v);
			}
			break;
		case OP_LOAD_DELTA:
			pushf(delta);
			break;
		case OP_MUL:
			pushf(popf() * popf());
			break;
		case OP_CALL_ROTATE_Y:
			rdpq_draw_look(popf(), 0.0f);
			break;
		case OP_READY:
			s_ready = 1;
			break;
		case OP_PROCESS:
			break;
		case OP_HAS_FEATURE_N64:
			pushf(1.0f);
			break;
		case OP_SWAP_PACK:
			pack_io_swap((int)popf());
			break;
		case OP_PLAY_SFX:
			sfx_io_play((int)popf());
			break;
		case OP_SET_RUMBLE:
			pad_io_set_rumble(1);
			break;
		case OP_STOP_RUMBLE:
			pad_io_stop_rumble();
			break;
		case OP_INPUT_TRANSLATE: {
			float sx, sy;
			pad_io_stick(&sx, &sy);
			s_px += sx * delta * 60.0f;
			s_pz += sy * delta * 60.0f;
			if (s_kit_player) {
				sys_io_slide(&s_px, &s_py, &s_pz, sx, s_grav_vy, sy);
			}
			break;
		}
		case OP_JUST_ACCEPT_SFX:
			if (pad_io_pressed(0)) {
				sfx_io_play(0);
			}
			break;
		case OP_USER_SAVE:
			pack_io_user_save(&s_px, sizeof(s_px));
			break;
		case OP_USER_LOAD:
			pack_io_user_load(&s_px, sizeof(s_px));
			break;
		case OP_KIT_TICK:
			if (sys_io_entered_kit() || sys_io_overlap_refresh()) {
				do_checkpoint();
			}
			if (s_kit_follow) {
				sys_io_path_follow(0);
			}
			break;
		case OP_SAY:
		case OP_SAY_TEXT:
			(void)sys_io_say_done();
			break;
		case OP_PLAY_FMV:
			break;
		case OP_SYS_TICK:
			sys_io_tick(delta);
			break;
		case OP_PLAY_MUSIC:
			sfx_io_music_play();
			break;
		case OP_STOP_MUSIC:
			sfx_io_music_stop();
			break;
		case OP_MUSIC_VOL:
			sfx_io_music_vol(popf());
			break;
		case OP_LOOK_STICK:
		case OP_LOOK_CAMERA:
			sys_io_look(popf(), popf());
			break;
		case OP_ORBIT_CAMERA:
			rdpq_draw_orbit_sph(popf(), popf(), popf());
			break;
		case OP_SHAKE_CAMERA:
			rdpq_draw_shake(popf());
			break;
		case OP_NEXT_CAMERA:
			sys_io_next_cam();
			break;
		case OP_TWEEN:
			sys_io_tween_start((int)popf(), popf());
			break;
		case OP_TIMER:
			sys_io_timer_start((int)popf(), popf());
			break;
		case OP_SLIDE:
			sys_io_slide(&s_px, &s_py, &s_pz, popf(), popf(), popf());
			break;
		case OP_OVERLAP:
			pushf((float)sys_io_overlaps(s_px, s_py, s_pz));
			break;
		case OP_RAYCAST:
			pushf((float)sys_io_raycast(s_px, s_py, s_pz, 0, 0, 1));
			break;
		case OP_TILE_AT:
			pushf((float)sys_io_tile_solid_at((int)s_px, (int)s_pz));
			break;
		case OP_SET_FADE:
			sys_io_set_fade(popf());
			break;
		case OP_SCENE_FADE:
			sys_io_scene_fade((int)popf(), popf());
			break;
		case OP_CHANGE_SCENE:
		case OP_LOAD_SCENE:
			sys_io_load_pack((int)popf());
			break;
		case OP_INSTANTIATE:
			(void)pack_io_can_fit(128);
			break;
		case OP_SEEK_ANIM:
			sys_io_seek_anim(popf());
			break;
		case OP_JMP:
			if (i < s_len) {
				i = s_code[i];
			}
			break;
		case OP_JZ:
			if (popf() == 0.0f && i < s_len) {
				i = s_code[i];
			} else if (i < s_len) {
				i++;
			}
			break;
		case OP_CALL_NATIVE:
			if (i < s_len) {
				do_native(s_code[i++]);
			}
			break;
		default:
			break;
		}
	}
}

int script_vm_ready(void)
{
	return s_ready;
}

int script_vm_kit_spawn(void)
{
	return 1;
}

int script_vm_kit_player(void)
{
	return s_kit_player;
}

int script_vm_kit_portal(void)
{
	return 1;
}

int script_vm_kit_checkpoint(void)
{
	return 1;
}

int script_vm_kit_hazard(void)
{
	return 1;
}

int script_vm_kit_pickup(void)
{
	return 1;
}

int script_vm_kit_load(void)
{
	return 1;
}

int script_vm_kit_save(void)
{
	return 1;
}
