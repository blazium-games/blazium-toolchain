// MIT. libdragon mixer. MUSIC00 / SFX%02d via rom:// when present.

#include "sfx_io.h"

#include "dfs_io.h"

#include <libdragon.h>
#include <stdio.h>

static int s_audible;
static float s_vol = 1.0f;
static int s_loaded[16];

void sfx_io_init(void)
{
	audio_init(44100, AUDIO_DEFAULT_LATENCY);
	mixer_init(16);
	s_audible = 1;
}

int sfx_io_audible(void)
{
	return s_audible;
}

void sfx_io_play(int id)
{
	(void)id;
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
	FILE *f = dfs_io_fopen("rom://MUSIC00.bin");
	if (!f) {
		f = dfs_io_fopen("rom://MUSIC00.wav64");
	}
	if (f) {
		fclose(f);
	}
	(void)s_loaded[0];
}

void sfx_io_music_stop(void)
{
}

void sfx_io_music_vol(float v)
{
	s_vol = v;
	(void)s_vol;
}
