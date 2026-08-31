# Blazium N64 guest — libdragon preview Makefile (Unlicense libs only).
# Spawned by blazium-toolchain n64 build. Link libdragon only (preview).
# Makefile-first: n64.mk owns the z64 recipe. CMakeLists.txt is documentation.

BUILD_DIR = build
ROMNAME = runtime

ifeq ($(N64_INST),)
  $(error N64_INST is not set)
endif
ifeq ($(wildcard $(N64_INST)/include/n64.mk),)
  $(error $(N64_INST)/include/n64.mk does not exist: N64_INST may be incorrect)
endif

LIBDRAGON_PREVIEW ?= 2
include $(N64_INST)/include/n64.mk

CXXFLAGS += -fno-exceptions -fno-rtti -DBLAZIUM_N64_COOK_ABI=1

all: $(ROMNAME).z64
.PHONY: all

CXX_FILES := main.cpp rdpq_draw.cpp pad_io.cpp sfx_io.cpp pack_io.cpp script_vm.cpp sys_io.cpp dfs_io.cpp
CXX_FILES += $(wildcard extra/*.cpp)
OBJS := $(addprefix $(BUILD_DIR)/,$(CXX_FILES:.cpp=.o))

-include cook.mk

$(BUILD_DIR)/$(ROMNAME).elf: $(OBJS)

$(ROMNAME).z64: N64_ROM_TITLE = "Blazium N64"
$(ROMNAME).z64: N64_ROM_SAVETYPE = eeprom4k

clean:
	$(RM) -r $(BUILD_DIR) *.z64
.PHONY: clean

ifneq ($(wildcard $(BUILD_DIR)),)
  -include $(wildcard $(BUILD_DIR)/*.d) $(wildcard $(BUILD_DIR)/extra/*.d)
endif
