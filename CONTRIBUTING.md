# Contributing

This repository is the official Blazium console toolchain manager. It is GPL-3.0-or-later on purpose so it can fetch and spawn compilers the MIT editor must never absorb.

## Requirements

- Go 1.23.8 or later
- `go test ./...` and `go build -o blazium-toolchain.exe ./cmd/blazium-toolchain` (drop `.exe` on Unix)
- Windows or Linux for `ps1` / `ps2` / `n64` setup, build, and run (macOS can still build the CLI and master Interactive DVD)

## Rules

- Do not add sibling-workspace path walks (`../PSn00bSDK`, parent `third_party`, editor checkout assumptions in production code). Optional `n64_stuff` / `ps2_stuff` next to the executable or cwd is the only local-tree shortcut.
- Discover host tools via env, `--prefix` / `BLAZIUM_TOOLCHAIN_PREFIX`, `third_party/<plat>` next to the executable, then PATH.
- Never search `C:\ultra`. Never fetch a Sony BIOS or an N64 PIF dump.
- Keep first-party product sources in the embed (`guest/ps1/runtime`, `guest/ps2/runtime`, `guest/n64/runtime`, `internal/embedfs/files`). Do not embed GCC, PSn00bSDK, ps2dev, libdragon toolchain, pcsx-redux, PCSX2, Ares, or mkpsxiso zips.
- After changing `manifest.json`, `LICENSE`, or `NOTICE` at the repo root, copy the same bytes into `internal/embedfs/files/` so the embed tests stay green.
- After changing PS1 fetch URLs or hashes, edit `internal/embedfs/files/pins.json`.
- Guest cook ABIs must stay in lockstep with the editor cooker. Do not reuse numbers across platforms:
  - PS1: `guest/ps1.CookABI` and `BLAZIUM_PS1_COOK_ABI`
  - PS2: `guest/ps2.CookABI` and `BLAZIUM_PS2_COOK_ABI`
  - N64: `guest/n64.CookABI` and `BLAZIUM_N64_COOK_ABI`
- Keep `cmd/blazium-toolchain/doc.go`, `README.md`, and `blazium-toolchain` usage() aligned when adding commands.
- Do not merge toolchain components into `blazium.git`.

## Pull requests

Use the PR template. Tests and `go vet` must pass. CI also compiles `hello.elf` (PS2) and `hello.z64` (N64) on Ubuntu and Windows, then builds linux, windows, and darwin binaries.
