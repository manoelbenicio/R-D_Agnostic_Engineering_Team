# Current owner decisions required for integration (co-lead brief)

- Author: Kiro/Opus-4.8 (co-lead adjudicator) — 2026-07-18
- Scope: consolidates ONLY decisions that require **owner authority** before acceptance/push. Co-lead **does not select**; each item lists an evidence-backed default recommendation, exact consequence, the smallest reversible choice, and what stays blocked.
- Read-only except this file. No git/index/product/test/spec-checkbox/credential/network/service change.
- Counts unchanged: build 51/85 · chat 4/10 · cred 4/21 · native 9/17 · persist-prodex 0/16 (PROGRAM HOLD).

## Source evidence (SHA-256, verified on disk 2026-07-18)

| Artifact | SHA-256 |
|---|---|
| `credential-isolation-4.4-fresh-review.md` | `cdb70e85fab9131c3aff59e52d037a57a8ee080265a51440bde071c526d08d95` |
| `credential-isolation-1.1-1.3-architecture-decision.md` | `e9949a7fc8cfb02228256fdb709631acda01a4a374bab8c9f046d793f53dbc1a` |
| `credential-isolation-config-env-audit.md` (arch input) | `7fb12ec8b1a4e85209cef4f85f4d88c82dc5520b5797bb29df1339ddd7abcef4` |
| `persist-prodex-vs-omniroute-reconciliation-audit.md` | `e1a654162c57e3e2ea7a8330007105c6647d4adfd4a1b0c4132fe57ee28c7504` |
| `packetb-staged-frontend-push-ownership-review.md` | `1a4d58ddebfb2b3728d478d30497e850a0aff269a3c4b6a8666aca14800df6be` |
| `credential-isolation-task-5.3-automatic-rotation.md` | `aece8372e620e6dbf572b9dce70e4abedc675f2bd614b84c10abccfae20367b7` |

---

## D1 — Credential 4.4 alerting scope: daemon-only vs required frontend hooks

- **Question:** Is task 4.4 ("Registrar/alertar a troca — aproveitar `useSessionMonitor`/`isExpiringSoon`") satisfied by **daemon structured-log alerting alone**, or does it **require the frontend `useSessionMonitor`/`isExpiringSoon` surface**?
- **Evidence:** task text + `proposal.md:57` explicitly name the frontend hooks; `spec.md:84` THEN is mechanism-agnostic ("o sistema sinaliza o esgotamento"). Fresh review (`cdb70e85…`) delivers daemon logs and **non-claims** the frontend UI; distinct reviewer + 120 PASS confirmed.
- **Default recommendation (evidence-backed):** treat the frontend hook wiring as in-scope because the task/proposal name it; accept daemon-only ONLY via an explicit owner re-scope.
- **Exact consequence:** daemon-only = operators see logs but end users get no in-app expiry/switch alert the proposal envisioned; frontend-required = additional UI lane + evidence before 4.4 closes.
- **Smallest reversible choice:** owner declares 4.4 = daemon-signal-only and **moves the frontend hook wiring to a separate frontend task** (reversible re-scope note; UI can still be added later).
- **Stays blocked regardless:** the missing **fail-closed failure-path test** (synthetic `RecordRotation` failure → assert no success alert) is required for 4.4 acceptance under either scope.

## D2 — Credential architecture 1.1–1.3: Option A vs dev-only B (+ provider/session/migration boundaries)

- **Question:** Adopt **Option A** (explicit tenant-scoped account, no implicit global fallback) or **Option B** (dev/test-only gated implicit fallback)? Plus the provider-discovery-root, session-API, and legacy-migration boundaries.
- **Evidence:** `e9949a7f…` (RECOMMENDATION READY; PRODUCT-OWNER DECISION REQUIRED), input audit `7fb12ec8…`. Option A = strongest; production already fails closed (no regression). Option B = weaker, adds a second launch path + support surface.
- **Default recommendation:** **Option A** (matches the artifact's recommendation and fail-closed posture).
- **Exact consequence:** A = one-time bootstrap/import + explicit assignment for legacy/workstation setup; B = retains ambient daemon-user state, must be mutually exclusive with gateway-required/rotation/multi-account/session APIs and validated at startup.
- **Smallest reversible choice:** **Option A now** — reversible in the permissive direction (a gated B mode can be added later if a real dev bootstrap need appears); A cannot be safely "added" after B leaks ambient state.
- **Stays blocked:** cred-iso 1.1–1.3 implementation + independent evidence; **PP task 3.5 purge** (deletes credential files) is downstream of this decision + PD-08.

## D3 — persist-prodex PROGRAM HOLD: Options A / B / C

- **Question:** Disposition of `persist-prodex-runtime-integration`, which conflicts with OmniRoute **0.5** (blocks concurrent superseded-Prodex execution) and **7.8/AB-REQ-37** (Prodex default-off/drain/removal) — per audit `e1a65416…`.
- **Options:** **A** sanctioned transitional + sunset (default-off/flag reconciliation with 7.8, sunset bound to AB-REQ-37/10.4/10.5, subordinate to AB-REQ-36, artifacts in REMOVAL_REGISTER); **B** defer/decline new persistence (rely on 7.8 default-off drain flag); **C** minimal reversible continuity (documented manual/foreground start, no durable service/launcher).
- **Default recommendation:** **B** (lowest removal debt, honors 0.5/7.8/AB-REQ-37); escalate to **C** only if restart-continuity during drain is genuinely required; **A** only with an explicit sunset waiver.
- **Exact consequence:** A = entrenches legacy path, enlarges deletion surface (10.4/10.5), risks slowing cutover; B/C = minimal debt, may require manual restart during drain.
- **Smallest reversible choice:** **C** if any continuity is needed (smallest reversible mechanism, no new durable legacy artifact); otherwise **B**.
- **Stays blocked:** ALL persist product/test implementation is FROZEN and persist code EXCLUDED from push scope until this decision.

## D4 — Packet B task ownership + `client.ts` disentanglement

- **Question:** Which traceable OpenSpec task owns the model/runtime **picker UI**, and how is `client.ts` (dual-claimed) disentangled? Per ownership review `1a4d58dd…`.
- **Evidence:** no accepted task owns the picker UI (1.4 backend-only `[x]`; **1.5/1.6 `[ ]` BLOCKED**); Packet B evidence = "PRODUCED — pending independent re-review"; `client.ts` dual-claimed by native 1.5/1.7 auth-login (Bucket C REJECTED/REOPENED) ∧ Packet B AbortSignal passthrough (PRODUCED/pending); `types/agent.*` UNKNOWN/UNOWNED.
- **Default recommendation:** owner authorizes/identifies a traceable task for the picker UI (or re-scopes 1.5/1.6), then a **distinct** re-review accepts against it; require `client.ts` split into its two lanes before any commit.
- **Exact consequence:** committing now would push unaccepted UI and **entangle a REOPENED lane (auth-login) with a PENDING lane (vendor UI)** in one file.
- **Smallest reversible choice:** **keep all 11 staged files excluded** (status quo, fully reversible; git index untouched) until owning task + `client.ts` split + distinct acceptance.
- **Stays blocked:** push of the 11 staged frontend files; G1 remains frozen.

## D5 — Credential 5.3 governance: waiver vs clean re-execution

- **Question:** The 5.3 rotation reproduction is genuine (distinct GLM52-auth-QA + TL re-reproduction, 60/60, race) BUT the producer **missed the pre-edit Golden Rule check-in** (`aece8372…`; retrospective disclosure does not cure it). Accept via **owner waiver** or require **clean re-execution**?
- **Default recommendation (co-lead governance ruling):** the missing check-in is **not self-waivable**; prefer **clean re-execution with a proper pre-edit check-in**, OR an explicit, bounded, one-time owner waiver.
- **Exact consequence:** waiver = accepts existing genuine reproduction but sets a precedent (must be explicitly bounded/one-time to avoid eroding pre-edit-check-in integrity); re-execution = clean provenance, small repeat cost.
- **Smallest reversible choice:** **owner-issued one-time documented waiver** (reversible policy act) that explicitly does not generalize — accepts the existing reproduction without re-running.
- **Stays blocked:** 5.3 acceptance until waiver or re-execution; process-exception pattern (also Codex56#D, Gemini chat 1.2/1.3, Opus48#A persist, cred 4.4-prior) stays on record.

---

## Actionable decision table

| ID | Decision | Default rec | Smallest reversible | If undecided, blocked |
|---|---|---|---|---|
| D1 | 4.4 alerting scope | Frontend in-scope unless re-scoped | Re-scope to daemon-signal-only + separate FE task | 4.4 acceptance (+ failure-path test always required) |
| D2 | cred arch 1.1–1.3 | Option A (no implicit fallback) | Option A now (B addable later) | cred 1.1–1.3 impl/evidence; PP 3.5 purge |
| D3 | persist PROGRAM HOLD | B (defer) | C (minimal reversible continuity) | all persist impl/test (frozen) + push exclusion |
| D4 | Packet B ownership + client.ts | Authorize owning task + split client.ts | Keep 11 excluded (status quo) | push of 11 staged files; G1 |
| D5 | 5.3 governance | Clean re-exec (or bounded waiver) | One-time documented owner waiver | 5.3 acceptance |

**Co-lead selects none of the above.** All are owner-authority gates. Nothing here changes a checkbox, product, test, git index, credential, or spec. PD-01/PD-08, PROGRAM HOLD, and all STOP/OPEN gates remain in force.
