#ifndef BLAZIUM_N64_PACK_IO_H
#define BLAZIUM_N64_PACK_IO_H

#ifdef __cplusplus
extern "C" {
#endif

void pack_io_init(void);
int pack_io_swap(int pack);
int pack_io_can_fit(int extra_bytes);
int pack_io_instantiate(int pack);
int pack_io_prefetch(const char *path);
const char *pack_io_find_path(const char *res);
const unsigned char *pack_io_cur_mesh(unsigned *sz);
const unsigned char *pack_io_cur_ntex(unsigned *sz);
const unsigned char *pack_io_cur_node(unsigned *sz);
int pack_io_user_save(const void *data, unsigned sz);
int pack_io_user_load(void *data, unsigned sz);
const char *pack_io_user_error(void);
void pack_io_poke(unsigned off, unsigned char v);
unsigned char pack_io_peek(unsigned off);

#ifdef __cplusplus
}
#endif

#endif
