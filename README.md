# blazium-toolchain

A CLI that fetches console compilers into a cache and builds PS1, PS2, N64, and Interactive DVD products. GPL-3.0-or-later; the MIT Blazium editor only spawns the binary. `ps3` and `ps4` are reserved and exit `2`.

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

`n64` has no ISO path. The product is a big-endian `.z64`. `n64 iso` is rejected.

## Install

```
git clone https://github.com/blazium-games/blazium-toolchain.git
cd blazium-toolchain
go test ./...
go build -o blazium-toolchain.exe ./cmd/blazium-toolchain
```

Drop `.exe` on Unix. Go 1.23.8 or later. GitHub Releases publish linux/windows/darwin binaries.

## Project settings

Copy [blazium-toolchain.yml](blazium-toolchain.yml) into a game repo, or use `.blazium-toolchain.yml`. The CLI looks for `--settings FILE`, then `$BLAZIUM_TOOLCHAIN_SETTINGS`, then those two filenames walking up from cwd.

Omitted keys keep built-in defaults. `blazium-toolchain settings` dumps the resolved file (`--json settings` for a machine). SHA-pinned PS1 and FFmpeg zips stay in the embedded `pins.json`.

Knobs you actually change per game: smoke/download timeouts, `download_max_bytes`, default profiles, N64 display/RDRAM/emu/rom title, PS2 ISO labels and SYSTEM.CNF, size gates (`ps2.elf_*_max`, `n64.cart_max`), fetch URLs, and host search paths.

`--tim`, `--node`, and the other cook flags read the file you name. The CLI does not sandbox those paths.

`interdvd` `menu_language`, `audio_language`, `subtitle_language`, `region_mask`, and `parental_level` are stored in `disc.interdvd.json`. They are not written into IFO or UDF.

## How it talks

Stdout is process status. One verb per line:

```
fetching   libdragon toolchain  https://…
installing libdragon preview  <prefix>/n64/inst
skip       project64  PROJECT64_EXE missing
wrote      hello.z64
ready      n64 compile
```

A failure prints a block you can paste into a GitHub issue:

```
error      missing-tool
           n64 compile needs N64_INST (mips64-elf-gcc, n64.mk, libdragon.a)
fix        blazium-toolchain n64 setup --profile compile
---
command    blazium-toolchain n64 build --out hello.z64
host       windows/amd64
version    blazium-toolchain 0.1.0
prefix     C:\Users\…\Blazium\blazium-toolchain
```

`--json` writes the same fields as one object. There is no session log.

## Cache

| Item | Use |
|------|-----|
| [manifest.json](manifest.json) | Name, version, platforms, commands (also embedded) |
| `blazium-toolchain --json version` | `{ "name", "version", "license" }` |
| `blazium-toolchain --json list` | Platform catalog |
| Exit `2` | Reserved platform, or this host cannot compile/run |
| Exit `3` | Host tool missing |
| `--prefix` / `BLAZIUM_TOOLCHAIN_PREFIX` | Cache root |

Default cache: `%LOCALAPPDATA%\Blazium\blazium-toolchain` (Windows), `~/.local/share/blazium-toolchain` (elsewhere).

```
<prefix>/
  ps1/gcc/  psn00bsdk/  pcsx-redux/  openbios/  cmake/  ninja/
  ps1/guest/runtime/    work/    state.json
  ps2/ps2dev/           guest/runtime/    work/    state.json
  n64/inst/             src/libdragon/    src/tiny3d/    ares/
  n64/guest/runtime/    work/    state.json
  interdvd/ffmpeg/      state.json
```

Lookup order: tool env, then `<prefix>/<plat>/`, then `<exeDir>/<plat>/`, then PATH.

The MIT guests (CMake/Makefile + C++) live inside the CLI. `ps1|ps2|n64 export-guest --out DIR` writes the stub. `build` without `--src` uses that stub. `--overlay DIR` copies extra or replacement `*.cpp` on top. `--export-src DIR` writes the resolved tree.

Compilers are not in the binary. `ps1 setup`, `ps2 setup`, and `n64 setup` fetch upstream zips into `<prefix>/<plat>/`. Sony BIOS images and N64 PIF dumps are not fetched. `C:\ultra` is ignored.

## PS1

| Profile | What you get |
|---------|--------------|
| `compile` | gcc + PSn00bSDK 0.24 + elf2x |
| `dev` | compile + OpenBIOS + pcsx-redux CLI |
| `iso` | dev + mkpsxiso |

`ps1 setup --profile compile` fetches PSn00bSDK v0.24 (GCC 12.3.0 + SDK/`elf2x`) when they are missing. `--offline` skips the network.

`ps1 setup --profile dev` also fetches the pcsx-redux Windows x64 CLI nightly and copies `openbios.bin` into `<prefix>/ps1/openbios/`. `ps1 run` smokes that CLI. Default is headless (`-no-ui`); `--ui` opens the window.

`ps1 build --sample template` builds the official zip template to a `PS-X EXE`. CMake and Ninja come from PATH, or a Windows fetch into the cache.

## PS2

| Profile | What you get |
|---------|--------------|
| `compile` | ps2dev EE gcc + PS2SDK |
| `dev` | compile + your `PCSX2_EXE` (BIOS is not fetched) |
| `iso` | dev + `ps2 iso --dir TREE` |

`ps2 setup --profile compile` fetches the ps2dev host tarball into `<prefix>/ps2/ps2dev/`. On Windows it also stages the MinGW DLLs that tarball leaves out.

Set `PCSX2_EXE` (and `PS2_BIOS_DIR` if you have a legal BIOS). `ps2 run` smokes the ELF; `--ui` shows PCSX2.

`ps2 build` without `--src` links the bundled EE guest. Sample `cube` comes from ps2sdk. Size gates (`ps2.elf_text_max`, `ps2.elf_file_max`) apply to `ps2 build` and `ps2 elf-info`. `ps2 chd` wraps an ISO with chdman when that tool is on PATH.

## N64

| Profile | What you get |
|---------|--------------|
| `compile` | libdragon preview `N64_INST` + `mips64-elf-gcc` + `n64.mk` + `libdragon.a` |
| `dev` | compile + Ares and/or Project64 (a missing emulator is a skip) |
| `rom` | compile + `mkdfs` / `n64tool` |

`n64 setup --profile compile` fetches the libdragon mips64-elf toolchain into `<prefix>/n64/inst/` (Windows zip or Linux `.deb`, extracted in-prefix), preview sources into `<prefix>/n64/src/libdragon/`, Tiny3D into `<prefix>/n64/src/tiny3d/`, then builds libdragon. Samples (`helloworld`, `rdpqdemo`, `t3dquad`, `ovldemo`) resolve from that cache.

`n64 setup --profile dev` also fetches Ares v148 on Windows x64. Linux has no zip in that release: install `ares` and set `ARES_EXE`. Project64 is discovered via `PROJECT64_EXE`. `n64 run` uses Parallel-RDP or Angrylion, not Jabo.

`n64 build` without `--src` links the bundled rdpq guest and writes a big-endian `.z64` (`80 37 12 40`).

| Flag | Effect |
|------|--------|
| `--display 320\|640` | Framebuffer; default 320 |
| `--rumble` | Drive a Rumble Pak when present |
| `--rdram 8\|4` | 8 MiB Expansion Pak (default) or 4 MiB so Ares can boot without the pak |
| `--sample helloworld\|rdpqdemo\|t3dquad\|ovldemo` | libdragon / Tiny3D samples |
| `--ntex/--mesh/--node/--inp/--sfx/--music/--pack/--pack-dir` | Cooked slices / DragonFS extras |

`n64 run --emu both` (default) needs both validators when both are installed. A missing single emulator is a skip. `--rdram 4` matches the 4 MiB build.

`n64 rom --dir TREE` packs a DragonFS tree with `mkdfs` and wraps an ELF with `n64tool`.

## Hosts

PS1/PS2/N64 setup, build, run, iso, and rom are Windows and Linux. Other hosts exit `2`. `env`, `status`, and `export-guest` still work. The CLI itself builds on macOS so Interactive DVD mastering works there.

CI runs `go test ./...`, `go vet ./...`, `ps2 build --out hello.elf`, and `n64 build --out hello.z64` on Ubuntu and Windows.

## Interactive DVD

`interdvd iso` masters ISO9660 + UDF 1.02 from a folder that already has `VIDEO_TS/` (and optional `AUDIO_TS/`). No mkisofs. Extra PC files (`--extra`, `--recursive`) sit beside the video folders. Title and related fields live in `disc.interdvd.json`.

`interdvd setup` can fetch pinned FFmpeg. `interdvd ffmpeg` / `ffprobe` spawn that binary.

## License

[GPL-3.0-or-later](LICENSE). Each fetched tree keeps its upstream license; see [THIRDPARTY.md](THIRDPARTY.md). The editor stays MIT and only spawns this binary. libpsn00b / ps2sdk / libdragon link into the exported guest, not the editor.

[CONTRIBUTING.md](CONTRIBUTING.md), [SECURITY.md](SECURITY.md).
