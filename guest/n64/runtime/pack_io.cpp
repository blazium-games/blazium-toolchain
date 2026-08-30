// MIT. rom:// packs + user:// EEPROM wrap. 24KiB poke/peek scratch.

#include "pack_io.h"

#include "dfs_io.h"

#include <libdragon.h>
#include <stdio.h>
#include <string.h>

#define N64_LAYERS 16
#define N64_EXTRA_RDRAM_CAP (2 * 1024 * 1024)
#define POKE_BYTES (24 * 1024)

static unsigned char s_poke[POKE_BYTES];
static int s_pack;
static int s_cost_rdram;
static char s_user_err[64];
static const char *s_paths[] = {
	"rom://MESH%02d.bin",
	"rom://NTEX%02d.bin",
	"rom://NODE%02d.bin",
	"rom://STREAM.bin",
	NULL
};

void pack_io_init(void)
{
	s_pack = 0;
	s_cost_rdram = 0;
	s_user_err[0] = 0;
	memset(s_poke, 0, sizeof(s_poke));
}

int pack_io_swap(int pack)
{
	s_pack = pack;
	char name[64];
	snprintf(name, sizeof(name), "MESH%02d.bin", pack);
	FILE *f = dfs_io_fopen(name);
	if (f) {
		fclose(f);
	}
	snprintf(name, sizeof(name), "MESH01.bin");
	f = dfs_io_fopen(name);
	if (f) {
		fclose(f);
	}
	(void)N64_LAYERS;
	return 0;
}

int pack_io_can_fit(int extra_bytes)
{
	return (s_cost_rdram + extra_bytes) <= N64_EXTRA_RDRAM_CAP;
}

int pack_io_prefetch(const char *path)
{
	FILE *f = dfs_io_fopen(path);
	if (!f) {
		return -1;
	}
	fclose(f);
	return 0;
}

const char *pack_io_find_path(const char *res)
{
	(void)res;
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
