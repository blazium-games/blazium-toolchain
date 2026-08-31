// MIT. libdragon mixer + wav64. Toolchain audioconv64 writes rom:/SFX00.wav64 and MUSIC00.wav64.

#include "sfx_io.h"

#include "dfs_io.h"

#include <libdragon.h>
#include <string.h>

#if defined(__has_include)
#if __has_include("cook_flags.h")
#include "cook_flags.h"
#endif
#endif

static int s_audible;
static float s_vol = 1.0f;
static int s_loaded[16];

#ifdef BLAZIUM_N64_HAS_SFX
static wav64_t s_sfx;
static int s_sfx_ok;
#endif
#ifdef BLAZIUM_N64_HAS_MUSIC
static wav64_t s_music;
static int s_music_ok;
#endif

enum {
	CH_SFX = 0,
	CH_MUSIC = 2
};

void sfx_io_init(void)
{
	audio_init(44100, 4);
	mixer_init(16);
	s_audible = 1;
#ifdef BLAZIUM_N64_HAS_SFX
	if (dfs_io_ready()) {
		wav64_open(&s_sfx, "rom:/SFX00.wav64");
		s_sfx_ok = 1;
		wav64_play(&s_sfx, CH_SFX);
	}
#endif
#ifdef BLAZIUM_N64_HAS_MUSIC
	if (dfs_io_ready()) {
		wav64_open(&s_music, "rom:/MUSIC00.wav64");
		wav64_set_loop(&s_music, true);
		s_music_ok = 1;
		wav64_play(&s_music, CH_MUSIC);
		mixer_ch_set_vol(CH_MUSIC, s_vol, s_vol);
	}
#endif
}

void sfx_io_tick(void)
{
	mixer_try_play();
}

int sfx_io_audible(void)
{
	return s_audible;
}

void sfx_io_play(int id)
{
#ifdef BLAZIUM_N64_HAS_SFX
	if (s_sfx_ok && (id == 0 || id == -1)) {
		wav64_play(&s_sfx, CH_SFX);
	}
#else
	(void)id;
#endif
}

void sfx_io_load(int id)
{
	if (id >= 0 && id < 16) {
		s_loaded[id] = 1;
	}
}

void sfx_io_unload(int id)
{
	if (id >= 0 && id < 16) {
		s_loaded[id] = 0;
	}
}

void sfx_io_music_play(void)
{
#ifdef BLAZIUM_N64_HAS_MUSIC
	if (s_music_ok) {
		wav64_set_loop(&s_music, true);
		wav64_play(&s_music, CH_MUSIC);
		mixer_ch_set_vol(CH_MUSIC, s_vol, s_vol);
	}
#endif
}

void sfx_io_music_stop(void)
{
	mixer_ch_stop(CH_MUSIC);
}

void sfx_io_music_vol(float v)
{
	if (v < 0.0f) {
		v = 0.0f;
	}
	if (v > 1.0f) {
		v = 1.0f;
	}
	s_vol = v;
	mixer_ch_set_vol(CH_MUSIC, s_vol, s_vol);
}
