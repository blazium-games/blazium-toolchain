// blazium-toolchain is the official Blazium console toolchain manager.
//
// It is a separate project from the MIT Blazium editor. A 3rd-party
// application downloads this binary; the editor only spawns it.
//
//	blazium-toolchain [--json] [--prefix DIR] version
//	blazium-toolchain [--json] list
//	blazium-toolchain [--json] [--prefix DIR] ps1 setup [--profile compile|dev|iso] [--offline]
//	blazium-toolchain [--json] [--prefix DIR] ps1 env
//	blazium-toolchain [--json] [--prefix DIR] ps1 status
//	blazium-toolchain [--prefix DIR] ps1 build --src DIR --out FILE
//	blazium-toolchain [--prefix DIR] ps1 run [--iso CUE] GAME.EXE
//	blazium-toolchain [--prefix DIR] ps1 iso --xml FILE [--out PATH]
//
// Only the ps1 platform is implemented. ps2, ps3, and ps4 are reserved
// and exit with code 2.
package main
