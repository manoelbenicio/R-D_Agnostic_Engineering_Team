# AgentVerse — Technical Architecture (Local vs Cloud)

This document maps **every component** and **where it lives** in the two
deployment topologies: **Local (Docker on your machine)** and **Cloud
(GCP: Cloud Run + Firebase Hosting)**. Read it side-by-side: the *same code*
runs in both; only the *hosting* and *auth* change.

---

## 1. The three layers (conceptual)

```
┌──────────────────────────────────────────────────────────────────────┐
│  LAYER 1 — SPA (frontend)         AgentVerse React app (this repo, src/)│
│            React + Vite + Zustand + react-query                         │
│            Talks to the runtime over HTTP/REST + WebSocket              │
├──────────────────────────────────────────────────────────────────────┤
│  LAYER 2 — RUNTIME (backend)      cli-agent-orchestrator ("CAO")        │
│            Python/uvicorn server, port 9889 (local) / $PORT (cloud)     │
│            Owns sessions, terminals (tmux), profiles, flows             │
│            Bundled worker CLIs: codex · kiro-cli · agy (Antigravity)    │
├──────────────────────────────────────────────────────────────────────┤
│  LAYER 3 — WORKERS (agents)       The CLIs the runtime spawns in tmux   │
│            codex (OpenAI) · kiro-cli (Kiro/Opus) · agy (Antigravity)    │
│            Authenticated via mounted creds (local) / secrets (cloud)    │
└──────────────────────────────────────────────────────────────────────┘
```

- **Layer 1 (SPA)** is what you build from this repository (`src/`).
- **Layer 2 (Runtime)** is the upstream `cli-agent-orchestrator` package,
  wrapped + patched by `infra/runtime/Dockerfile`.
- **Layer 3 (Workers)** are the agent CLIs the runtime launches inside tmux
  sessions to actually do work.
h
---

## 2. Component inventory (where each thing lives)

| Component | Source in repo | Local home | Cloud home |
|-----------|----------------|------------|------------|
| SPA bundle | `src/`, built to `dist/` | Vite dev server `:5173` (or `dist/` served by any static host) | **Firebase Hosting** (`dist/`) |
| Runtime engine (CAO) | `infra/runtime/Dockerfile` (wraps `cli-agent-orchestrator`) | Docker container `:9889` on your machine | **Cloud Run** service, `$PORT` |
| Worker CLIs | bundled via `WORKER_CLI` build-arg | inside the container image | inside the *same* image on Cloud Run |
| CLI credentials | `~/.codex`, `~/.kiro`, `~/.gemini`, `~/.aws` | **bind-mounted** into the container | **Secret Manager** → env (not mounted) |
| Runtime state/DB/logs | CAO writes to `/root/.aws/cli-agent-orchestrator` | Docker **named volume** `agentverse-runtime-state` | Cloud Run gen2 volume / GCS (stateless otherwise) |
| SPA persistence (canvases, keys, usage) | `src/shared/storage/` (IndexedDB) | **browser IndexedDB** | **browser IndexedDB** (same — it's client-side) |
| Auth | `src/shell/auth*.ts` | **disabled / optional** (`VITE_AUTH_REQUIRED=false`) | **Firebase Auth** + JWT, enforced |
| FinOps token cost | `src/finops/` | computed in browser from usage events | same (client-side) |
| Topology guard | `src/shared/topology-guard.ts` + `validation-proxy.ts` | runs SPA-side | SPA-side (+ optional CAO-side proxy, deferred) |

> **Key insight:** IndexedDB, FinOps, topology guard, and the whole SPA are
> **client-side** — they live in the *browser*, identical in local and cloud.
> What moves between local and cloud is only the **runtime hosting** and the
> **credential/auth mechanism**.

---

## 3. LOCAL topology (Docker on your machine)

```
   YOUR MACHINE (WSL/Windows)
   ┌─────────────────────────────────────────────────────────────────┐
   │                                                                   │
   │  Browser ──HTTP──► Vite dev server  http://localhost:5173         │
   │     │              (SPA, `npm run dev`)                           │
   │     │  IndexedDB (canvases, provider keys, usage events)          │
   │     │                                                             │
   │     └──HTTP/WS──► Docker container  http://localhost:9889         │
   │                   ┌──────────────────────────────────────────┐   │
   │                   │ agentverse-runtime:local                  │   │
   │                   │  • cao-server (uvicorn :9889)             │   │
   │                   │  • tmux sessions                          │   │
   │                   │  • workers: codex · kiro-cli · agy        │   │
   │                   └──────────────────────────────────────────┘   │
   │                          ▲ bind mounts (read-only)                │
   │     ~/.codex  ~/.kiro  ~/.gemini  ~/.aws/{credentials,config} ─────┘
   │     named volume: agentverse-runtime-state → /root/.aws/cli-agent-orchestrator
   └───────────────────────────────────────────────────────────────────┘
```

**How auth works locally (your question):** the worker CLIs use the **exact
same login as your terminal/IDE**. We bind-mount your `~/.codex`, `~/.kiro`,
`~/.gemini` (and AWS credential files) into the container, so `codex`,
`kiro-cli` and `agy` see the credentials you already authenticated on the host.
No tokens, no re-login. (`~/.aws` is mounted *file-by-file* read-only because
the runtime needs to write its own DB/logs under `/root/.aws`.)

**Commands (local):**
```bash
bash infra/runtime/run-local.sh build   # build image with the 3 CLIs (one-off)
bash infra/runtime/run-local.sh up       # runtime on :9889 + mount your creds
npm run dev                              # SPA on :5173  (VITE_CAO_BASE_URL=http://127.0.0.1:9889)
bash infra/runtime/run-local.sh logs     # follow runtime logs
bash infra/runtime/run-local.sh down     # stop + remove container
```

**Local facts (verified):** image builds; `/health` → `200 {"status":"ok"}`;
`codex` @ `/usr/bin/codex`, `kiro-cli` + `agy` @ `/root/.local/bin`.

---

## 4. CLOUD topology (GCP)

```
   INTERNET
   ┌──────────────────────────────────────────────────────────────────┐
   │  Browser ──HTTPS──► Firebase Hosting   https://<project>.web.app   │
   │     │               (static dist/, SPA fallback, CSP headers)      │
   │     │  IndexedDB (same client-side stores as local)                │
   │     │                                                              │
   │     │  Firebase Auth (Google sign-in) → JWT                        │
   │     │                                                              │
   │     └──HTTPS/WSS (Bearer JWT)──► Cloud Run  https://<svc>-run.app  │
   │                                  ┌─────────────────────────────┐   │
   │                                  │ agentverse-runtime image    │   │
   │                                  │  • cao-server (:$PORT)      │   │
   │                                  │  • tmux + workers           │   │
   │                                  │    codex · kiro-cli · agy   │   │
   │                                  └─────────────────────────────┘   │
   │                                     ▲ env from Secret Manager       │
   │                          (per-tenant secrets, CORS origin, keys)    │
   │                          state → Cloud Run gen2 volume / GCS        │
   └──────────────────────────────────────────────────────────────────┘
```

**What changes vs local:**
- SPA is **pre-built** (`npm run build` → `dist/`) and served by **Firebase
  Hosting**, not the Vite dev server.
- Runtime is the **same Docker image**, but pushed to Artifact Registry and
  run on **Cloud Run** (binds `$PORT`, not 9889).
- **Credentials are NOT bind-mounted** (there's no host filesystem). Worker
  CLI auth must come from **Secret Manager → env vars** (this is the headless-
  auth requirement — still a pending decision/blocker).
- **Auth is enforced**: `VITE_AUTH_REQUIRED=true`, Firebase JWT attached to
  every runtime call by `src/shell/app-fetch.ts`; Cloud Run rejects
  unauthenticated requests (IAM vs custom proxy — pending decision).
- **Per-tenant isolation**: one Cloud Run service per tenant
  (`agentverse-runtime-${TENANT_ID}`), parameterised in
  `infra/runtime/service.yaml`.

**Commands (cloud):**
```bash
./start.sh cloud-deploy   # build SPA → Cloud Build → Cloud Run → Firebase Hosting
./start.sh cloud          # open the deployed URL
```

**Cloud IaC files:** `infra/runtime/{Dockerfile,cloudbuild.yaml,service.yaml}`,
`scripts/deploy-cloud.sh`, `firebase.json`, `.firebaserc`.

---

## 5. Local vs Cloud — quick diff

| Concern | Local (Docker) | Cloud (GCP) |
|---------|----------------|-------------|
| SPA host | Vite dev server `:5173` | Firebase Hosting (CDN) |
| Runtime host | Docker container `:9889` | Cloud Run `:$PORT` |
| Runtime image | `agentverse-runtime:local` | same image in Artifact Registry |
| Worker auth | bind-mount host `~/.codex` etc. | Secret Manager → env (**pending**) |
| App auth | optional / off | Firebase JWT, enforced |
| Cost | free | billable (Cloud Run, Build, Hosting) |
| State | named Docker volume | Cloud Run volume / GCS |
| Multi-tenant | single instance | one service per tenant |
| Command | `run-local.sh up` + `npm run dev` | `./start.sh cloud-deploy` |

---

## 6. Request flow (end-to-end, both topologies)

1. User opens the SPA (browser) → React boots, reads IndexedDB.
2. SPA calls the runtime at `VITE_CAO_BASE_URL` (`127.0.0.1:9889` local /
   Cloud Run URL in cloud).
3. Runtime creates a **session** + **terminals** (tmux), launching a worker
   CLI (`codex` / `kiro-cli` / `agy`) per canvas node.
4. Worker authenticates using mounted creds (local) or env secrets (cloud).
5. Terminal output streams back to the SPA over WebSocket; the terminal grid
   renders it via xterm.
6. FinOps records token usage (when the runtime exposes it — **pending CAO
   side**) into IndexedDB; the topology guard validates agent-to-agent calls
   against the deployed canvas edges.

---

## 7. Status & open items (as of this writing)

- ✅ **Local runtime**: builds, healthy, 3 CLIs bundled, host creds mounted.
- ✅ **SPA**: 399 tests passing, builds, runs against the local runtime.
- ⏳ **Cloud auth (headless worker creds)**: needs Secret Manager wiring +
  decision (Cloud Run IAM vs custom proxy).
- ⏳ **Worker CLI choice for cloud image**: same 3 CLIs, but cloud needs
  non-interactive auth tokens.
- ⏳ **CAO-side usage capture & topology enforcement**: depends on the runtime
  exposing per-turn usage and an interception point (CAO source not in this
  repo).

> See `BACKLOG.md` and `openspec/changes/*/tasks.md` for the per-item detail
> and what is blocked on external factors (GCP creds, CAO source).
