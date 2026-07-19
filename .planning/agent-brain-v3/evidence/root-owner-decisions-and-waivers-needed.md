# Root/owner decisions & waivers needed to unblock push — consolidated (ADVISORY)

Read-only consolidation of every **owner-only** decision currently blocking push. **Advisory only:** each
item lists evidence hashes, the safest reversible options, the exact consequence, and a **recommended
choice — without making the choice.** No decision is enacted here.

- Author: **Kiro/Opus-4.8, session `w8:p2`** (read-only synthesis). Adjudicator/decider: **Kiro TL /
  root owner**. Integrator: root.
- HEAD `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f`. Counts (per `AGENT_LEDGER`): build 51/85, chat 4/10,
  **cred 4/21**, native 9/17, **persist-prodex 0/16**.
- **Identity caveat:** producer/reviewer/adjudicator roles across these items are pervasively the same
  **Kiro/Opus-4.8** model family in different sessions; separation is by session/pane, not identity.
  Where a producer/reviewer is unnamed in the records, this doc says so and **fabricates nothing**.
- Related prior consolidation (not superseded): `current-owner-decisions-for-integration.md`
  (`120cfe7b7e240e1e137e04ebbecf3c49e126735c981965337bea0c056bcdfca3`). This artifact is additive.

## CHECK-IN 2026-07-18T21:56:00Z
Mode: READ-ONLY synthesis. Sole writable deliverable = this file. Excluded (honored): no
STATE/LEDGER/EVIDENCE_INDEX/tasks/spec/source/test/git/index/refs edit; no credentials/env values; no
DB/network/services. No checkbox changed; no acceptance; no push authorized.

---

## Cross-cutting decision classes (recur across items)
- **W = Waiver** (owner accepts an irrecoverable governance gap, e.g. unattributed producer / missing
  pre-edit check-in — *never backdate*).
- **R = Remediation** (produce a named artifact / distinct review / pinned manifest / disjoint test —
  reversible, additive).
- **H = Hold** (do nothing; item stays OPEN).
All three are reversible except an actual push; no option below authorizes a push.

---

## 1. Credential **4.1** — provenance thinness (task `[x]`)
- Evidence: review `credential-isolation-4.1-push-eligibility-independent-review.md`
  (`d41bbc21ba54a6128e138e81e51914ce714b3151e941ed58953085406cca9324`); files
  `detector_discovery.go` `bc61a46c…45b55`, `detector_discovery_test.go` `4e8092ff…5a4f` (both untracked).
  Technically GREEN + clean-room 2-file build closed; EV-CREDISO-4.1 exists **only as a ledger/index row**
  (no standalone artifact, **producer unnamed**); sibling matrix flags **HOLD**.
- Options: **(R)** owner authors/accepts a named EV-4.1 artifact with a source manifest → artifact-grade;
  **(W)** owner waiver accepting ledger-row provenance as sufficient; **(H)** leave (no push).
- Consequence: R = push-grade provenance, small doc cost; W = fast but sets a precedent that ledger-row
  acceptance suffices; H = 4.1 detector cannot ship even though technically inert/complete.
- **Recommended (not decided): R** — cheapest durable fix; 4.1 is technically proven and additive.

## 2a. Credential **4.4** — D1 scope + contract (task `[ ]`)
- Evidence: record-failure-alert review `credential-isolation-4.4-record-failure-alert-independent-review.md`
  (`e9c43ea8…19fa`, corrected provenance); fresh review `credential-isolation-4.4-fresh-review.md`
  (`cdb70e85…d08d95`). Ledger:250 = **technical PASS, contract REJECT** (missing EV/AB-REQ/provenance/
  distinct-reviewer). Task text names the **frontend seam** `useSessionMonitor/isExpiringSoon`.
- **Owner decision "D1" (scope):** does 4.4 include the **frontend session-monitor integration**, or is it
  **backend record/alert-only**? (No literal "D1" token found in-artifact; described by content, not
  fabricated.) Also: the non-atomic **Assign-before-RecordRotation** behavior (documented, not cured) is an
  owner architecture call.
- Options: **(D1-scope)** owner declares 4.4 = backend-alert-only (frontend tracked separately) **or**
  4.4 includes the frontend seam; **(R)** docs-only contract repair (pin EV/AB-REQ/provenance/distinct
  reviewer — already coordinated w/ Codex56#D per ledger); **(H)** hold.
- Consequence: backend-alert-only + R clears 4.4 quickly; including the frontend seam enlarges scope and
  pulls in unresolved frontend-owner questions. Non-atomicity left as-is is acceptable for *alerting* but
  is a latent correctness residual.
- **Recommended (not decided): declare D1 = backend-alert-only + R (docs-only contract repair)**; track the
  frontend seam and non-atomicity as separate owner items.

## 2b. Credential **4.3** — production reachability (task `[ ]`)
- Evidence: trace `credential-isolation-4.3-production-integration-gap-trace.md`
  (`d793712503932cee5eb9757f18d252e6a89da8c2ff63a4cdd6874ba0790454b4`). **PRIMARY GAP:** the discovery
  **producer/emitter is test-only** — no production code emits `daemon:credential_session_discovery`
  (consumer/monitor/alerts are fully wired). Secondary owner gates: cross-process lock (in-proc mutex
  only), non-atomic Assign+Record, destructive logout-before-login hazard on shared home.
- Options: **(R, owner-gated)** implement the smallest-safe NEW producer/emitter (daemon-owner one-line
  wiring) + a rotation-owner atomic `AssignAndRecordRotation` + disjoint tests; **(W/scope)** accept
  reactive **Path A** (already wired) as satisfying "no manual intervention" and defer discovery-driven
  Path B producer; **(H)** hold.
- Consequence: R = true production auto-reassignment but needs daemon-owner (Codex1) + rotation-owner
  (W-PGSTORE) + concurrency-policy sign-off; scope-accept Path A ships less but is honest about the gap;
  H = 4.3 stays open.
- **Recommended (not decided): owner picks concurrency/atomicity policy first, then R** on new files only;
  do **not** push a "4.3 complete" claim while the producer is test-only.

## 3. Credential **5.3** — missed producer pre-edit check-in (task `[ ]`)
- Evidence: my independent review `credential-isolation-5.3-clean-reexecution-independent-review.md`
  (`572e166120daa19bcbc80d1789cbbb854eebfe70e7e090b6f76d617c00dc3343`); reviewed re-exec `80a930d0…`;
  14/14 source hashes match; **3 independent technical reproductions** (test genuine). Standing ruling
  (ledger): a reproduction does **not** cure the original missing pre-edit check-in.
- Options: **(W)** owner-accepted, documented process-exception waiver; **(R, option-b)** a *producer*
  re-does the original edit under a proper pre-edit check-in; **(H)** hold.
- Consequence: W = closes 5.3 immediately given 3 genuine reproductions (precedent: process-exception
  waiver); R = strictest, but re-edits already-correct bytes; H = 5.3 stays open indefinitely.
- **Recommended (not decided): W** (documented waiver) given the triple genuine reproduction; R only if the
  owner wants zero process-exception precedent.

## 4a. Credential **5.4** — redaction-core provenance + canonical EV (task `[ ]`)
- Evidence: provenance audit `credential-isolation-5.4-redact-core-provenance-audit.md`
  (`dbf7033bc8bb7dc96c24fcfcb4d03d94282397dfc854c25d73c1642c56814bde`); clean-room review
  `…clean-room-independent-review.md` (`129025cc…24e4`); core files `redact.go` `f409ba8a…f68a5c`,
  `redact_test.go` `5a37941a…602fec9`; generated Claude delta `c7922b7b…5ede9`. Gaps: **unattributed
  "16:15:46" producer / no pre-edit check-in**; **missing distinct Gemini/GLM review** (adjudication gate,
  ledger:242/245); **no pinned canonical EV-CREDISO-5.4-CORE hash** (core-review `521cef31…` unpinned).
- Options: **(R)** obtain the pending Gemini/GLM distinct review + pin the canonical EV-CORE hash + add
  internal provenance to the core-review; **(W)** owner waiver for the unattributed producer/check-in;
  **(H)** hold.
- Consequence: R+W together make the 4-file atomic unit (core + Claude 2-hunk delta `c7922b7b`) push-ready;
  Claude side is already technically PASS. Missing Gemini review is the hard gate.
- **Recommended (not decided): R (get Gemini review + pin EV hash) + W (producer attribution waiver)**;
  push the Claude slice only as the regenerated 2-hunk delta, never the mixed working-tree `claude.go`
  (`3f9dc4fb`).

## 4b. Credential **5.4** — Google/Cloud external-body seam disposition
- Evidence: residual audit `credential-isolation-5.4-remaining-absolute-log-safety-gaps-independent-review.md`
  (`5a927fbdf8543a1b1000a2f7820f45971944ee4dcfbfe84ba255c30ccab3fc4c`). Two external-response-body sinks —
  `internal/handler/auth.go:656` **and** `internal/auth/cloud_pat.go:359` — logged under `"body"`,
  **pattern-dependently** redacted via the confirmed `slog.Default` `SanitizeSlogAttr` hook; **no
  site-level test evidence**; `cloud_pat.go:359` was not enumerated by the producer's sweep.
- Options: **(R)** add disjoint handler tests for **both** sinks; **(W)** owner accepts them as
  non-blocking residual **R-5.4-B** with the honest bar "absolute for known secret shapes routed through a
  redacting logger"; **(H)** hold 5.4.
- Consequence: R = closes the test-evidence gap; W = accepts an inherent pattern-dependency residual (a
  token in an unmatched shape/key could leak); H = 5.4 stays open.
- **Recommended (not decided): W with the bounded wording + enumerate `cloud_pat.go:359`**; R optional as
  cheap hardening.

## 5. Chat **1.1 / 1.4** — provenance & checkbox authority (tasks `[x]`)
- Evidence: provenance reconciliation `chat-orchestration-1.1-1.4-provenance-reconciliation.md`
  (`2fd701ba100fbba80f60686cb7bf9717593509ebd115a1524f416657002d2198`); clean-room `e8d1d1ce…` (Codex56#B,
  distinct from reviewer Codex#56#A); EV artifact `c7064375…`. Technical COMPLETE (24 AST assertions +
  daemon ×20/race; handler tests **compile-only**, DB-gated). Gaps: **producer identity + pre-edit
  check-in irrecoverable**; **checkbox contradiction** (`[x]` set vs "no checkbox set" in accept rows).
- Options: **(W)** owner waiver for the unattributable producer/check-in **and** reconciliation of who set
  `1.1/1.4 [x]`; **(R)** producer self-attests going forward; **(H)** hold.
- Consequence: W+reconcile clears the only READY-1 blockers (dependency-complete, two distinct reviewers);
  H leaves accepted-but-unpushable state. Handler-runtime remains unproven (DB-gated) regardless.
- **Recommended (not decided): W + checkbox-authority reconciliation**; note the compile-only handler bound
  in the commit rationale.

## 6. Native **1.7** — atomic-push boundaries (task `[x]`)
- Evidence: my independent review `native-onboarding-1.7-push-eligibility-independent-review.md`
  (`1bc6ca4385ee184b8c7d047732b90ee3ca33a4f8a30dae6ab813e5ed2c818dba`); eligibility review `1937b6fc…`
  (Codex56#A = **HOLD**); `auth.go` `d69877a9…` (all hunks 1.7); EV-AUTH-1.7 = 17-file manifest. Blockers:
  **producer + accepting reviewer unnamed**; essential **`auth_routes_test.go` (`7e814662…`) outside the
  accepted manifest**; **`.env.example` needs env-permitted reconciliation** (bytes not read).
- Options: **(R)** name producer + a **distinct** reviewer (or waiver) **+** pin `auth_routes_test.go` into
  the manifest (+re-hash) **+** env-permitted `.env.example` reconciliation; **(W)** waiver for the
  irrecoverable attribution only; **(H)** hold.
- Consequence: R closes all three HOLD conditions; partial R leaves push blocked. CLI/rotation/14 topology
  tests/frontend-1.5/mobile stay out of the atom regardless.
- **Recommended (not decided): R (all three) + W for irrecoverable attribution**; keep the 14-file backend
  atom exact.

## 7. Persist-prodex — program hold (0/16)
- Evidence: TL-reproduced review `persist-prodex-runtime-1.1-1.3-review.md`
  (`ed27595e600a33594b1003cddb2c14b4f60594065ca40e7356c97c0d87fe825d`). Ledger:243/247/250 — 1.1–1.3
  **REOPENED** (Opus48#A **self-check**, PENDING independent QA); TL independently reproduced the 9 named
  tests (`-count=20 -race`, PASS) but the **review's evidence contract is INCOMPLETE** (0 source SHA-256,
  no in-artifact reviewer identity/provenance, no AB-REQ/EV mapping). 2.1–2.2 design + 3.5 purge design are
  separate pending items.
- Options: **(R, docs-only)** add source-hash manifest + distinct-reviewer identity/provenance (Gemini ≠
  producer Opus48#A) + AB-REQ/EV mapping, then re-adjudicate; **(H)** keep program hold; **(full re-QA)**
  require a fresh independent QA rather than docs-only.
- Consequence: R = reclosure path with test proof already genuine (cheapest); H = 0/16 persists; full re-QA
  = highest assurance, highest cost.
- **Recommended (not decided): R (docs-only contract completion) for 1.1–1.3**, then re-adjudicate;
  keep 2.1–2.2 / 3.5 on their own tracks.

---

## Owner decision summary (advisory)
| Item | State | Class of owner decision | Recommended (not decided) |
|---|---|---|---|
| cred 4.1 | `[x]`, tech-green, provenance thin | R vs W | R (named EV-4.1 artifact) |
| cred 4.4 (D1) | `[ ]`, tech-PASS/contract-REJECT | D1 scope + R | D1=backend-alert-only + R |
| cred 4.3 | `[ ]`, producer test-only | concurrency/atomicity policy + R | policy first, then R (new files) |
| cred 5.3 | `[ ]`, 3× reproduced | W vs R(option-b) | W (documented waiver) |
| cred 5.4-core | `[ ]`, tech-green | R (Gemini review + EV pin) + W | R + W |
| cred 5.4-seam | `[ ]` | W (R-5.4-B bounded) vs R(tests) | W + enumerate cloud_pat.go:359 |
| chat 1.1/1.4 | `[x]`, tech-complete | W + checkbox reconcile | W + reconcile |
| native 1.7 | `[x]`, HOLD | R(3 conditions) + W | R + W |
| persist 1.1–1.3 | REOPENED 0/16 | R(docs-only) vs full re-QA | R (docs-only) |

**Recurring owner themes:** (1) unattributed producer / missing pre-edit check-in → **Waiver-or-attribute**
(4.1, 5.3, 5.4-core, chat 1.1/1.4, native 1.7); (2) `[x]` set vs "no checkbox set" → **checkbox-authority
reconciliation** (4.1, chat 1.1/1.4); (3) ledger-row vs artifact-grade provenance + unpinned EV hashes →
**pin canonical EV** (4.1, 5.4-core, persist); (4) genuine architecture gates → **owner policy** (4.3
concurrency/atomicity; 4.4 non-atomicity/frontend scope; DEC-CREDISO-1.1-1.3).

## Non-claims
- Advisory only. **No decision made**; no checkbox/STATE/LEDGER/INDEX/tasks/spec/source/test/git/index/ref
  change; no credentials/env values; no DB/network/services. No identity fabricated — unknowns recorded as
  unknown. Kiro TL / root owner decides; TL must re-hash immediately before any integration.

## CHECK-OUT 2026-07-18T22:00:00Z — DONE
Only this file created. Everything else unchanged.
