# blazium-toolchain

Official **Blazium console toolchain manager**. This is a **separate GPLv3 project** from the MIT Blazium engine so it can **contain, cache, and redistribute** compilers, SDKs, packers, and emulators. A 3rd-party application already in use downloads this CLI; the editor **spawns** it. It does not live inside `blazium.git`.

**Now:** `ps1` only.  
**Later:** `ps2`, `ps3`, `ps4` (ids reserved; commands exit `2`).

```
blazium-toolchain [--json] [--prefix DIR] version
blazium-toolchain [--json] list
blazium-toolchain [--json] [--prefix DIR] ps1 setup [--profile compile|dev|iso] [--offline]
blazium-toolchain [--json] [--prefix DIR] ps1 env
blazium-toolchain [--json] [--prefix DIR] ps1 status
blazium-toolchain [--prefix DIR] ps1 build --src DIR --out FILE
blazium-toolchain [--prefix DIR] ps1 run [--iso CUE] GAME.EXE
blazium-toolchain [--prefix DIR] ps1 iso --xml FILE [--out PATH]
```

## For the 3rd-party downloader

| Item | Use |
|------|-----|
| [manifest.json](manifest.json) | Name, version, platforms, commands |
| `blazium-toolchain --json version` | `{ "name", "version" }` |
| `blazium-toolchain --json list` | Platform catalog |
| Exit `2` | Platform reserved but not implemented |
| Exit `3` | Required host tool missing |
| `--prefix` / `BLAZIUM_TOOLCHAIN_PREFIX` | Cache root |

Default cache: `%LOCALAPPDATA%\Blazium\blazium-toolchain` (Windows).

## PS1 profiles

| Profile | Contents |
|---------|----------|
| `compile` | gcc + PSn00bSDK 0.24 + elf2x |
| `dev` | compile + OpenBIOS + pcsx-redux CLI |
| `iso` | dev + mkpsxiso (GPLv2+, spawn) |

`setup` records pins and **reuses** tools already vendored under `third_party/ps1/`, the cache prefix, PATH, or env (`PSN00BSDK_*`, `MIPS_GCC`, `PCSX_EXE`, `OPENBIOS`). Full HTTP fetch of official mirrors can be added without changing the CLI.

## License

This CLI is **[GPL-3.0-or-later](LICENSE)** ([NOTICE](NOTICE)). That is intentional: this repo may vendor GCC, PSn00bSDK, mkpsxiso, pcsx-redux, and OpenBIOS. Each tree keeps its upstream license — see [THIRDPARTY.md](THIRDPARTY.md).

**Blazium stays MIT.** Never merge these components into `blazium.git`. The editor only spawns this binary. `libpsn00b` links into the **exported guest** only.

## Build

```powershell
go test ./...
go build -o blazium-toolchain.exe ./cmd/blazium-toolchain
```
