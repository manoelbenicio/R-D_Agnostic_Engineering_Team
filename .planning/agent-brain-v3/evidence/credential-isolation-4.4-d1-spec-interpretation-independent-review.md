# Independent review: agent-credential-isolation 4.4 D1 spec interpretation (producer Codex56#B)

**Reviewer:** Kiro/Sonnet, pane `w7:p1` — independent review only, distinct
from producer Codex56#B (Codex-root) and from both prior GLM traces
(GLM52-auth-QA, GLM52#B).
**Preparatory pass date:** 2026-07-18T18:51:02-03:00 to 19:05:00-03:00.
**Finalization pass date:** 2026-07-18T19:26:25-03:00 to completion.
**Adjudication authority:** Kiro TL adjudicates; owner decides D1. This
review does not accept, reject, or select D1-A/B/C on its own authority —
it grades the producer's artifact only.

## STATUS: FINALIZED

The producer artifact
`.planning/agent-brain-v3/evidence/credential-isolation-4.4-d1-spec-interpretation.md`
now exists on disk.

## Golden Rule check-in (finalization pass)

- **Check-IN:** 2026-07-18T19:26:25-03:00 — claimed scope: re-verify the
  producer's artifact hash, spot-check its most load-bearing source/history
  citations (the AgentVerse deletion commit, the historical hook contents,
  the current-repo absence of `useSessionMonitor`/`isExpiringSoon`, the
  backend production-reachability claim), then finalize this document only.
  No product/test/shared/spec/task edit. No git/index mutation beyond
  read-only `git show`/`git log`/`grep`. No credential/env/DB/network/service
  access.
- Note on tooling: an initial broad `grep -r` search hung and was
  interrupted (per live steering); the same verification was re-run with a
  scoped tool call and completed normally with zero matches, confirming the
  producer's absence claim rather than being blocked by it. No work was
  duplicated — the interrupted call produced no output and was abandoned,
  not repeated.
- **Check-OUT:** 2026-07-18T19:35:00-03:00 — DONE. Verdict below is final.

## Producer artifact hash — verified

```
57eb8a30800c925fea5367461f39381f5c92da45412c97eddca6559f7f466be9  .planning/agent-brain-v3/evidence/credential-isolation-4.4-d1-spec-interpretation.md
```

Confirmed via independent `sha256sum` against the exact hash reported in the
task instruction (`57eb8a30...`) — **exact match**. Producer: **Codex56#B
(Codex-root)**, self-identified in the artifact's own Golden Rule
check-in/out section with timestamps `2026-07-18T21:52:22Z` (check-in) and
`21:54:38Z` (check-out).

## Independent re-verification of the producer's load-bearing claims

I did not accept the producer's citations on faith; I independently
re-executed the checks that matter most for the D1 question.

### 1. The AgentVerse deletion commit — CONFIRMED, exact match

```
git log -1 --format='%H %s' a61281e963961adeba546332e182b088286caed2
→ a61281e963961adeba546332e182b088286caed2 chore(cleanup): remove AgentVerse SPA (wrong frontend) — keep Multica + prodex only
```
The commit exists, and its subject line independently corroborates the
producer's framing (not merely "stale prose" but an actual deleted SPA).

### 2. Historical `useSessionMonitor.ts` content — CONFIRMED, exact match

```
git show a61281e963961adeba546332e182b088286caed2^:src/sessions/useSessionMonitor.ts
```
Read the full historical file independently. It matches the producer's
description precisely: a `useEffect`-based hook that hydrates/refreshes
every 5 minutes and on window focus, and **logs a `console.warn`** for
sessions with `status === 'expiring'`, listing `${s.cli_provider}:
${s.account_email}`. This independently confirms two of the producer's
specific, checkable claims: (a) the historical hook only warned about
*impending* expiry (not completed reassignment — a distinction the producer
uses to argue the parenthetical cannot be a literal reuse target for task
4.4's actual subject, "a troca"), and (b) the historical warning exposed
`account_email` — a materially more sensitive field than anything the
current backend allowlist exposes. This is a real, independently-verified
strengthening of the producer's "cannot recreate as-is" argument that
neither prior GLM trace surfaced (both traces stopped at "these symbols
don't exist"; the producer went further to characterize what they did when
they existed and why that characterization still doesn't satisfy 4.4).

### 3. Historical `isExpiringSoon` (`session-security.ts` / `session-discovery.ts`) — CONFIRMED, exact match

```
git show a61281e963961adeba546332e182b088286caed2^:src/api/session-discovery.ts
```
Confirms `isExpiringSoon` was imported from `./session-security` and used
to derive a client-side `'expiring'` status band from `expires_at`,
consistent with the producer's citation (`session-discovery.ts:16-25`) and
its characterization of the mechanism.

### 4. Current-repo absence of both symbols — CONFIRMED, exact match (zero matches)

Scoped search across `*.ts/*.tsx/*.js/*.jsx/*.go/*.py` under
`multica-auth-work/` and `openspec/` for `useSessionMonitor`/`isExpiringSoon`
returned **zero matches** in product/test source (matching both the
producer's claim and both prior GLM traces' independent findings). The
initial attempt at this exact check used an unscoped `grep -r` that hung on
this large monorepo and was correctly interrupted per live steering; the
scoped re-run completed cleanly and is the result cited here — no
duplicated effort, just a corrected tool invocation.

### 5. Production producer/emitter reachability — CONFIRMED, exact match

```
grep -rn 'NewCredentialSessionDiscoveryProducer(' internal/daemon/
→ credential_session_discovery_producer_test.go (5 matches)
→ credential_session_discovery_producer.go (3 matches — definition only)
→ credential_rotation_task53_test.go (2 matches)
```
**Zero non-test call sites.** This independently confirms the producer's
central claim: "every current call to `NewCredentialSessionDiscoveryProducer`
is in `_test.go` files" and "every current implementation of
`EmitCredentialSessionDiscovery` is in `credential_session_discovery_producer_test.go`."
This is the single most consequential claim in the artifact for the D1
question, because it means **no D1 option, by itself, proves task 4.4 is
production-reachable** — a distinction the producer draws explicitly and
correctly, and which neither prior GLM trace stated as sharply (the GLM
traces mention the 4.3 gap but frame it as "orthogonal" to the 4.4
frontend-scope question; the producer instead treats it as a **gating
dependency on whether 4.4 can be closed at all**, regardless of which D1
option is chosen).

### 6. Spot-check of backend allowlist/source citations — CONFIRMED

`credential_session_monitor.go:45-56`'s `credentialSessionDiscoveryOutcome`
struct fields (`Handled`, `Reassigned`, `AgentID`, `PreviousAccountID`,
`NextAccountID`, `Provider`, `TenantID`) match the producer's characterization
of a non-secret metadata allowlist exactly, on direct read.

## Verification limits (disclosed)

I did not re-verify every cited line range in `discovery_reassignment.go`,
`service.go`, `wakeup.go`, or the two cited independent-review artifacts'
SHA-256 values (`credential-isolation-4.4-fresh-review.md`,
`credential-isolation-4.4-record-failure-alert-codex-independent-review.md`)
byte-for-byte in this finalization pass — those were already independently
verified in this session's earlier work on this same task chain (the D1
frontend-scope trace review, and the redact-core ownership trace, both of
which cross-checked overlapping backend files). I focused this pass's
re-verification budget on the claims most specific to **this** artifact and
most consequential to the D1 decision: the historical-hook correction (new,
not in either GLM trace) and the 4.3 reachability framing (sharper than
either GLM trace). This is a scoped, not exhaustive, re-verification,
disclosed rather than silently assumed complete.

## Grading: D1-A/B/C textual fidelity

| Claim | Producer's position | Independent grade |
|---|---|---|
| Symbols absent from current source | Confirmed absent, but with historical correction: they existed pre-cleanup and are real deleted code, not merely "external/stale prose" | **PASS** — stronger and more precise than both prior GLM traces, independently re-verified via `git show` on the actual historical blob |
| Historical hook's actual behavior | Warned only about *impending* expiry via `console.warn`, not completed reassignment; exposed `account_email` | **PASS** — independently confirmed by reading the full historical file; this is a genuinely new, checkable finding not present in either GLM trace |
| Spec scenario is mechanism-agnostic | Confirmed, "o sistema" is broader than "the frontend," no UI/event named | **PASS** — matches independent primary-text reading in the preparatory pass |
| Task 4.4's textual scope | "a troca" (the reassignment) is the object, not only early-expiry detection; slash should not be read as record-OR-alert (weakened) but record-AND-alert | **PASS** — a careful, textually grounded reading; does not overclaim a UI mandate while also not underclaiming the "alert" half as optional |
| Desktop already offers passive visibility | `daemon-panel.tsx`/`parse-daemon-log.ts` already surface WARN/ERROR from the daemon log, which the GLM traces also found but the producer correctly narrows to "passive/filterable observability — not an attention-grabbing toast" | **PASS** — accurate, appropriately hedged (does not claim this satisfies a stronger reading it does not) |
| 4.3 producer/emitter gap | Frames this as a **gate on 4.4 closure regardless of D1 choice**, not orthogonal | **PASS, and an improvement over both GLM traces** — independently re-verified (zero non-test call sites) and is the correct framing: D1 answers a scope question, not a completion question |
| Assign→RecordRotation non-atomicity residual | Explicitly notes this exists, is separate from the D1 question, and "D1 cannot cure or waive it" | **PASS** — correctly scoped; does not let D1 be used to paper over an unrelated open risk |
| No false premise asserted | Does not claim the hooks currently exist; does not claim the spec mandates a UI; does not claim the backend path is production-live | **PASS** |

**No textual-fidelity defect found.** The producer's artifact is, if
anything, more rigorous than either prior GLM trace on the two dimensions
that matter most: it corrects the "these are just stale references" framing
with actual verified git history, and it correctly refuses to let any D1
answer stand in for proof of production reachability.

## Grading: is D1-C advisory only?

**Yes — confirmed advisory, not a self-acceptance or scope decision.**

- The artifact's own check-in/out explicitly states: "no shared ledger
  entry was permitted or made," "advisory interpretation only," "does not
  decide D1, accept task 4.4, or change its open checkbox."
- Its final "Recommendation without owner decision" section explicitly
  frames its 4-point recommendation as something "Kiro TL may instead
  select D1-B" over, i.e. it does not treat its own recommendation as
  binding.
- It does not touch `tasks.md`, does not set or unset the 4.4 checkbox, and
  (per its own non-claims section) did not modify any OpenSpec, shared
  ledger/state/index, product, test, or git state.
- The producer's D1-C variant carries an explicit **sequencing
  qualification** absent from both prior GLM traces' D1-C framing: "Do not
  close 4.4 merely from the bounded backend tests" until 4.3's production
  path is proven. This makes the producer's D1-C **more conservative and
  more reversible** than the GLM traces' D1-C (which recommended accepting
  4.4 now under D1-A without this explicit gate) — a genuine refinement, not
  a mere restatement.

## Divergence from the two prior GLM traces — explicitly assessed

The producer's artifact does not merely repeat the GLM traces; it refines
them in three independently-verified ways:
1. **Historical correction** (§1-3 above): the symbols were real, not just
   stale prose — verified by this reviewer via direct git archaeology, not
   accepted on the producer's word.
2. **Reachability framing** (§5 above): 4.3's producer/emitter gap is a
   closure gate for 4.4 under *any* D1 choice, not an orthogonal concern —
   verified by this reviewer via an independent grep showing zero
   non-test call sites.
3. **More conservative D1-C**: explicit sequencing qualification against
   premature closure, which the GLM traces' D1-C did not carry.

I independently agree with all three refinements, based on my own
re-verification, not merely because the producer asserted them.

## Owner decision required (not decided here)

Per the producer's own framing, and independently confirmed by this review,
**Kiro TL / the owner must still decide:**
1. Whether task 4.4's frontend criterion is **not mandatory** (D1-A/D1-C
   baseline) or whether proactive in-app alerting is a **deliberate product
   requirement** (D1-B) — this is a scope-strengthening choice, not a
   discovery of an existing mandate; both this review and the producer agree
   the current text does not compel D1-B.
2. Whether to gate any 4.4 closure on proof of 4.3's production
   producer/emitter reachability (both the producer and this review say
   yes, but this is still an owner/TL ruling, not self-executing from either
   artifact).
3. Whether the Assign→RecordRotation non-atomicity residual (noted by the
   producer, out of scope for D1 itself) needs its own separate remediation
   task before or independent of any 4.4 closure.

## Final verdict

**PASS.**

The producer's artifact is textually faithful, appropriately hedged,
correctly scoped as advisory, does not self-accept, does not touch any
checkbox or shared state, and — on independent re-verification of its most
load-bearing and most checkable claims (deletion commit, historical hook
content, current absence, production-reachability gap) — every claim I
checked held up exactly as stated. It represents a genuine improvement over
the two prior GLM traces on textual/historical rigor and on correctly
gating D1 against the separate 4.3 production-reachability question, rather
than merely restating their conclusions. I found no false premise, no
overclaim, and no irreversible action anywhere in the artifact.

**This PASS grades the producer's artifact only. It does not select
D1-A/B/C, does not accept task 4.4, and does not adjudicate the owner
decision items listed above** — those remain with Kiro TL / the owner.

## Non-claims (finalization pass)

- Does not accept, reject, or select D1-A/B/C.
- Does not re-verify every line-range citation in the producer's artifact
  byte-for-byte (disclosed scope limit above).
- No product/test/shared/spec/task/git/index file was edited. No
  credential/env value read. No DB/network/live-provider/service accessed
  in either the preparatory or finalization pass.
- Does not assert task 4.4 is production-ready, safe to close, or that any
  residual risk (Assign→RecordRotation atomicity, 4.3 reachability) is
  resolved.
