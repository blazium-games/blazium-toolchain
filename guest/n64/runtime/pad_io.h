#ifndef BLAZIUM_N64_PAD_IO_H
#define BLAZIUM_N64_PAD_IO_H

#ifdef __cplusplus
extern "C" {
#endif

void pad_io_init(void);
void pad_io_poll(void);
int pad_io_pressed(int action);
void pad_io_stick(float *x, float *y);
void pad_io_set_deadzone(float dz);
void pad_io_set_rumble(int on);
void pad_io_stop_rumble(void);

#ifdef __cplusplus
}
#endif

#endif
