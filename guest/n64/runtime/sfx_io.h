#ifndef BLAZIUM_N64_SFX_IO_H
#define BLAZIUM_N64_SFX_IO_H

#ifdef __cplusplus
extern "C" {
#endif

void sfx_io_init(void);
void sfx_io_tick(void);
int sfx_io_audible(void);
void sfx_io_play(int id);
void sfx_io_load(int id);
void sfx_io_unload(int id);
void sfx_io_music_play(void);
void sfx_io_music_stop(void);
void sfx_io_music_vol(float v);

#ifdef __cplusplus
}
#endif

#endif
