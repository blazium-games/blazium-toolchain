// MIT. Stream pack 1 from host: (ELF dir HostFs) then cdrom0:. SAVE.BIN is user:// stub.

#include "pack_io.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

static int s_loaded;

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

static FILE *open_pack(void)
{
	static const char *paths[] = {
		"host:PACK01.bin",
		"cdrom0:\\PACK01.BIN;1",
		"cdrom0:PACK01.BIN;1",
		NULL
	};
	for (int i = 0; paths[i]; i++) {
		FILE *f = fopen(paths[i], "rb");
		if (f) {
			return f;
		}
	}
	return NULL;
}

static void write_save_stub(void)
{
	FILE *f = fopen("host:SAVE.BIN", "wb");
	if (!f) {
		f = fopen("mc0:SAVE.BIN", "wb");
	}
	if (!f) {
		return;
	}
	const char mag[] = { 'S', 'A', 'V', 'E', 1, 0 };
	fwrite(mag, 1, sizeof(mag), f);
	fclose(f);
}

int pack_io_init(void)
{
	s_loaded = 0;
	FILE *f = open_pack();
	if (!f) {
		write_save_stub();
		return 0;
	}
	if (fseek(f, 0, SEEK_END) != 0) {
		fclose(f);
		write_save_stub();
		return 0;
	}
	long n = ftell(f);
	if (n < 8) {
		fclose(f);
		write_save_stub();
		return 0;
	}
	rewind(f);
	unsigned char *buf = (unsigned char *)malloc((size_t)n);
	if (!buf) {
		fclose(f);
		write_save_stub();
		return 0;
	}
	if (fread(buf, 1, (size_t)n, f) != (size_t)n) {
		free(buf);
		fclose(f);
		write_save_stub();
		return 0;
	}
	fclose(f);
	s_loaded = parse_pack(buf, (unsigned)n);
	free(buf);
	write_save_stub();
	return s_loaded;
}

int pack_io_loaded(void)
{
	return s_loaded;
}
