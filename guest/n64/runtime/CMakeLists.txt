# Makefile-first N64 guest. CMake is export-guest completeness only.
# Do not add a Godot engine port. Link libdragon via n64.mk (not this file).
cmake_minimum_required(VERSION 3.16)
project(blazium_n64_guest LANGUAGES CXX)

file(GLOB extra_cpp CONFIGURE_DEPENDS extra/*.cpp)
add_library(blazium_n64_guest_sources OBJECT
	main.cpp
	rdpq_draw.cpp
	pad_io.cpp
	sfx_io.cpp
	pack_io.cpp
	script_vm.cpp
	sys_io.cpp
	dfs_io.cpp
	${extra_cpp}
)
target_include_directories(blazium_n64_guest_sources PRIVATE ${CMAKE_CURRENT_SOURCE_DIR})
target_compile_definitions(blazium_n64_guest_sources PRIVATE BLAZIUM_N64_COOK_ABI=1)
target_compile_options(blazium_n64_guest_sources PRIVATE -fno-exceptions -fno-rtti)
