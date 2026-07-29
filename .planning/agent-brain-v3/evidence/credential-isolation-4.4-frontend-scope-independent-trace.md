# READ-ONLY independent trace — agent-credential-isolation 4.4 frontend scope (D1 decision support)

- Author: **GLM52#B** (independent trace; taking over the bounded 4.4 frontend-scope lane because Herdr pane `w4:p3` — the prior `GLM52-auth-QA` reviewer — is provider-rate-limited).
- Mode: **READ-ONLY**. No product/test/spec/task/shared-ledger/git/index edit. No DB/network/credential/env-value/live-service access.
- Method: deterministic **static source inspection only** (grep + read + sha256). No typecheck/test execution (the offline frontend typechecks are slow on this WSL2/mount host — minutes each — and this is a read-only scope trace, not a verification lane; static inspection of the typed `WSEventType` union, the `Agent` type, and the daemon event const is sufficient and deterministic for the scope question).
- Kiro TL adjudicates the D1 owner decision; this trace supplies the grounded basis, does not self-accept, and does not unilaterally re-scope.

## Golden Rule check-IN / check-OUT

- **Check-IN** 2026-07-18T21:27:10Z — claimed: read-only static trace + this single artifact `credential-isolation-4.4-frontend-scope-independent-trace.md` only.
- Excluded (honored): no product/test/spec/`tasks.md`/shared-ledger/`EVIDENCE_INDEX`/`STATE`/OpenSpec edit; no git/index op; no DB/network/live-provider/credential/env-value access; no typecheck/test run.
- **Check-OUT** 2026-07-18T21:36:56Z — DONE; trace below; nothing else modified. `tasks.md` 4.4 confirmed `[ ]` (OPEN) before and after.

## Provenance

- **Reviewer:** GLM52#B (distinct from producer `Codex/root`, distinct from prior reviewer `GLM52-auth-QA`/`w4:p3`, distinct from adjudicator `Kiro/Opus-4.8`). Identity basis: the dispatch explicitly routed this to GLM52#B because `w4:p3` (the prior 4.4 reviewer) is provider-rate-limited; GLM52#B is a different pane/identity taking the bounded frontend-scope trace.
- **Host:** WSL2 linux/amd64 (the opencode execution environment).
- **Toolchain:** static inspection only — `grep`, `Read`, `sha256sum`. No Go/node/pnpm toolchain invocation (not needed for a scope trace).
- **Repository HEAD:** `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f` (the commit pinned by the prior 4.4 reviews; current working tree carries the producer's 4.4 daemon edits as uncommitted `M`, consistent with the prior review manifests).
- **Review window:** 2026-07-18T21:27:10Z through 2026-07-18T21:36:56Z UTC.
- **No credential, auth home, session file, token, environment secret, database, network, live provider/daemon/CLI, or multi-node state was read or used.** Only repository source/spec/evidence files were inspected.

## The D1 scope question (what this trace answers)

AGENT_LEDGER row `credio-4.4-fresh-review` (and the co-lead adjudication at `:300`, and the record-failure review at `:330`) left task 4.4 OPEN **solely** pending the explicit **D1 frontend scope decision**: does task 4.4 require the named frontend `useSessionMonitor`/`isExpiringSoon` integration, or is it satisfied by the daemon structured-log alerting already delivered (EV-CREDISO-4.4 + EV-CREDISO-4.4-RECORDFAIL)?

Task 4.4 wording (`tasks.md:28`): "Registrar/alertar a troca (aproveitar useSessionMonitor/isExpiringSoon)."
Spec acceptance scenario (`spec.md:82-84`): "Sem conta disponível" — "o sistema sinaliza o esgotamento (alerta) sem sobrescrever credenciais".

This trace locates the actual current implementation of the named hooks, maps the smallest safe backend alert payload to existing frontend surfaces, and answers whether the task wording truly requires frontend delivery.

## Finding 1 — `useSessionMonitor` / `isExpiringSoon` do NOT exist in this repo

Repo-wide grep (excluding `node_modules`/`.git`/`.next`/`tsbuildinfo`) for `useSessionMonitor` and `isExpiringSoon` across `*.ts`/`*.tsx`/`*.go`/`*.py`/`*.md` returns matches **only in planning docs** (AGENT_LEDGER.md, EVIDENCE_INDEX.md) and `evidence/*.md` review artifacts — **zero matches in product/test source**.

The prior trace `credential-isolation-4.3-production-integration-gap-trace.md` and `credential-isolation-session-api-architecture-review.md:43` already established why: `proposal.md:57`'s parenthetical "o monitor de expiração já existente (`useSessionMonitor`/`isExpiringSoon`)" references the **archived/external AgentVerse/AOP SPA reference model** (`design.md:54-56`: "O AOP é a fonte de verdade; o `auth_routes.py` do AgentVerse é a versão…"). The AgentVerse SPA was removed from this repo (see `README.md`/`LEGACY_ARCHIVE_REFERENCE.md`); `src/api/session-discovery.ts`, `src/api/session-store.ts`, `src/canvas-reconciler/reconciler.ts` (`resolveSessionEnv`) are all absent (confirmed by `session-api-architecture-review.md:40` — repo-wide zero matches). The names are **stale external references**, not hooks that exist to be "reused".

**Conclusion:** there is nothing to "aproveitar" (reuse) in this repo's frontend. The parenthetical in `tasks.md:28`/`proposal.md:57` names symbols that do not exist here. Any "reuse" is **inspiration for the shape of a future hook**, not a deliverable integration against existing code.

## Finding 2 — actual current event/API plumbing (where the alert lives today)

### Backend (delivered; daemon-internal)

The reassignment alert is **already delivered on the backend** as a daemon-internal structured log + daemon-internal WS frame, both secret-free:

- **WS frame (daemon-internal):** `server/internal/daemon/credential_session_monitor.go:13` defines `const eventDaemonCredentialSessionDiscovery = "daemon:credential_session_discovery"`. The producer (`credential_session_discovery_producer.go`) and consumer (`credential_session_monitor.go`) are **both daemon-side**; this frame is **not** published to the web client. (Note: the production producer/emitter is itself a separate 4.3 gap — see `credential-isolation-4.3-production-integration-gap-trace.md` — but that is orthogonal to the 4.4 frontend-scope question.)
- **Structured log (operator-visible):** `server/internal/daemon/wakeup.go:325-377` emits the four alert outcomes (success WARN, no-account WARN, failure ERROR, no-op DEBUG) via `d.logger`. The payload is a **whitelist** (`credential_session_monitor.go:45-56`): `agent_id`, `provider`, `tenant_id`, `previous_account_id`, `next_account_id`, `reason=quota_exhausted_reactive` / `alert=no_account_available` / `alert=reassignment_failed` / bounded error class (`deadline_exceeded`/`canceled`/`service_unavailable`). **No** `HomeDir`, `ConfigDir`, `LastError`, raw error string, or session id enters the outcome (proven by `credential_session_alert_test.go:17-140` synthetic sentinels; covered by EV-CREDISO-4.4 + EV-CREDISO-4.4-RECORDFAIL).
- **Durable record:** `server/internal/rotation/service.go:150` `Assign` → `:156` `RecordRotation` → `:157` return on record failure → `:159` return success. The WARN completion alert cannot be reached before `RecordRotation` succeeds (record-failure fail-closed proven by `TestDispatchCredentialSessionRecordRotationFailureAlertsWithoutSuccessOrLeak`, EV-CREDISO-4.4-RECORDFAIL).

### Frontend (current subscription surface — does NOT include the reassignment event)

The frontend's typed WS event union is `packages/core/types/events.ts:11-82` (`WSEventType`) — **50+ event types**, and **`daemon:credential_session_discovery` is NOT in it**. The closest existing event is `"agent:status"` (`events.ts:20`) with payload `AgentStatusPayload { agent: Agent }` (`:113-115`), used for online/offline presence (mobile `apps/mobile/data/realtime/use-presence-realtime.ts:51` subscribes to invalidate the agent list; web uses it via `packages/core/realtime/`). The `Agent` type (`packages/core/types/agent.ts`) has a `provider` field (`:21`, `:534`) but **no** `account_id`, `credential`, `rotation`, or `session_id` field — there is no existing field on the frontend Agent model to carry reassignment info.

**Conclusion:** the backend alert is daemon-internal (log + daemon→daemon WS). The web client has no subscription to it and no existing Agent field to render it. Surfacing the reassignment in the UI would require a **new server→web-client WS event** + a **new Agent-model field or a side-channel notification** + a frontend subscription + a UI surface.

## Finding 3 — smallest safe backend reassignment alert payload → existing frontend surfaces

Mapping the **already-delivered backend whitelist payload** to the **closest existing frontend surfaces** (if the owner chooses a UI alert):

| Backend payload field (whitelist; no secrets) | Closest existing frontend surface | Gap to surface it |
|---|---|---|
| `agent_id` | `Agent.id` (`types/agent.ts`); `agent-detail-inspector.tsx`, `agents-page.tsx`, `activity-tab.tsx` (all import `toast` from `sonner`) | none — agent id is already a first-class UI key |
| `provider` | `Agent.provider` (`types/agent.ts:21,534`); runtime/provider picker UI | none — provider is already rendered |
| `tenant_id` (= workspace id) | workspace context (already scoped by middleware) | none — tenant scoping is the existing auth model |
| `previous_account_id` / `next_account_id` | **NONE** — `Agent` type has no account field; no rotation/account UI exists (grep for `rotation`/`account_id`/`AccountID` in `packages/views/`+`apps/web/app/` returns only false positives — "Rotat" in onboarding step-welcome, comment-card, etc., none credential-related) | **new Agent field OR a side-channel notification** (the account ids are opaque uuids; surfacing them needs a new typed field + a label resolution path) |
| `reason=quota_exhausted_reactive` / `alert=no_account_available` / `alert=reassignment_failed` | `sonner` toast (`packages/ui/components/ui/sonner.tsx`) — the standard alert surface, used pervasively across agents views (agent-detail-inspector, agent-detail-page, agent-row-actions, agents-page, create-agent-dialog, activity-tab, etc.) | **new toast call** behind a new WS subscription |

### Smallest safe UI surfacing path (if the owner chooses frontend delivery)

1. **New server→web WS event** (e.g. `agent:credential_rotated` or extend `agent:status`) — requires extending both `packages/core/types/events.ts:11-82` (`WSEventType`) AND the Go `server/pkg/protocol/events.go` mirror, plus a handler that publishes it tenant-scoped. The payload must be the **same whitelist** the daemon log already uses (`agent_id`/`provider`/`tenant_id`/`previous_account_id`/`next_account_id`/`reason`/`alert`) — **no** `HomeDir`/`ConfigDir`/`LastError`/raw error (the backend whitelist is the redaction boundary; it is already enforced and tested).
2. **Frontend subscription + toast** — a `useWSEvent("agent:credential_rotated", …)` hook (`packages/core/realtime/hooks.ts:13`) that calls `toast.warning(...)` / `toast.error(...)` from `sonner` with a localized message. The `agent-detail-page.tsx` / `agents-page.tsx` are the natural hosts (they already import `toast`).
3. **No new Agent field is strictly required** for a toast-only alert (the toast can carry `agent_id`→name resolution via the existing React Query `workspaceKeys` cache). Surfacing `previous/next account` in the Agent inspector would need a new field + label resolution — that is a larger change and out of the smallest-safe scope.

### Redaction / tenant / provider boundaries (already enforced on the backend; the frontend inherits them)

- **Redaction:** the backend outcome is a **whitelist** (`credential_session_monitor.go:45-56`); `HomeDir`/`ConfigDir`/`LastError`/raw error cannot enter. The frontend toast would render only the whitelisted fields. The existing `pkg/redact.SanitizeSlogAttr` is supplemental, not the only protection (per EV-CREDISO-4.4 review). **No frontend redaction work needed** — the backend is the trust boundary.
- **Tenant:** the bridge copies `tenant_id` (= workspace id) from the signed event (`credential_session_monitor.go:83-97`); a server→web event MUST preserve workspace scoping via the existing WS auth (the web client already only receives its own workspace's events). **No new tenant-leak risk** if the event is published through the existing workspace-scoped WS hub.
- **Provider:** the 4.3 service rejects a current assignment outside the canonical provider/tenant (`discovery_reassignment.go:65-70`); the alert carries the signed `provider` value. **No new provider-boundary risk.**

## Finding 4 — does the task wording truly require frontend delivery?

### Literal task wording

`tasks.md:28`: "Registrar/alertar a troca (aproveitar useSessionMonitor/isExpiringSoon)."
- "Registrar" (record) = durable `RecordRotation` + structured log → **ALREADY DELIVERED** (EV-CREDISO-4.4 + EV-CREDISO-4.4-RECORDFAIL).
- "alertar" (alert) = the open clause. The parenthetical "aproveitar useSessionMonitor/isExpiringSoon" references **symbols that do not exist in this repo** (Finding 1) — they are archived/external AgentVerse references, not reusable hooks.

### Spec acceptance scenario (the binding acceptance gate)

`spec.md:82-84` "Sem conta disponível": "o sistema sinaliza o esgotamento (alerta) sem sobrescrever credenciais."
- "sinaliza o esgotamento (alerta)" is **mechanism-agnostic** — it does not name a UI, a WS event, a toast, or a frontend hook. It requires a *signal*.
- "sem sobrescrever credenciais" (no credential overwrite) is **already proven** (`service.go:97-122` returns without Logout/Login on `ErrNoAccountAvailable`; `credential_rotation_task53_test.go:59-98` asserts `assignment == "current"` + `len(auth calls) == 0`).

### Verdict on the wording

**The task wording does NOT truly require frontend delivery.** Reasoning:
1. The parenthetical names **non-existent symbols** (stale external references) — there is nothing to "aproveitar" in this repo. The wording cannot require integrating against hooks that do not exist.
2. The binding spec scenario is **mechanism-agnostic** ("sinaliza o esgotamento (alerta)") — a daemon structured WARN/ERROR log is a valid "alerta" (operator-visible signal), and it is already delivered + tested + secret-free.
3. The proposal's "monitor de expiração já existente" framing was **factually stale at write time** — the expiration monitor lives **backend-side** (`rotation/detector_discovery.go`, `credential_session_discovery_producer.go`), not in a frontend hook. The intent ("reuse an existing expiration monitor") is already satisfied by the backend detection that feeds the alert.

**However**, the prior reviewers and co-lead correctly flagged this as an **owner/spec scope decision**, not a reviewer call (AGENT_LEDGER `:300`: "co-lead does not unilaterally re-scope"). The daemon-log alert satisfies the literal spec; a frontend toast is a **stronger operator UX** but is **not required** by the binding acceptance scenario. The choice between "daemon-signal-only" and "frontend-hook delivery" is a product/owner call.

## Explicit owner decision required (D1 — not decided by this trace)

Per the dispatch, this trace does **not** select the scope. It supplies the grounded basis for the owner's D1 decision:

- **Option D1-A — Daemon-signal-only (spec-literal):** task 4.4 is satisfied by the delivered daemon structured WARN/ERROR alert + durable `RecordRotation`. No frontend work. **Pro:** matches the mechanism-agnostic spec scenario; zero new public WS surface; zero new authz/tenant/redaction work; the backend is already delivered and tested (EV-CREDISO-4.4 + RECORDFAIL). **Con:** the operator sees the alert in logs/observability, not in the app UI.
- **Option D1-B — Frontend toast alert (stronger UX):** add a new server→web WS event (`agent:credential_rotated` or extend `agent:status`) carrying the existing backend whitelist payload + a `sonner` toast subscription. **Pro:** operator sees the reassignment in the app. **Con:** new public WS surface + new `WSEventType`/`protocol/events.go` entries + workspace-scoped publish + a frontend subscription + i18n strings; the account-id labels need a resolution path. This is a **new frontend feature**, not a "reuse" of existing hooks.
- **Option D1-C — Split:** accept 4.4 now under D1-A (daemon-signal-only, spec-literal) and open a **separate follow-on task** for the frontend toast (D1-B) under a new task id (e.g. 4.5 or a cred-iso frontend-alert lane). **Pro:** unblocks 4.4 on the delivered backend evidence without re-scoping; the frontend UX becomes its own owned, evidenced lane. **Con:** two lanes instead of one.

**Recommendation (advisory, not binding):** D1-C is the smallest safe path — it accepts 4.4 on the spec-literal daemon signal (already delivered + tested + secret-free) and routes the frontend toast as a separate owned lane, so the frontend work is not smuggled through a stale "reuse" parenthetical. But the owner may prefer D1-B if the operator UX gap is unacceptable. **Kiro TL adjudicates.**

## Source SHA-256 manifest (read-only; files inspected this trace)

| SHA-256 | Source |
|---|---|
| `3bdbc1e1916629d87c14db544aee88050daf0e0aaa2c07c3c933798678e7a0e3` | `openspec/changes/agent-credential-isolation/tasks.md` (4.4 `[ ]` OPEN, confirmed before+after) |
| `a15b62c1d77c899c61b4f5fa39bf975ac4318cc99941f6eab10fbdca8d618636` | `openspec/changes/agent-credential-isolation/proposal.md` (L57 stale `useSessionMonitor`/`isExpiringSoon` reference) |
| `02b0a5c131c0fd2cee2732dbf4116155d29e87c55d37cc7391994cfdf20da63b` | `openspec/changes/agent-credential-isolation/specs/agent-credential-isolation/spec.md` (L82-84 mechanism-agnostic "Sem conta disponível") |
| `92ffdf6b414a76d3fbd8baa8f32a6497af05b5fc325b0a00ef180cafd10fc40a` | `openspec/changes/agent-credential-isolation/design.md` (L54-56 AgentVerse/AOP external reference) |
| `83c3c272feeeee714a686b0630103cabf8832cc5a93bd1cbfd0e69a7e29f990e` | `multica-auth-work/packages/core/types/events.ts` (`WSEventType` union — no `daemon:credential_session_discovery`; `agent:status` payload `{ agent: Agent }`) |
| `1cabdc4e4375c7ef31572417ab0e3854a8dfb65db60d26ed0df8447618c4f957` | `multica-auth-work/packages/ui/components/ui/sonner.tsx` (standard toast surface) |
| `18b286089ad818442a0ef193b11039c579fc6c55ab6cb214cf08dff841ffbe00` | `multica-auth-work/packages/views/agents/components/agent-detail-inspector.tsx` (imports `toast` from `sonner`; no credential/account/rotation field) |
| `71b89980c9b8cb7c03a81a502da5b9f4bdab4c0c522c117d2147b670e050f730` | `multica-auth-work/server/internal/daemon/wakeup.go` (delivered alert path L325-377) |
| `936b3e40f19c699078e994740c9ae63fdce5c172516b678d289f05c3cc38d1e2` | `multica-auth-work/server/internal/daemon/credential_session_monitor.go` (L13 daemon-internal event const; L45-56 whitelist outcome) |

Prior-review artifacts read for context (not re-hashed here; their SHAs are recorded in AGENT_LEDGER): `credential-isolation-reassignment-alerting.md` (producer), `credential-isolation-reassignment-alerting-review.md` (REVIEW), `credential-isolation-4.4-fresh-review.md` (GLM52-auth-QA), `credential-isolation-4.4-record-failure-alert-independent-review.md` (Codex Agent-6), `credential-isolation-4.3-production-integration-gap-trace.md`, `credential-isolation-session-api-architecture-review.md`.

## Explicit non-claims

- This is a **read-only scope trace**, not a verification lane. No Go/node/pnpm typecheck or test was executed (deterministic static inspection only); the offline frontend typechecks are slow on this WSL2/mount host and are not needed to answer the scope question.
- No claim that the backend alert is **production-deployed** — the 4.3 production producer/emitter gap is separate (`credential-isolation-4.3-production-integration-gap-trace.md`); this trace is scoped to the 4.4 frontend-scope question only.
- No claim that a frontend toast is **required** — the spec scenario is mechanism-agnostic; D1-A (daemon-signal-only) is spec-literal.
- No claim that the proposed D1-B WS event `agent:credential_rotated` is the **chosen** design — it is the smallest-safe illustration; the owner may pick any shape.
- No claim of `useSessionMonitor`/`isExpiringSoon` existence — they do not exist in this repo (Finding 1).
- No claim of live WebSocket delivery, live vendor behavior, PostgreSQL execution, transactionality, multi-node CAS, production deployment, full task 4.3 completion, or Kiro/TL acceptance.
- No edits to: OpenSpec (`tasks.md`/`proposal.md`/`design.md`/`specs/`), `STATE.md`, `AGENT_LEDGER.md`, `EVIDENCE_INDEX.md`, any product/test file, any checkbox. `tasks.md` 4.4 confirmed `[ ]` (OPEN) before and after.
- No credential, auth home, session file, token, environment secret, database, network, live provider/daemon/CLI, or multi-node state was read or used.

## Verdict (advisory; Kiro TL adjudicates D1)

- **Finding 1:** `useSessionMonitor`/`isExpiringSoon` do NOT exist in this repo — the task parenthetical references stale/external AgentVerse symbols; there is nothing to "reuse".
- **Finding 2:** the reassignment alert is already delivered backend-side as a daemon-internal structured log (whitelist payload, secret-free) + durable `RecordRotation` (EV-CREDISO-4.4 + RECORDFAIL). The web client does not subscribe to `daemon:credential_session_discovery` (it is daemon-internal) and the `Agent` type has no account/credential field.
- **Finding 3:** the smallest safe UI surfacing path (if the owner chooses frontend) is a new server→web WS event carrying the existing backend whitelist + a `sonner` toast subscription; no frontend redaction work needed (backend is the trust boundary).
- **Finding 4:** the task wording does NOT truly require frontend delivery — the parenthetical names non-existent hooks and the binding spec scenario is mechanism-agnostic ("sinaliza o esgotamento (alerta)"). The daemon structured log is a valid "alerta" and is already delivered.
- **Owner decision D1 (not decided here):** D1-A (daemon-signal-only, spec-literal, accept 4.4 on delivered backend evidence) vs D1-B (frontend toast, new WS event + sonner) vs D1-C (split: accept 4.4 now, frontend toast as a separate follow-on lane). **Advisory recommendation: D1-C** — unblocks 4.4 on the spec-literal daemon signal and routes the frontend UX as its own owned, evidenced lane. **Kiro TL adjudicates.**

Task 4.4 stays `[ ]` (OPEN). This trace does not self-accept and does not unilaterally re-scope.
