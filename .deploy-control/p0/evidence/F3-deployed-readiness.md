# F3 — Deployed OmniRoute Readiness Proof for Task 6.1 (F3-DEPLOYED-READINESS)

- agent: `Codex56#A` · lane: `F3` · pane: `w7:p3` · task: `F3-DEPLOYED-READINESS`
- lock: `.deploy-control/p0/evidence/F3-deployed-readiness.md` (only mutable artifact) + receipts
- covers OpenSpec: **6.1** (immutable revision + protocol/model readiness) — **against the DEPLOYED instance**
- MODE: read-only repo/ORQ1 probing; **zero inference, zero secret, no deploy/restart/Docker**. Bodies never printed.

> STATUS: **BLOCKED (external) — CORRECTED RE-PROBE (§5 authoritative).** The authoritative ORQ1
> Tailscale endpoint `http://100.118.244.61:20128` **IS reachable**; reachability and API-key
> enforcement are now PROVEN deployed. Immutable revision + selected model/protocol readiness remain
> UNPROVEN because that surface is auth-gated (`/v1/models` → 401) and this lane may not use the secret.
> **§1–§3 below used the WRONG topology (ORQ2 loopback + retired LAN) and are SUPERSEDED by §5.**

---

## 0. Preflight

| Field | Value |
|---|---|
| pane | `w7:p3` |
| cwd | `/home/ec2-user/workspace/R-D_Agnostic_Engineering_Team` |
| git HEAD | `a6d50986aee7f1981a323b29c9d90f175b4b6f8b` |
| git status count | 331 |
| go | `go1.26.1` (`/home/ec2-user/goroot/go/bin/go`) |
| disk | root 187M (100% used); `/tmp` 7.6G free |

---

## 1. Deployed-endpoint probes (read-only, non-inference, no bodies)

Endpoints derived from source config (README host default `AGENT_BRAIN_GATEWAY_BASE_URL=http://127.0.0.1:20128`;
`gateway/client.go` EndpointSet Liveness/Readiness; architect-declared LAN `192.168.1.27:20128`).

Environment: no `AGENT_BRAIN_GATEWAY_BASE_URL`/`OMNIROUTE*` in this pane.
Listening sockets: **nothing listening on `:20128` or `:8080`** (`ss -ltnp` / `netstat`).

| Endpoint | Purpose | HTTP status | curl rc | Interpretation |
|---|---|---:|---:|---|
| `http://127.0.0.1:20128/health/live` | liveness (unauthenticated) | `000` | 7 | connection refused — no listener |
| `http://127.0.0.1:20128/health/ready` | readiness/catalog | `000` | 7 | connection refused — no listener |
| `http://127.0.0.1:20128/v1/models` | registry/models metadata | `000` | 7 | connection refused — no listener |
| `http://192.168.1.27:20128/health/live` | LAN liveness | `000` | 28 | timeout — not reachable |
| `X-OmniRoute-Registry-Version` header | immutable revision | — | — | no response; header unavailable |

Commands (exit codes captured; bodies discarded with `-o /dev/null`; header probe used `-D -`
filtered to the single registry-version header only):
```
env | grep -i 'AGENT_BRAIN_GATEWAY_BASE_URL\|OMNIROUTE'   -> (none)
ss -ltnp | grep -E ':20128|:8080'                          -> (nothing listening)
curl -sS -o /dev/null -w '%{http_code}' --max-time 4 <url> -> 000 (rc 7 / 28)
curl -sS -D - -o /dev/null --max-time 4 .../v1/models | grep -i X-OmniRoute-Registry-Version -> (none)
```

---

## 2. 6.1 deployed-proof result (per required fields)

| Required field (6.1) | Deployed result |
|---|---|
| endpoint | `127.0.0.1:20128` (host default) and `192.168.1.27:20128` (LAN) |
| HTTP status | `000` on all (rc 7 connection refused / rc 28 timeout) |
| immutable image/revision | **UNPROVEN** — no reachable endpoint; no `X-OmniRoute-Registry-Version` obtainable; image digest not verifiable from this host without Docker (prohibited) |
| health/readiness | **UNPROVEN** — liveness/readiness unreachable |
| registry revision consistency (header vs body) | **UNPROVEN** — no response |
| selected-model availability (`cp/cline-pass/glm-5.2`) | **UNPROVEN** — registry not reachable |
| selected-protocol readiness (openai-chat) | **UNPROVEN** — registry not reachable |

Deployed proof of 6.1 is therefore **not attainable** in the current environment.

---

## 3. Concrete external blocker

- **Blocker:** the deployed OmniRoute runtime is not reachable from pane `w7:p3` — nothing is
  listening on `127.0.0.1:20128` (rc 7 connection refused) and the declared LAN `192.168.1.27:20128`
  times out (rc 28). No non-secret liveness endpoint responds, so immutable revision, health/readiness,
  registry revision consistency, and selected model/protocol readiness cannot be proven against the
  deployed instance without inference.
- **Owner:** OmniRoute operator / Principal Orchestrator (`w5:p9`) for endpoint provisioning; ORQ1 host
  owner for reachability.
- **Next action:** expose a reachable OmniRoute endpoint to this pane (host loopback `127.0.0.1:20128`
  or the declared LAN address) and confirm at least the unauthenticated liveness path; provide non-secret
  guidance for the authenticated readiness/`/v1/models` probe under `REQUIRE_API_KEY=true` (readiness
  gate needs auth). Then F3 re-runs the §1 probes and records real HTTP status + `X-OmniRoute-Registry-Version`
  + bounded selected-model/protocol availability. Additionally gated by D-V3-25(B) key revocation and
  `live_runs.*=false` for any authenticated step.

---

## 4. Non-claims / limitations

- No inference, no secret, no deploy/restart/Docker/systemd; no bodies printed.
- Source-level 6.1 contract correctness is evidenced in `.deploy-control/p0/evidence/R3-deploy-readiness.md`
  (gateway `ReadinessChecker` fail-closed gates + immutable `registry.Version` handling). **That is
  source proof, NOT deployed proof**, per this lane's explicit standard.
- No repository files edited; F3 is read-only + this evidence artifact.
- Prior assignment P0-GLM-LIVE-ACCEPTANCE remains BLOCKED; no lock overlap.

---

## 5. CORRECTED RE-PROBE — authoritative ORQ1 Tailscale endpoint (supersedes §1–§4 topology)

Corrected authoritative endpoint (manager, 2026-07-22T11:36Z): **`http://100.118.244.61:20128`**
(ORQ1 Tailscale). Prior §1 probes used the wrong topology (ORQ2 `127.0.0.1` + retired LAN
`192.168.1.27`) and are invalid. Re-probe is read-only, non-secret, no inference, no bodies printed.
Resumed via `p0_control.py heartbeat --resume`.

### 5.1 Deployed probe results (status only; headers = metadata only)

| Endpoint (`http://100.118.244.61:20128`) | HTTP status | curl rc | Note |
|---|---:|---:|---|
| `/health/live` | 404 | 0 | source-configured path not served on this deployment |
| `/health/ready` | 404 | 0 | idem |
| `/health` | 404 | 0 | — |
| `/healthz` `/livez` `/readyz` `/api/health` `/health/liveness` `/health/readiness` `/ping` `/version` `/api/version` `/metrics` | 404 | 0 | no standard non-secret health/version path found |
| `/status` | 200 | 0 | `Content-Type: text/html` (dashboard UI, not machine-readable status; no version field) |
| `/` | 307 | 0 | `Location: /dashboard` (management UI) |
| `/v1/models` | **401** | 0 | **API-key required (REQUIRE_API_KEY enforced)**; header `x-omniroute-route-class: CLIENT_API`; NO `X-OmniRoute-Registry-Version`, no `Server`, no `WWW-Authenticate` on the unauthenticated 401 |
| `/v1/health` | 401 | 0 | auth-gated |

Latencies sub-40ms; `curl_rc=0` on all → the endpoint is genuinely reachable (TCP+HTTP responding).

### 5.2 What is PROVEN deployed (non-secret)

- **Reachability:** `http://100.118.244.61:20128` responds over Tailscale (rc=0). Endpoint is live.
- **API-key enforcement / fail-closed for unauthenticated:** `/v1/models` and `/v1/health` return
  `401` without a key — consistent with `REQUIRE_API_KEY=true`.
- **OmniRoute client-API surface confirmed:** `/v1/models` carries `x-omniroute-route-class: CLIENT_API`.
- Management UI present (`/` → 307 `/dashboard`, `/status` 200 HTML).

### 5.3 What remains UNPROVEN (and why) — precise new blocker

| 6.1 field | Deployed result | Reason |
|---|---|---|
| immutable image/revision | UNPROVEN | no unauthenticated revision/version endpoint; `X-OmniRoute-Registry-Version` only on an **authenticated** `/v1/models` response; image digest needs Docker (prohibited) |
| health/readiness | PARTIAL | reachability + 401 enforcement proven; source health path (`/health/live|ready`) returns 404 here, so the deployment's exact readiness path is unknown/undisclosed |
| registry revision consistency (header vs body) | UNPROVEN | requires authenticated `/v1/models` (header + body); unauthenticated 401 carries neither |
| selected-model availability (`cp/cline-pass/glm-5.2`) | UNPROVEN | registry rows are behind the auth-gated `/v1/models` |
| selected-protocol readiness (openai-chat) | UNPROVEN | idem |

### 5.4 NEW concrete blocker (supersedes §3)

- **Blocker:** the ORQ1 OmniRoute endpoint is reachable but the registry/revision/readiness surface is
  **auth-gated** — `/v1/models` → `401` without the scoped OmniRoute key, and no non-secret
  liveness/readiness/version path is exposed (source `/health/live|ready` → 404; only an HTML
  `/status` and `/dashboard`). This lane must not read/use the secret and must not run inference, so
  immutable revision, registry consistency, and selected model/protocol availability cannot be proven
  deployed by this lane alone.
- **Owner:** OmniRoute operator (endpoint/health exposure) + Principal Orchestrator `w5:p9`
  (authorization for a non-inference authenticated readiness read).
- **Next action (any ONE unblocks):**
  1. Operator exposes a **non-secret** liveness/readiness/revision endpoint (or documents the real
     health path on this deployment) that returns the `X-OmniRoute-Registry-Version` and per-model
     availability/protocol without a key; **or**
  2. Principal authorizes a **single non-inference** authenticated `GET /v1/models` (metadata read
     only, no chat/completions) performed by an auth-permitted lane, capturing the registry version +
     `cp/cline-pass/glm-5.2` availability + protocol — headers/bounded fields only, never raw body,
     still `live_runs.*=false` (a models read is not inference); **or**
  3. Operator publishes the pinned image digest + registry revision out-of-band as signed metadata.
  Still additionally gated by D-V3-25(B) key revocation for any authenticated step.
- **Standard reaffirmed:** source-only proof (`R3-deploy-readiness.md`) is NOT deployed proof; this
  section records the real deployed endpoint, HTTP statuses, and metadata actually observed.

---

## 6. Principal adjudication (2026-07-22T11:44Z) — FINAL for this window

- **Verdict:** partial **non-secret deployed proof ACCEPTED**; task **6.1 stays OPEN / BLOCKED**.
- **Accepted deployed evidence:** `/status` → 200 (mgmt UI); `/v1/models` → 401 with `REQUIRE_API_KEY`
  enforcement + `x-omniroute-route-class: CLIENT_API`; `/` → 307 `/dashboard`; ORQ1 Tailscale
  `http://100.118.244.61:20128` reachable (rc=0).
- **NON-CLAIMS (explicitly NOT proven):** immutable revision, registry revision consistency,
  selected-model (`cp/cline-pass/glm-5.2`) availability, selected-protocol (openai-chat) readiness.
- **Blocker (external owner action required; NEITHER authorized):**
  - Option A: a sanitized **non-secret readiness/revision endpoint** exposing revision + per-model
    availability/protocol without a key; OR
  - Option B: an owner-**authorized credentialed metadata-only** (`/v1/models`) token, non-inference.
  Owner: OmniRoute operator + Principal Orchestrator (`w5:p9`). D-V3-25(B) gates any credentialed step.
- **Directive honored:** no credential read/use/copy, no deploy/endpoint change, no inference, no
  further probing. Lane terminal = BLOCKED; evidence preserved for F8.
