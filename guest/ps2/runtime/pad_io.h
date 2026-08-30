// MIT. libpad + rom0 or embedded IRX. Digital + sticks nudge the P4 camera.

#ifndef BLAZIUM_PS2_PAD_IO_H
#define BLAZIUM_PS2_PAD_IO_H

#ifdef __cplusplus
extern "C" {
#endif

int pad_io_init(void);
void pad_io_poll(float *yaw, float *dolly, int *cross_down);
void pad_io_set_rumble(int small_on, int large);
int pad_io_get_pressure(int button);

#ifdef __cplusplus
}
#endif

#endif
