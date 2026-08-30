#ifndef BLAZIUM_N64_DFS_IO_H
#define BLAZIUM_N64_DFS_IO_H

#include <stdio.h>

#ifdef __cplusplus
extern "C" {
#endif

void dfs_io_init(void);
int dfs_io_ready(void);
FILE *dfs_io_fopen(const char *path);

#ifdef __cplusplus
}
#endif

#endif
