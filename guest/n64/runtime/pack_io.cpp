// MIT. rom:// packs + user:// EEPROM wrap. 24KiB poke/peek scratch.
// Extra packs stream from DragonFS. Do not write to rom://. EEPROM persist is P5 test 4.

#include "pack_io.h"

#include "dfs_io.h"

#include <libdragon.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#ifndef BLAZIUM_N64_COOK_ABI
#define BLAZIUM_N64_COOK_ABI 1
#endif

#define N64_LAYERS 16
#define N64_EXTRA_RDRAM_CAP (2 * 1024 * 1024)
#define POKE_BYTES (24 * 1024)

static unsigned char s_poke[POKE_BYTES];
static int s_pack;
static int s_cost_rdram;
static char s_user_err[64];
static unsigned char *s_blob;
static unsigned s_blob_sz;
static const char *s_paths[] = {
	"rom://MESH%02d.bin",
	"rom://NTEX%02d.bin",
	"rom://NODE%02d.bin",
	"rom://STREAM.bin",
	"rom://PACK%02d.bin",
	NULL
};

static uint16_t ru16le(const unsigned char *p)
{
	return (uint16_t)p[0] | ((uint16_t)p[1] << 8);
}

static uint32_t ru32le(const unsigned char *p)
{
	return (uint32_t)p[0] | ((uint32_t)p[1] << 8) | ((uint32_t)p[2] << 16) | ((uint32_t)p[3] << 24);
}

static void free_blob(void)
{
	free(s_blob);
	s_blob = NULL;
	s_blob_sz = 0;
	s_cost_rdram = 0;
}

void pack_io_init(void)
{
	s_pack = 0;
	s_cost_rdram = 0;
	s_user_err[0] = 0;
	s_blob = NULL;
	s_blob_sz = 0;
	memset(s_poke, 0, sizeof(s_poke));
	(void)N64_LAYERS;
	(void)pack_io_swap(1);
}

int pack_io_can_fit(int extra_bytes)
{
	if (extra_bytes < 0) {
		return 0;
	}
	return (s_cost_rdram + extra_bytes) <= N64_EXTRA_RDRAM_CAP;
}

int pack_io_swap(int pack)
{
	if (pack <= 0) {
		free_blob();
		s_pack = 0;
		return 0;
	}
	if (pack > 23) {
		return -1;
	}
	char path[64];
	snprintf(path, sizeof(path), "rom://PACK%02d.bin", pack);
	FILE *f = dfs_io_fopen(path);
	if (!f) {
		return -1;
	}
	if (fseek(f, 0, SEEK_END) != 0) {
		fclose(f);
		return -1;
	}
	const long sz = ftell(f);
	if (sz < 12 || sz > N64_EXTRA_RDRAM_CAP) {
		fclose(f);
		return -1;
	}
	if (!pack_io_can_fit((int)sz)) {
		fclose(f);
		return -1;
	}
	if (fseek(f, 0, SEEK_SET) != 0) {
		fclose(f);
		return -1;
	}
	unsigned char *buf = (unsigned char *)malloc((size_t)sz);
	if (!buf) {
		fclose(f);
		return -1;
	}
	const size_t n = fread(buf, 1, (size_t)sz, f);
	fclose(f);
	if (n != (size_t)sz || buf[0] != 'P' || buf[1] != 'A' || buf[2] != 'C' || buf[3] != 'K') {
		free(buf);
		return -1;
	}
	if (buf[0] == 0x4D && buf[1] == 0x5A) {
		free(buf);
		return -1;
	}
	if (ru16le(buf + 4) != (uint16_t)BLAZIUM_N64_COOK_ABI || ru16le(buf + 6) != (uint16_t)pack) {
		free(buf);
		return -1;
	}
	if (ru32le(buf + 8) + 12u > (unsigned)sz) {
		free(buf);
		return -1;
	}
	free_blob();
	s_blob = buf;
	s_blob_sz = (unsigned)sz;
	s_cost_rdram = (int)sz;
	s_pack = pack;
	return 0;
}

int pack_io_prefetch(const char *path)
{
	FILE *f = dfs_io_fopen(path);
	if (!f) {
		return -1;
	}
	unsigned char hdr[12];
	const size_t n = fread(hdr, 1, 12, f);
	fclose(f);
	if (n < 12) {
		return -1;
	}
	return 0;
}

const char *pack_io_find_path(const char *res)
{
	if (res && (strstr(res, "PACK") || strstr(res, "pack") || strstr(res, "STREAM"))) {
		if (strstr(res, "STREAM") || strstr(res, "stream")) {
			return s_paths[3];
		}
		return "rom://PACK01.bin";
	}
	return s_paths[0];
}

static int s_eep_ok;
static const eepfs_entry_t s_eep[] = {
	{ .path = "/user.bin", .size = 256, .checksum = false, .backup = false },
};

static int ensure_eep(void)
{
	if (s_eep_ok) {
		return 0;
	}
	if (eepfs_init(s_eep, 1) != EEPFS_ESUCCESS) {
		snprintf(s_user_err, sizeof(s_user_err), "user:// eeprom");
		return -1;
	}
	s_eep_ok = 1;
	return 0;
}

int pack_io_user_save(const void *data, unsigned sz)
{
	if (!data || sz == 0 || sz > 256) {
		snprintf(s_user_err, sizeof(s_user_err), "user:// size");
		return -1;
	}
	if (ensure_eep() != 0) {
		return -1;
	}
	if (eepfs_write("/user.bin", data, sz) != EEPFS_ESUCCESS) {
		snprintf(s_user_err, sizeof(s_user_err), "user:// eeprom");
		return -1;
	}
	s_user_err[0] = 0;
	return 0;
}

int pack_io_user_load(void *data, unsigned sz)
{
	if (!data || sz == 0) {
		return -1;
	}
	if (ensure_eep() != 0) {
		return -1;
	}
	if (eepfs_read("/user.bin", data, sz) != EEPFS_ESUCCESS) {
		snprintf(s_user_err, sizeof(s_user_err), "user:// eeprom");
		return -1;
	}
	return 0;
}

const char *pack_io_user_error(void)
{
	return s_user_err;
}

void pack_io_poke(unsigned off, unsigned char v)
{
	if (off < POKE_BYTES) {
		s_poke[off] = v;
	}
}

unsigned char pack_io_peek(unsigned off)
{
	if (off < POKE_BYTES) {
		return s_poke[off];
	}
	return 0;
}
