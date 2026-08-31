# blazium-toolchain

Official **Blazium console toolchain manager**. This is a **separate GPLv3 project** from the MIT Blazium engine so it can **contain, cache, and redistribute** compilers, SDKs, packers, and emulators. A 3rd-party application downloads this CLI; the editor **spawns** it. It does not live inside `blazium.git`.

**Now:** `ps1`, `ps2`, `n64`, and `interdvd`.  
**Later:** `ps3`, `ps4` (ids reserved; commands exit `2`).

```
blazium-toolchain [--json] [--prefix DIR] version
blazium-toolchain [--json] list
blazium-toolchain [--json] [--prefix DIR] ps1 setup [--profile compile|dev|iso] [--offline]
blazium-toolchain [--json] [--prefix DIR] ps1 env
blazium-toolchain [--json] [--prefix DIR] ps1 status
blazium-toolchain [--prefix DIR] ps1 build --out FILE [--src DIR | --sample template|gte]
  [--overlay DIR] [--export-src DIR]
  [--tim|--mesh|--vag|--sprite|--script|--gdbc|--luau|--str|--xa|--node|--hud|--tile|--scene|--anim|--cam|--hit|--nav|--path|--way]
blazium-toolchain [--prefix DIR] ps1 export-guest [--out DIR]
blazium-toolchain [--prefix DIR] ps1 run [--iso CUE] [--timeout 120s] [--ui] GAME.EXE
blazium-toolchain [--prefix DIR] ps1 iso --xml FILE [--out PATH]
blazium-toolchain [--json] [--prefix DIR] ps1 fmv
blazium-toolchain [--json] [--prefix DIR] interdvd setup [--offline]
blazium-toolchain [--json] [--prefix DIR] interdvd env
blazium-toolchain [--json] [--prefix DIR] interdvd status
blazium-toolchain [--prefix DIR] interdvd ffmpeg -- <args>
blazium-toolchain [--prefix DIR] interdvd ffprobe -- <args>
blazium-toolchain interdvd iso --dir DIR --out FILE [--meta FILE] [--write-meta FILE] [--extra HOST[:DISC]] [--recursive]
blazium-toolchain interdvd meta init --out disc.interdvd.json
blazium-toolchain interdvd meta validate --meta disc.interdvd.json
blazium-toolchain [--prefix DIR] ps2 setup [--profile compile|dev|iso] [--offline]
blazium-toolchain [--prefix DIR] ps2 env
blazium-toolchain [--prefix DIR] ps2 status
blazium-toolchain [--prefix DIR] ps2 build --out FILE.elf [--src DIR | --sample cube]
  [--overlay DIR] [--export-src DIR] [--node|--mesh|--gtex|--script]
blazium-toolchain [--prefix DIR] ps2 export-guest [--out DIR]
blazium-toolchain [--prefix DIR] ps2 run [--iso FILE.iso] [--timeout 120s] [--ui] GAME.elf
blazium-toolchain [--prefix DIR] ps2 iso --dir TREE --out FILE.iso
blazium-toolchain [--prefix DIR] ps2 elf-info FILE.elf
blazium-toolchain [--prefix DIR] ps2 chd --iso FILE.iso --out FILE.chd
blazium-toolchain [--prefix DIR] n64 setup [--profile compile|dev|rom] [--offline]
blazium-toolchain [--prefix DIR] n64 env
blazium-toolchain [--prefix DIR] n64 status
blazium-toolchain [--prefix DIR] n64 build --out FILE.z64 [--src DIR | --sample helloworld|rdpqdemo|t3dquad|ovldemo]
  [--overlay DIR] [--export-src DIR] [--display 320|640] [--rumble] [--rdram 8|4]
  [--ntex|--mesh|--node|--inp|--sfx|--music|--pack|--pack-dir|--script|--gdbc|--luau]
blazium-toolchain [--prefix DIR] n64 export-guest [--out DIR]
blazium-toolchain [--prefix DIR] n64 run [--emu ares|project64|both] [--timeout 120s] [--rdram 8|4] GAME.z64
blazium-toolchain [--prefix DIR] n64 rom --dir TREE --out FILE.z64 [--elf FILE.elf]
```

`n64` has no ISO/CUE path. The product is a big-endian `.z64`. `n64 iso` is rejected.

## Install

Clone this repository (private today; the same layout is ready to go public later):

```
git clone https://github.com/blazium-games/blazium-toolchain.git
cd blazium-toolchain
go test ./...
go build -o blazium-toolchain.exe ./cmd/blazium-toolchain
```

On Unix, drop the `.exe` suffix. Go **1.23.8** or later. GitHub Releases publish linux/windows/darwin binaries.

The **MIT guests** (CMake/Makefile + C++) are **embedded** in the CLI. `ps1|ps2|n64 export-guest --out DIR` writes the stub for that platform. `build` without `--src` uses that stub. `--overlay DIR` (or `BLAZIUM_PS1_OVERLAY` / `BLAZIUM_PS2_OVERLAY` / `BLAZIUM_N64_OVERLAY`) copies extra or replacement `*.cpp` on top — drop hooks in `extra/` or replace `script_vm.cpp`. `--export-src DIR` writes the resolved tree.

Compilers, SDKs, packers, and emulators are **not** in the binary. `ps1 setup`, `ps2 setup`, and `n64 setup` fetch official zips into the prefix when a profile needs them. Sony BIOS images and N64 PIF dumps are **never** fetched. `C:\ultra` is ignored.

## Cache and discovery

| Item | Use |
|------|-----|
| [manifest.json](manifest.json) | Name, version, platforms, commands (also embedded) |
| `blazium-toolchain --json version` | `{ "name", "version", "license" }` |
| `blazium-toolchain --json list` | Platform catalog |
| Exit `2` | Platform reserved but not implemented, or host OS cannot compile/run |
| Exit `3` | Required host tool missing |
| `--prefix` / `BLAZIUM_TOOLCHAIN_PREFIX` | Cache root |

Default cache: `%LOCALAPPDATA%\Blazium\blazium-toolchain` (Windows), `~/.local/share/blazium-toolchain` (elsewhere).

Discovery order (no sibling workspace walks in production paths except optional `n64_stuff` / `ps2_stuff` next to the executable or cwd for local libdragon/ps2dev trees):

1. Tool env (`PSN00BSDK_*`, `MIPS_GCC`, `ELF2X`, `PCSX_EXE`, `OPENBIOS`, `MKPSXISO`, `EE_GCC`, `PS2SDK`, `PS2DEV`, `PCSX2_EXE`, `PS2_BIOS_DIR`, `N64_INST`, `N64_GCC`, `ARES_EXE`, `PROJECT64_EXE`, …)
2. `--prefix` / `BLAZIUM_TOOLCHAIN_PREFIX` (`<prefix>/<plat>/…` or `<prefix>/third_party/<plat>/…`)
3. `third_party/<plat>/` next to the **executable**
4. `PATH`

Official PS1 fetch URLs and SHA-256 pins live in the embedded `pins.json`. PS2 fetches the official [ps2dev](https://github.com/ps2dev/ps2dev/releases) host tarball. N64 fetches the official [libdragon](https://github.com/DragonMinded/libdragon/releases) `gcc-toolchain-mips64` zip and, on Windows `dev`, Ares **v148** (no PIF).

## PS1 profiles

| Profile | Contents |
|---------|----------|
| `compile` | gcc + PSn00bSDK 0.24 + elf2x |
| `dev` | compile + OpenBIOS + pcsx-redux CLI |
| `iso` | dev + mkpsxiso (GPLv2+, spawn) |

`ps1 setup --profile compile` fetches official PSn00bSDK **v0.24** zips (GCC **12.3.0** + SDK/`elf2x`) into the cache when they are missing. `--offline` never hits the network.

`ps1 setup --profile dev` also fetches the official pcsx-redux **Windows x64 CLI** nightly (AppDistrib `dev-win-cli-x64`) and installs `openbios.bin` from that zip into `<prefix>/ps1/openbios/`. `ps1 run` spawns that CLI with OpenBIOS, `-loadexe` / `-iso`, and a smoke timeout. Default is headless (`-no-ui`); `--ui` opens the window. `iso` still only records the mkpsxiso pin.

`ps1 build --sample template` configures the official zip template with Ninja, builds only the EXE target (not ISO), and writes a `PS-X EXE`. CMake and Ninja are looked up on PATH or fetched on Windows into the cache.

## PS2 profiles

| Profile | Contents |
|---------|----------|
| `compile` | ps2dev EE gcc (`mips64r5900el-ps2-elf-gcc`) + PS2SDK |
| `dev` | compile + user-supplied `PCSX2_EXE` (BIOS never fetched) |
| `iso` | dev + in-tree ISO9660 writer (`ps2 iso --dir TREE`) |

`ps2 setup --profile compile` fetches the official ps2dev host tarball into `<prefix>/ps2/ps2dev/` when EE gcc is missing. On Windows it also stages the MinGW DLLs the official tarball omits.

`ps2 setup --profile dev` does **not** download PCSX2 or a BIOS. Set `PCSX2_EXE` (and `PS2_BIOS_DIR` when you have a legal BIOS). `ps2 run` smokes the ELF; `--ui` shows the PCSX2 window.

`ps2 build` without `--src` links the bundled EE guest. Default sample is `cube` (ps2sdk). `ps2 elf-info` prints `.text` and file size gates used by CI (512 KiB `.text`, 2 MiB file). `ps2 chd` wraps an ISO with chdman when that tool is on PATH.

## N64 profiles

| Profile | Contents |
|---------|----------|
| `compile` | libdragon preview `N64_INST` + `mips64-elf-gcc` + `n64.mk` + `libdragon.a` |
| `dev` | compile + Ares and/or Project64 (missing validator is a skip, not a hard fail) |
| `rom` | compile + `mkdfs` / `n64tool` for `n64 rom` |

`n64 setup --profile compile` fetches the official libdragon **mips64-elf** toolchain zip and builds libdragon **preview** into `N64_INST`. It never searches `C:\ultra` and never fetches a PIF/BIOS dump.

`n64 setup --profile dev` also fetches Ares **v148** on Windows x64 (`ares-windows-x64.zip`). Linux has no official Ares zip in that release — install `ares` and set `ARES_EXE`. Project64 is discovered via `PROJECT64_EXE` or a local install; `n64 run` pins Parallel-RDP or Angrylion (never Jabo).

`n64 build` without `--src` links the bundled rdpq guest and writes a big-endian `.z64` (`80 37 12 40`). Flags:

| Flag | Effect |
|------|--------|
| `--display 320\|640` | Framebuffer; default 320 |
| `--rumble` | Drive a Rumble Pak when present |
| `--rdram 8\|4` | 8 MiB Expansion Pak (default) or 4 MiB so Ares can boot without the pak |
| `--sample helloworld\|rdpqdemo\|t3dquad\|ovldemo` | Official libdragon / Tiny3D samples (not linked into the default guest) |
| `--ntex/--mesh/--node/--inp/--sfx/--music/--pack/--pack-dir` | Cooked slices / DragonFS extras |

`n64 run --emu both` (default) requires both validators to boot when both are installed; a missing single emulator is a skip so One-click can continue. `--rdram 4` matches the 4 MiB build.

`n64 rom --dir TREE` packs a DragonFS tree with `mkdfs` and wraps an ELF with `n64tool`.

## Host support

PS1/PS2/N64 **setup / build / run / iso / rom** are **Windows and Linux** only (other hosts exit `2`). `env`, `status`, and `export-guest` still work for inspection. The Go CLI itself builds on macOS so Interactive DVD mastering works everywhere.

CI gates `go test ./...`, `go vet ./...`, `ps2 build --out hello.elf` (MIPS ELF + size), and `n64 build --out hello.z64` (`.z64` magic + 64 MiB cart cap) on Ubuntu and Windows.

## Interactive DVD

`interdvd iso` masters a DVD-Video **ISO9660 + UDF 1.02** bridge from a folder that already contains `VIDEO_TS/` (and optional `AUDIO_TS/`). No mkisofs or oscdimg. Extra PC files (`--extra`, `--extras-dir`, `--recursive`) sit beside the video folders. Disc title, copyright, license, and related fields live in `disc.interdvd.json` (`interdvd meta init` / `--meta` / `--write-meta`).

`interdvd setup` can fetch pinned FFmpeg into the prefix. `interdvd ffmpeg` / `ffprobe` spawn that binary.

## License

This CLI is **[GPL-3.0-or-later](LICENSE)** ([NOTICE](NOTICE)). That is intentional: this repo may vendor GCC, PSn00bSDK, mkpsxiso, pcsx-redux, OpenBIOS, ps2dev, libdragon toolchain, and Ares. Each tree keeps its upstream license — see [THIRDPARTY.md](THIRDPARTY.md).

**Blazium stays MIT.** Never merge these components into `blazium.git`. The editor only spawns this binary. `libpsn00b` / ps2sdk / libdragon link into the **exported guest** only.

See [CONTRIBUTING.md](CONTRIBUTING.md) and [SECURITY.md](SECURITY.md).
