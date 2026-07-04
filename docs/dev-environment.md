# Development Environment

This repo is checked out on a Windows path (`C:\VMs\Projetos\Automonous_Agentic`,
visible from WSL as `/mnt/c/...`). To avoid `node_modules` corruption and
filesystem errors, the Node toolchain has **one canonical environment**.

## Canonical environment: Windows (PowerShell)

Run all Node toolchain commands from **Windows PowerShell**, where `node_modules`
lives on native NTFS:

```powershell
cd C:\VMs\Projetos\Automonous_Agentic
npm ci            # clean install from package-lock.json
npm run build     # production build
npm run dev       # dev server at http://localhost:5173
npm test          # vitest
npm run lint
npm run typecheck
```

### The one rule

Do **not** run `npm install` / `npm ci` / `npm run dev` against this working tree
from **WSL/Linux**. npm resolves platform-specific native binaries (rollup,
esbuild) at install time; mixing a Windows install with a WSL install leaves the
tree broken (`Cannot find module @rollup/rollup-<platform>`). Worse, recursive
deletes and large installs over the WSL→Windows filesystem bridge (`/mnt/c`,
DrvFs) intermittently fail with I/O, ENOENT, and EACCES errors. Keeping all Node
operations on Windows avoids both classes of failure.

The CAO backend may still run in WSL/Docker — it communicates with the SPA over
HTTP at `http://127.0.0.1:9889`, so a split (CAO in WSL, SPA toolchain on Windows)
is fine.

## Recovering a corrupted node_modules

If you see a missing-native-binary error or the preflight doctor fails, reinstall
cleanly from the OS that owns the disk (Windows):

```powershell
cd C:\VMs\Projetos\Automonous_Agentic
Get-Process node,esbuild -ErrorAction SilentlyContinue | Stop-Process -Force
Remove-Item -Recurse -Force node_modules    # or: cmd /c "rmdir /s /q node_modules"
npm ci
```

## Preflight doctor

`scripts/check-dev-env.mjs` runs automatically via the `predev` and `prebuild`
hooks. It loads rollup to trigger native-binary resolution; if the binary for the
current host does not resolve (cross-platform corruption), it fails fast with the
exact remediation instead of a cryptic build stack trace.

## File watching

The dev server uses native file watching by default. If you ever run the dev
server from WSL on `/mnt/c` (not recommended), enable polling so HMR detects
edits:

```bash
VITE_WATCH_POLLING=true npm run dev
```

## Long-term alternative (best performance)

For native Linux performance and watching, move the repo into the WSL filesystem
(ext4), e.g. `~/projects/Automonous_Agentic`, and run everything in WSL. Windows
can access it via `\\wsl$\`. This eliminates the `/mnt/c` bridge entirely but
changes Windows-side paths, so it is optional and deferred.
