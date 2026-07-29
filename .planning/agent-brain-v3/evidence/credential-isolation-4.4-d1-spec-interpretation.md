# Credential isolation 4.4 — D1 independent spec interpretation

## Golden Rule check-in / check-out

- **CHECK-IN:** 2026-07-18T21:52:22Z — Codex56#B (Codex-root), independent read-only interpretation. The user-visible check-in and initial source reads preceded this formal timestamp; no shared ledger entry was permitted or made.
- **CHECK-OUT:** 2026-07-18T21:54:38Z — interpretation complete; formatting validated; advisory D1-C recommendation recorded without owner decision or task acceptance.
- **Boundary honored:** only this uniquely named evidence artifact was created. No product, test, OpenSpec, task, shared planning/state/ledger/index, or git/index content was changed. No credentials, environment values, database, network, provider, or live service were accessed. No test or typecheck was needed for this text-and-source interpretation.
- **Authority:** advisory interpretation only. Kiro TL adjudicates D1; root integrates. This artifact does not decide D1, accept task 4.4, or change its open checkbox.

## Bottom line

**A dedicated new frontend notification is not a mandatory acceptance criterion in the current text.** The binding behavior is a durable record plus an operator-visible, secret-safe signal; neither task 4.4 nor the SHALL scenario specifies a browser/mobile UI, toast, banner, or WebSocket contract.

The named `useSessionMonitor`/`isExpiringSoon` parenthetical is more than a random inspiration: both symbols genuinely existed in the historical AgentVerse SPA when this change was framed, and the hook issued frontend `console.warn` messages for expiring sessions. But that historical code monitored impending expiry; it did not announce a completed account reassignment. It was deleted with the “wrong frontend” SPA and has no current product-source counterpart. The parenthetical therefore preserves **implementation intent/pattern**, not a viable mandatory integration target in the present Multica architecture.

The most text-faithful reversible recommendation is **D1-C, with a sequencing qualification**: interpret 4.4's minimum as durable recording plus an operator-visible backend signal, and treat any dedicated toast/banner as a separately owned follow-on. Do **not** close 4.4 merely from the bounded backend tests: current source still has no production construction of `CredentialSessionDiscoveryProducer` and no production `CredentialSessionDiscoveryEmitter`, so production reachability remains an upstream 4.3 gate. This recommendation is advisory, not the owner decision.

## Controlling text and force

### Task 4.4

`openspec/changes/agent-credential-isolation/tasks.md:24-28` places 4.4 after detection, selection, and reassignment. Its minimal wording is “Registrar/alertar a troca,” followed by the `useSessionMonitor/isExpiringSoon` reuse parenthetical.

Textual consequences:

1. The object is **the reassignment** (“a troca”), so 4.4 is not only an early-expiry detector.
2. The slash is compressed checklist prose. It should not be used to weaken the task into “record *or* alert”; the safest reading requires both a record and an alert/signal. The completed backend slice implements both.
3. “Aproveitar” gives an implementation direction—reuse an existing monitor if available—but it does not say the alert **SHALL** be rendered by frontend code. It names no platform, component, toast, banner, public event, or user-interaction acceptance test.
4. Because 4.1-4.3 already own detection/selection/reassignment, the historical hook reference most naturally points to a monitoring/notification pattern, not a second source of truth for rotation.

### Proposal

- `proposal.md:51-57` defines Phase 2 as automatic reassignment without manual intervention and says it uses discovery `status`/`expires_at` and the existing expiration monitor.
- `proposal.md:64-67` lists the historical session frontend and current execenv among affected code for the capability as a whole; it does not make a specific 4.4 frontend UI acceptance scenario.
- `proposal.md:68-69` makes secret safety explicit.

The proposal supports the intent to reuse existing observation machinery. It does not require that the final reassignment alert be a new app notification.

### Normative spec

- `spec.md:72-80` requires automatic same-provider reassignment and continued execution.
- `spec.md:82-84` requires that, when no account is available, the system “sinaliza o esgotamento (alerta)” without overwriting credentials.

The scenario is mechanism-agnostic. “O sistema” is broader than “the frontend,” and no UI/event mechanism is named. It establishes a signal and safety outcome, not a specific presentation layer. It also focuses on **no replacement available**, while task 4.4's “troca” additionally covers successful reassignment. The current backend reports success, unavailable, failure, and no-op outcomes, which is broader than the single explicit spec alert scenario.

## Historical meaning of the named symbols

The GLM trace is correct that the symbols do not exist in current product source, but its description of them as merely stale/external references loses material history.

Repository history establishes:

- Cleanup commit `a61281e963961adeba546332e182b088286caed2` (2026-07-11, subject: “remove AgentVerse SPA (wrong frontend)”) deleted `src/sessions/useSessionMonitor.ts`, `src/api/session-discovery.ts`, `src/api/session-security.ts`, session tests, and the SPA shell.
- Immediately before that commit, `src/sessions/useSessionMonitor.ts:4-12` described a global OAuth-session monitor; lines 16-35 hydrated/refreshed periodically and on focus; lines 37-46 emitted a `console.warn` for sessions already classified as `expiring`.
- Historical `src/api/session-discovery.ts:16-25` used `isExpiringSoon` to derive `expiring`; lines 28-44 fetched `/auth/sessions` with a provider fallback.
- Historical `src/api/session-security.ts:22-28` implemented the 30-minute threshold.
- Historical tests at `src/sessions/__tests__/useSessionMonitor.test.ts:24-90` covered refresh cadence/focus, the warning, and cleanup.

Minimal semantic conclusion: the parenthetical originally pointed to real frontend polling and warning code. It is therefore **historical implementation guidance**, not invented prose. But the historical warning exposed provider plus account email and warned about impending expiration; it neither recorded rotation nor proved a successful/no-account reassignment notification. Recreating it literally would revive a deleted SPA contract and would not by itself satisfy current tenant/provider/redaction architecture.

Current verification:

- current product-source search across `.ts/.tsx/.js/.jsx/.go/.py` has zero `useSessionMonitor` or `isExpiringSoon` matches;
- the names remain only in `tasks.md:28`, `proposal.md:57`, and planning/evidence discussions;
- the old `src/api/session-discovery.ts`, `src/api/session-store.ts`, and `src/canvas-reconciler/reconciler.ts` paths are absent from the current checkout.

Thus “frontend delivery” and “reuse these exact hooks” are not equivalent requirements today.

## Completed backend behavior and its boundary

### Behavior present and boundedly evidenced

- `credential_session_monitor.go:45-56` defines an outcome allowlist containing assignment metadata only; lines 66-108 validate the dedicated event and call the reassignment service.
- `discovery_reassignment.go:42-70` validates agent/account/provider/tenant identity, rejects stale assignment and cross-provider/cross-tenant input, and lines 72-80 marks exhausted then rotates.
- `service.go:97-122` selects a replacement before destructive logout, so `ErrNoAccountAvailable` does not disturb the current assignment/session. Lines 150-159 assign, record rotation, and return success only after `RecordRotation` succeeds.
- `wakeup.go:325-361` emits bounded structured outcomes: WARN for unavailable and completed, ERROR for failure, DEBUG for no-op. Lines 364-376 reduce errors to bounded classes; no raw error is logged.
- The current bounded reviews support the alert invariant: `credential-isolation-4.4-fresh-review.md` SHA-256 `cdb70e85…d95` covers the four outcomes; `credential-isolation-4.4-record-failure-alert-codex-independent-review.md` SHA-256 `2f4094f6…b5` proves a `RecordRotation` failure produces no false completion alert or secret/raw-error leak.

One residual is important but separate from the frontend interpretation: `service.go:150-158` writes assignment before the audit record and has no rollback on record failure. The bounded record-failure review explicitly observes that non-atomic state. D1 cannot cure or waive it.

### Production reachability is not yet proved

The GLM trace correctly mentions the separate 4.3 producer/emitter gap, but phrases such as “already delivered” must be read as **implemented/tested consumer behavior**, not end-to-end production delivery:

- `credential_session_discovery_producer.go:75-147` contains a bounded, concurrency-safe producer type.
- Every current call to `NewCredentialSessionDiscoveryProducer` is in `_test.go` files.
- Every current implementation of `EmitCredentialSessionDiscovery` is in `credential_session_discovery_producer_test.go`.
- `wakeup.go:263-288` can consume and asynchronously dispatch the event, but current non-test source does not construct the producer/emitter that supplies it.

Therefore D1-A or D1-C can answer **whether a frontend criterion exists**, but neither option alone proves task 4.4 production completion. Task 4.3's production path remains a dependency for a live automatic alert.

### Existing frontend visibility is narrower than a dedicated alert

The GLM trace says a new server-to-web event is needed for a web toast; that is correct for web/mobile. It omits an existing desktop surface:

- `apps/desktop/.../daemon-panel.tsx:44-51` recognizes WARN/ERROR; lines 66-108 subscribe to the local daemon log stream; lines 125 onward filter logs; lines 240-340 provide search/filter and display.
- `parse-daemon-log.ts:1-11,29-48,76-95` parses slog WARN/ERROR plus structured fields without dropping unmatched input.

Accordingly, a reachable daemon alert would already be operator-viewable in the desktop's **Local daemon logs** panel when opened. This is frontend visibility, but it is passive/filterable observability—not an attention-grabbing toast/banner, and not web/mobile delivery. The spec does not state which of those levels is required.

## Assessment of the GLM trace

Source reviewed: `.planning/agent-brain-v3/evidence/credential-isolation-4.4-frontend-scope-independent-trace.md`, SHA-256 `7a9020944efd74f2c512fe93933a4e68106d44041d2c240dad26c4b6453ea911`.

| GLM proposition | Independent assessment |
|---|---|
| exact hooks are absent from current product | **CONFIRMED** |
| references are external/archived rather than reusable current hooks | **CONFIRMED, but incomplete** — they were real in-repo AgentVerse SPA code before cleanup, not merely names from planning prose |
| binding spec is mechanism-agnostic | **CONFIRMED** |
| backend outcome is allowlisted and secret-safe | **CONFIRMED by current source and bounded evidence** |
| frontend toast requires new transport/subscription | **CONFIRMED for web/mobile; OVERBROAD for all frontend** — desktop already displays daemon logs |
| backend alert is already delivered | **PARTIAL** — consumer/reporting behavior exists and is tested; production producer/emitter construction remains absent |
| frontend is not textually mandatory | **CONFIRMED** |
| D1-C is the smallest safe recommendation | **CONFIRMED with sequencing qualification** — separate dedicated UX, but do not close 4.4 until production dispatch reachability is proved |

## D1 consequences

| Option | Text fidelity | Consequences | Independent grade |
|---|---|---|---|
| **D1-A — daemon signal only** | High against `spec.md:82-84`; medium-high against task 4.4 because it preserves record+alert but not the historical frontend reuse pattern | No new public event or UI. Operator signal is logs/desktop log panel, not proactive web/mobile notification. Still requires 4.3 producer/emitter production wiring before whole-path acceptance; does not waive Assign→RecordRotation non-atomicity. | **VALID minimum interpretation, not completion proof** |
| **D1-B — dedicated frontend alert mandatory** | Medium: honors the old monitoring/warning intent, but adds a mandatory mechanism absent from SHALL text and cannot reuse the deleted hook directly | Requires an owner-selected platform and contract. Web/mobile needs a tenant-scoped server event, typed payload, subscription, i18n and tests; desktop-only could classify the existing allowlisted log and show a toast. Larger attack/compatibility surface and harder to reverse once public event semantics ship. | **PERMISSIBLE stronger product requirement, not compelled by current text** |
| **D1-C — split dedicated UI follow-on** | Highest combined fidelity: keeps current record+signal semantics and preserves historical UX intent without pretending deleted code is reusable | 4.4 remains backend/observability-scoped; a separately owned frontend task can define platform, UX and EV explicitly. Reversible because no public contract is introduced until chosen. Must not be read as permission to close 4.4 before production reachability and existing backend residuals are adjudicated. | **RECOMMENDED, advisory** |

## Recommendation without owner decision

Recommend that Kiro TL choose the **interpretive shape of D1-C**:

1. define task 4.4's current frontend criterion as **not mandatory**;
2. retain the behavioral acceptance bar of durable recording plus an operator-visible, secret-safe signal;
3. require production-path evidence before task closure (the 4.3 producer/emitter dependency is not erased);
4. place any dedicated desktop/web/mobile toast or banner in a separately owned follow-on with an explicit platform, tenant-scoped transport, allowlisted payload and tests.

This is the most reversible choice because it does not manufacture a new public event from an ambiguous parenthetical, and the most text-faithful because it preserves both the task's record/alert outcome and the proposal's historical monitoring intent. Kiro TL may instead select D1-B if proactive in-app attention is a product requirement; that would be a deliberate scope strengthening, not discovery of an existing mandatory criterion.

## SHA-256 / immutable history manifest

Current checkout: `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f`.

| SHA-256 / object | Source |
|---|---|
| `3bdbc1e1916629d87c14db544aee88050daf0e0aaa2c07c3c933798678e7a0e3` | `openspec/changes/agent-credential-isolation/tasks.md` |
| `a15b62c1d77c899c61b4f5fa39bf975ac4318cc99941f6eab10fbdca8d618636` | `.../proposal.md` |
| `92ffdf6b414a76d3fbd8baa8f32a6497af05b5fc325b0a00ef180cafd10fc40a` | `.../design.md` |
| `02b0a5c131c0fd2cee2732dbf4116155d29e87c55d37cc7391994cfdf20da63b` | `.../specs/agent-credential-isolation/spec.md` |
| `7a9020944efd74f2c512fe93933a4e68106d44041d2c240dad26c4b6453ea911` | GLM frontend-scope independent trace |
| `71b89980c9b8cb7c03a81a502da5b9f4bdab4c0c522c117d2147b670e050f730` | `server/internal/daemon/wakeup.go` |
| `936b3e40f19c699078e994740c9ae63fdce5c172516b678d289f05c3cc38d1e2` | `server/internal/daemon/credential_session_monitor.go` |
| `4e240c09af1653fd2ecdbcd0763f8c05d3eb3f57ee8b9c339045edfa32a9ce6c` | `server/internal/daemon/credential_session_discovery_producer.go` |
| `c655c9c4f94716d2c93109b1cc4d3c14e62d6fe8dea615dddec3ec93769b2832` | `server/internal/rotation/discovery_reassignment.go` |
| `f20951a36765d2b6b8576232871f1991697de56d8577377b0226ed5e30130cf0` | `server/internal/rotation/service.go` |
| `276297a73dad98fde931f922872cab7855a977a237045b0db2e8c23181a529a9` | desktop `daemon-panel.tsx` |
| `2eded73ea53577d2b90da56c476d0c05d9177c1df9ae2e66200bdaa1c66aeb4c` | desktop `parse-daemon-log.ts` |
| `a61281e963961adeba546332e182b088286caed2` | cleanup commit deleting the historical AgentVerse SPA |
| `a3958e57e87e152d838a98fba08c9552e4cb979598f7ff4e5a78031994e52375` | historical `a61281e^:src/sessions/useSessionMonitor.ts` blob content |
| `ec51dbc7ff7422986685529a5748961cf3369e8cc6f550e0cd4b513dd58795c4` | historical `a61281e^:src/api/session-discovery.ts` blob content |
| `b388d10bb886667f2154af8e42a80fd6363b4218368df7d04540418cb9766132` | historical `a61281e^:src/api/session-security.ts` blob content |

## Method and non-claims

Read-only commands used: `openspec list --json`; `rg`, `nl`, `sed`, `find`, `sha256sum`; `git show`, `git grep`, `git log -S`, `git rev-parse`, and `git status` limited to repository source/history. One proposed helper command containing temporary-file cleanup was rejected before execution; no temporary file was created. No test, typecheck, build, formatter, DB, network, credential, environment-value, or service command ran.

- No assertion that task 4.4 is accepted, production-deployed, or safe to close.
- No assertion that desktop log visibility equals a dedicated notification UX.
- No assertion that the 4.3 production producer/emitter, cross-process locking, Assign/RecordRotation atomicity, or destructive logout risks are resolved.
- No resurrection of the deleted AgentVerse frontend is recommended.
- No OpenSpec wording, checkbox, EV, shared ledger/state/index, source, test, or git state was modified.
