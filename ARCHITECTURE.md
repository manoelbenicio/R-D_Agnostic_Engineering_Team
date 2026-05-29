# AgentVerse v1 Architecture

This document summarises the cross-cutting decisions (D1–D15) and risks
(R1–R9) that span the v1 capabilities. Source of truth lives in the
OpenSpec change `milestone-1-canvas-deploy-run/design.md`.

## Decisions (D1–D15)

| # | Decision                                  | One-line                                                                     |
| - | ----------------------------------------- | ---------------------------------------------------------------------------- |
| D1  | **Stack**                              | React 18 + Vite + TypeScript (master spec v4.2). Rework risk accepted.       |
| D2  | **Canvas graph library**               | `@xyflow/react` for node-graph editor primitives.                            |
| D3  | **State**                              | Zustand for UI state · TanStack Query for server state · `useState` local.   |
| D4  | **Local persistence**                  | IndexedDB via `idb`, behind typed interfaces (cloud-swap later).             |
| D5  | **Atomic deploy state**                | Persist `deploy_state` before AND after every CAO call.                      |
| D6  | **No Validation Proxy in v1**          | Supervisor obedience is best-effort via prompt-level topology.               |
| D7  | **WebGL mandatory**                    | Production refuses Canvas2D fallback. Dev opt-in via `VITE_ALLOW_CANVAS2D`.  |
| D8  | **Parallel-from-day-zero**             | All owners share `main`; supervisor reviews every PR.                        |
| D9  | **`src/shared/` is gated**             | Only the supervisor merges shared types; no sideways capability imports.    |
| D10 | **Testing strategy**                   | Vitest · MSW · Playwright · `CAO_LIVE=1` contract suite (nightly).          |
| D11 | **CAO client surface**                 | Full v1 endpoint set inside `src/api/cao-client.ts` — only entry point.     |
| D12 | **Single-branch coordination**         | No long-lived feature branches. Broken main triggers immediate revert.       |
| D13 | **Inter as default sans, JetBrains Mono reserved for code/terminal** | Deviation from original SENTINEL default. See `openspec/changes/design-system-indra-alignment/`. User-configurable in Settings → Appearance via `applyFontOverrides()`. |
| D14 | **Diff-based edit-after-deploy**       | Reconciler applies the delta; full redeploy is rarely needed.                |
| D15 | **Voice NLU on user's BYOK key**       | Cheapest validated model wins (Gemini Flash → GPT-4o-mini → Haiku).         |

## Risks (R1–R9)

| # | Risk                                                | Mitigation                                                            |
| - | --------------------------------------------------- | --------------------------------------------------------------------- |
| R1 | Stack rework risk                                   | Framework-agnostic layers (CAO client, schema, IDB, reconciler).      |
| R2 | Pattern divergence under parallel-from-day-zero     | Supervisor review of every PR · `docs/patterns/` · early-cycle thrash budgeted. |
| R3 | Supervisor LLM ignores topology (no Validation Proxy) | Topology baked into supervisor prompt · Validation Proxy on roadmap. |
| R4 | Same-branch breakage                                | CI gates merge; supervisor reverts promptly.                          |
| R5 | Diff-based edit under partial failure               | Diff against canonical `terminal_map`, not fresh GET; user ack required. |
| R6 | No incremental customer feedback                    | Playwright smoke is the synthetic customer; weekly demos.             |
| R7 | WebGL hardware diversity                            | Production refuses fallback; CI runs WebGL-forced Chrome.             |
| R8 | CAO API drift                                       | Nightly `CAO_LIVE=1` contract suite (D10).                            |
| R9 | Plaintext key leak via dev-tools                    | Documented v1 threat model; UI masks values; encrypted Firestore later. |

## Owners (master spec §14.1)

- **SUP** — supervisor: `src/shell/`, `src/design-system/`, `src/shared/`, CI, docs.
- **CV** — canvas: `src/canvas-builder/`, `src/canvas-document/`, `src/canvas-reconciler/`, `src/canvas-templates/`.
- **TM** — terminal: `src/terminal/`, `src/terminal-grid/`, `src/chat-view/`.
- **DB** — dashboard: `src/dashboard/`, `src/finops/`, `src/health/`.
- **ST** — studio: `src/agent-studio/`, `src/flows/`, `src/memory-viewer/`.
- **VX** — voice: `src/voice/`.
- **IF** — infra: `src/api/`, `src/settings/`.

## Quality gates

- `npm run lint` → ESLint + custom `agentverse/*` rules (D9, 4.13).
- `npm run typecheck` → strict TS, project references.
- `npm run test` → Vitest + MSW.
- `npm run test:smoke` → Playwright critical path.
- `npm run test:contract` → live CAO (gated `CAO_LIVE=1`).
- `npm run build` → Vite production bundle (≤ 1.5 MB gzipped budget).
- `node scripts/check-bundle-size.mjs` runs after build.

See `openspec/changes/milestone-1-canvas-deploy-run/design.md` for the
authoritative version of every decision.
