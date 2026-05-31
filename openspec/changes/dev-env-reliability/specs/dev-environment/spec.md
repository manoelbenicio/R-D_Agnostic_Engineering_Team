## ADDED Requirements

### Requirement: Canonical development environment
The project SHALL designate a single canonical environment (Windows/PowerShell,
where the repository's NTFS disk lives) for dependency installation, building,
and running the dev server. Documentation SHALL instruct that `npm ci`,
`npm run build`, and `npm run dev` are run from that environment only, and that
the Node toolchain MUST NOT be run from WSL/Linux against the `/mnt/c` working
tree (the DrvFs bridge causes native-binary mismatch and I/O failures).

#### Scenario: Documented canonical workflow
- **WHEN** a developer or agent opens `docs/dev-environment.md`
- **THEN** it states Windows/PowerShell as the canonical environment and lists the
  exact install/build/dev commands to run there

#### Scenario: Cross-platform install is prevented from silently breaking the tree
- **WHEN** `node_modules` was last populated by an install from a different OS
  than the current host
- **THEN** the preflight doctor (see below) fails fast before build/dev with a
  remediation message, rather than surfacing a cryptic native-module stack trace

### Requirement: Platform-correct dependency integrity
A clean install from the committed `package-lock.json` on the canonical
environment SHALL resolve the correct platform-specific native dependencies
(including the rollup native binary) with no manual `--no-save` steps.

#### Scenario: Clean install builds successfully
- **WHEN** a developer runs `rm -rf node_modules && npm ci` on WSL/Linux and then
  `npm run build`
- **THEN** the build completes successfully without any manually-installed
  `@rollup/rollup-*` binary

#### Scenario: No transient binaries are required
- **WHEN** the production build runs
- **THEN** it depends only on dependencies recorded in `package-lock.json`, not on
  any binary installed outside the lockfile

### Requirement: Preflight environment doctor
The project SHALL provide a preflight check, invoked automatically before
`dev` and `build`, that verifies the installed native dependencies match the
current host platform and fails fast with actionable remediation when they do not.

#### Scenario: Mismatched platform binary is caught
- **WHEN** the rollup native binary for the current `process.platform`/`arch` is
  not resolvable
- **THEN** the doctor exits non-zero and prints the exact fix
  (`rm -rf node_modules && npm ci` from the canonical environment)

#### Scenario: Healthy environment passes silently
- **WHEN** the correct platform native binary is installed
- **THEN** the doctor exits zero and `dev`/`build` proceed

### Requirement: Reliable dev-server file watching on mounted filesystems
The Vite dev server SHALL detect source edits reliably when the repository is on
a mounted filesystem (e.g. `/mnt/c` under WSL), so HMR does not serve stale
modules.

#### Scenario: Edit on a mounted filesystem triggers reload
- **WHEN** a source file is edited while the repo is on `/mnt/c` and the dev
  server is running
- **THEN** the dev server detects the change and updates the module without a
  manual restart

### Requirement: Scoped dependency scanning
Vite dependency optimization SHALL scan only the real SPA entry (`index.html`),
so unrelated `*.html` files anywhere in the repository do not break dependency
discovery.

#### Scenario: Stray HTML files do not break startup
- **WHEN** the repository contains `*.html` files outside the SPA entry (docs,
  reports, samples)
- **THEN** `npm run dev` and `npm run build` start dependency optimization
  against `index.html` only and do not fail scanning those files

### Requirement: Production bundle contains no mock or placeholder infrastructure
The production build output SHALL NOT contain mock-service (MSW) runtime code or
placeholder/sample data in executable bundle files.

#### Scenario: No mock runtime in shipped JS
- **WHEN** `npm run build` completes
- **THEN** no `dist/assets/*.js` runtime file contains MSW worker code
  (`setupWorker`, `msw/browser`) and `dist/mockServiceWorker.js` is absent
