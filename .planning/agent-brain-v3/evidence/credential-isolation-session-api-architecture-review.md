# Credential Isolation — Session API Architecture Review (principal, read-only)

- author: Kiro (principal)
- date: 2026-07-18T20:30:00Z
- mode: READ-ONLY decision support. No source/spec/impl/STATE/ledger edits. No DB/network/Docker/credentials/env values touched.
- scope: independent validation of the two Gemini audit artifacts for `agent-credential-isolation` tasks 0.1–0.3 and 2.1–2.3, staleness assessment, current equivalents, and minimal ordered artifact/design update recommendation.
- check-in: `.deploy-control/Kiro__CREDISO-ARCH-REVIEW__20260718T202400Z.md`

## Audited inputs (SHA-256)

| File | SHA-256 |
|------|---------|
| credential-isolation-source-contract-map.md | `3e704696adcfdf144eadbe126f85f09cb541fec3736d6f2709adb12e8d5b4fdd` |
| credential-isolation-session-api-audit.md | `9573b058d838273dd0d9a058a9d33f51b88873bfa9335e1b1e603cedb71b1554` |
| openspec/.../proposal.md | `a15b62c1d77c899c61b4f5fa39bf975ac4318cc99941f6eab10fbdca8d618636` |
| openspec/.../design.md | `92ffdf6b414a76d3fbd8baa8f32a6497af05b5fc325b0a00ef180cafd10fc40a` |

## Current-source references cited (SHA-256, captured this review)

| File | SHA-256 |
|------|---------|
| server/cmd/server/router.go | `5c6492bfd64347d48bb13749ac3f1b38ef84b4275fd4d20b1fc44e0f1cdb5a74` |
| server/internal/handler/auth_provider.go | `3c8f75b5ac2e9a4e2ca83b228285c3ff21d2ced484aab3b29f71cb7b67d70857` |
| server/internal/daemon/execenv/execenv.go | `8099867594f18f6d5e5c47fa88581f06ac771e456f203fc0161d09c6584fc0e6` |
| server/internal/daemon/execenv/codex_home.go | `aa3f8bbd82ffb3fe70ed72da037eaae3028af4b558a7b1d0c207ccaeb4cb3fe0` |
| server/internal/rotation/store_pg.go | `e47757784ba22d9c4d749c51028247523a6d8cfd3db002343a503d5fba67edf8` |
| server/internal/rotation/contract.go | `eef92d45127137f5f339a430490c7b188f5704c9ebf55b545a54fc0c3c85fd4e` |
| server/internal/daemon/credential_session_discovery_producer.go | `4e240c09af1653fd2ecdbcd0763f8c05d3eb3f57ee8b9c339045edfa32a9ce6c` |
| server/internal/rotation/discovery_reassignment.go | `c655c9c4f94716d2c93109b1cc4d3c14e62d6fe8dea615dddec3ec93769b2832` |
| server/internal/rotation/detector_discovery.go | `bc61a46c0c700010001a49c9075b87f97e89f802b138e63f90b53193a4b45b55` |

## Claim-by-claim adjudication

### Artifact 1 (source-contract-map.md) — tasks 0.1–0.3

| # | Gemini claim | Verdict | Grounded evidence |
|---|--------------|---------|-------------------|
| 1 | `infra/cao/auth_routes.py` not found | **CONFIRMED (conclusion) / evidence CORRECTION** | `infra/` **does exist** at repo root but contains only `infra/host-bootstrap/{wsl,windows}` — there is **no `infra/cao/` and no `auth_routes.py`**. Gemini's stated reason ("the `infra/` directory no longer exists at the project root") is factually wrong; the file's absence is correct. |
| 2 | `src/canvas-reconciler/reconciler.ts` not found (root `src/` gone) | **CONFIRMED** | No root-level `src/` directory exists. |
| 3 | `resolveSessionEnv` / `session-discovery` / `session-store` zero matches | **CONFIRMED for the literal identifiers** | Repo-wide search for `resolveSessionEnv`, `session-discovery`, `session-store`, `canvas-reconciler`, `auth_routes.py` returns zero matches. |
| 4 | (implicit) no current equivalents identified | **INCOMPLETE / MISLEADING** | Functional equivalents **do exist** under new names (see "Current equivalents" below). Artifact 1 explicitly declined to map them ("Non-claims"), so it is not wrong, but it leaves tasks 0.1–0.3 blocked without pointing at the real successors. |

Additional grounding: proposal.md L17/L30–31/L35/L40 and tasks.md 0.1–0.3 anchor the reference model to `infra/cao/auth_routes.py`, `src/api/session-discovery.ts`, `src/api/session-store.ts`, `src/canvas-reconciler/reconciler.ts`. **design.md L54–56 clarifies these are the AgentVerse/AOP reference model** ("O AOP é a fonte de verdade; o `auth_routes.py` do AgentVerse é a versão…"), i.e. an external/archived project (the AgentVerse SPA was removed — see repo `README.md` / `LEGACY_ARCHIVE_REFERENCE.md`), not files expected to live in this repo. Therefore 0.1–0.3 cannot be satisfied "in place" and must be re-anchored.

### Artifact 2 (session-api-audit.md) — tasks 2.1–2.3

| # | Gemini claim | Verdict | Grounded evidence |
|---|--------------|---------|-------------------|
| 5 | `GET /auth/sessions` absent | **CONFIRMED** | `router.go` registers only `/auth/login` (L487), `/auth/google` (L488), `/auth/logout` (L489). No `/auth/sessions`. |
| 6 | `POST /auth/login` with account `config_dir` absent | **CONFIRMED** | The existing `POST /auth/login` → `h.Login` is Multica **user** auth. `handler/auth_provider.go` defines `AuthProvider.Login(ctx, email, password) (AuthIdentity{UserID}, error)` with bcrypt/password store — no `config_dir`/provider concept. |
| 7 | `DELETE /auth/sessions/:id` absent | **CONFIRMED** | No such route in `router.go`; `session` routes are only chat-sessions (e.g. `/chat-sessions/{sessionId}/gc-check` L538) and `/tasks/{taskId}/session` (L543). |
| 8 | No request/response structs for the contract | **CONFIRMED** | No Go structs for provider-session listing/login/revoke. |
| 9 | "No tests exist" for these | **CONFIRMED for HTTP contract; NUANCE** | No HTTP handler tests for provider-session list/login/revoke. However `server/cmd/server/auth_routes_test.go` exists and tests the **user** `/auth/login` (401 path) — it is user-auth, not the credential contract. |
| 10 | proposal.md/design.md are **stale** (assume AS-IS port of Python CAO contract) | **CONFIRMED (grounded)** | proposal.md anchors the API to the legacy `auth_routes.py` "mesmo contrato de API" (L40); those sources are absent. The staleness is in the **source anchors**, not necessarily the intent. |

## Distinguishing user-auth login from provider-session login (explicitly required)

- **User auth (exists):** `POST /auth/login` (`router.go:487`) → `handler.Handler.Login` → `AuthProvider.Login(email,password)` (`auth_provider.go`). Authenticates a **Multica platform user**, issues JWT cookies. Also `/auth/google` (OAuth) and `/auth/logout`. This is identity for the human/tenant operating Multica.
- **Provider-session login (does NOT exist as HTTP):** tasks 2.1–2.3 want per-**agent-account** provider credential sessions (Codex/Claude/Gemini/Kiro), keyed by `config_dir`/`CODEX_HOME`, listed/created/revoked. There is **no** REST surface for this.
- These are **different security domains** and must not be conflated: user-auth guards the control plane; provider-session governs which isolated credential home a runtime executes under.

## Current equivalents (the true successors of the legacy reference model)

The legacy names are gone, but the *model* was reimplemented in Go — as an **internal daemon discovery + rotation model**, not (yet) as an HTTP contract:

- **`PROVIDERS` (config_dir/env per CLI) →** per-provider isolation levers in `server/internal/daemon/execenv/execenv.go` (L66–69, L158) and `execenv/codex_home.go`; native env injection `runtimeenv/env.go:214` (`CODEX_HOME`), policy `runtimeenv/policy.go:34` (`CODEX_HOME: DenyCredentialRoot`); provider env vars `CODEX_HOME`/`CLAUDE_CONFIG_DIR`/`GEMINI_CONFIG_DIR`/`KIRO_HOME` (tasks.md 1.2). This is the current, grounded successor for **task 0.1**.
- **`resolveSessionEnv` (env assembly) →** Go execenv assembly + rotation account `HomeDir`→env; **tasks.md 0.4 already `[x]`** ("como a versão pura (execenv) monta hoje o home de credencial"). Current successor for **task 0.2**.
- **`session-discovery` / `session-store` (contract) →** account inventory `rotation/store_pg.go:52 ListAccounts(ctx, vendor, tenantID)` + `rotation/contract.go`; live discovery `daemon/credential_session_discovery_producer.go:75 NewCredentialSessionDiscoveryProducer`, `credential_session_monitor.go`, `rotation/detector_discovery.go`, `rotation/discovery_reassignment.go`. Current successor for **task 0.3** (data/discovery layer) — **as an internal producer/monitor, not an HTTP API**.

**Implication:** the data + discovery + env-isolation layers that 2.1–2.3 would sit on already exist. What is missing is only the **HTTP contract** (and the product decision of whether it is even wanted, given the internal model).

## Security / tenant / revocation boundaries (grounded observations)

- Account inventory is already **tenant-scoped**: `ListAccounts(ctx, vendor, tenantID)`; L2/prodex reconciliation reads it per tenant (`daemon/prodex_profiles.go`). Any `GET /auth/sessions` MUST enforce the same `tenantID` scope and MUST NOT leak cross-tenant accounts.
- Credential homes are governed by `runtimeenv/policy.go` (`DenyCredentialRoot`) and execenv mode/root checks. Any login-with-`config_dir` endpoint MUST validate against the approved slot root and modes (mirrors the fail-closed checks already in `prodex_profiles.go`).
- **Revocation** semantics currently live as account state in the rotation store + discovery detection (`expired`/`expires_at`, tasks.md 4.1 `[x]`), not as `DELETE`. A REST `DELETE /auth/sessions/:id` would need to define whether it revokes the account row, clears the credential home, or only marks unavailable — this changes rotation/fallback behavior.
- **No secret exposure:** any listing endpoint must return opaque identifiers/config-source labels only (consistent with Golden Rule 5 and `pkg/redact`).

## Recommendation (minimal, grounded) — with the owner decision flagged

**OpenSpec IS stale in its source anchors** (0.1–0.3 point at removed/external AgentVerse paths). Recommended minimal update, in order:

1. **Re-anchor tasks 0.1–0.3** (do not delete them) to the current Go successors:
   - 0.1 → `execenv/execenv.go` + `execenv/codex_home.go` + `runtimeenv/policy.go` (provider→config_dir/env map).
   - 0.2 → mark satisfied-by-0.4 (execenv env assembly) or merge into 0.4.
   - 0.3 → `rotation/store_pg.go` `ListAccounts` + `rotation/contract.go` + `daemon/credential_session_discovery_producer.go`.
2. **Update proposal.md L17/L30–31/L35/L40 and design.md L54–77** to state the reference model is archived/external (AgentVerse/AOP) and the *authoritative implementation basis is now the Go rotation store + execenv*, not a live `auth_routes.py`.
3. **Correct artifact 1's evidence line** (infra/ exists but has no `cao/`) so the record is accurate.
4. **Reframe tasks 2.1–2.3** pending the decision below.

Artifact files that must be updated (no code): `openspec/changes/agent-credential-isolation/proposal.md`, `.../design.md`, `.../tasks.md` (0.1–0.3 anchors), and (optionally) refresh the two Gemini evidence artifacts with the corrected `infra/` finding + current-equivalent pointers.

## OWNER / PRODUCT DECISION REQUIRED — STOP POINT (not decided here)

Tasks 2.1–2.3 assume an HTTP contract ported from the removed Python CAO. The current architecture instead exposes credential-session behavior **internally** (daemon discovery producer + rotation store + execenv isolation). The owner must choose the target shape before implementation:

- **Option A — Build the REST contract** (`GET /auth/sessions`, `POST /auth/login`+config_dir, `DELETE /auth/sessions/:id`) as new Go handlers over the existing rotation store. Pros: matches original spec/frontend expectation (`packages/core` client had `listAuthSessions` historically). Cons: new public surface, new authz/tenant/revocation semantics to design and secure.
- **Option B — Redefine 2.1–2.3 against the internal model** (discovery producer + rotation store), exposing read-only status via existing health/observability rather than a new mutable `/auth/sessions` API. Pros: reuses shipped, tested internals; smaller attack surface. Cons: diverges from the literal spec wording and any legacy frontend contract.
- **Ownership if Option A:** routes in `server/cmd/server/router.go` under an authenticated group; handlers in `server/internal/handler/` (new `auth_session.go`, distinct from user `auth.go`); store reads via `rotation.Store`; strict `tenantID` scoping; opaque responses.

I am **not** selecting A vs B — it is a product/owner decision. This review supplies the grounded basis for that call.

## Explicit non-claims

- I did **not** run the daemon, DB, Docker, or any live service; all findings are static (source/tree/hashes) as of the SHAs above.
- I did **not** read or emit any credential, token, `auth.json`, or env value.
- I did **not** infer the *name/shape* of a future REST resource beyond noting the spec's literal paths; the A/B design is framed, not chosen.
- I did **not** edit any OpenSpec, source, STATE, or ledger file; I marked no task checkboxes.
- The historical ledger references (`.planning/AGENT_LEDGER*.md`) are treated as secondary; primary evidence is the current tree + hashes.
- "Current equivalents" are asserted from path + symbol anchors; I did not exhaustively read every file end-to-end, so semantic parity (not mere existence) should be confirmed during implementation.
