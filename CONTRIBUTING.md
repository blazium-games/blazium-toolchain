# Contributing

Go 1.23.8 or later.

```
go test ./...
go vet ./...
go build -o blazium-toolchain.exe ./cmd/blazium-toolchain
```

Drop `.exe` on Unix. Windows and Linux can run `ps1` / `ps2` / `n64` setup, build, and run. macOS can still build the CLI and master Interactive DVD.

Setup writes compilers and SDK zips into `<prefix>/<plat>/` (`inst`, `src`, `guest`, `work`). Discovery is env, then that tree, then a portable copy next to the exe, then PATH. `C:\ultra` is ignored. Sony BIOS and N64 PIF dumps are not fetched.

Project knobs (URLs, timeouts, volumes, host paths, size gates) live in `blazium-toolchain.yml`. Read them with `settings.Current()`. Keep that file and `Defaults()` in step.

Stdout is process only: `report.Line` / `report.Setup`. Failures go through `report.Usage` / `report.Missing` / `report.Offline` / `report.Fail` so a pasted ticket has kind, message, fix, command, host, version, prefix.

After you change `manifest.json`, `LICENSE`, or `NOTICE` at the repo root, copy the same bytes into `internal/embedfs/files/`. PS1 fetch URL or hash changes go in `internal/embedfs/files/pins.json`. Guest cook ABIs stay aligned with the editor cooker, and the numbers are not shared across platforms:

- PS1: `guest/ps1.CookABI` / `BLAZIUM_PS1_COOK_ABI`
- PS2: `guest/ps2.CookABI` / `BLAZIUM_PS2_COOK_ABI`
- N64: `guest/n64.CookABI` / `BLAZIUM_N64_COOK_ABI`

When you add a command or change the prefix layout, update `cmd/blazium-toolchain/doc.go`, `README.md`, and `usage()`.

CI compiles `hello.elf` (PS2) and `hello.z64` (N64) on Ubuntu and Windows, then builds linux, windows, and darwin binaries.
