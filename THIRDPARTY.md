# Components this project may contain and manage

`blazium-toolchain` is **GPL-3.0-or-later** so this repository can **vendor, cache, redistribute, and spawn** the full console toolchain. That is the point of a separate project: Blazium stays MIT; this tree is allowed to hold copyleft tools.

Each component keeps its **upstream** license. Shipping them together is an aggregate of separate programs (the manager **spawns** compilers, packers, and the emulator; it does not statically link them into Blazium).

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

Do **not** copy any of these trees into `blazium/` or link them into the editor binary.

## Layout

```
third_party/ps1/gcc/bin/mipsel-none-elf-gcc[.exe]
third_party/ps1/psn00bsdk/          (SDK prefix / libpsn00b)
third_party/ps1/elf2x/elf2x[.exe]
third_party/ps1/mkpsxiso/mkpsxiso[.exe]
third_party/ps1/pcsx-redux/pcsx-redux[.exe]
third_party/ps1/openbios/openbios.bin
```

The same layout is accepted under `--prefix` / `BLAZIUM_TOOLCHAIN_PREFIX` (`<prefix>/ps1/…` or `<prefix>/third_party/ps1/…`). Discovery order: environment variables, then these vendor trees, then PATH.

Official compile-profile downloads (unmodified):

- `https://github.com/Lameguy64/PSn00bSDK/releases/download/v0.24/gcc-mipsel-none-elf-12.3.0-windows.zip`
- `https://github.com/Lameguy64/PSn00bSDK/releases/download/v0.24/PSn00bSDK-0.24-win32.zip`
- `https://github.com/Kitware/CMake/releases/download/v3.28.6/cmake-3.28.6-windows-x86_64.zip` (if cmake is not on PATH)
- `https://github.com/ninja-build/ninja/releases/download/v1.12.1/ninja-win.zip` (if ninja is not on PATH)
