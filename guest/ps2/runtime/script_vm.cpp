// MIT. Interpret cooked SCRP ABI 1 (rotate_y + kit name/id bind). No Godot/Luau VM.

#include "script_vm.h"

#include "gs_draw.h"
#include "pack_io.h"
#include "pad_io.h"
#include "sfx_io.h"
#include "sys_io.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#ifndef BLAZIUM_PS2_COOK_ABI
#define BLAZIUM_PS2_COOK_ABI 1
#endif

enum {
	OP_END = 0,
	OP_LOADK_F = 1,
	OP_LOAD_DELTA = 2,
	OP_MUL = 3,
	OP_CALL_ROTATE_Y = 4,
	OP_READY = 5,
	OP_PROCESS = 6,
	OP_HAS_FEATURE_PS2 = 7,
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
	OP_LOAD_SCENE = 50,
	NAT_EE_USED = 51,
	NAT_EE_FREE = 52,
	NAT_EE_LIMIT = 53,
	NAT_GS_USED = 54,
	NAT_GS_FREE = 55,
	NAT_GS_LIMIT = 56,
	NAT_PACK_COST_EE = 57,
	NAT_PACK_COST_GS = 58,
	NAT_CAN_LOAD = 59,
	NAT_CAN_INSTANTIATE = 60,
	NAT_IS_LOADED = 61,
	NAT_LOADED_COUNT = 62
};

static char s_str[8][128];
static int s_str_n;
static int s_last_str;

static const unsigned char *s_tape;
static unsigned s_tape_n;
static unsigned s_process_off;
static int s_ready;
static int s_kit_spawn;
static int s_kit_player;
static int s_kit_portal;
static int s_kit_checkpoint;
static int s_kit_hazard;
static int s_kit_pickup;
static int s_kit_music;
static int s_kit_music_on;
static int s_kit_unload;
static int s_kit_load;
static int s_kit_talk;
static int s_kit_spawner;
static int s_kit_save;
static float s_yaw;

static unsigned ru16(const unsigned char *p)
{
	return (unsigned)p[0] | ((unsigned)p[1] << 8);
}

static unsigned ru32(const unsigned char *p)
{
	return (unsigned)p[0] | ((unsigned)p[1] << 8) | ((unsigned)p[2] << 16) | ((unsigned)p[3] << 24);
}

static float rf32(const unsigned char *p)
{
	unsigned u = ru32(p);
	float f;
	memcpy(&f, &u, 4);
	return f;
}

static int name_is_ci(const char *n, const char *want)
{
	if (!n || !want) {
		return 0;
	}
	unsigned i = 0;
	while (n[i] && want[i]) {
		char a = n[i];
		char b = want[i];
		if (a >= 'A' && a <= 'Z') {
			a = (char)(a - 'A' + 'a');
		}
		if (b >= 'A' && b <= 'Z') {
			b = (char)(b - 'A' + 'a');
		}
		if (a != b) {
			return 0;
		}
		i++;
	}
	return n[i] == 0 && want[i] == 0;
}

static void apply_kit(unsigned char kit, const char *name)
{
	if (kit == 1 || name_is_ci(name, "Spawn")) {
		s_kit_spawn = 1;
	}
	if (kit == 2 || name_is_ci(name, "Checkpoint")) {
		s_kit_checkpoint = 1;
	}
	if (kit == 3 || name_is_ci(name, "Portal")) {
		s_kit_portal = 1;
	}
	if (kit == 4 || name_is_ci(name, "Player") || name_is_ci(name, "Hero")) {
		s_kit_player = 1;
	}
	if (kit == 5 || name_is_ci(name, "Hazard")) {
		s_kit_hazard = 1;
	}
	if (kit == 6 || name_is_ci(name, "Pickup")) {
		s_kit_pickup = 1;
	}
	if (kit == 7 || name_is_ci(name, "Music") || name_is_ci(name, "MusicZone")) {
		s_kit_music = 1;
	}
	if (kit == 8 || name_is_ci(name, "Unload") || name_is_ci(name, "UnloadVolume")) {
		s_kit_unload = 1;
	}
	if (kit == 9 || name_is_ci(name, "Load") || name_is_ci(name, "LoadZone")) {
		s_kit_load = 1;
	}
	if (kit == 10 || name_is_ci(name, "Talk") || name_is_ci(name, "TalkZone")) {
		s_kit_talk = 1;
	}
	if (kit == 11 || name_is_ci(name, "Spawner") || name_is_ci(name, "Instance")) {
		s_kit_spawner = 1;
	}
	if (kit == 12 || name_is_ci(name, "Save") || name_is_ci(name, "SaveZone")) {
		s_kit_save = 1;
	}
}

static void bind_nodes(const unsigned char *blob, unsigned sz)
{
	if (!blob || sz < 8) {
		return;
	}
	if (blob[0] != 'N' || blob[1] != 'O' || blob[2] != 'D' || blob[3] != 'E') {
		return;
	}
	if (ru16(blob + 4) != BLAZIUM_PS2_COOK_ABI) {
		return;
	}
	const unsigned count = ru16(blob + 6);
	const unsigned rec = 74;
	if (8 + count * rec > sz) {
		return;
	}
	for (unsigned i = 0; i < count; i++) {
		const unsigned char *n = blob + 8 + i * rec;
		char name[33];
		memcpy(name, n + 42, 32);
		name[32] = 0;
		apply_kit(n[5], name);
	}
}

static void try_bind_node_pack(int pack)
{
	char host[40];
	char iso_bs[48];
	char iso[48];
	sprintf(host, "host:NODE%02d.bin", pack);
	sprintf(iso_bs, "cdrom0:\\NODE%02d.BIN;1", pack);
	sprintf(iso, "cdrom0:NODE%02d.BIN;1", pack);
	const char *paths[4];
	paths[0] = host;
	paths[1] = iso_bs;
	paths[2] = iso;
	paths[3] = NULL;
	for (int i = 0; paths[i]; i++) {
		FILE *f = fopen(paths[i], "rb");
		if (!f) {
			continue;
		}
		if (fseek(f, 0, SEEK_END) != 0) {
			fclose(f);
			continue;
		}
		long n = ftell(f);
		if (n < 8) {
			fclose(f);
			continue;
		}
		rewind(f);
		unsigned char *buf = (unsigned char *)malloc((size_t)n);
		if (!buf) {
			fclose(f);
			continue;
		}
		if (fread(buf, 1, (size_t)n, f) == (size_t)n) {
			bind_nodes(buf, (unsigned)n);
		}
		free(buf);
		fclose(f);
		return;
	}
}

static void rebind_current_pack(void)
{
	const unsigned char *node = 0;
	unsigned sz = 0;
	pack_io_current_node(&node, &sz);
	if (node && sz) {
		bind_nodes(node, sz);
	}
}

static void kit_tick(void)
{
	if (pad_io_just_pressed(4)) {
		if (s_kit_portal || s_kit_load) {
			const int cur = pack_io_current();
			const int next = cur + 1;
			if (!pack_io_swap(next)) {
				if (!pack_io_swap(1)) {
					pack_io_swap(0);
				}
			}
			rebind_current_pack();
		}
		if (s_kit_checkpoint || s_kit_save) {
			const unsigned char slot[2] = { 1, 0 };
			if (!pack_io_user_save(slot, 2)) {
				(void)pack_io_user_error();
			}
		}
		if (s_kit_pickup || s_kit_talk) {
			sfx_io_play();
		}
		if (s_kit_spawner) {
			const int next = pack_io_current() + 1;
			(void)pack_io_instantiate(next > 0 ? next : 1);
		}
		if (s_kit_unload) {
			const int cur = pack_io_current();
			if (cur > 0) {
				pack_io_unload(cur);
			} else {
				pack_io_unload(pack_io_max());
			}
		}
	}
	if (s_kit_hazard && pad_io_pressed(4)) {
		pad_io_set_rumble(1, 64);
		sfx_io_play();
	}
	if (s_kit_music && !s_kit_music_on) {
		sfx_io_music_play();
		s_kit_music_on = 1;
	}
}

static void run_range(unsigned from, unsigned to, float delta)
{
	float stack[8];
	int sp = 0;
	unsigned i = from;
	while (i < to && i < s_tape_n) {
		const unsigned char op = s_tape[i++];
		if (op == OP_END) {
			break;
		}
		if (op == OP_READY) {
			continue;
		}
		if (op == OP_PROCESS) {
			break;
		}
		if (op == OP_LOADK_F) {
			if (i + 4 > s_tape_n || sp >= 8) {
				return;
			}
			stack[sp++] = rf32(s_tape + i);
			i += 4;
			continue;
		}
		if (op == OP_LOAD_DELTA) {
			if (sp >= 8) {
				return;
			}
			stack[sp++] = delta;
			continue;
		}
		if (op == OP_MUL) {
			if (sp < 2) {
				return;
			}
			const float b = stack[--sp];
			const float a = stack[--sp];
			stack[sp++] = a * b;
			continue;
		}
		if (op == OP_CALL_ROTATE_Y) {
			if (sp < 1) {
				return;
			}
			s_yaw += stack[--sp];
			gs_draw_set_world_yaw(s_yaw);
			continue;
		}
		if (op == OP_HAS_FEATURE_PS2) {
			if (sp < 8) {
				stack[sp++] = 1.0f;
			}
			continue;
		}
		if (op == OP_SWAP_PACK) {
			if (sp < 1) {
				return;
			}
			pack_io_swap((int)stack[--sp]);
			rebind_current_pack();
			continue;
		}
		if (op == OP_PLAY_SFX) {
			sfx_io_play();
			continue;
		}
		if (op == OP_SET_RUMBLE) {
			if (sp < 2) {
				return;
			}
			const int large = (int)stack[--sp];
			const int small = (int)stack[--sp];
			pad_io_set_rumble(small != 0, large);
			continue;
		}
		if (op == OP_STOP_RUMBLE) {
			pad_io_set_rumble(0, 0);
			continue;
		}
		if (op == OP_INPUT_TRANSLATE) {
			if (sp < 1) {
				return;
			}
			const float rate = stack[--sp];
			float sx = 0.0f;
			float sy = 0.0f;
			pad_io_stick(0, &sx, &sy);
			sys_io_slide(sx * rate, 0.0f, sy * rate, delta);
			continue;
		}
		if (op == OP_JUST_ACCEPT_SFX) {
			if (pad_io_just_pressed(4)) {
				sfx_io_play();
			}
			continue;
		}
		if (op == OP_USER_SAVE) {
			const unsigned char slot[2] = { 1, 0 };
			if (!pack_io_user_save(slot, 2)) {
				(void)pack_io_user_error();
			}
			continue;
		}
		if (op == OP_USER_LOAD) {
			unsigned char slot[8];
			(void)pack_io_user_load(slot, 8);
			continue;
		}
		if (op == OP_MEMCARD_PRESENT) {
			if (sp < 8) {
				stack[sp++] = pack_io_user_present() ? 1.0f : 0.0f;
			}
			continue;
		}
		if (op == OP_KIT_TICK) {
			kit_tick();
			continue;
		}
		if (op == OP_SAY) {
			if (s_last_str >= 0 && s_last_str < 8 && s_str[s_last_str][0]) {
				sys_io_say(s_str[s_last_str]);
			} else {
				sys_io_say("PS2");
			}
			continue;
		}
		if (op == OP_NAV_FOLLOW) {
			sys_io_nav_follow(1.0f, delta);
			continue;
		}
		if (op == OP_SAVE_SLOT) {
			int slot = 0;
			if (sp >= 1) {
				slot = (int)stack[--sp];
			}
			const unsigned char data[2] = { 1, 0 };
			if (!pack_io_user_save_slot(slot, data, 2)) {
				(void)pack_io_user_error();
			}
			continue;
		}
		if (op == OP_PLAY_FMV) {
			(void)sys_io_play_fmv();
			(void)sys_io_fmv_error();
			continue;
		}
		if (op == OP_SYS_TICK) {
			sys_io_tick(delta);
			continue;
		}
		if (op == OP_PLAY_MUSIC) {
			sfx_io_music_play();
			continue;
		}
		if (op == OP_STOP_MUSIC) {
			sfx_io_music_stop();
			continue;
		}
		if (op == OP_MUSIC_VOL) {
			if (sp < 1) {
				return;
			}
			sfx_io_music_set_vol(stack[--sp]);
			continue;
		}
		if (op == OP_LOOK_STICK) {
			sys_io_look_stick(delta);
			continue;
		}
		if (op == OP_LOOK_CAMERA) {
			if (sp < 3) {
				return;
			}
			const float roll = stack[--sp];
			const float pitch = stack[--sp];
			const float yaw = stack[--sp];
			sys_io_look(yaw, pitch, roll);
			continue;
		}
		if (op == OP_ORBIT_CAMERA) {
			if (sp < 3) {
				return;
			}
			const float dist = stack[--sp];
			const float pitch = stack[--sp];
			const float yaw = stack[--sp];
			sys_io_orbit(yaw, pitch, dist);
			continue;
		}
		if (op == OP_SHAKE_CAMERA) {
			if (sp < 2) {
				return;
			}
			const float ms = stack[--sp];
			const float amp = stack[--sp];
			sys_io_shake(amp, ms);
			continue;
		}
		if (op == OP_NEXT_CAMERA) {
			if (pad_io_just_pressed(8)) {
				sys_io_next_cam();
			}
			if (pad_io_just_pressed(9)) {
				sys_io_prev_cam();
			}
			continue;
		}
		if (op == OP_ATTACH_CAMERA) {
			if (sp < 3) {
				return;
			}
			const float oz = stack[--sp];
			const float oy = stack[--sp];
			const float ox = stack[--sp];
			sys_io_attach(ox, oy, oz);
			continue;
		}
		if (op == OP_TWEEN) {
			if (sp < 4) {
				return;
			}
			const float kind = stack[--sp];
			const float sec = stack[--sp];
			const float to = stack[--sp];
			const float from = stack[--sp];
			(void)sys_io_tween_start(from, to, sec, (int)kind);
			continue;
		}
		if (op == OP_TIMER) {
			if (sp < 1) {
				return;
			}
			(void)sys_io_timer_start(stack[--sp]);
			continue;
		}
		if (op == OP_KILL_TWEENS) {
			sys_io_kill_tweens();
			continue;
		}
		if (op == OP_SLIDE) {
			if (sp < 3) {
				return;
			}
			const float vz = stack[--sp];
			const float vy = stack[--sp];
			const float vx = stack[--sp];
			sys_io_slide(vx, vy, vz, delta);
			continue;
		}
		if (op == OP_OVERLAP) {
			sys_io_overlap_refresh();
			if (sp < 8) {
				stack[sp++] = sys_io_overlaps() ? 1.0f : 0.0f;
			}
			continue;
		}
		if (op == OP_RAYCAST) {
			if (sp < 4) {
				return;
			}
			const float dist = stack[--sp];
			const float dz = stack[--sp];
			const float dy = stack[--sp];
			const float dx = stack[--sp];
			float px = 0.0f, py = 0.0f, pz = 0.0f;
			gs_draw_look_point(&px, &py, &pz);
			const int hit = sys_io_raycast(px, py, pz, dx, dy, dz, dist, 255);
			if (sp < 8) {
				stack[sp++] = hit ? 1.0f : 0.0f;
			}
			continue;
		}
		if (op == OP_TILE_AT) {
			if (sp < 2) {
				return;
			}
			const float y = stack[--sp];
			const float x = stack[--sp];
			if (sp < 8) {
				stack[sp++] = (float)sys_io_tile_at(x, y);
			}
			continue;
		}
		if (op == OP_SET_FADE) {
			if (sp < 4) {
				return;
			}
			const float b = stack[--sp];
			const float g = stack[--sp];
			const float r = stack[--sp];
			const float a = stack[--sp];
			sys_io_set_fade(a, r, g, b);
			continue;
		}
		if (op == OP_SCENE_FADE) {
			if (sp < 1) {
				return;
			}
			if (s_last_str >= 0 && s_last_str < 8 && s_str[s_last_str][0]) {
				sys_io_set_fade_pack(pack_io_find_path(s_str[s_last_str]));
			}
			sys_io_scene_fade(stack[--sp]);
			continue;
		}
		if (op == OP_LOADK_S) {
			if (i >= s_tape_n) {
				return;
			}
			const unsigned n = s_tape[i++];
			if (i + n > s_tape_n) {
				return;
			}
			if (s_str_n >= 8) {
				s_str_n = 0;
			}
			unsigned c = 0;
			while (c < n && c < 127) {
				s_str[s_str_n][c] = (char)s_tape[i + c];
				c++;
			}
			s_str[s_str_n][c] = 0;
			i += n;
			s_last_str = s_str_n;
			if (sp < 8) {
				stack[sp++] = (float)s_str_n;
			}
			s_str_n++;
			continue;
		}
		if (op == OP_CHANGE_SCENE) {
			int pack = -1;
			if (s_last_str >= 0 && s_last_str < 8 && s_str[s_last_str][0]) {
				pack = pack_io_find_path(s_str[s_last_str]);
			}
			if (pack < 0 && sp >= 1) {
				pack = (int)stack[--sp];
			}
			if (pack >= 0) {
				pack_io_swap(pack);
				rebind_current_pack();
			}
			continue;
		}
		if (op == OP_INSTANTIATE || op == OP_LOAD_SCENE) {
			int pack = -1;
			if (s_last_str >= 0 && s_last_str < 8 && s_str[s_last_str][0]) {
				pack = pack_io_find_path(s_str[s_last_str]);
			}
			if (pack < 0 && sp >= 1) {
				pack = (int)stack[--sp];
			}
			if (sp < 8) {
				stack[sp++] = pack_io_instantiate(pack) ? 1.0f : 0.0f;
			} else {
				(void)pack_io_instantiate(pack);
			}
			continue;
		}
		if (op == OP_UNLOAD_PACK) {
			int pack = pack_io_current();
			if (s_last_str >= 0 && s_last_str < 8 && s_str[s_last_str][0]) {
				const int found = pack_io_find_path(s_str[s_last_str]);
				if (found >= 0) {
					pack = found;
				}
			}
			pack_io_unload(pack);
			rebind_current_pack();
			continue;
		}
		if (op == OP_SAY_TEXT) {
			if (s_last_str >= 0 && s_last_str < 8) {
				sys_io_set_hud_text(0, s_str[s_last_str]);
			}
			(void)sys_io_say_done();
			continue;
		}
		if (op == OP_SEEK_ANIM) {
			float t = 0.0f;
			if (sp >= 1) {
				t = stack[--sp];
			}
			sys_io_seek_anim(t);
			continue;
		}
		if (op == OP_JMP) {
			if (i + 2 > s_tape_n) {
				return;
			}
			const short off = (short)ru16(s_tape + i);
			i += 2;
			i = (unsigned)((int)i + off);
			continue;
		}
		if (op == OP_JZ) {
			if (i + 2 > s_tape_n || sp < 1) {
				return;
			}
			const short off = (short)ru16(s_tape + i);
			i += 2;
			if (stack[--sp] == 0.0f) {
				i = (unsigned)((int)i + off);
			}
			continue;
		}
		if (op == OP_CALL_NATIVE) {
			if (i >= s_tape_n) {
				return;
			}
			const unsigned char nid = s_tape[i++];
			if (nid == OP_HAS_FEATURE_PS2 && sp < 8) {
				stack[sp++] = 1.0f;
			} else if (nid == OP_OVERLAP) {
				sys_io_overlap_refresh();
				if (sp < 8) {
					stack[sp++] = sys_io_overlaps() ? 1.0f : 0.0f;
				}
			} else if (nid == OP_TILE_AT && sp >= 2) {
				const float y = stack[--sp];
				const float x = stack[--sp];
				stack[sp++] = (float)sys_io_tile_at(x, y);
			} else if (nid == NAT_EE_USED && sp < 8) {
				stack[sp++] = (float)pack_io_ee_used();
			} else if (nid == NAT_EE_FREE && sp < 8) {
				stack[sp++] = (float)pack_io_ee_free();
			} else if (nid == NAT_EE_LIMIT && sp < 8) {
				stack[sp++] = (float)pack_io_ee_limit();
			} else if (nid == NAT_GS_USED && sp < 8) {
				stack[sp++] = (float)pack_io_gs_used();
			} else if (nid == NAT_GS_FREE && sp < 8) {
				stack[sp++] = (float)pack_io_gs_free();
			} else if (nid == NAT_GS_LIMIT && sp < 8) {
				stack[sp++] = (float)pack_io_gs_limit();
			} else if (nid == NAT_PACK_COST_EE && sp < 8) {
				int pack = -1;
				if (s_last_str >= 0 && s_last_str < 8 && s_str[s_last_str][0]) {
					pack = pack_io_find_path(s_str[s_last_str]);
				}
				stack[sp++] = (float)pack_io_cost_ee(pack);
			} else if (nid == NAT_PACK_COST_GS && sp < 8) {
				int pack = -1;
				if (s_last_str >= 0 && s_last_str < 8 && s_str[s_last_str][0]) {
					pack = pack_io_find_path(s_str[s_last_str]);
				}
				stack[sp++] = (float)pack_io_cost_gs(pack);
			} else if ((nid == NAT_CAN_LOAD || nid == NAT_CAN_INSTANTIATE) && sp < 8) {
				int pack = -1;
				if (s_last_str >= 0 && s_last_str < 8 && s_str[s_last_str][0]) {
					pack = pack_io_find_path(s_str[s_last_str]);
				}
				stack[sp++] = (pack >= 0 && pack_io_can_fit(pack)) ? 1.0f : 0.0f;
			} else if (nid == NAT_IS_LOADED && sp < 8) {
				int pack = -1;
				if (s_last_str >= 0 && s_last_str < 8 && s_str[s_last_str][0]) {
					pack = pack_io_find_path(s_str[s_last_str]);
				}
				stack[sp++] = pack_io_is_loaded(pack) ? 1.0f : 0.0f;
			} else if (nid == NAT_LOADED_COUNT && sp < 8) {
				stack[sp++] = (float)pack_io_loaded_count();
			} else if (sp < 8) {
				stack[sp++] = (float)sys_io_get_hp();
			}
			continue;
		}
	}
}

int script_vm_init(const unsigned char *scrp, unsigned scrp_sz,
		const unsigned char *node, unsigned node_sz)
{
	s_ready = 0;
	s_tape = 0;
	s_tape_n = 0;
	s_process_off = 0;
	s_kit_spawn = 0;
	s_kit_player = 0;
	s_kit_portal = 0;
	s_kit_checkpoint = 0;
	s_kit_hazard = 0;
	s_kit_pickup = 0;
	s_kit_music = 0;
	s_kit_music_on = 0;
	s_kit_unload = 0;
	s_kit_load = 0;
	s_kit_talk = 0;
	s_kit_spawner = 0;
	s_kit_save = 0;
	s_yaw = 0.0f;
	s_str_n = 0;
	s_last_str = -1;
	memset(s_str, 0, sizeof(s_str));
	bind_nodes(node, node_sz);
	try_bind_node_pack(1);
	try_bind_node_pack(2);
	if (!scrp || scrp_sz < 12) {
		return 0;
	}
	if (scrp[0] != 'S' || scrp[1] != 'C' || scrp[2] != 'R' || scrp[3] != 'P') {
		return 0;
	}
	if (ru16(scrp + 4) != BLAZIUM_PS2_COOK_ABI) {
		return 0;
	}
	const unsigned tape_n = ru32(scrp + 8);
	if (12 + tape_n > scrp_sz) {
		return 0;
	}
	s_tape = scrp + 12;
	s_tape_n = tape_n;
	s_process_off = 0;
	for (unsigned i = 0; i < s_tape_n; i++) {
		if (s_tape[i] == OP_PROCESS) {
			s_process_off = i + 1;
			break;
		}
		if (s_tape[i] == OP_LOADK_F) {
			i += 4;
		} else if (s_tape[i] == OP_JMP || s_tape[i] == OP_JZ) {
			i += 2;
		} else if (s_tape[i] == OP_CALL_NATIVE) {
			i += 1;
		} else if (s_tape[i] == OP_LOADK_S && i + 1 < s_tape_n) {
			i += 1 + s_tape[i + 1];
		}
	}
	run_range(0, s_tape_n, 0.0f);
	s_ready = 1;
	return 1;
}

void script_vm_process(float delta)
{
	if (!s_ready || !s_tape) {
		return;
	}
	run_range(s_process_off, s_tape_n, delta);
	sfx_io_tick(delta);
}

int script_vm_ready(void)
{
	return s_ready;
}

int script_vm_kit_spawn(void)
{
	return s_kit_spawn;
}

int script_vm_kit_player(void)
{
	return s_kit_player;
}

int script_vm_kit_portal(void)
{
	return s_kit_portal;
}

int script_vm_kit_checkpoint(void)
{
	return s_kit_checkpoint;
}

int script_vm_kit_hazard(void)
{
	return s_kit_hazard;
}

int script_vm_kit_pickup(void)
{
	return s_kit_pickup;
}

int script_vm_kit_load(void)
{
	return s_kit_load;
}

int script_vm_kit_save(void)
{
	return s_kit_save;
}
