# blazium-toolchain

Official **Blazium console toolchain manager**. This is a **separate project** from the MIT Blazium engine. A 3rd-party application already in use downloads this CLI; the editor **spawns** it. It does not live inside `blazium.git`.

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
| `iso` | dev + mkpsxiso (**spawn only**, GPLv2+) |

`setup` records pins and **reuses** tools already on PATH / env (`PSN00BSDK_*`, `MIPS_GCC`, `PCSX_EXE`, `OPENBIOS`). Full HTTP fetch of official mirrors can be added without changing the CLI.

## License

[MIT](LICENSE) for this repo. Fetched compilers and SDKs stay in the cache and are spawned or linked only into the **exported guest**. See [THIRDPARTY.md](THIRDPARTY.md). Never merge those trees into Blazium.

## Build

```powershell
go test ./...
go build -o blazium-toolchain.exe ./cmd/blazium-toolchain
```
