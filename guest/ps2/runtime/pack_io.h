// MIT. Open host: then cdrom0: for PACK01 / MESH01 / GTEX01 / NODE01 / DISP.
// user:// stub is host:SAVE.BIN. Pack 0 stays ELF-embedded.

#ifndef BLAZIUM_PS2_PACK_IO_H
#define BLAZIUM_PS2_PACK_IO_H

#ifdef __cplusplus
extern "C" {
#endif

int pack_io_init(void);
int pack_io_loaded(void);

void pack_io_set_pack0(const unsigned char *mesh, unsigned mesh_sz,
		const unsigned char *gtex, unsigned gtex_sz,
		const unsigned char *node, unsigned node_sz);

// 0 = embed, 1 = MESH01/GTEX01/NODE01 from host: or ISO. Re-inits gs_draw.
int pack_io_swap(int pack);
int pack_io_current(void);
int pack_io_has_pack1(void);

// user:// → host:SAVE.BIN then mc0:SAVE.BIN. Errors if neither is writable.
int pack_io_user_present(void);
int pack_io_user_ready(void);
int pack_io_user_save(const void *data, unsigned n);
int pack_io_user_load(void *data, unsigned maxn);
const char *pack_io_user_error(void);

#ifdef __cplusplus
}
#endif

#endif
