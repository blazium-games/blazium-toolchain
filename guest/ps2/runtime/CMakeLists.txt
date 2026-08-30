# Guest-only ps2sdk target. Host compilers are spawned by blazium-toolchain
# (ps2 build). Prefer the Makefile when PS2SDK samples are present.

cmake_minimum_required(VERSION 3.10)

project(blazium_ps2_runtime LANGUAGES CXX)

add_executable(runtime main.cpp)
file(GLOB _ps2_extra RELATIVE "${CMAKE_CURRENT_SOURCE_DIR}"
	"extra/*.cpp" "extra/*.c")
if(_ps2_extra)
	target_sources(runtime PRIVATE ${_ps2_extra})
endif()
target_include_directories(runtime PRIVATE
	"${CMAKE_CURRENT_SOURCE_DIR}"
	"$ENV{PS2SDK}/ee/include"
	"$ENV{PS2SDK}/common/include")
target_compile_options(runtime PRIVATE -D_EE -G0 -fno-exceptions -fno-rtti)
target_compile_definitions(runtime PRIVATE BLAZIUM_PS2_COOK_ABI=1)
target_link_libraries(runtime PRIVATE draw graph packet dma)
set_target_properties(runtime PROPERTIES OUTPUT_NAME "runtime" SUFFIX ".elf")
