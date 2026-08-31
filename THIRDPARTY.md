# Components this project may contain and manage

`blazium-toolchain` is **GPL-3.0-or-later** so this repository can **vendor, cache, redistribute, and spawn** the full console toolchain. That is the point of a separate project: Blazium stays MIT; this tree is allowed to hold copyleft tools.

Each component keeps its **upstream** license. Shipping them together is an aggregate of separate programs (the manager **spawns** compilers, packers, and emulators; it does not statically link them into Blazium).

| Component | Upstream license | Where it may live | Role |
|-----------|------------------|-------------------|------|
| This CLI | GPL-3.0-or-later | this repo | Manager |
| mipsel-none-elf GCC | GPLv3 (unmodified binaries; sources from GNU) | `third_party/ps1/gcc/`, cache | Spawned at `ps1 build` |
| PSn00bSDK / libpsn00b | MPL 2.0 | `third_party/ps1/psn00bsdk/`, cache | Linked into the **guest** only |
| elf2x | PSn00bSDK / MPL 2.0 | `third_party/ps1/elf2x/`, cache | Spawned |
| mkpsxiso | GPLv2 or later | `third_party/ps1/mkpsxiso/`, cache | Spawned by `ps1 iso` |
| pcsx-redux | GPLv2 | `third_party/ps1/pcsx-redux/`, cache | Spawned by `ps1 run` |
| OpenBIOS | pcsx-redux tree | `third_party/ps1/openbios/`, cache | No Sony BIOS |
| CMake | BSD-3 | `third_party/ps1/cmake/`, cache | Spawned at `ps1 build` if not on PATH |
| Ninja | Apache-2.0 | `third_party/ps1/ninja/`, cache | Spawned at `ps1 build` if not on PATH |
| ps2dev (EE gcc + PS2SDK) | various (ps2dev / GPL / Academic) | `<prefix>/ps2/ps2dev/`, `third_party/ps2/` | Spawned at `ps2 build` |
| PCSX2 | GPLv3 | user `PCSX2_EXE` | Spawned by `ps2 run`; BIOS never fetched |
| libdragon mips64-elf GCC | GPLv3 (unmodified binaries) | `<prefix>/n64/`, `third_party/n64/` | Spawned at `n64 build` |
| libdragon preview | Unlicense | built into `N64_INST` | Linked into the **N64 guest** only |
| Tiny3D | Unlicense | sibling `n64_stuff/tiny3d` (sample only) | `n64 build --sample t3dquad` |
| Ares | ISC | `<prefix>/n64/ares/` | Spawned by `n64 run`; no PIF dump |
| Project64 | GPLv2 | user `PROJECT64_EXE` | Spawned by `n64 run` (Parallel-RDP or Angrylion) |
| FFmpeg | GPL/LGPL (pinned build) | `<prefix>/interdvd/ffmpeg/` | Spawned by `interdvd ffmpeg` / `ffprobe` |

Do **not** copy any of these trees into `blazium/` or link them into the editor binary.

## Layout

```
third_party/ps1/gcc/bin/mipsel-none-elf-gcc[.exe]
third_party/ps1/psn00bsdk/          (SDK prefix / libpsn00b)
third_party/ps1/elf2x/elf2x[.exe]
third_party/ps1/mkpsxiso/mkpsxiso[.exe]
third_party/ps1/pcsx-redux/pcsx-redux[.exe]
third_party/ps1/openbios/openbios.bin

<prefix>/ps2/ps2dev/ee/bin/mips64r5900el-ps2-elf-gcc[.exe]
<prefix>/ps2/guest/runtime/         (exported MIT stub)

<prefix>/n64/                       (N64_INST: mips64-elf-gcc, include/n64.mk, libdragon.a)
<prefix>/n64/ares/ares[.exe]
<prefix>/n64/guest/runtime/         (exported MIT stub)
```

The same layout is accepted under `--prefix` / `BLAZIUM_TOOLCHAIN_PREFIX` (`<prefix>/<plat>/…` or `<prefix>/third_party/<plat>/…`) and next to the executable (`<exeDir>/third_party/<plat>/…`). Discovery order: environment variables, then those vendor trees, then PATH. The CLI does not walk the current working directory or sibling source checkouts except optional `n64_stuff` / `ps2_stuff` for local libdragon/ps2dev trees. `C:\ultra` is always ignored.

Official compile-profile downloads (unmodified):

- `https://github.com/Lameguy64/PSn00bSDK/releases/download/v0.24/gcc-mipsel-none-elf-12.3.0-windows.zip`
- `https://github.com/Lameguy64/PSn00bSDK/releases/download/v0.24/PSn00bSDK-0.24-win32.zip`
- `https://github.com/Kitware/CMake/releases/download/v3.28.6/cmake-3.28.6-windows-x86_64.zip` (if cmake is not on PATH)
- `https://github.com/ninja-build/ninja/releases/download/v1.12.1/ninja-win.zip` (if ninja is not on PATH)
- `https://github.com/ps2dev/ps2dev/releases/download/latest/ps2dev-windows-latest.tar.gz`
- `https://github.com/ps2dev/ps2dev/releases/download/latest/ps2dev-ubuntu-latest.tar.gz`
- `https://github.com/DragonMinded/libdragon/releases/download/toolchain-continuous-prerelease/gcc-toolchain-mips64-win64.zip`

Official dev-profile downloads (unmodified nightlies / releases):

- AppDistrib catalog `https://distrib.app/storage/manifests/pcsx-redux/dev-win-cli-x64/manifest.json` → latest `pcsx-redux-nightly-*-x64-cli.zip` (contains the CLI and `openbios.bin`)
- `https://github.com/ares-emulator/ares/releases/download/v148/ares-windows-x64.zip` (Windows x64 only; Linux has no zip in that release)
