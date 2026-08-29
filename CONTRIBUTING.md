# Contributing

This repository is the official Blazium console toolchain manager. It is GPL-3.0-or-later on purpose so it can fetch and spawn compilers the MIT editor must never absorb.

## Requirements

- Go 1.23.8 or later
- `go test ./...` and `go build -o blazium-toolchain.exe ./cmd/blazium-toolchain` (drop `.exe` on Unix)

## Rules

- Do not add sibling-workspace path walks (`../PSn00bSDK`, parent `third_party`, editor checkout assumptions in production code).
- Discover host tools via env, `--prefix` / `BLAZIUM_TOOLCHAIN_PREFIX`, `third_party/ps1` next to the executable, then PATH.
- Keep first-party product sources in the embed (`guest/ps1/runtime`, `internal/embedfs/files`). Do not embed GCC, PSn00bSDK, pcsx-redux, or mkpsxiso zips.
- After changing `manifest.json`, `LICENSE`, or `NOTICE` at the repo root, copy the same bytes into `internal/embedfs/files/` so the embed tests stay green.
- After changing fetch URLs or hashes, edit `internal/embedfs/files/pins.json`.
- Guest cook ABI (`guest.CookABI` and `BLAZIUM_PS1_COOK_ABI` in CMakeLists.txt) must stay in lockstep.
- Do not merge toolchain components into `blazium.git`.

## Pull requests

Use the PR template. Tests and `go vet` must pass. CI builds linux, windows, and darwin binaries on every change.
