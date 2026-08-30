// MIT. Open host: then cdrom0: for STREAM / PACK01 / MESH%02d / GTEX%02d / NODE%02d / DISP.
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

// 0 = embed, n>0 = MESH%02d/GTEX%02d/NODE%02d from host: or ISO. Re-inits gs_draw.
int pack_io_swap(int pack);
int pack_io_current(void);
int pack_io_has_pack1(void);
int pack_io_max(void);
int pack_io_has_stream(void);
void pack_io_current_node(const unsigned char **out, unsigned *sz);

// user:// → host:SAVE.BIN then mc0:SAVE.BIN. Errors if neither is writable.
int pack_io_user_present(void);
int pack_io_user_ready(void);
int pack_io_user_save(const void *data, unsigned n);
int pack_io_user_load(void *data, unsigned maxn);
int pack_io_user_save_slot(int slot, const void *data, unsigned n);
int pack_io_user_load_slot(int slot, void *data, unsigned maxn);
int pack_io_user_format(void);
const char *pack_io_user_error(void);

#ifdef __cplusplus
}
#endif

#endif
