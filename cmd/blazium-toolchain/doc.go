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
//	  [--tim|--mesh|--vag|--sprite|--script|--gdbc|--luau|--str|--xa|--node|--hud|--tile]
//	  --src is optional; the bundled MIT guest stub is the default.
//	blazium-toolchain [--prefix DIR] ps1 run [--iso CUE] [--timeout 120s] GAME.EXE
//	blazium-toolchain [--prefix DIR] ps1 iso --xml FILE [--out PATH]
//	blazium-toolchain [--json] [--prefix DIR] ps1 fmv
//	blazium-toolchain interdvd iso --dir DIR --out FILE [--meta FILE] [--write-meta FILE]
//	blazium-toolchain interdvd meta init --out FILE
//	blazium-toolchain interdvd meta validate --meta FILE
//
// ps1 and interdvd are implemented. PS1 host tools (setup/build/run)
// are Windows and Linux only; other hosts exit with code 2.
// ps2, ps3, and ps4 are reserved and exit with code 2.
package main
