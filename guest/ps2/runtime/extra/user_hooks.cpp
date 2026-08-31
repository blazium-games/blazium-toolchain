// Overlay extra/*.cpp. Define strong user_init / user_tick / user_pad to replace these.

#include "../guest_hooks.h"

#if defined(__GNUC__)
#define BLAZIUM_WEAK __attribute__((weak))
#else
#define BLAZIUM_WEAK
#endif

BLAZIUM_WEAK void user_init(void)
{
}

BLAZIUM_WEAK void user_tick(float delta)
{
	(void)delta;
}

BLAZIUM_WEAK void user_pad(void)
{
}
