# blazium-toolchain

Official **Blazium console toolchain manager**. This is a **separate GPLv3 project** from the MIT Blazium engine so it can **contain, cache, and redistribute** compilers, SDKs, packers, and emulators. A 3rd-party application downloads this CLI; the editor **spawns** it. It does not live inside `blazium.git`.

**Now:** `ps1` and `interdvd`.  
**Later:** `ps2`, `ps3`, `ps4` (ids reserved; commands exit `2`).

```
blazium-toolchain [--json] [--prefix DIR] version
blazium-toolchain [--json] list
blazium-toolchain [--json] [--prefix DIR] ps1 setup [--profile compile|dev|iso] [--offline]
blazium-toolchain [--json] [--prefix DIR] ps1 env
blazium-toolchain [--json] [--prefix DIR] ps1 status
blazium-toolchain [--prefix DIR] ps1 build --out FILE [--src DIR | --sample template|gte]
blazium-toolchain [--prefix DIR] ps1 run [--iso CUE] [--timeout 120s] GAME.EXE
blazium-toolchain [--prefix DIR] ps1 iso --xml FILE [--out PATH]
blazium-toolchain interdvd iso --dir DIR --out FILE [--meta FILE] [--write-meta FILE] [--extra HOST[:DISC]] [--recursive]
blazium-toolchain interdvd meta init --out disc.interdvd.json
blazium-toolchain interdvd meta validate --meta disc.interdvd.json
```

## Install

Clone this repository (private today; the same layout is ready to go public later):

```
git clone https://github.com/blazium-games/blazium-toolchain.git
cd blazium-toolchain
go test ./...
go build -o blazium-toolchain.exe ./cmd/blazium-toolchain
```

On Unix, drop the `.exe` suffix. Go **1.23.8** or later. GitHub Releases publish linux/windows/darwin binaries.

The **MIT guest** (CMake + C++) is **embedded** in the CLI. `ps1 build` without `--src` installs it into the cache. Compilers, PSn00bSDK, pcsx-redux, and mkpsxiso are **not** in the binary; `ps1 setup` fetches official zips into the prefix.

## Cache and discovery

| Item | Use |
|------|-----|
| [manifest.json](manifest.json) | Name, version, platforms, commands (also embedded) |
| `blazium-toolchain --json version` | `{ "name", "version", "license" }` |
| `blazium-toolchain --json list` | Platform catalog |
| Exit `2` | Platform reserved but not implemented |
| Exit `3` | Required host tool missing |
| `--prefix` / `BLAZIUM_TOOLCHAIN_PREFIX` | Cache root |

Default cache: `%LOCALAPPDATA%\Blazium\blazium-toolchain` (Windows), `~/.local/share/blazium-toolchain` (elsewhere).

Discovery order (no sibling workspace walks):

1. Tool env (`PSN00BSDK_*`, `MIPS_GCC`, `ELF2X`, `PCSX_EXE`, `OPENBIOS`, `MKPSXISO`)
2. `--prefix` / `BLAZIUM_TOOLCHAIN_PREFIX` (`<prefix>/ps1/…` or `<prefix>/third_party/ps1/…`)
3. `third_party/ps1/` next to the **executable**
4. `PATH`

Official fetch URLs and SHA-256 pins live in the embedded `pins.json`.

## PS1 profiles

| Profile | Contents |
|---------|----------|
| `compile` | gcc + PSn00bSDK 0.24 + elf2x |
| `dev` | compile + OpenBIOS + pcsx-redux CLI |
| `iso` | dev + mkpsxiso (GPLv2+, spawn) |

`ps1 setup --profile compile` fetches official PSn00bSDK **v0.24** zips (GCC **12.3.0** + SDK/`elf2x`) into the cache when they are missing. `--offline` never hits the network.

`ps1 setup --profile dev` also fetches the official pcsx-redux **Windows x64 CLI** nightly (AppDistrib `dev-win-cli-x64`) and installs `openbios.bin` from that zip into `<prefix>/ps1/openbios/`. `ps1 run` spawns that CLI with OpenBIOS, `-loadexe` / `-iso`, and a smoke timeout. `iso` still only records the mkpsxiso pin.

`ps1 build --sample template` configures the official zip template with Ninja, builds only the EXE target (not ISO), and writes a `PS-X EXE`. CMake and Ninja are looked up on PATH or fetched on Windows into the cache.

PS1 host tools are **Windows and Linux** only (other hosts exit `2`). The Go CLI itself builds on macOS so Interactive DVD mastering works everywhere.

## Interactive DVD

`interdvd iso` masters a DVD-Video **ISO9660 + UDF 1.02** bridge from a folder that already contains `VIDEO_TS/` (and optional `AUDIO_TS/`). No mkisofs or oscdimg. Extra PC files (`--extra`, `--extras-dir`, `--recursive`) sit beside the video folders. Disc title, copyright, license, and related fields live in `disc.interdvd.json` (`interdvd meta init` / `--meta` / `--write-meta`).

## License

This CLI is **[GPL-3.0-or-later](LICENSE)** ([NOTICE](NOTICE)). That is intentional: this repo may vendor GCC, PSn00bSDK, mkpsxiso, pcsx-redux, and OpenBIOS. Each tree keeps its upstream license — see [THIRDPARTY.md](THIRDPARTY.md).

**Blazium stays MIT.** Never merge these components into `blazium.git`. The editor only spawns this binary. `libpsn00b` links into the **exported guest** only.

See [CONTRIBUTING.md](CONTRIBUTING.md) and [SECURITY.md](SECURITY.md).
