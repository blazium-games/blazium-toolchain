# Blazium PS3 GDIR guest: the PS2 guest's scene runtime (GDIR VM, sys_io,
# pack_io) over the PS3's RSX, pads, audio port and hard drive. Built with
# PSL1GHT (ps3dev) by blazium-toolchain (ps3 build --sample gdir), which
# stages the PS2 export's cooked files in pkg/USRDIR/DATA/ first.
#
# make           gdir.elf, and pkg/ as a hard drive game: PARAM.SFO,
#                ICON0.PNG, USRDIR/EBOOT.BIN (an NPDRM SELF) and USRDIR/DATA
# make package   gdir.pkg from pkg/ as well (install it on a PS3 or in RPCS3)

ifeq ($(strip $(PSL1GHT)),)
$(error PSL1GHT is not set (the ps3dev folder))
endif

include $(PSL1GHT)/ppu_rules

TITLE		?= Blazium GDIR
APPID		?= BLAZ00001
CONTENTID	?= UP0001-$(APPID)_00-0000000000000000
APP_VER		?= 01.00
ICON0		?= $(PS3DEV)/bin/ICON0.PNG

SOURCES		:= $(wildcard *.cpp) $(wildcard extra/*.cpp)
OFILES		:= $(SOURCES:.cpp=.o)

# the GDIR VM's tables at the PS2's sizes (and the PS2's pack budgets, which
# the scripts can read); "ps3" for OS.has_feature
DEFINES		:= -DBLAZIUM_PS3_GDIR=1 '-DGDIR_FEATURE="ps3"' -DBLAZIUM_PS2_COOK_ABI=1 \
		   -DPS2_POKE_N=2048u -DGDIR_POKE_BYTES=2048 -DGDIR_ARRAYS=128 -DGDIR_MAX_NODES=512 \
		   '-DPS3_TITLE_ID="$(APPID)"' $(GDIR_EXTRA_FLAGS)

CXXFLAGS	:= -O2 -mcpu=cell $(MACHDEP) -fno-exceptions -fno-rtti -fno-strict-aliasing -Wall -Wno-unused-function \
		   -Wno-unused-variable -Wno-unused-but-set-variable -Wno-sign-compare -I. $(LIBPSL1GHT_INC) \
		   -I$(PORTLIBS)/include $(DEFINES)
LDFLAGS		:= $(MACHDEP) -Wl,-Map,gdir.map
LIBS		:= -lrsx -lgcm_sys -lio -lsysutil -laudio -lrt -llv2 -lm
LIBPATHS	:= $(LIBPSL1GHT_LIB) -L$(PORTLIBS)/lib

.PHONY: all package clean
all: pkg/USRDIR/EBOOT.BIN pkg/PARAM.SFO pkg/ICON0.PNG

gdir.elf: $(OFILES)
	$(VERB) echo linking ... $@
	$(VERB) $(CXX) $^ $(LDFLAGS) $(LIBPATHS) $(LIBS) -o $@

# the ELF stripped, its PRX imports linked, signed as an NPDRM SELF (the
# EBOOT.BIN of a game on the hard drive)
pkg/USRDIR/EBOOT.BIN: gdir.elf
	$(VERB) mkdir -p pkg/USRDIR
	$(VERB) $(STRIP) $< -o gdir.stripped.elf
	$(VERB) $(SPRX) gdir.stripped.elf
	$(VERB) $(SELF_NPDRM) gdir.stripped.elf $@ $(CONTENTID) > /dev/null
	$(VERB) echo wrote $@

pkg/PARAM.SFO: sfo.xml
	$(VERB) mkdir -p pkg
	$(VERB) $(SFO) --title "$(TITLE)" --appid "$(APPID)" -f sfo.xml $@
	$(VERB) echo wrote $@

pkg/ICON0.PNG:
	$(VERB) mkdir -p pkg
	$(VERB) cp "$(ICON0)" $@

package: all
	$(VERB) $(PKG) --contentid $(CONTENTID) pkg/ gdir.pkg > /dev/null
	$(VERB) echo wrote gdir.pkg

clean:
	rm -f $(OFILES) gdir.elf gdir.stripped.elf gdir.map gdir.pkg
	rm -rf pkg/USRDIR/EBOOT.BIN pkg/PARAM.SFO pkg/ICON0.PNG
