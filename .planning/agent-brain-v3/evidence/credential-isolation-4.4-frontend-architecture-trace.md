# EV-CREDISO-4.4-FRONTEND — read-only architecture trace: surfacing daemon reassignment/no-account/failure alerts to operators

Read-only architecture trace for the **frontend scope** of
agent-credential-isolation task 4.4 ("Registrar/alertar a troca — aproveitar
useSessionMonitor/isExpiringSoon"). This is a design/trace artifact, not
implementation, not self-acceptance, not a task checkbox change. Submitted to
Kiro/Opus-4.8 (TL) for adjudication and the daemon-only-vs-frontend-required
owner decision.

## Provenance

- **Reviewer/trace author:** GLM52-auth-QA (Herdr pane `w4:p3`, workspace `w4`).
- **Host:** `manoelneto-laptop` (WSL2, Linux amd64).
- **Repository commit (HEAD):** `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f`.
- **Toolchain read:** `/home/dataops-lab/go-sdk/bin/go` `go1.26.4` (not exercised
  here — this is a static trace; the 4.4 backend alert tests were already
  reproduced in `evidence/credential-isolation-4.4-fresh-review.md`).
- **Trace window:** 2026-07-18T21:07Z through 2026-07-18T21:35Z UTC.
- **Method:** `grep`/`glob`/`read` over `multica-auth-work/{apps,packages}` and
  `multica-auth-work/server/internal/daemon` only. No product/test/spec/task
  file was edited. No installs, network, DB, live provider, credentials,
  staging, commit, or push.

## AB-REQ / EV / spec mapping

- **OpenSpec spec:** `agent-credential-isolation/specs/agent-credential-isolation/spec.md:72-84`,
  requirement "Rotação automática ao esgotar conta (Fase 2)", acceptance
  scenario "Sem conta disponível" (`spec.md:82-84`): "o sistema sinaliza o
  esgotamento (alerta) sem sobrescrever credenciais".
- **OpenSpec task:** `agent-credential-isolation/tasks.md:28` "4.4
  Registrar/alertar a troca (aproveitar useSessionMonitor/isExpiringSoon)" —
  unchecked `[ ]`.
- **Proposal context:** `proposal.md:51-57` says the rotation "Aproveita o
  `status`/`expires_at` que o discovery já expõe e o monitor de expiração já
  existente (`useSessionMonitor`/`isExpiringSoon`)." **Finding: those
  referenced monitors do not exist in the current codebase** (see below).
- **AB-REQ mapping (REQUIREMENTS.md):**
  - **AB-REQ-12** (credential+quota lifecycle; ORR) — backend alert emission.
  - **AB-REQ-21** (secret-safe evidence; CLE) — payload redaction.
  - **AB-REQ-38** (operational handover: dashboards/alerts; BCO) —
    operator-visible alert delivery. **The frontend alert-surfacing slice is
    the delivery side of AB-REQ-38; no dedicated frontend AB-REQ exists.**
- **EV mapping:** no existing EV for frontend alerting. This trace proposes
  **`EV-CREDISO-4.4-FRONTEND` = MISSING** (no implementation, no test, no
  indexed evidence). The backend alert slice is `EV-CREDISO-4.4` (PARTIAL —
  see `credential-isolation-4.4-fresh-review.md`).

## Conflict scan (Golden Rule 2: disjunta)

- `FILE_OWNERSHIP.md` bounds a frontend grant only for
  `packages/core/runtimes/models.ts`,
  `packages/views/agents/components/inspector/model-picker.tsx`,
  `packages/views/agents/components/model-dropdown.tsx` (Codex3, vendor/model
  visibility round). **No desktop file is in any bounded grant** → no active
  owner collision for this trace.
- Concurrent worktree changes by other agents (mobile auth-migrate,
  model-picker/dropdown/runtime-picker, server cmd tests) do not overlap the
  desktop daemon-panel / parse-daemon-log / daemon-manager seam analyzed here.
- This trace is read-only; no `files_locked` claimed.

## Exact current files and symbols (frontend alert-surfacing surface)

The codebase has **three** frontend tiers; only the **desktop** app has any
daemon-alert surface.

### Tier 1 — Desktop (Electron): the ONLY existing alert channel

| File | Role | Key symbols |
| --- | --- | --- |
| `apps/desktop/src/main/daemon-manager.ts` | Spawns daemon, tails its log file | `profileLogPath()` `:107` → `…/daemon.log`; `startLogTail()` `:1000`; `watchFile(logPath,…)` `:1051`; `win.webContents.send("daemon:log-line", line)` `:991,:1026`; `stopLogTail()` `:1056` |
| `apps/desktop/src/preload/index.ts` | IPC bridge | `startLogStream` `:258`, `stopLogStream` `:259`, `onLogLine` `:260` → `ipcRenderer.send("daemon:start-log-stream")` / `daemon:log-line` |
| `apps/desktop/src/renderer/src/components/parse-daemon-log.ts` | Pure slog/tint log-line parser | `parseLogLine()` `:76`; `HEADER_RE` `:36` (matches `HH:MM:SS.mmm LEVEL …`); `LogLevel = DEBUG|INFO|WARN|ERROR` `:13`; `ParsedLogLine{id,timestamp,level,message,fields,raw}` `:15` |
| `apps/desktop/src/renderer/src/components/parse-daemon-log.test.ts` | Parser tests | existing pure-parser coverage |
| `apps/desktop/src/renderer/src/components/daemon-panel.tsx` | Generic log viewer (modal) | `DaemonPanel` `:60`; `window.daemonAPI.onLogLine()` `:94`; level filter chips `LEVELS` `:44` (DEBUG off by default, INFO/WARN/ERROR on `:72`); `LEVEL_BADGE_CLASS` `:46` (WARN=warning, ERROR=destructive); search/filter/copy/clear; group-collapse for repeated lines |
| `apps/desktop/src/shared/daemon-types.ts` | Daemon state model | `DaemonState` incl. `"auth_expired"` `:11`; `DAEMON_STATE_COLORS/LABELS` `:42,:52`; **no rotation/no-account/reassignment state** |
| `apps/desktop/src/renderer/src/platform/daemon-reauth.ts` | Dedicated reauth flow for `auth_expired` only | `reauthenticateDaemon()` `:21`; `toast.error` on transient `:39`; `useAuthStore.logout()` on 401 `:35`. **No equivalent for rotation alerts.** |

### Tier 2 — Web (Next.js): NO daemon alert surface

- `apps/web/components/web-notification-bridge.tsx` — routes **inbox
  notification clicks** to the source workspace (`SystemNotificationPayload`
  → `push(paths.workspace(slug).inbox())`). Not daemon/rotation.
- `apps/web/app/[workspaceSlug]/(dashboard)/agents/page.tsx:3` —
  `"Web has no bundled daemon"`; no credential/rotation UI.
- No `useWebSocket`/`ws://`/`EventSource` in `apps/web/app|lib|platform`.
- Realtime sync (`packages/core/realtime/use-realtime-sync.ts`) subscribes to
  `issue:*`, `inbox:new`, `comment:*`, `activity:created`, `reaction:*`,
  `subscriber:*`, `workspace:updated/deleted`, `member:*`, `invitation:*`
  (`:574-788`). **No `daemon:*`, `rotation:*`, `credential_session:*`, or
  `reassignment:*` event type is handled.**

### Tier 3 — Mobile (React Native): NO daemon alert surface

- Mobile references to "session"/"credential" are chat-session/credential
  migration only (e.g. `data/auth-store.ts`, `data/api.ts` —
  `MOBILE-AUTH-MIGRATE` lane). No daemon/rotation/reassignment UI.

## Backend alert sources (the log lines the desktop already tails)

The daemon writes 4.4 alerts to its slog log (the file the desktop tails):

| Alert | Backend anchor (wakeup.go) | slog level | message prefix | fields |
| --- | --- | --- | --- | --- |
| Reassignment completed | `:357-361` `d.logger.Warn("rotation: automatic credential account reassignment completed", attrs...)` | WARN | `rotation: automatic credential account reassignment completed` | `agent_id`, `provider`, `tenant_id`, `previous_account_id`, `next_account_id`, `reason=quota_exhausted_reactive` |
| No account available | `:345-347` `d.logger.Warn("rotation: automatic credential account reassignment unavailable", attrs...)` | WARN | `rotation: automatic credential account reassignment unavailable` | `agent_id`, `provider`, `tenant_id`, `previous_account_id`, `alert=no_account_available` |
| Other failure | `:349` `d.logger.Error("rotation: automatic credential account reassignment failed", attrs...)` | ERROR | `rotation: automatic credential account reassignment failed` | `agent_id`, `provider`, `tenant_id`, `previous_account_id`, `alert=deadline_exceeded|canceled|service_unavailable|reassignment_failed` (`:364-376`) |
| No-op (stale/duplicate/future) | `:352-354` `d.logger.Debug("rotation: credential session discovery produced no reassignment", attrs...)` | DEBUG | `rotation: credential session discovery produced no reassignment` | `agent_id`, `provider`, `tenant_id`, `previous_account_id` |

Attr whitelist at `wakeup.go:336-340`; error class bounded at `:364-376`;
`redact.SanitizeSlogAttr` applied in the alert test logger
(`credential_session_alert_test.go:138`). **These 4 log lines already flow
through the desktop `DaemonPanel` today** as filterable WARN/ERROR/DEBUG rows.

## What is MISSING (the gap)

1. **`useSessionMonitor`/`isExpiringSoon` do not exist.** `grep -rn` over
   `multica-auth-work/{apps,packages}` returns **zero** matches for either
   symbol. The proposal's `proposal.md:55-57` claim "monitor de expiração já
   existente" is stale: the cited legacy paths
   (`src/api/session-discovery.ts`, `src/api/session-store.ts`,
   `src/canvas-reconciler/reconciler.ts`, `infra/cao/auth_routes.py`) **do not
   exist** in the current repo (glob-confirmed; legacy Python/frontend layout
   migrated away). Task 4.4 cannot "aproveitar" hooks that are absent.
2. **No structured event channel daemon→frontend for rotation.** The backend
   `daemon:credential_session_discovery` event is **intra-daemon**:
   `CredentialSessionDiscoveryProducer.Produce` →
   `wakeup.go:274` (daemon's own WS read loop) →
   `dispatchCredentialSessionDiscoveryEvent` → reassignment → slog alert.
   `EmitCredentialSessionDiscovery` has **only test implementations**
   (`producerLoopbackEmitter`, `producerRecordingEmitter`); no production
   emitter and no outbound WS emit. The desktop receives alerts **only** via
   raw `daemon.log` file tailing — there is no `daemon:rotation` / `daemon:reassign`
   IPC event and no `ws.on("rotation:*")` in the realtime layer.
3. **No dedicated rotation-alert UI.** The desktop `DaemonPanel` is a generic
   log viewer; 4.4 alerts appear as ordinary WARN/ERROR rows among all other
   daemon logs. There is no toast, banner, badge, or notification keyed to
   "a rotation happened / no account available / reassignment failed" —
   unlike `auth_expired` which has a dedicated `DaemonState` + `reauthenticateDaemon()`.
4. **Frontend blocker depends on backend 4.3 blocker.** No production
   `NewCredentialSessionDiscoveryProducer` call site exists (4.3 blocker 1);
   therefore no real discovery observation flows into the alert path. Any
   frontend alerting built now would be exercisable only via synthetic logs,
   not live events. (Documented in `credential-isolation-auto-reassignment.md`
   blocker 1 and `credential-isolation-4.4-fresh-review.md` non-claims.)

## Smallest integration seam (design only — no edits)

The **smallest, no-backend-change seam** is a pure message-prefix classifier
on the desktop side, layered on the existing log-tail pipe, emitting a toast.
No new transport, no backend change, no `useSessionMonitor` invention:

1. **`apps/desktop/src/renderer/src/components/parse-daemon-log.ts`** — add a
   pure exported classifier, e.g. `classifyRotationAlert(line: ParsedLogLine):
   {type: "completed"|"no_account"|"failed"|"noop", fields: Record<string,string>}
   | null` that pattern-matches the 4 stable message prefixes above. Pure,
   unit-testable, no React, no IPC. (Message prefixes are stable because they
   are string literals in `wakeup.go:346/349/353/361`.)
2. **`apps/desktop/src/renderer/src/components/parse-daemon-log.test.ts`** —
   add 4 cases asserting each prefix maps to its type and the bounded field
   set (and that a non-rotation line returns `null`).
3. **`apps/desktop/src/renderer/src/components/daemon-panel.tsx`** — in the
   `onLogLine` callback (`:94`), call the classifier; on `completed`/
   `no_account`/`failed`, emit `toast.message/warning/error` (sonner is
   already a dep, `:26`) with a short human message + the `provider`/`tenant`
   fields. No new store needed; the toast is ephemeral. No-op (DEBUG) is
   ignored (matches the backend's DEBUG-only intent).

Optional (only if the TL wants a durable surface beyond ephemeral toasts):
4. **`apps/desktop/src/renderer/src/hooks/use-reassignment-alerts.ts`** (new)
   — subscribe to the log stream, classify, and feed a small Zustand store of
   recent rotation events for a future badge/list. **Not required for the
   minimal acceptance scenario** ("sinaliza o esgotamento (alerta)").

## Safe payload contract (secret-safe — AB-REQ-21)

The frontend must **never** parse or display raw error strings, home dirs,
config paths, session IDs, or tokens. The contract is already enforced
upstream by the daemon (`wakeup.go:336-340` attr whitelist +
`credentialSessionReassignmentErrorClass` `:364-376` bounded class +
`redact.SanitizeSlogAttr`). The frontend classifier must restrict itself to:

```ts
type RotationAlert = {
  type: "completed" | "no_account" | "failed" | "noop";
  agent_id: string;        // ok
  provider: string;        // ok
  tenant_id: string;        // ok (= workspace_id)
  previous_account_id: string; // ok (account id, not a secret)
  next_account_id?: string;    // ok (only on "completed")
  reason?: "quota_exhausted_reactive"; // fixed literal
  alert?: "no_account_available" | "deadline_exceeded" | "canceled"
        | "service_unavailable" | "reassignment_failed"; // bounded class
};
```

**Must NOT pass through:** `HomeDir`, `ConfigDir`, `LastError`, raw `error`
text, session IDs, tokens. The existing `ParsedLogLine.fields` is a
`Record<string,string>` of all trailing `key=value` pairs — the classifier
must **explicitly allowlist** the fields above, not echo `fields` wholesale
(a future non-rotation log line could carry an unrelated secret field into a
toast). Backend redaction is the primary defense; the frontend allowlist is
secondary/defense-in-depth.

## Tenant/account metadata boundaries

- **tenant_id** = `task.WorkspaceID` (the daemon passes `task.WorkspaceID` as
  the tenant scope to `OnExhaustion`/`ReassignDiscoverySession`,
  `daemon.go:4275`). The frontend already keys everything by workspace slug/id;
  `tenant_id` maps directly to the workspace selector. A rotation alert must
  be routed to the UI of its own workspace, not the active one (cf.
  `web-notification-bridge.tsx:14-17` "the SOURCE workspace — not the active
  one").
- **provider** = the vendor (`codex`/`kiro`/`antigravity`/`claude`/`glm`/
  `cline`/`nim`); canonicalized via `canonicalDiscoveryProvider`
  (`detector_discovery.go:56-67`).
- **account_id** = the vendor subscription id, **not** a credential. Safe to
  display. No `HomeDir`/`ConfigDir` (those are credential paths — secret).

## Tests (design — no edits)

The minimal test surface is **pure and offline** (no daemon, no IPC, no
React render):

- `parse-daemon-log.test.ts` additions: feed each of the 4 raw slog lines
  (constructable from the message prefixes in `wakeup.go:346/349/353/361` +
  the field sets in `credential_session_alert_test.go:49-56/75-76`) through
  `parseLogLine` then `classifyRotationAlert`; assert type + allowlisted
  fields. Feed one non-rotation line; assert `null`.
- **No new backend test needed** — the backend alert emission is already
  covered by `credential_session_alert_test.go` (reproduced in
  `credential-isolation-4.4-fresh-review.md`: 6 tests ×20 = 120 PASS/0 FAIL).
- **No DB/network/live** — the classifier is a pure string function.

## Risks

- **Log-line pattern matching is brittle.** If `wakeup.go` message wording
  changes, the classifier silently stops emitting toasts. Mitigation: the
  test pins the exact prefixes; a wording change breaks the test, not
  silence. A structured `daemon:rotation` IPC event would be more robust but
  is a larger backend change (out of minimal-seam scope).
- **No guaranteed delivery.** Log rotation/truncation (`daemon-manager.ts:1039`
  "File rotated/truncated — restart from the new beginning") can drop lines
  during rotation; a toast could be missed. The existing `DaemonPanel` buffer
  (MAX_LOG_LINES=500) has the same limitation. A structured event channel
  would fix this but is not minimal.
- **4.3 blocker makes live alerting moot.** Until a production
  `NewCredentialSessionDiscoveryProducer` call site exists, no real discovery
  observation enters the path; the classifier is exercisable only by feeding
  it synthetic log lines (which is exactly what the test does). Frontend
  alerting is ready-to-wire but dead until 4.3's producer is wired.
- **Secret-safety defense-in-depth.** `ParsedLogLine.fields` is a generic
  `Record<string,string>`; if the classifier ever echoes `fields` wholesale
  into a toast, a future non-rotation line's secret field could leak. The
  allowlist above is mandatory, not optional.
- **Desktop-only.** Web and mobile have no daemon alert surface and no
  bundled daemon; the minimal seam is desktop-only. If the TL wants web
  alerts, that requires a backend→server→WS event path (large, out of scope).

## Owner decision boundary: daemon-only vs frontend-required

The spec acceptance scenario (`spec.md:82-84`) requires only "o sistema
**sinaliza** o esgotamento (alerta) sem sobrescrever credenciais." Two
readings:

| Reading | Verdict | What suffices | Owner |
| --- | --- | --- | --- |
| **(A) "sinaliza" = any operator-visible signal** | **PARTIAL — already met (daemon-only)** | The 4.4 WARN/ERROR slog lines already flow through the desktop `DaemonPanel` as filterable WARN/ERROR rows (`daemon-panel.tsx:44-51`); the operator can filter WARN+ERROR and search "rotation". Backend alert emission is DONE (`wakeup.go:325-377`); secret-safe (attr whitelist + bounded error class + `redact.SanitizeSlogAttr`). **No frontend work strictly required.** | Codex/root (backend, DONE) |
| **(B) "sinaliza" = a dedicated, attention-grabbing alert** (toast/banner/notification, not buried in a log modal) | **MISSING — frontend-required** | The minimal seam above: classifier in `parse-daemon-log.ts` + toast in `daemon-panel.tsx` + 4 unit tests. Desktop-only. | New owner (desktop frontend) — **not Codex3's bounded grant** (model-picker/dropdown only); needs TL assignment |

**Recommendation to TL:** the spec text does not demand a dedicated UI; the
generic WARN/ERROR log viewer arguably satisfies "sinaliza (alerta)". If the
TL accepts reading (A), task 4.4's frontend scope is **N/A — daemon-only
suffices** and the backend `EV-CREDISO-4.4` (PARTIAL, pending 4.3 producer)
closes the alert slice. If the TL wants reading (B), the minimal seam is ~3
file edits + 4 tests on the desktop, owner to be assigned, and it is blocked
on nothing technical (only on the 4.3 producer for live events; the
classifier test runs offline today).

## Non-claims

This trace does **not** claim:
- that `useSessionMonitor`/`isExpiringSoon` exist (they do not);
- that the legacy `src/api/session-discovery.ts`/`session-store.ts`/`reconciler.ts`
  paths exist (they do not);
- that any frontend implementation was produced (none was — read-only trace);
- that the backend alert path is live (it is PARTIAL — 4.3 producer blocker);
- live WebSocket delivery, live daemon, real credential, DB, or network
  behavior;
- TL acceptance. **Not self-accepted.** Kiro/Opus-4.8 adjudicates the
  daemon-only-vs-frontend-required decision and any `EV-CREDISO-4.4-FRONTEND`
  index entry.

## Golden Rule check-in/out

Per `GOLDEN_RULES_E_CHECKIN.md`:
- **Rule 1 (sign-in/out before touching):** this trace touched no file;
  evidence artifact + ledger row only.
- **Rule 2 (disjunta):** no `files_locked` — read-only; no overlap with
  Codex3's bounded frontend grant or `MOBILE-AUTH-MIGRATE`.
- **Rule 4 (nada inventado):** all file/symbol references are from `grep`/`glob`/
  `read` over current source; the non-existence of `useSessionMonitor`/
  `isExpiringSoon`/legacy paths is glob-confirmed, not assumed.
- **Rule 5 (sem segredo):** no credential/token/home/session content read or
  recorded; the safe payload contract is allowlist-only.
- **Rule 9 (só o TL commita / PARE e escale):** no commit; the
  daemon-only-vs-frontend-required decision is escalated to the TL.

## Source SHA-256 manifest (15 traced files, read-only)

Hashes computed only over the paths below. No credential, auth home, session,
token, environment secret, or live service path was traversed or hashed.
`wakeup.go`/`credential_session_monitor.go`/`credential_session_alert_test.go`
match the 4.4 fresh-review manifest; the 3 openspec paths match the
agent-credential-isolation baseline.

```text
5d1a71c97d993b102ee48b889a1f17ce0ca7c4fbf8f3535bd5f54c6b40aad813  multica-auth-work/apps/desktop/src/shared/daemon-types.ts
2eded73ea53577d2b90da56c476d0c05d9177c1df9ae2e66200bdaa1c66aeb4c  multica-auth-work/apps/desktop/src/renderer/src/components/parse-daemon-log.ts
aa101211ad2faf0ecfcaa1cecdbcb7919a89c708a7b935643581cc0b9b561cc2  multica-auth-work/apps/desktop/src/renderer/src/components/parse-daemon-log.test.ts
276297a73dad98fde931f922872cab7855a977a237045b0db2e8c23181a529a9  multica-auth-work/apps/desktop/src/renderer/src/components/daemon-panel.tsx
ad1f5528a2947c86a08aa6ff64e8b252beb24ebceafac541a7a73e6a9f0f5075  multica-auth-work/apps/desktop/src/main/daemon-manager.ts
fe176d429077093620d37066b379530b53c5696895e8f59607d138a2a3a3f0a8  multica-auth-work/apps/desktop/src/renderer/src/platform/daemon-reauth.ts
e02ff0b115dfd1199477820b71d6e389cb2b2e4ddfc624ec18dbc3048044a374  multica-auth-work/apps/desktop/src/preload/index.ts
5541c7c610072e16716e1a2c69ab96e1199aac754eea925d767802875cbb0fc2  multica-auth-work/packages/core/realtime/use-realtime-sync.ts
a5c087f39e3f72cd76295d563db342fae4444a7e3497f09d0b941d74e8137762  multica-auth-work/apps/web/components/web-notification-bridge.tsx
71b89980c9b8cb7c03a81a502da5b9f4bdab4c0c522c117d2147b670e050f730  multica-auth-work/server/internal/daemon/wakeup.go
936b3e40f19c699078e994740c9ae63fdce5c172516b678d289f05c3cc38d1e2  multica-auth-work/server/internal/daemon/credential_session_monitor.go
8e369510e814ff2f5743bf60dc76ee3074110a98da3daa172301aae1fda12bea  multica-auth-work/server/internal/daemon/credential_session_alert_test.go
02b0a5c131c0fd2cee2732dbf4116155d29e87c55d37cc7391994cfdf20da63b  openspec/changes/agent-credential-isolation/specs/agent-credential-isolation/spec.md
a15b62c1d77c899c61b4f5fa39bf975ac4318cc99941f6eab10fbdca8d618636  openspec/changes/agent-credential-isolation/proposal.md
3bdbc1e1916629d87c14db544aee88050daf0e0aaa2c07c3c933798678e7a0e3  openspec/changes/agent-credential-isolation/tasks.md
```
