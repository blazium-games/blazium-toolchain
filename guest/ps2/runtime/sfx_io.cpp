// MIT IMA ADPCM of ABI 1 "SFX ". Load host: then cdrom0:. freesd IRX; no audsrv.

#include "sfx_io.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#if defined(__has_include)
#if __has_include("irx_flags.h")
#include "irx_flags.h"
#endif
#if __has_include(<loadfile.h>)
#include <loadfile.h>
#define BLAZIUM_PS2_HAS_LOADFILE 1
#endif
#endif

#ifdef BLAZIUM_PS2_HAS_LIBSD
/* Do not call sceSdInit: this newlib -lsdr does not export it. */
#endif

#ifdef BLAZIUM_PS2_HAS_IRX_FREESD
extern "C" {
extern const unsigned char irx_freesd[];
extern const unsigned int size_irx_freesd;
}
#endif

static const int s_ima_step[89] = {
	7, 8, 9, 10, 11, 12, 13, 14, 16, 17,
	19, 21, 23, 25, 28, 31, 34, 37, 41, 45,
	50, 55, 60, 66, 73, 80, 88, 97, 107, 118,
	130, 143, 157, 173, 190, 209, 230, 253, 279, 307,
	337, 371, 408, 449, 494, 544, 598, 658, 724, 796,
	876, 963, 1060, 1166, 1282, 1411, 1552, 1707, 1878, 2066,
	2272, 2499, 2749, 3024, 3327, 3660, 4026, 4428, 4871, 5358,
	5894, 6484, 7132, 7845, 8630, 9493, 10442, 11487, 12635, 13899,
	15289, 16818, 18500, 20350, 22385, 24623, 27086, 29794, 32767
};

static const int s_ima_index[16] = {
	-1, -1, -1, -1, 2, 4, 6, 8,
	-1, -1, -1, -1, 2, 4, 6, 8
};

static short *s_pcm;
static int s_pcm_n;
static int s_rate;
static int s_ready;
static int s_played;
static int s_audible;
static int s_sd_ok;

static unsigned ru16(const unsigned char *p)
{
	return (unsigned)p[0] | ((unsigned)p[1] << 8);
}

static unsigned ru32(const unsigned char *p)
{
	return (unsigned)p[0] | ((unsigned)p[1] << 8) | ((unsigned)p[2] << 16) | ((unsigned)p[3] << 24);
}

static int load_blob(unsigned char **out, unsigned *sz)
{
	static const char *paths[] = {
		"host:SFX00.bin",
		"cdrom0:\\SFX00.BIN;1",
		"cdrom0:SFX00.BIN;1",
		NULL
	};
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
		if (n < 12) {
			fclose(f);
			continue;
		}
		rewind(f);
		unsigned char *buf = (unsigned char *)malloc((size_t)n);
		if (!buf) {
			fclose(f);
			continue;
		}
		if (fread(buf, 1, (size_t)n, f) != (size_t)n) {
			free(buf);
			fclose(f);
			continue;
		}
		fclose(f);
		*out = buf;
		*sz = (unsigned)n;
		return 1;
	}
	return 0;
}

static int decode_ima(const unsigned char *nibbles, unsigned nbytes)
{
	int pred = 0;
	int idx = 0;
	const unsigned samples = nbytes * 2;
	s_pcm = (short *)malloc(samples * sizeof(short));
	if (!s_pcm) {
		return 0;
	}
	s_pcm_n = (int)samples;
	unsigned o = 0;
	for (unsigned i = 0; i < nbytes; i++) {
		for (int half = 0; half < 2; half++) {
			const unsigned nibble = (half == 0) ? (nibbles[i] & 0x0f) : (nibbles[i] >> 4);
			const int step = s_ima_step[idx];
			int delta = step >> 3;
			if (nibble & 4) {
				delta += step;
			}
			if (nibble & 2) {
				delta += step >> 1;
			}
			if (nibble & 1) {
				delta += step >> 2;
			}
			if (nibble & 8) {
				pred -= delta;
			} else {
				pred += delta;
			}
			if (pred > 32767) {
				pred = 32767;
			}
			if (pred < -32768) {
				pred = -32768;
			}
			idx += s_ima_index[nibble];
			if (idx < 0) {
				idx = 0;
			}
			if (idx > 88) {
				idx = 88;
			}
			s_pcm[o++] = (short)pred;
		}
	}
	return 1;
}

int sfx_io_init(void)
{
	s_ready = 0;
	s_played = 0;
	s_audible = 0;
	s_sd_ok = 0;
#ifdef BLAZIUM_PS2_HAS_IRX_FREESD
#ifdef BLAZIUM_PS2_HAS_LOADFILE
	SifExecModuleBuffer((void *)irx_freesd, (int)size_irx_freesd, 0, NULL, NULL);
#endif
#endif
	unsigned char *blob = 0;
	unsigned sz = 0;
	if (!load_blob(&blob, &sz)) {
		return 0;
	}
	if (sz < 12 || blob[0] != 'S' || blob[1] != 'F' || blob[2] != 'X' || blob[3] != ' ') {
		free(blob);
		return 0;
	}
	if (ru16(blob + 4) != 1) {
		free(blob);
		return 0;
	}
	s_rate = (int)ru16(blob + 6);
	const unsigned nbytes = ru32(blob + 8);
	if (12 + nbytes > sz) {
		free(blob);
		return 0;
	}
	const int ok = decode_ima(blob + 12, nbytes);
	free(blob);
	s_ready = ok;
	return ok;
}

void sfx_io_play(void)
{
	if (!s_ready || !s_pcm) {
		return;
	}
	s_played = 1;
#ifdef BLAZIUM_PS2_HAS_LIBSD
	/* PCM is decoded on the EE. Voice DMA is not linked (sceSdInit missing
	   from -lsdr). Cook WARNINGS.txt tells the developer play may be silent. */
	s_sd_ok = 1;
	s_audible = 0;
	(void)s_rate;
	(void)s_pcm_n;
#else
	(void)s_rate;
	(void)s_pcm_n;
#endif
}

void sfx_io_stop(void)
{
	s_played = 0;
}

int sfx_io_ready(void)
{
	return s_ready;
}

int sfx_io_audible(void)
{
	return s_audible;
}
