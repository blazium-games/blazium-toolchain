## Summary

-

## Test plan

- [ ] `go test ./...`
- [ ] `go build` of `./cmd/blazium-toolchain`
- [ ] No new hardcoded machine or sibling-workspace paths (no `C:\ultra`, no BIOS/PIF fetch)
- [ ] Embed files updated if `manifest.json`, `LICENSE`, `NOTICE`, or pins changed
- [ ] README / `cmd/blazium-toolchain/doc.go` / usage() stay aligned if commands changed
- [ ] PS2 `hello.elf` and N64 `hello.z64` CI jobs still apply when those platforms change
