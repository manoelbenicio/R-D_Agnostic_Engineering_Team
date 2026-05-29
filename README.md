# AgentVerse

A multi-agent orchestration SPA on top of CAO. v1 ships the full surface
described in `openspec/changes/milestone-1-canvas-deploy-run/`.

## Requirements

- Node ≥ 20.10
- npm ≥ 10.2
- A reachable CAO server (default: `http://127.0.0.1:9889`).

## Install

```bash
npm ci
```

`npm ci` (not `npm install`) so the committed `package-lock.json` is the
source of truth.

## Run

```bash
cp .env.example .env.local        # configure VITE_CAO_BASE_URL if needed
npm run dev                       # http://localhost:5173
```

The dev server is fixed to port 5173 to match the documented CAO
`CAO_CORS_ORIGINS` allow-list. See `docs/cao-cors.md`.

## Test

```bash
npm run lint              # ESLint + agentverse local rules
npm run typecheck         # TS strict project references
npm run format:check      # prettier
npm test                  # vitest unit + MSW integration
npm run test:smoke        # playwright (boots dev server)
CAO_LIVE=1 npm run test:contract   # live CAO contract suite
```

## Build

```bash
npm run build             # → dist/
node scripts/check-bundle-size.mjs   # 1.5 MB gzipped budget
```

## Layout

See [`ARCHITECTURE.md`](./ARCHITECTURE.md). Capability ownership is in
[`.github/CODEOWNERS`](./.github/CODEOWNERS).

```
src/
├── shell/              # SUP — routing, layout, error boundary, fetch wrapper
├── design-system/      # SUP — locked tokens + base components
├── shared/             # SUP — cross-cutting types, IDB infra
├── api/                # IF  — CaoClient, KeyStore, MSW + contract tests
├── settings/           # IF  — settings store + pages
├── canvas-builder/     # CV
├── canvas-document/    # CV
├── canvas-reconciler/  # CV
├── canvas-templates/   # CV
├── terminal/           # TM
├── terminal-grid/      # TM
├── chat-view/          # TM
├── dashboard/          # DB
├── finops/             # DB
├── health/             # DB
├── agent-studio/       # ST
├── flows/              # ST
├── memory-viewer/      # ST
└── voice/              # VX
```

## Documentation

- [`ARCHITECTURE.md`](./ARCHITECTURE.md) — D1–D15 + R1–R9 summary.
- [`docs/patterns/`](./docs/patterns/) — established conventions per topic.
- [`docs/cao-cors.md`](./docs/cao-cors.md) — CAO env vars to allow this SPA.
- [`docs/key-storage-v1.md`](./docs/key-storage-v1.md) — v1 BYOK threat model.
- `openspec/changes/milestone-1-canvas-deploy-run/` — authoritative spec.
