// blazium-toolchain is the official Blazium console toolchain manager.
//
// It is GPL-3.0-or-later so it can contain and manage toolchain
// components (GCC, PSn00bSDK, mkpsxiso, pcsx-redux). A 3rd-party
// application downloads this binary; the MIT Blazium editor only
// spawns it.
//
//	blazium-toolchain [--json] [--prefix DIR] version
//	blazium-toolchain [--json] list
//	blazium-toolchain [--json] [--prefix DIR] ps1 setup [--profile compile|dev|iso] [--offline]
//	blazium-toolchain [--json] [--prefix DIR] ps1 env
//	blazium-toolchain [--json] [--prefix DIR] ps1 status
//	blazium-toolchain [--prefix DIR] ps1 build --out FILE [--src DIR | --sample template|gte]
//	  [--overlay DIR] [--export-src DIR]
//	  [--tim|--mesh|--vag|--sprite|--script|--gdbc|--luau|--str|--xa|--node|--hud|--tile]
//	  --src is optional; the bundled MIT guest stub is the default.
//	  --overlay copies extra or replacement *.cpp on top of the stub (or --src).
//	  --export-src writes the resolved guest C++ tree (same files as export-guest plus overlay).
//	blazium-toolchain [--prefix DIR] ps1 export-guest [--out DIR]
//	  writes bundled guest *.cpp / *.h / CMakeLists.txt (default: prefix/ps1/guest/runtime).
//	blazium-toolchain [--prefix DIR] ps1 run [--iso CUE] [--timeout 120s] [--ui] GAME.EXE
//	blazium-toolchain [--prefix DIR] ps1 iso --xml FILE [--out PATH]
//	blazium-toolchain [--json] [--prefix DIR] ps1 fmv
//	blazium-toolchain [--json] [--prefix DIR] interdvd setup [--offline]
//	blazium-toolchain [--json] [--prefix DIR] interdvd env
//	blazium-toolchain [--json] [--prefix DIR] interdvd status
//	blazium-toolchain [--prefix DIR] interdvd ffmpeg -- <args>
//	blazium-toolchain [--prefix DIR] interdvd ffprobe -- <args>
//	blazium-toolchain interdvd iso --dir DIR --out FILE [--meta FILE] [--write-meta FILE]
//	blazium-toolchain interdvd meta init --out FILE
//	blazium-toolchain interdvd meta validate --meta FILE
//	blazium-toolchain [--prefix DIR] ps2 setup [--profile compile|dev|iso] [--offline]
//	blazium-toolchain [--prefix DIR] ps2 env
//	blazium-toolchain [--prefix DIR] ps2 status
//	blazium-toolchain [--prefix DIR] ps2 build --out FILE.elf [--src DIR | --sample cube]
//	blazium-toolchain [--prefix DIR] ps2 export-guest [--out DIR]
//	blazium-toolchain [--prefix DIR] ps2 run [--iso FILE.iso] [--timeout 120s] [--ui] GAME.elf
//	blazium-toolchain [--prefix DIR] ps2 iso --dir TREE --out FILE.iso
//	blazium-toolchain [--prefix DIR] n64 setup [--profile compile|dev|rom] [--offline]
//	blazium-toolchain [--prefix DIR] n64 env
//	blazium-toolchain [--prefix DIR] n64 status
//	blazium-toolchain [--prefix DIR] n64 build --out FILE.z64 [--src DIR | --sample helloworld|rdpqdemo|t3dquad] [--display 320|640] [--rumble]
//	blazium-toolchain [--prefix DIR] n64 export-guest [--out DIR]
//	blazium-toolchain [--prefix DIR] n64 run [--emu ares|project64|both] [--timeout 120s] GAME.z64
//	blazium-toolchain [--prefix DIR] n64 rom --dir TREE --out FILE.z64 [--elf FILE.elf]
//
// ps1, ps2, n64, and interdvd are implemented. The N64 guest C++ is bundled in
// this CLI (n64 export-guest / default n64 build). Product is a big-endian .z64
// (no ISO/CUE). PS1/PS2/N64 host tools (setup/build/run/iso/rom) are Windows
// and Linux only; other hosts exit with code 2. env/status/export-guest still
// work for inspection. ps3 and ps4 are reserved and exit with code 2.
package main
