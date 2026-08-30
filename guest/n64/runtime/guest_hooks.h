// Weak overlay hooks. Developers add extra/*.cpp (res://n64_guest/extra) and
// define user_init / user_tick / user_pad. No plugin ABI / second VM.

#ifndef BLAZIUM_N64_GUEST_HOOKS_H
#define BLAZIUM_N64_GUEST_HOOKS_H

#ifdef __cplusplus
extern "C" {
#endif

#if defined(__GNUC__)
void user_init(void) __attribute__((weak));
void user_tick(float delta) __attribute__((weak));
void user_pad(void) __attribute__((weak));
#else
void user_init(void);
void user_tick(float delta);
void user_pad(void);
#endif

#ifdef __cplusplus
}
#endif

#endif
