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
//
// ps1, ps2, and interdvd are implemented. The PS2 guest C++ is bundled in
// this CLI (ps2 export-guest / default ps2 build). PS1/PS2 host tools
// (setup/build/run/iso) are Windows and Linux only; other hosts exit with
// code 2. env/status/export-guest still work for inspection.
// ps3, ps4, and n64 are reserved and exit with code 2.
package main
