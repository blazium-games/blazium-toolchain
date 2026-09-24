# @blazium-engine/toolchain

Command-line tool that fetches console compilers into a cache and builds PS1, PS2, N64, and Interactive DVD products.

```bash
npx @blazium-engine/toolchain
npm install -g @blazium-engine/toolchain
blazium-toolchain
```

Supported platforms are Linux and Windows, x64 and ia32. When npm installs the matching optional package (`@blazium-engine/toolchain-linux-x64`, `toolchain-linux-ia32`, `toolchain-win32-x64`, or `toolchain-win32-ia32`), that binary is used and nothing is downloaded. The CDN download runs only if that optional package is absent.

- Repository: https://github.com/blazium-games/blazium-toolchain
- Docs: https://docs.blazium.app
- Discord: https://discord.gg/sZaf9KYzDp
- License: GPL-3.0-or-later
