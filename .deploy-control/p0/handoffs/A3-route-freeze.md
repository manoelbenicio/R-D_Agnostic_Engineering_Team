# A3 — GLM/Kimi RouteModel Freeze (P0-ROUTE-FREEZE)

- agent: `Codex56#A`  ·  lane: `A3`  ·  task: `P0-ROUTE-FREEZE`  ·  pane: `w7:p3`
- output lock: `.deploy-control/p0/handoffs/A3-route-freeze.md` (this file, only mutable artifact)
- check-in: `.deploy-control/p0/checkins/Codex56-A__P0-ROUTE-FREEZE__20260721T223004Z.json`
- covers: `5.6` (Cline→GLM-5.2), `5.7` (Cline→Kimi-K2.7), `8.1` (exact model/protocol/availability)
- product source: READ-ONLY. This is a docs-only handoff to A1 (Cline base) / W1 (integrator).
- rule applied: exact provenance or `BLOCKED_EXTERNAL`; no alias chosen by plausibility; no ID invented.

> STATUS: **1 route FROZEN with provenance (GLM primary), 1 fallback FROZEN (OmniRoute-owned,
> not Brain-selectable), 1 route BLOCKED_EXTERNAL (Kimi exact model undeclared), availability
> BLOCKED_EXTERNAL for both Cline routes.** See §5 blockers.

---

## 0. Preflight (exact versions/paths/results)

| Tool | Path | Version | Result |
|---|---|---|---|
| HERDR_PANE_ID | env | `w7:p3` | matches ASSIGNMENTS row for Codex56#A ✓ |
| HERDR_ENV | env | `1` | Herdr context confirmed ✓ |
| git | `/usr/bin/git` | `2.50.1` | present ✓ |
| python3 | `/usr/bin/python3` | `3.9.25` | present ✓ (p0_control.py OK) |
| rg | `/usr/local/bin/rg` | `15.2.0` | present ✓ |
| herdr | `/home/ec2-user/.local/bin/herdr` | `0.7.4` | present ✓ |
| p0_control.py | `scripts/orchestration/p0_control.py` | — | present; check-in succeeded ✓ |

**Registry/catalog access check (no secrets):** No live OmniRoute registry endpoint was queried.
The OmniRoute runtime is a container at `http://127.0.0.1:20128` requiring the single scoped
inference secret (`REQUIRE_API_KEY=true`), and live-provider access is under the **D-V3-25(B)
security stop** until the exposed key is confirmed revoked. Per PROTOCOL/EVIDENCE_CONTRACT, the
authoritative offline source is the **frozen accepted evidence + acceptance checklist**, which is
what this freeze reconciles against. No credential value, prefix, header, or account identity is
recorded here.

---

## 1. OmniRoute source of truth (provenance chain)

Ordered by authority; each exact ID is cited to a specific line.

1. **`authoritative-route-matrix-D-V3-27.md`** — FROZEN owner decision (Decision 8). Names the
   route *families* only: row 4 `Cline (main provider/runtime) → Kimi-K2.7`; row 5 `Cline → GLM52`;
   row 6 `NVIDIA = GLM52 fallback`, selection/fallback **OmniRoute-owned & bounded**, Agent Brain
   holds no creds and makes no fallback decision. **No exact model-ID strings.**
2. **`omniroute-architecture-acceptance-checklist.md`** — exact route rows:
   - `:173` → `cp/cline-pass/glm-5.2` (Codex/OpenAI-compatible adapter, "4 clinepass accounts expected").
   - `:174` → `nvidia/z-ai/glm-5.2` (Codex or NIM adapter, "3 NVIDIA connections expected").
   - `:172` → `kimi-sub` / approved Kimi model — adapter "Kimi-compatible", exact model **"Architect to confirm"** (i.e. UNDECLARED).
   - `:58` → confirms the **`cp/cline-pass/...`** namespace is the canonical clinepass form.
3. **`OMNIROUTE_ARCHITECT_RESPONSE.md`** — protocol + operational combos:
   - `§2.1` → Kimi/GLM/NVIDIA use **OpenAI Chat Completions `POST /v1/chat/completions`** (resolves the checklist "Responses or Chat" ambiguity to **Chat**).
   - `§6` → operational combos: `kimi-sub` (priority) → `cline-kimi-k2.7-dedicated` (4 subs, RR, failover) → `group2-nvidia-glm5.2-fallback` (3 subs); Claude Code combo = `claude_code_kimi_2.7_Code`.
4. **`g1-model-route-matrix.md` (EV-G1-MODELMATRIX)** — the exactness reconciliation authority:
   - GLM exact `RouteModel` = **`cp/cline-pass/glm-5.2`** (route listed; pool/conformance PENDING).
   - NVIDIA fallback exact `RouteModel` = **`nvidia/z-ai/glm-5.2`** (alias/pool PENDING).
   - Kimi: **`kimi-sub` is a combo/route alias; exact model ID UNDECLARED**. `cline-kimi-k2.7-dedicated`
     is a *downstream combo* (4 subs), **not an exact versioned model row**. "UNDECLARED … must not be guessed or silently defaulted."
   - Claude Code combo = `claude_code_kimi_2.7_Code` (Anthropic Messages, `/v1/messages`).

---

## 2. FROZEN reconciliation — one canonical ID per route

| Route (task) | Canonical exact `RouteModel` | Protocol | Brain-selectable? | Status | Provenance |
|---|---|---|---|---|---|
| **Cline → GLM-5.2 (5.6, primary)** | **`cp/cline-pass/glm-5.2`** | OpenAI Chat Completions (`/v1/chat/completions`) | YES | **FROZEN** | checklist `:173`, `:58`; architect `§2.1`; g1-matrix GLM row |
| **GLM-5.2 → NVIDIA (5.6, fallback)** | **`nvidia/z-ai/glm-5.2`** | OpenAI Chat Completions | **NO — OmniRoute-owned bounded fallback** | **FROZEN (fallback only)** | checklist `:174`; architect `§6` (`group2-nvidia-glm5.2-fallback`); g1-matrix NVIDIA row; design.md:84; spec omniroute-agent-routing:4 |
| **Cline → Kimi-K2.7 (5.7)** | **UNDECLARED** | OpenAI Chat Completions (family known) | (pending ID) | **BLOCKED_EXTERNAL** | checklist `:172` ("Architect to confirm"); g1-matrix Kimi row (UNDECLARED); architect `§6` (combo names only) |
| Claude Code combo (context only — NOT a Cline route) | `claude_code_kimi_2.7_Code` | Anthropic Messages (`/v1/messages`) | YES (own family) | reference only | architect `§6`; model_projection.go:46 |

**Disambiguation (critical for A1/W1):** `claude_code_kimi_2.7_Code` is the **Claude Code** route
family (Anthropic Messages). It resolves upstream to a Kimi model but is **NOT** the
`Cline → Kimi-K2.7` route (task 5.7, OpenAI Chat via Cline runtime). Do **not** substitute it as the
Cline→Kimi RouteModel.

---

## 3. Divergence enumeration across the four surfaces

### 3.1 GLM-5.2

| Surface | Symbol / location | Current literal | vs canonical | Verdict |
|---|---|---|---|---|
| runtimeenv (source) | `runtimeenv/cline.go` — `NewClineConfigContract(gatewayRoot, brain.RouteModel, updatedAt)` | *(none — ID-agnostic; takes RouteModel param; only rejects `nvidia/` namespace via `isNVIDIAOwnedRoute` → `ErrClineRouteNotAgentBrainSelectable`)* | n/a | OK — no hardcoded ID; correctly fail-closes NVIDIA |
| runtimeenv (test) | `runtimeenv/cline_test.go:20,46,67,110,113,119` incl. `clineAgentBrainRoutes` | `cp/cline-pass/glm-5.2` | == canonical | **MATCH** |
| static catalog (Go) | `server/pkg/agent/models.go:562` `clineStaticModels()` | `cline-pass/glm-5.2` (Label "GLM-5.2", Provider "cline-pass") | missing `cp/` prefix | **SUBSTITUTE → `cp/cline-pass/glm-5.2`** (see §4) |
| static catalog (Go) | `server/pkg/agent/models.go:572` `nimStaticModels()` | `z-ai/glm-5.2` (`Default:true`, Provider "z-ai") | NIM-direct path, not the OmniRoute Cline route | **REVIEW** — legacy direct-provider default; see §4.3 |
| static catalog (Go, test) | `models_test.go:172,199-206`; `nim_test.go:82,86`; `nim.go:25 nimDefaultModel` | `z-ai/glm-5.2` | mirrors nimStaticModels | ties to §4.3 review |
| gateway projection (test) | `gateway/projection_test.go:57` | `nvidia/z-ai/glm-5.2` (fallback spec) | == canonical fallback | **MATCH** (fallback, not Brain-selectable) |
| gateway registry (source) | `gateway/registry.go` | *(none — generic; consumes OmniRoute snapshot; no hardcoded GLM)* | n/a | OK |
| static catalog (UI) | `packages/core/runtimes/models.ts` | *(none — discovery-based via `resolveRuntimeModels`; no static GLM IDs)* | n/a | OK — nothing to freeze |

Unrelated GLM literals (NOT part of 5.6/5.7, do not touch): `bailian-coding-plan/glm-4.7`,
`glm-coding-plan/glm-4.7` (GLM-4.7 coding-plan discovery fixtures in `models_test.go`), `glm-5.1-ioa`
(CodeBuddy catalog), `zhipu` provider-label mapping.

### 3.2 Kimi-K2.7

| Surface | Symbol / location | Current literal | vs source of truth | Verdict |
|---|---|---|---|---|
| runtimeenv (source) | `runtimeenv/cline.go` | *(none — ID-agnostic)* | n/a | OK |
| runtimeenv (test) | `runtimeenv/cline_test.go:19,79` incl. `clineAgentBrainRoutes` | `cline-kimi-k2.7-dedicated` | combo/route **alias** (4-sub combo), not an exact versioned model | **BLOCKED_EXTERNAL** — currently uses combo alias by convention; not an OmniRoute-declared exact model ID |
| static catalog (Go) | `server/pkg/agent/models.go:563` `clineStaticModels()` | `cline-pass/kimi-k2.7-code` (Label "Kimi K2.7 Code") | **NOT attested anywhere in OmniRoute source of truth** | **BLOCKED_EXTERNAL** — appears Multica-side; do not treat as canonical |
| static catalog (Go, test) | `models_test.go:172`; `handler/agent_thinking_test.go:164` | `cline-pass/kimi-k2.7-code` | mirrors catalog | ties to block |
| gateway projection (source) | `gateway/model_projection.go:46` `approvedProjectionRouteModel` | `claude_code_kimi_2.7_Code` | **Claude Code family**, not Cline→Kimi | **DO NOT CONFLATE** (see §2 disambiguation) |
| static catalog (UI) | `packages/core/runtimes/models.ts` | *(none — discovery-based)* | n/a | OK |

Three different literals target the "Kimi K2.7" concept (`cline-kimi-k2.7-dedicated`,
`cline-pass/kimi-k2.7-code`, `claude_code_kimi_2.7_Code`). None is an OmniRoute-declared **exact
versioned Kimi model row** for the `Cline → Kimi-K2.7` route → freeze is **BLOCKED_EXTERNAL**.

---

## 4. Substitution list for A1 / W1 (exact, apply only under proper lock)

> `server/pkg/agent/models.go` is a **W1/Codex1 hotspot** (FILE_OWNERSHIP + P0 override). A3 does
> **not** edit it. A1 owns `runtimeenv/cline.go`/`cline_test.go`. These are the exact edits to make
> once the owner holds the lock.

### 4.1 SUB-1 (GLM prefix) — REQUIRED, provenance-backed
- File: `server/pkg/agent/models.go:562` (function `clineStaticModels`)
- From: `{ID: "cline-pass/glm-5.2", Label: "GLM-5.2", Provider: "cline-pass"},`
- To:   `{ID: "cp/cline-pass/glm-5.2", Label: "GLM-5.2", Provider: "cline-pass"},`
- Reason: canonical exact `RouteModel` is `cp/cline-pass/glm-5.2` (checklist:173, :58; g1-matrix).
  Current value drops the `cp/` clinepass namespace segment and would not match the OmniRoute row.
- Owner: **W1** (hotspot). Verify: `go test ./server/pkg/agent/ -run TestClineStaticModels` and the
  `models_test.go:172` expectation string must be updated in lockstep to `cp/cline-pass/glm-5.2`.

### 4.2 SUB-2 (Kimi) — BLOCKED, do NOT substitute yet
- Files: `server/pkg/agent/models.go:563` (`cline-pass/kimi-k2.7-code`) and
  `runtimeenv/cline_test.go:19` (`cline-kimi-k2.7-dedicated`).
- Action: **HOLD**. No canonical exact Kimi RouteModel exists in the OmniRoute source of truth.
  Do not pick between `cline-kimi-k2.7-dedicated`, `cline-pass/kimi-k2.7-code`, or `kimi-sub`.
- Unblock condition: see §5 blocker BLK-KIMI. Once OmniRoute publishes the exact versioned Kimi
  model ID, apply the same shape as SUB-1 across models.go:563 + cline_test.go:19,79 +
  models_test.go:172 + handler/agent_thinking_test.go:164.

### 4.3 REVIEW-1 (NIM-direct GLM default) — flag for 5.7 "remove obsolete direct-provider alternatives"
- File: `server/pkg/agent/models.go:572` (`nimStaticModels`): `{ID: "z-ai/glm-5.2", ..., Default: true}`;
  also `pkg/agent/nim.go:25 nimDefaultModel`, `nim_test.go:75-86`, `models_test.go:199-206`.
- Note: This is a **NIM-direct** (NVIDIA-hosted, OpenAI-compatible NIM backend) path, distinct from
  the OmniRoute Cline→GLM route and from the OmniRoute-owned `nvidia/z-ai/glm-5.2` fallback. Task
  `5.7` says "remove obsolete direct-provider alternatives". Whether this NIM-direct default is an
  obsolete direct-provider alternative to retire is a **W1/A1 + Principal scope decision**, NOT a
  RouteModel-freeze decision. A3 flags it; A3 does not decide or edit it.

### 4.4 No-op confirmations (no change needed)
- `runtimeenv/cline.go` / `model.go` / `adapter.go`: ID-agnostic by design; GLM route flows through
  `brain.RouteModel` param; NVIDIA namespace correctly fail-closed. No literal to change for GLM.
- `gateway/registry.go`, `gateway/projection.go`: generic; consume OmniRoute snapshot; no hardcoded
  GLM/Kimi.
- `packages/core/runtimes/models.ts`: discovery-based; no static GLM/Kimi IDs.

---

## 5. Blockers (BLOCKED_EXTERNAL — exact owner + required action)

### BLK-KIMI — Cline→Kimi-K2.7 exact RouteModel undeclared
- Fact: OmniRoute source of truth declares only combo aliases (`kimi-sub`,
  `cline-kimi-k2.7-dedicated`) and the checklist marks the approved Kimi model "Architect to confirm"
  (`omniroute-architecture-acceptance-checklist.md:172`; `g1-model-route-matrix.md` Kimi row = UNDECLARED).
- Owner: **OmniRoute architect/operator** (via Principal Orchestrator).
- Required action: publish the **exact versioned Kimi model `RouteModel` string** the Agent Brain
  must set for the `Cline → Kimi-K2.7` route (the value that goes into Cline `providers.json`
  `settings.model` / the `brain.RouteModel` passed to `NewClineConfigContract`), plus its protocol
  (expected OpenAI Chat) and availability. Until then task `5.7` route freeze cannot close and
  §4.2 SUB-2 must not be applied.

### BLK-AVAIL — enriched OmniRoute registry rows not published (task 8.1)
- Fact: `gateway/model_projection.go` DEV-compat projection marks **only** `claude_code_kimi_2.7_Code`
  `available=true`; every other row (including `cp/cline-pass/glm-5.2`) is projected `available=false`
  (fail-closed → `LookupModel` returns `ErrorUnknownModel`). In normal (non-DEV) operation the
  registry requires an **enriched** OmniRoute `/v1/models` row (protocol, streaming, tools, reasoning,
  structured_output, context_limit, account_pool, rotation, affinity, available). Per g1-matrix gate
  "publish versioned model/capability registry", no such enriched row exists yet for the Cline routes.
- Owner: **OmniRoute architect/operator**.
- Required action: publish enriched registry rows for `cp/cline-pass/glm-5.2` (and the Kimi exact ID
  from BLK-KIMI) with `protocol=openai-chat`, `available=true`, and pool/rotation/affinity, so the
  gateway can admit the routes. **Exact-ID freeze for GLM is independent of this and is done** (§2);
  only availability/admissibility is blocked.
- Note: live acceptance is additionally gated by the **D-V3-25(B)** security stop (key revocation)
  per `authoritative-route-matrix-D-V3-27.md` — out of A3 scope; recorded for A1/W1 awareness.

---

## 6. Impacted symbols index (for A1 / W1)

| Symbol | File:line | Lane owner | Note |
|---|---|---|---|
| `clineStaticModels()` GLM row | `server/pkg/agent/models.go:562` | W1 (hotspot) | SUB-1: `cline-pass/glm-5.2` → `cp/cline-pass/glm-5.2` |
| `clineStaticModels()` Kimi row | `server/pkg/agent/models.go:563` | W1 (hotspot) | SUB-2 HOLD (BLK-KIMI) |
| `nimStaticModels()` GLM default | `server/pkg/agent/models.go:572` | W1 (hotspot) | REVIEW-1 (5.7 obsolete-alt) |
| `nimDefaultModel` | `server/pkg/agent/nim.go:25` | W1/A-runtime | tied to REVIEW-1 |
| `clineAgentBrainRoutes` (test) | `runtimeenv/cline_test.go:18-20` | A1 | GLM `cp/cline-pass/glm-5.2` OK; Kimi `cline-kimi-k2.7-dedicated` = alias (BLK-KIMI) |
| `NewClineConfigContract` GLM fixtures | `runtimeenv/cline_test.go:46,67,110,113,119` | A1 | already canonical GLM |
| `NewClineConfigContract` (source) | `runtimeenv/cline.go` | A1 | ID-agnostic; no change |
| `isNVIDIAOwnedRoute` / `ErrClineRouteNotAgentBrainSelectable` | `runtimeenv/cline.go` | A1 | keeps `nvidia/z-ai/glm-5.2` non-Brain-selectable ✓ |
| `approvedProjectionRouteModel` | `gateway/model_projection.go:46` | W2/W1 | `claude_code_kimi_2.7_Code` — Claude family; do NOT reuse for Cline→Kimi |
| GLM catalog expectation (test) | `server/pkg/agent/models_test.go:172` | W1 | update with SUB-1 in lockstep |
| Kimi thinking test | `server/internal/handler/agent_thinking_test.go:164` | W1 | ties to SUB-2 (BLK-KIMI) |

---

## 7. Summary for the Principal / A1 / W1

- **GLM (5.6): FROZEN.** Canonical `cp/cline-pass/glm-5.2` (OpenAI Chat). One real substitution
  (SUB-1) in the W1-owned static catalog to add the missing `cp/` prefix; runtimeenv already
  canonical. NVIDIA fallback `nvidia/z-ai/glm-5.2` is FROZEN as OmniRoute-owned and must stay
  non-Brain-selectable (already enforced in `cline.go`).
- **Kimi (5.7): BLOCKED_EXTERNAL** (BLK-KIMI). No OmniRoute-declared exact versioned model ID; three
  divergent Multica-side literals exist; none may be chosen by plausibility. Owner action required.
- **Availability (8.1): BLOCKED_EXTERNAL** (BLK-AVAIL). Enriched OmniRoute registry rows for the
  Cline routes not yet published; exact-ID freeze for GLM is nonetheless complete and independent.
- Product source untouched; A3 output limited to this handoff.
