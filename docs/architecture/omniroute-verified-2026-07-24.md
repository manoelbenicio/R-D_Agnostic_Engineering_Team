# OmniRoute Architecture — Live MCP Verification (2026-07-24)

Verified read-only against the running OmniRoute instance via its MCP server
(`/api/mcp/stream`, initialize handshake + `tools/list` + read-only tool calls).
No provider/credential/secret inspection; no changes made.

## Endpoint / liveness
- Gateway published at `http://100.118.244.61:20128` (Tailscale IP) — **NOT** on `127.0.0.1`.
  Anything checking `localhost:20128` will fail; use `100.118.244.61` (or point
  `AGENT_BRAIN_GATEWAY_BASE_URL` there).
- `omniroute` container: **Up 2 days (healthy)**.
- `GET /api/health/ping` = **200**; `GET /v1/models` = **401** (up; needs API key).
- MCP `initialize` = live → `serverInfo {name: omniroute, version: 1.8.1}`; gateway version **3.8.48**;
  **99 MCP tools** advertised (doc catalog says 104 — minor version drift).
- `omniroute_get_health`: cryptography **aes-256-gcm healthy**, circuit breakers **[] (none open)**,
  rate limits **[] (none active)**.

## Provider / model inventory (`omniroute_list_models_catalog`)
- **144 models total.** Providers: `nvidia` = 119 (mostly `available`), `agy` (Antigravity) = 15 (`degraded`),
  `clinepass` = 10. Status split: 119 `available`, 25 `degraded`.
- **Codex / OpenAI-family:** only `openai/gpt-oss-120b` and `openai/gpt-oss-20b` on **nvidia** (`available`)
  and `gpt-oss-120b-medium` on `agy` (`degraded`). These are OpenAI-**architecture open-weight** chat models
  served over the chat format — **there is NO OpenAI Codex / Responses-protocol route**.

## Combo topology (`omniroute_list_combos`)
- `claude_code_kimi_2.7_Code` — strategy **priority**, enabled:
  1. `group2-nvidia-glm5.2-fallback`  (priority 1)
  2. `cline-kimi-k2.7-dedicated`      (priority 2 / fallback)
- `group2-nvidia-glm5.2-fallback` — **round-robin** across 3× `nvidia/z-ai/glm-5.2`.
- `cline-kimi-k2.7-dedicated` — **round-robin** across 4× `clinepass/cline-pass/kimi-k2.7-code`.
- `kimi-sub` — **priority**: `cline-kimi-k2.7-dedicated` → `group2-nvidia-glm5.2-fallback`.

**Interpretation:** our daemon's `AGENT_BRAIN_ROUTE_MODEL=claude_code_kimi_2.7_Code` is an OmniRoute
**priority combo**, not a single model. `AGENT_BRAIN_CLI_KIND=claude-code` speaks the Anthropic Messages
protocol to the gateway; OmniRoute resolves the combo to **NVIDIA GLM-5.2 (primary) → Kimi K2.7 via clinepass
(fallback)**. Main Brain stays **credentialless** — OmniRoute owns all provider auth, selection, fallback and quota.

## Architecture invariants (owner-confirmed + MCP-verified)
- OmniRoute is the **sole router** and credential owner; Main Brain connects to the one HTTP endpoint and is
  **credentialless** (no model-side login; does not know auth type/provider).
- Runtime online/offline + credential START/STOP are **OmniRoute's domain**, not Main Brain's.
- `nim` native runtime was **removed** from the server (NVIDIA is reached as an OmniRoute provider —
  confirmed: `nvidia` is a live provider with 119 models). See the nim-removal change (Lane A).
- Main Brain keeps **strict fail-closed readiness**; OmniRoute is fail-open internally for routing continuity.

## OpenSpec open-question reconciliation
| Change / task | Status | Verified against OmniRoute |
|---|---|---|
| build-omniroute-agent-brain **6.3** Tier-20 acceptance | OPEN | Not an OmniRoute question — needs a real instrumented Tier-20 run (gated on an owner-approved consistent deploy). |
| build-omniroute-agent-brain **6.4** Tier-50/100 | OPEN | Gated behind 6.3. |
| chat-orchestration-standard **2.2** `@codex` direct | OPEN — **externally blocked (verified)** | OmniRoute exposes **no Codex/OpenAI-Codex route**; only NVIDIA `gpt-oss` chat models. Cannot close without an approved+ready Codex route (OmniRoute-side). |
| chat-orchestration-standard **2.3** final receipts | OPEN | Blocked behind 2.2. |
| native-runtimes-onboarding **2.4 / 3.2** (nim/cline online) | OPEN — **stale vs current architecture** | `nim` removed; NVIDIA is an OmniRoute provider; runtime online/offline is OmniRoute's domain. Needs owner scope decision (rewrite/close vs excluded). |
| native-runtimes-onboarding **1.5 / 1.6 / 2.5 / 3.1 / 3.3 / 3.4** | OPEN — **excluded scope** | Frontend onboarding / design parity / UAT — explicitly excluded unless owner reverses. |

_Method: OmniRoute MCP read-only tools (`tools/list`, `get_health`, `list_models_catalog`, `list_combos`)
on 2026-07-24 ~12:03–12:06Z. No writes, no secrets._

## Per-topic verification log

### Topic 1 — MB 6.3 (Tier-20 capacity acceptance) — verified 2026-07-24 ~12:10Z
OmniRoute-side readiness for the `claude_code_kimi_2.7_Code` route (read-only MCP):
- `check_quota`: route connections (clinepass Kimi K2.7, agy, NVIDIA GLM-5.2) **token valid, 100% remaining**.
- `get_provider_metrics` nvidia & clinepass: **circuit breaker CLOSED, successRate 1.0, errorRate 0** (requestCount 0 = no load yet).
- `simulate_route` (2000 tok): primary `group2-nvidia-glm5.2-fallback` (p=0.85) → fallback `cline-kimi-k2.7-dedicated` (p=0.15); both healthy, quota 100, ~$0.006/req.

**Verdict:** OmniRoute agrees the route can sustain Tier-20 (healthy, full quota, breakers closed, fallback resolves).
**6.3 remains OPEN** — closing requires a real instrumented Tier-20 run (20 concurrent leases, continuous 9-ID trace,
20 persisted terminal rows, dropped=0, independent audit), which is gated on an **owner-approved consistent deploy**
(fresh daemon+backend binaries; currently frozen). No checkbox changed.

### Stack state — recovered & version-consistent (verified 2026-07-24 ~12:27Z)
After a backend crash-loop (recreate had not loaded the canonical env-file), forward-fix applied with
`--env-file /home/ec2-user/.config/multica-transition/dev.env` + new backend image. Independently verified:
- Backend container **Up**, `/health` = **200** (port 18080); OmniRoute gateway ping **200**.
- Daemon `status=running`, `router_owner=omniroute`, `admission_limit=20`, `readiness=unavailable`
  (= expected lazy fail-closed default, not a regression).
- **Version-consistent, same-tree (post-nim-removal) pair:** daemon binary `b635556…` + backend server
  binary `a05415c4…` (authoritative docker build; differs from the earlier local `dc8b28d1` which predates
  nim removal). Backend image id `994aa2284b55`; frontend `cf8017e3d2fd`.
- Known hygiene item (deferred, non-disruptive): running image **tag name** `multica-backend:t20obs-8f524054`
  embeds an OLD server hash though the image content is the new `994aa2284b55`; owner-lead will retag +
  update `images.yml` in one step during an idle window.
- Recovery guardrails that held: single-source env-file, bounded health gate, armed auto-rollback (unused).
- JWT preserved (canonical dev.env reused) → no user logout.


---
**Deploy hard rule (see `docs/deploy/rollback-runbook.md` + `docs/deploy/INCIDENTS.md`, 2026-07-24):** all backend/daemon container recreates MUST use `--env-file /home/ec2-user/.config/multica-transition/dev.env` as the single env source — a bare `docker compose up` drops deployment secrets and crash-loops the service.
