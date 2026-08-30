# Blazium PS2 guest — ps2sdk EE Makefile (AFL libs only: draw/graph/packet/dma).
# Spawned by blazium-toolchain ps2 build. Do not link gsKit or audsrv.

EE_BIN = runtime.elf
EE_OBJS = main.o
EE_LIBS = -ldraw -lgraph -lpacket -ldma
EE_CXXFLAGS += -fno-exceptions -fno-rtti -DBLAZIUM_PS2_COOK_ABI=1

# Variables only (no rules). Must come before eeglobal so $(EE_BIN): $(EE_OBJS)
# lists cook_embed.o. Do not add a cook_embed.o recipe here — that would become
# the default goal. eeglobal already has %.o: %.S.
-include cook.mk

ifeq ($(wildcard $(PS2SDK)/samples/Makefile.pref),)
include $(PS2SDK)/samples/Makefile.pref_sample
include $(PS2SDK)/samples/Makefile.eeglobal_sample
else
include $(PS2SDK)/samples/Makefile.pref
include $(PS2SDK)/samples/Makefile.eeglobal
endif

# Windows cmd mkdir is not GNU mkdir -p; objects live next to sources.
DIR_GUARD =
