# Third-party components this CLI may fetch or spawn

`blazium-toolchain` itself is MIT. It must not statically link the following into the Blazium **editor**.

| Component | License | How used |
|-----------|---------|----------|
| mipsel-none-elf GCC | GPLv3 (unmodified binaries) | Spawned at `ps1 build` |
| PSn00bSDK / libpsn00b | MPL 2.0 | Linked into the **guest** only |
| elf2x | PSn00bSDK | Spawned |
| mkpsxiso | GPLv2+ | Spawned by `ps1 iso` only |
| pcsx-redux + OpenBIOS | their trees | Spawned by `ps1 run` |

Do not copy those sources into this repository’s link line or into `blazium/`.
