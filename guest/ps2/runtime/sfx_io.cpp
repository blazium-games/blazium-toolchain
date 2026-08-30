// MIT IMA ADPCM of ABI 1 "SFX ". Load host: then cdrom0:.
// Voice: sdrdrv SIF RPC (rSdInit / sce_SDR_DEV) + freesd. Do not include
// libsd.h/libsdr.h or call sceSdInit/sceSdRemote (this -lsdr lacks those).

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
#if __has_include(<iopheap.h>)
#include <iopheap.h>
#define BLAZIUM_PS2_HAS_IOPHEAP 1
#endif
#if __has_include(<kernel.h>)
#include <kernel.h>
#endif
#if __has_include(<libsdr-common.h>) && __has_include(<sifrpc.h>)
extern "C" {
#include <libsd-common.h>
#include <libsdr-common.h>
#include <sifrpc.h>
#if __has_include(<sifdma.h>)
#include <sifdma.h>
#endif
}
#define BLAZIUM_PS2_HAS_LIBSDR 1
#endif
#endif

#ifdef BLAZIUM_PS2_HAS_IRX_FREESD
extern "C" {
extern const unsigned char irx_freesd[];
extern const unsigned int size_irx_freesd;
}
#endif
#ifdef BLAZIUM_PS2_HAS_IRX_SDRDRV
extern "C" {
extern const unsigned char irx_sdrdrv[];
extern const unsigned int size_irx_sdrdrv;
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
static unsigned char *s_vag;
static unsigned s_vag_sz;
static int s_ready;
static int s_played;
static int s_audible;
static int s_sd_ok;
#ifdef BLAZIUM_PS2_HAS_LIBSDR
static SifRpcClientData_t s_sd_cd __attribute__((aligned(64)));
static int s_sbuff[16] __attribute__((aligned(64)));
static int s_sd_bound;
#endif

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

static int encode_vag(const short *pcm, int n)
{
	if (!pcm || n < 1) {
		return 0;
	}
	if (n > 11200) {
		n = 11200;
	}
	const int blocks = (n + 27) / 28;
	s_vag_sz = (unsigned)blocks * 16;
	s_vag = (unsigned char *)malloc(s_vag_sz + 16);
	if (!s_vag) {
		s_vag_sz = 0;
		return 0;
	}
	memset(s_vag, 0, s_vag_sz);
	for (int b = 0; b < blocks; b++) {
		unsigned char *blk = s_vag + b * 16;
		int maxabs = 0;
		short samp[28];
		for (int i = 0; i < 28; i++) {
			const int idx = b * 28 + i;
			const short s = (idx < n) ? pcm[idx] : 0;
			samp[i] = s;
			int a = s < 0 ? -s : s;
			if (a > maxabs) {
				maxabs = a;
			}
		}
		int shift = 12;
		while (shift > 0 && (maxabs >> (12 - shift)) > 7) {
			shift--;
		}
		if (shift < 0) {
			shift = 0;
		}
		blk[0] = (unsigned char)(shift & 0x0f);
		blk[1] = (b == blocks - 1) ? 1 : 0;
		for (int i = 0; i < 14; i++) {
			int n0 = samp[i * 2] >> (12 - shift);
			int n1 = samp[i * 2 + 1] >> (12 - shift);
			if (n0 > 7) {
				n0 = 7;
			}
			if (n0 < -8) {
				n0 = -8;
			}
			if (n1 > 7) {
				n1 = 7;
			}
			if (n1 < -8) {
				n1 = -8;
			}
			blk[2 + i] = (unsigned char)((n0 & 0x0f) | ((n1 & 0x0f) << 4));
		}
	}
	return 1;
}

static void load_sound_irx(void)
{
#ifdef BLAZIUM_PS2_HAS_LOADFILE
#ifdef BLAZIUM_PS2_HAS_IRX_FREESD
	SifExecModuleBuffer((void *)irx_freesd, (int)size_irx_freesd, 0, NULL, NULL);
#endif
#ifdef BLAZIUM_PS2_HAS_IRX_SDRDRV
	SifExecModuleBuffer((void *)irx_sdrdrv, (int)size_irx_sdrdrv, 0, NULL, NULL);
#endif
#endif
}

#ifdef BLAZIUM_PS2_HAS_LIBSDR
static int sdr_bind(void)
{
	if (s_sd_bound) {
		return 1;
	}
	memset(&s_sd_cd, 0, sizeof(s_sd_cd));
	for (;;) {
		if (sceSifBindRpc(&s_sd_cd, sce_SDR_DEV, 0) < 0) {
			return 0;
		}
		if (s_sd_cd.server != NULL) {
			s_sd_bound = 1;
			return 1;
		}
		nopdelay();
	}
}

static int sdr_cmd(int cmd, int a1, int a2, int a3, int a4, int a5)
{
	if (!sdr_bind()) {
		return -1;
	}
	s_sbuff[0] = (int)(unsigned long)s_sbuff;
	s_sbuff[1] = a1;
	s_sbuff[2] = a2;
	s_sbuff[3] = a3;
	s_sbuff[4] = a4;
	s_sbuff[5] = a5;
	if (sceSifCallRpc(&s_sd_cd, cmd, 0, &s_sbuff[0], sizeof(s_sbuff), &s_sbuff[0], 16, NULL, NULL) < 0) {
		return -1;
	}
	return s_sbuff[0];
}

static int sdr_trans_to_iop(void *ee, void *iop, unsigned size)
{
	SifDmaTransfer_t dmat;
	dmat.src = ee;
	dmat.dest = iop;
	dmat.size = size;
	dmat.attr = 0;
	int id = 0;
	while (!(id = sceSifSetDma(&dmat, 1))) {
	}
	while (sceSifDmaStat(id) >= 0) {
	}
	return 0;
}
#endif

static int init_sdr(void)
{
#ifdef BLAZIUM_PS2_HAS_LIBSDR
	if (!sdr_bind()) {
		return 0;
	}
	sdr_cmd(rSdInit, 0, 0, 0, 0, 0);
	sdr_cmd(rSdSetParam, SD_PARAM_MVOLL, 0x3fff, 0, 0, 0);
	sdr_cmd(rSdSetParam, SD_PARAM_MVOLR, 0x3fff, 0, 0, 0);
	sdr_cmd(rSdSetSwitch, SD_SWITCH_VMIXL, 1, 0, 0, 0);
	sdr_cmd(rSdSetSwitch, SD_SWITCH_VMIXR, 1, 0, 0, 0);
	return 1;
#else
	return 0;
#endif
}

int sfx_io_init(void)
{
	s_ready = 0;
	s_played = 0;
	s_audible = 0;
	s_sd_ok = 0;
	s_vag = 0;
	s_vag_sz = 0;
#ifdef BLAZIUM_PS2_HAS_LIBSDR
	s_sd_bound = 0;
#endif
	load_sound_irx();
	s_sd_ok = init_sdr();
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
	if (ok) {
		encode_vag(s_pcm, s_pcm_n);
	}
	s_ready = ok;
	return ok;
}

void sfx_io_play(void)
{
	if (!s_ready || !s_pcm) {
		return;
	}
	s_played = 1;
	s_audible = 0;
#ifdef BLAZIUM_PS2_HAS_LIBSDR
#ifdef BLAZIUM_PS2_HAS_IOPHEAP
	if (!s_sd_ok || !s_vag || s_vag_sz < 16) {
		return;
	}
	void *iop = SifAllocIopHeap((int)s_vag_sz);
	if (!iop) {
		return;
	}
#ifdef BLAZIUM_PS2_HAS_LOADFILE
	FlushCache(0);
#endif
	if (sdr_trans_to_iop(s_vag, iop, s_vag_sz) < 0) {
		SifFreeIopHeap(iop);
		return;
	}
	const int pitch = s_rate > 0 ? (s_rate * 4096) / 44100 : 2048;
	const int voice = SD_VOICE(0, 0);
	sdr_cmd(rSdVoiceTrans, 0, SD_TRANS_WRITE | SD_TRANS_MODE_DMA, (int)(unsigned long)iop, 0x5000, (int)s_vag_sz);
	sdr_cmd(rSdVoiceTransStatus, 0, 1, 0, 0, 0);
	sdr_cmd(rSdSetParam, SD_VPARAM_VOLL | voice, 0x3fff, 0, 0, 0);
	sdr_cmd(rSdSetParam, SD_VPARAM_VOLR | voice, 0x3fff, 0, 0, 0);
	sdr_cmd(rSdSetParam, SD_VPARAM_PITCH | voice, pitch, 0, 0, 0);
	sdr_cmd(rSdSetParam, SD_VPARAM_ADSR1 | voice, SD_SET_ADSR1(0, 0x7f, 0xf, 0xf), 0, 0, 0);
	sdr_cmd(rSdSetParam, SD_VPARAM_ADSR2 | voice, SD_SET_ADSR2(0, 0, 0, 0x10), 0, 0, 0);
	sdr_cmd(rSdSetAddr, SD_VADDR_SSA | voice, 0x5000, 0, 0, 0);
	sdr_cmd(rSdSetSwitch, SD_SWITCH_KON, 1, 0, 0, 0);
	SifFreeIopHeap(iop);
	s_audible = 1;
	return;
#endif
#endif
	(void)s_rate;
	(void)s_pcm_n;
}

void sfx_io_stop(void)
{
	s_played = 0;
#ifdef BLAZIUM_PS2_HAS_LIBSDR
	if (s_sd_ok) {
		sdr_cmd(rSdSetSwitch, SD_SWITCH_KOFF, 1, 0, 0, 0);
	}
#endif
}

int sfx_io_ready(void)
{
	return s_ready;
}

int sfx_io_audible(void)
{
	return s_audible;
}
