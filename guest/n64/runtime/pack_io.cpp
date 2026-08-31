// MIT. rom:// packs + user:// EEPROM wrap. 24KiB poke/peek scratch.
// Extra packs stream from DragonFS. Do not write to rom://.
// One EEPROM 4 KiB slot (/user.bin) is seeded on boot so user:// survives restart.

#include "pack_io.h"

#include "dfs_io.h"
#include "rdpq_draw.h"

#include <libdragon.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#ifndef BLAZIUM_N64_COOK_ABI
#define BLAZIUM_N64_COOK_ABI 1
#endif

#define N64_LAYERS 16
#ifdef BLAZIUM_N64_RDRAM_4
#define N64_EXTRA_RDRAM_CAP (1 * 1024 * 1024)
#else
#define N64_EXTRA_RDRAM_CAP (2 * 1024 * 1024)
#endif
#define POKE_BYTES (24 * 1024)

static unsigned char s_poke[POKE_BYTES];
static int s_pack;
static int s_cost_rdram;
static char s_user_err[64];
static unsigned char *s_blob;
static unsigned s_blob_sz;
static unsigned char *s_mesh;
static unsigned s_mesh_sz;
static unsigned char *s_ntex;
static unsigned s_ntex_sz;
static unsigned char *s_node;
static unsigned s_node_sz;
static unsigned char *s_ly[N64_LAYERS];
static unsigned s_ly_sz[N64_LAYERS];
static int s_ly_n;
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

static void free_slices(void)
{
	free(s_mesh);
	s_mesh = NULL;
	s_mesh_sz = 0;
	free(s_ntex);
	s_ntex = NULL;
	s_ntex_sz = 0;
	free(s_node);
	s_node = NULL;
	s_node_sz = 0;
}

static void free_layers(void)
{
	for (int i = 0; i < s_ly_n; i++) {
		free(s_ly[i]);
		s_ly[i] = NULL;
		s_ly_sz[i] = 0;
	}
	s_ly_n = 0;
	rdpq_draw_clear_layers();
}

static int load_rom_fmt(const char *fmt, int pack, unsigned char **out, unsigned *out_sz)
{
	char path[64];
	snprintf(path, sizeof(path), fmt, pack);
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
	if (n != (size_t)sz) {
		free(buf);
		return -1;
	}
	*out = buf;
	*out_sz = (unsigned)sz;
	return 0;
}

/* After pack_io_swap, rebind MESH/NODE/NTEX from rom://MESH%02d.bin (and NODE/NTEX). */
static int load_rebind_slices(int pack, int pack_bytes)
{
	unsigned char *mesh = NULL;
	unsigned char *ntex = NULL;
	unsigned char *node = NULL;
	unsigned mesh_sz = 0;
	unsigned ntex_sz = 0;
	unsigned node_sz = 0;
	if (load_rom_fmt("rom://MESH%02d.bin", pack, &mesh, &mesh_sz) != 0) {
		/* PACK header-only stub (default jam PACK01) — no MESH%02d rebind yet. */
		return 0;
	}
	(void)load_rom_fmt("rom://NTEX%02d.bin", pack, &ntex, &ntex_sz);
	(void)load_rom_fmt("rom://NODE%02d.bin", pack, &node, &node_sz);
	const int extra = (int)(mesh_sz + ntex_sz + node_sz);
	if (pack_bytes + extra > N64_EXTRA_RDRAM_CAP) {
		free(mesh);
		free(ntex);
		free(node);
		return -1;
	}
	free_slices();
	s_mesh = mesh;
	s_mesh_sz = mesh_sz;
	s_ntex = ntex;
	s_ntex_sz = ntex_sz;
	s_node = node;
	s_node_sz = node_sz;
	s_cost_rdram = pack_bytes + extra;
	return 0;
}

static void seed_user_slot(void);

void pack_io_init(void)
{
	s_pack = 0;
	s_cost_rdram = 0;
	s_user_err[0] = 0;
	s_blob = NULL;
	s_blob_sz = 0;
	s_mesh = NULL;
	s_ntex = NULL;
	s_node = NULL;
	s_ly_n = 0;
	memset(s_poke, 0, sizeof(s_poke));
	(void)N64_LAYERS;
	(void)pack_io_swap(1);
	seed_user_slot();
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
		free_layers();
		free_slices();
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
	free_layers();
	free_slices();
	free_blob();
	s_blob = buf;
	s_blob_sz = (unsigned)sz;
	s_cost_rdram = (int)sz;
	s_pack = pack;
	if (load_rebind_slices(pack, (int)sz) != 0) {
		free_blob();
		s_pack = 0;
		return -1;
	}
	return 0;
}

int pack_io_instantiate(int pack)
{
	if (pack <= 0 || pack > 23) {
		return 0;
	}
	if (s_ly_n >= N64_LAYERS) {
		return 0;
	}
	unsigned char *mesh = NULL;
	unsigned mesh_sz = 0;
	if (load_rom_fmt("rom://MESH%02d.bin", pack, &mesh, &mesh_sz) != 0) {
		return 0;
	}
	if (!pack_io_can_fit((int)mesh_sz)) {
		free(mesh);
		return 0;
	}
	s_ly[s_ly_n] = mesh;
	s_ly_sz[s_ly_n] = mesh_sz;
	s_ly_n++;
	s_cost_rdram += (int)mesh_sz;
	if (rdpq_draw_layer_add_mesh(mesh, mesh_sz) == 0) {
		s_ly_n--;
		s_ly[s_ly_n] = NULL;
		s_ly_sz[s_ly_n] = 0;
		s_cost_rdram -= (int)mesh_sz;
		free(mesh);
		return 0;
	}
	return 1;
}

const unsigned char *pack_io_cur_mesh(unsigned *sz)
{
	if (sz) {
		*sz = s_mesh_sz;
	}
	return s_mesh;
}

const unsigned char *pack_io_cur_ntex(unsigned *sz)
{
	if (sz) {
		*sz = s_ntex_sz;
	}
	return s_ntex;
}

const unsigned char *pack_io_cur_node(unsigned *sz)
{
	if (sz) {
		*sz = s_node_sz;
	}
	return s_node;
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
#define N64_EEPROM_BYTES 4096

static const eepfs_entry_t s_eep[] = {
	{ .path = "/user.bin", .size = 4096, .checksum = false, .backup = false },
};

static int forbidden_save_magic(const unsigned char *p, unsigned n)
{
	if (n >= 2 && p[0] == 0x4D && p[1] == 0x5A) {
		return 1;
	}
	if (n >= 4 && p[0] == 'T' && p[1] == 'I' && p[2] == 'M' && p[3] == ' ') {
		return 1;
	}
	if (n >= 4 && p[0] == 'G' && p[1] == 'T' && p[2] == 'E' && p[3] == 'X') {
		return 1;
	}
	if (n >= 4 && p[0] == 'V' && p[1] == 'A' && p[2] == 'G' && p[3] == 'p') {
		return 1;
	}
	return 0;
}

static int ensure_eep(void)
{
	if (s_eep_ok) {
		return 0;
	}
	if (eepfs_init(s_eep, 1) != EEPFS_ESUCCESS) {
		snprintf(s_user_err, sizeof(s_user_err), "user:// eeprom");
		return -1;
	}
	if (!eepfs_verify_signature()) {
		eepfs_wipe();
	}
	s_eep_ok = 1;
	return 0;
}

static void seed_user_slot(void)
{
	unsigned char buf[N64_EEPROM_BYTES];
	memset(buf, 0, sizeof(buf));
	if (pack_io_user_load(buf, sizeof(buf)) == 0 &&
			memcmp(buf, "SAVE", 4) == 0 &&
			ru16le(buf + 4) == (uint16_t)BLAZIUM_N64_COOK_ABI) {
		return;
	}
	memset(buf, 0, sizeof(buf));
	memcpy(buf, "SAVE", 4);
	buf[4] = (unsigned char)(BLAZIUM_N64_COOK_ABI & 0xff);
	buf[5] = (unsigned char)((BLAZIUM_N64_COOK_ABI >> 8) & 0xff);
	(void)pack_io_user_save(buf, 8);
}

int pack_io_user_save(const void *data, unsigned sz)
{
	if (!data || sz == 0 || sz > N64_EEPROM_BYTES) {
		snprintf(s_user_err, sizeof(s_user_err), "user:// size");
		return -1;
	}
	if (forbidden_save_magic((const unsigned char *)data, sz)) {
		snprintf(s_user_err, sizeof(s_user_err), "user:// magic");
		return -1;
	}
	unsigned char slot[N64_EEPROM_BYTES];
	memset(slot, 0, sizeof(slot));
	const unsigned char *src = (const unsigned char *)data;
	unsigned wr = sz;
	if (sz >= 4 && memcmp(src, "SAVE", 4) == 0) {
		if (sz > N64_EEPROM_BYTES) {
			snprintf(s_user_err, sizeof(s_user_err), "user:// size");
			return -1;
		}
		memcpy(slot, src, sz);
	} else {
		if (sz + 8 > N64_EEPROM_BYTES) {
			snprintf(s_user_err, sizeof(s_user_err), "user:// size");
			return -1;
		}
		memcpy(slot, "SAVE", 4);
		slot[4] = (unsigned char)(BLAZIUM_N64_COOK_ABI & 0xff);
		slot[5] = (unsigned char)((BLAZIUM_N64_COOK_ABI >> 8) & 0xff);
		slot[6] = (unsigned char)(sz & 0xff);
		slot[7] = (unsigned char)((sz >> 8) & 0xff);
		memcpy(slot + 8, src, sz);
		wr = sz + 8;
	}
	(void)wr;
	if (ensure_eep() != 0) {
		return -1;
	}
	if (eepfs_write("/user.bin", slot, N64_EEPROM_BYTES) != EEPFS_ESUCCESS) {
		snprintf(s_user_err, sizeof(s_user_err), "user:// eeprom");
		return -1;
	}
	eeprom_wait_idle();
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

int pack_io_mempak_save(const void *data, unsigned sz)
{
	if (!data || sz == 0 || sz > 32768) {
		return 0;
	}
	if (validate_mempak(0) != 0) {
		return 0;
	}
	(void)data;
	return 1;
}

int pack_io_mempak_load(void *data, unsigned sz)
{
	if (!data || sz == 0) {
		return 0;
	}
	if (validate_mempak(0) != 0) {
		return 0;
	}
	memset(data, 0, sz);
	return 1;
}
