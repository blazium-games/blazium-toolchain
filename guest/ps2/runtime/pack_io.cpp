// MIT. Stream extra packs from host: (ELF dir HostFs) then cdrom0:. SAVE.BIN is user:// stub.

#include "pack_io.h"

#include "gs_draw.h"
#include "vu1_draw.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

static int s_toc_ok;
static int s_current;
static int s_has_pack1;
static int s_has_stream;
static int s_max_pack;

static const unsigned char *s_p0_mesh;
static const unsigned char *s_p0_gtex;
static const unsigned char *s_p0_node;
static unsigned s_p0_mesh_sz;
static unsigned s_p0_gtex_sz;
static unsigned s_p0_node_sz;

static unsigned char *s_px_mesh;
static unsigned char *s_px_gtex;
static unsigned char *s_px_node;
static unsigned s_px_mesh_sz;
static unsigned s_px_gtex_sz;
static unsigned s_px_node_sz;
static int s_px_id = -1;
static int s_user_present;
static int s_user_ready;
static char s_user_err[96];

static unsigned ru16(const unsigned char *p)
{
	return (unsigned)p[0] | ((unsigned)p[1] << 8);
}

static int parse_pack(const unsigned char *blob, unsigned sz)
{
	if (!blob || sz < 8) {
		return 0;
	}
	if (blob[0] != 'P' || blob[1] != 'A' || blob[2] != 'C' || blob[3] != 'K') {
		return 0;
	}
	if (ru16(blob + 4) != 1) {
		return 0;
	}
	return ru16(blob + 6) > 0;
}

static int parse_stream(const unsigned char *blob, unsigned sz)
{
	if (!blob || sz < 8) {
		return 0;
	}
	if (blob[0] != 'S' || blob[1] != 'T' || blob[2] != 'R' || blob[3] != 'M') {
		return 0;
	}
	if (ru16(blob + 4) != 1) {
		return 0;
	}
	const unsigned count = ru16(blob + 6);
	if (count == 0 || 8 + count * 2 > sz) {
		return 0;
	}
	unsigned maxp = 0;
	for (unsigned i = 0; i < count; i++) {
		const unsigned id = ru16(blob + 8 + i * 2);
		if (id > maxp) {
			maxp = id;
		}
	}
	s_max_pack = (int)maxp;
	return maxp > 0;
}

static FILE *open_named(const char *host, const char *iso_bs, const char *iso)
{
	const char *paths[4];
	paths[0] = host;
	paths[1] = iso_bs;
	paths[2] = iso;
	paths[3] = 0;
	for (int i = 0; paths[i]; i++) {
		FILE *f = fopen(paths[i], "rb");
		if (f) {
			return f;
		}
	}
	return NULL;
}

static int read_named(const char *host, const char *iso_bs, const char *iso,
		unsigned char **out, unsigned *sz)
{
	FILE *f = open_named(host, iso_bs, iso);
	if (!f) {
		return 0;
	}
	if (fseek(f, 0, SEEK_END) != 0) {
		fclose(f);
		return 0;
	}
	long n = ftell(f);
	if (n < 8) {
		fclose(f);
		return 0;
	}
	rewind(f);
	unsigned char *buf = (unsigned char *)malloc((size_t)n);
	if (!buf) {
		fclose(f);
		return 0;
	}
	if (fread(buf, 1, (size_t)n, f) != (size_t)n) {
		free(buf);
		fclose(f);
		return 0;
	}
	fclose(f);
	*out = buf;
	*sz = (unsigned)n;
	return 1;
}

static void set_user_err(const char *msg)
{
	unsigned i = 0;
	if (!msg) {
		s_user_err[0] = 0;
		return;
	}
	while (msg[i] && i + 1 < sizeof(s_user_err)) {
		s_user_err[i] = msg[i];
		i++;
	}
	s_user_err[i] = 0;
}

static FILE *open_save(const char *mode)
{
	FILE *f = fopen("host:SAVE.BIN", mode);
	if (f) {
		return f;
	}
	return fopen("mc0:SAVE.BIN", mode);
}

static void write_save_stub(void)
{
	set_user_err("");
	FILE *f = open_save("wb");
	if (!f) {
		s_user_present = 0;
		s_user_ready = 0;
		set_user_err("user:// save failed: no host:SAVE.BIN or mc0: (HostFs/memcard missing)");
		return;
	}
	const char mag[] = { 'S', 'A', 'V', 'E', 1, 0 };
	fwrite(mag, 1, sizeof(mag), f);
	fclose(f);
	s_user_present = 1;
	s_user_ready = 1;
}

static void free_px(void)
{
	free(s_px_mesh);
	free(s_px_gtex);
	free(s_px_node);
	s_px_mesh = 0;
	s_px_gtex = 0;
	s_px_node = 0;
	s_px_mesh_sz = 0;
	s_px_gtex_sz = 0;
	s_px_node_sz = 0;
	s_px_id = -1;
}

static int load_pack_n(int pack)
{
	if (pack < 1) {
		return 0;
	}
	if (s_px_id == pack && s_px_mesh && s_px_gtex && s_px_node) {
		return 1;
	}
	free_px();
	char host_m[40];
	char iso_m[48];
	char iso_m2[48];
	char host_g[40];
	char iso_g[48];
	char iso_g2[48];
	char host_n[40];
	char iso_n[48];
	char iso_n2[48];
	sprintf(host_m, "host:MESH%02d.bin", pack);
	sprintf(iso_m, "cdrom0:\\MESH%02d.BIN;1", pack);
	sprintf(iso_m2, "cdrom0:MESH%02d.BIN;1", pack);
	sprintf(host_g, "host:GTEX%02d.bin", pack);
	sprintf(iso_g, "cdrom0:\\GTEX%02d.BIN;1", pack);
	sprintf(iso_g2, "cdrom0:GTEX%02d.BIN;1", pack);
	sprintf(host_n, "host:NODE%02d.bin", pack);
	sprintf(iso_n, "cdrom0:\\NODE%02d.BIN;1", pack);
	sprintf(iso_n2, "cdrom0:NODE%02d.BIN;1", pack);
	unsigned char *mesh = 0;
	unsigned char *gtex = 0;
	unsigned char *node = 0;
	unsigned mesh_sz = 0;
	unsigned gtex_sz = 0;
	unsigned node_sz = 0;
	if (!read_named(host_m, iso_m, iso_m2, &mesh, &mesh_sz)) {
		return 0;
	}
	if (!read_named(host_g, iso_g, iso_g2, &gtex, &gtex_sz)) {
		free(mesh);
		return 0;
	}
	if (!read_named(host_n, iso_n, iso_n2, &node, &node_sz)) {
		free(mesh);
		free(gtex);
		return 0;
	}
	s_px_mesh = mesh;
	s_px_gtex = gtex;
	s_px_node = node;
	s_px_mesh_sz = mesh_sz;
	s_px_gtex_sz = gtex_sz;
	s_px_node_sz = node_sz;
	s_px_id = pack;
	if (pack == 1) {
		s_has_pack1 = 1;
	}
	if (pack > s_max_pack) {
		s_max_pack = pack;
	}
	return 1;
}

void pack_io_set_pack0(const unsigned char *mesh, unsigned mesh_sz,
		const unsigned char *gtex, unsigned gtex_sz,
		const unsigned char *node, unsigned node_sz)
{
	s_p0_mesh = mesh;
	s_p0_mesh_sz = mesh_sz;
	s_p0_gtex = gtex;
	s_p0_gtex_sz = gtex_sz;
	s_p0_node = node;
	s_p0_node_sz = node_sz;
	s_current = 0;
}

int pack_io_swap(int pack)
{
	if (pack <= 0) {
		if (!s_p0_mesh) {
			return 0;
		}
		if (!gs_draw_init(s_p0_mesh, s_p0_mesh_sz, s_p0_gtex, s_p0_gtex_sz, s_p0_node, s_p0_node_sz)) {
			return 0;
		}
		vu1_draw_init(s_p0_mesh, s_p0_mesh_sz);
		s_current = 0;
		return 1;
	}
	if (!load_pack_n(pack)) {
		return 0;
	}
	if (!gs_draw_init(s_px_mesh, s_px_mesh_sz, s_px_gtex, s_px_gtex_sz, s_px_node, s_px_node_sz)) {
		return 0;
	}
	vu1_draw_init(s_px_mesh, s_px_mesh_sz);
	s_current = pack;
	return 1;
}

int pack_io_init(void)
{
	s_toc_ok = 0;
	s_has_stream = 0;
	s_max_pack = 0;
	unsigned char *buf = 0;
	unsigned sz = 0;
	if (read_named("host:STREAM.bin", "cdrom0:\\STREAM.BIN;1", "cdrom0:STREAM.BIN;1", &buf, &sz)) {
		s_has_stream = parse_stream(buf, sz);
		s_toc_ok = s_has_stream;
		free(buf);
		buf = 0;
	}
	if (read_named("host:PACK01.bin", "cdrom0:\\PACK01.BIN;1", "cdrom0:PACK01.BIN;1", &buf, &sz)) {
		if (parse_pack(buf, sz)) {
			s_toc_ok = 1;
			if (s_max_pack < 1) {
				s_max_pack = 1;
			}
		}
		free(buf);
	}
	if (s_toc_ok) {
		load_pack_n(1);
	}
	write_save_stub();
	return s_toc_ok;
}

int pack_io_loaded(void)
{
	return s_toc_ok;
}

int pack_io_current(void)
{
	return s_current;
}

int pack_io_has_pack1(void)
{
	return s_has_pack1;
}

int pack_io_max(void)
{
	return s_max_pack;
}

int pack_io_has_stream(void)
{
	return s_has_stream;
}

void pack_io_current_node(const unsigned char **out, unsigned *sz)
{
	if (!out || !sz) {
		return;
	}
	if (s_current <= 0) {
		*out = s_p0_node;
		*sz = s_p0_node_sz;
		return;
	}
	*out = s_px_node;
	*sz = s_px_node_sz;
}

int pack_io_user_present(void)
{
	return s_user_present;
}

int pack_io_user_ready(void)
{
	return s_user_ready;
}

int pack_io_user_save(const void *data, unsigned n)
{
	FILE *f = open_save("wb");
	if (!f) {
		s_user_present = 0;
		s_user_ready = 0;
		set_user_err("user:// save failed: no host:SAVE.BIN or mc0: (HostFs/memcard missing)");
		return 0;
	}
	const char mag[] = { 'S', 'A', 'V', 'E', 1, 0 };
	fwrite(mag, 1, sizeof(mag), f);
	if (data && n) {
		fwrite(data, 1, n, f);
	}
	fclose(f);
	s_user_present = 1;
	s_user_ready = 1;
	set_user_err("");
	return 1;
}

int pack_io_user_load(void *data, unsigned maxn)
{
	FILE *f = open_save("rb");
	if (!f) {
		set_user_err("user:// load failed: SAVE.BIN not found on host: or mc0:");
		return -1;
	}
	unsigned char mag[6];
	if (fread(mag, 1, 6, f) != 6 || mag[0] != 'S' || mag[1] != 'A' || mag[2] != 'V' || mag[3] != 'E') {
		fclose(f);
		set_user_err("user:// load failed: SAVE.BIN is not a PS2 ABI 1 save");
		return -1;
	}
	int n = 0;
	if (data && maxn) {
		n = (int)fread(data, 1, maxn, f);
	}
	fclose(f);
	s_user_present = 1;
	s_user_ready = 1;
	set_user_err("");
	return n;
}

const char *pack_io_user_error(void)
{
	return s_user_err;
}
