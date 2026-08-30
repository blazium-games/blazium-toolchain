// MIT. DragonFS rom:// — disc analog. No ISO/CUE.

#include "dfs_io.h"

#include <libdragon.h>
#include <string.h>

static int s_ready;

void dfs_io_init(void)
{
	if (dfs_init(DFS_DEFAULT_LOCATION) == DFS_ESUCCESS) {
		s_ready = 1;
	}
}

int dfs_io_ready(void)
{
	return s_ready;
}

FILE *dfs_io_fopen(const char *path)
{
	if (!path) {
		return NULL;
	}
	if (strncmp(path, "rom://", 6) == 0) {
		return fopen(path, "rb");
	}
	char buf[256];
	snprintf(buf, sizeof(buf), "rom://%s", path);
	return fopen(buf, "rb");
}
