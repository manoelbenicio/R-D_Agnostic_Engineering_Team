# EV-CREDISO-4.4-RECORDFAIL — INDEPENDENT REVIEW (reviewer ≠ producer, reviewer ≠ adjudicator)

Independent review of the credential-isolation task 4.4 RecordRotation failure-alert test.
Reviewer: **Kiro/Opus-4.8 — reviewer session `w8:p2`**, distinct from the **producer**
(Kiro/Opus-4.8, session `w7:p2`) and from the **adjudicator** (Kiro TL, session `w3:p3`).
**This is a reviewer report, not an acceptance.** Kiro TL adjudicates task 4.4.

> **Provenance correction (2026-07-18, rehash below).** A prior version of this artifact
> **incorrectly identified the reviewer as "Codex Agent-6."** That label was wrong: this review was
> produced by Kiro/Opus-4.8 reviewer session `w8:p2`. Only these provenance/identity lines and this
> disclosure were changed; **no technical claim, result, count, hash-of-reviewed-artifact, or verdict
> was altered.**
>
> **Independence vs. model-family equality.** The reviewer (`w8:p2`) and the producer (`w7:p2`) are
> the **same model family (Kiro/Opus-4.8)** running in **separate panes/sessions with independent
> context** — the reviewer did not share the producer's working state and re-derived every finding
> from the source and re-ran the pinned offline reproductions itself. Process/context independence
> therefore holds. The honest caveat: shared model family means **shared inductive biases**, so this
> is *not* equivalent to an independent second implementation or a different-model/human reviewer;
> a blind spot in the model family could be common-mode. Kiro TL (`w3:p3`) should weigh this when
> adjudicating and may require a different-family or human cross-check if stronger independence is
> needed.

## Golden Rule check-IN — 2026-07-18T21:18:00Z
- Mode: READ-ONLY REVIEW. Owned/created file = **only this artifact**:
  `.planning/agent-brain-v3/evidence/credential-isolation-4.4-record-failure-alert-independent-review.md`.
- Will NOT edit producer files, shared ledger/state, product/spec/tasks, git index; no stage/commit/push;
  no credentials/env values/network/live services. Go runs are offline, pinned, cache-only.
- Inputs reviewed (read-only):
  - `multica-auth-work/server/internal/daemon/credential_session_record_failure_alert_test.go`
  - `.planning/agent-brain-v3/evidence/credential-isolation-4.4-record-failure-alert.md`
  - Corroborating product/fixtures (read-only): `internal/daemon/wakeup.go`,
    `internal/daemon/credential_session_monitor.go`, `internal/rotation/service.go`,
    `internal/rotation/discovery_reassignment.go`, `internal/daemon/credential_session_alert_test.go`,
    `internal/daemon/credential_session_discovery_producer_test.go`.

## Hashes / provenance (current, verified)
- git HEAD: `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f` (working tree dirty; multi-agent WIP).
- Test file SHA-256: `5b4d82caba027d4dd6b2650d9cd5a2ad78ebdd07504060fd84a415d40be739f5`
  — **matches** the producer evidence's claimed hash (artifact integrity confirmed).
- Producer evidence md SHA-256: `fba59bc2936bafeb0b7d47c07d6c3417ed9b8b6a3588366b45efd294ee31651f`.
- Both producer files are **untracked** and were **not modified** by this review (read-only).
- Toolchain (matches producer): `/home/dataops-lab/go-sdk/bin/go` = `go1.26.4 linux/amd64`,
  `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`; go.mod `go 1.26.1`. Offline.

## Independent findings — does the test truly exercise real code?

**YES — genuine integration test, not tautological / not over-mocked.**

1. **Real `rotation.Service`, real dispatch.** The test constructs
   `rotation.NewService(store, producerNoopDetector{}, producerSyntheticAuthenticator{})` and drives it
   through the real daemon bridge `dispatchAndReportCredentialSessionDiscoveryEvent`
   (wakeup.go) → `dispatchCredentialSessionDiscoveryEventWithOutcome` (credential_session_monitor.go)
   → the **concrete** `(*rotation.Service).ReassignDiscoverySession` (discovery_reassignment.go), which
   calls `onExhaustionLocked` (service.go). The narrower `syntheticDiscoveryReassigner` stub that exists
   in `credential_session_monitor_test.go` is **NOT** used here. Confirms producer claim "real service,
   not a stub reassigner."
2. **Fault injected only at the durable-record step.** `recordRotationFailingStore` embeds the real
   `producerSyntheticStore` and overrides **only** `RecordRotation` to return a sentinel error; every
   other Store op (List/Get/UpdateStatus/CurrentAssignment/Assign) runs the real fixture. So the real
   service executes CurrentAssignment → UpdateAccountStatus(exhausted) → selectNext → Logout(current)
   → Login(next) → WaitAuthenticated → Assign(next) → **RecordRotation (FAILS)**.
3. **`login:account-next` proof is real.** `producerSyntheticAuthenticator.Login` appends
   `"login:"+AccountID`; `WaitAuthenticated` returns `(true,nil)` so control genuinely reaches `Assign`
   and then the failing `RecordRotation`. The assertion therefore proves real execution to the record step.
4. **Alert text/fields come from product code, not the test.** The ERROR line and all attrs are emitted by
   `dispatchAndReportCredentialSessionDiscoveryEvent`; the test only supplies a buffer logger.

### Claim-by-claim verification
- **ERROR alert present.** For a generic (non-`ErrNoAccountAvailable`, non-deadline/canceled/unavailable)
  error, `credentialSessionReassignmentErrorClass` returns `"reassignment_failed"` (verified default arm),
  and the bridge calls `logger.Error("rotation: automatic credential account reassignment failed", …)`.
  Test asserts `level=ERROR`, `alert=reassignment_failed`, the message, and `agent_id/provider/tenant_id/
  previous_account_id`. ✔ Backed by real code.
- **No success / no `next_account_id`.** `next_account_id` and `reason` are appended **only** on the
  success arm, which also logs `logger.Warn("… completed")`. On the failure arm neither is added and the
  level is ERROR. Test forbids `"… completed"`, `"next_account_id="`, and `"level=WARN"`. ✔ Correct for
  this failure class. (Scoping note below.)
- **No sentinel / raw-error leak.** The test logger is a real `slog.NewTextHandler` with the **production**
  `redact.SanitizeSlogAttr` ReplaceAttr — so the no-leak assertions exercise the real redaction pipeline.
  Test asserts absence of the credential-home sentinel, the record-rotation sentinel, and the raw
  `"persist rotation failed"` text. ✔ Meaningful (not a stub redactor).
- **Honest non-atomic observation.** service.go `onExhaustionLocked` persists `store.Assign(next)` **before**
  `store.RecordRotation(...)`, and on RecordRotation failure it `return Account{}, err` with **no rollback**
  of the assignment (rollback exists only for the Assign-failure→Logout branch, not the record branch).
  The test asserts `assignment("agent-1") == "account-next"` and explicitly claims no rollback. ✔ Accurate
  description of current behavior; no false rollback claim.

## Reproduction — pinned offline Go (this reviewer, independent run)
- `gofmt -l internal/daemon/credential_session_record_failure_alert_test.go` → empty (formatted), exit 0.
- `go vet ./internal/daemon/` → exit 0.
- `go test … -run '^TestDispatchCredentialSessionRecordRotationFailureAlertsWithoutSuccessOrLeak$'
  -count=20 -v` → **RUN=20, PASS=20, FAIL=0**, `ok … 0.029s`.
- Same `-race -count=20 -v` → **RUN=20, PASS=20, FAIL=0, DATA RACE=0**, `ok … 1.151s`.
- Assertion surface in the test: **5 `t.Fatalf` guard sites** covering **15 conditions**
  (1 login-call + 7 required substrings + 3 forbidden success markers + 3 secret/raw-error checks +
  1 non-atomic assignment). One test function; 20 independent executions (×2 with race).

## Reviewer notes / limitations for the adjudicator (not test defects)
- The `level=WARN` prohibition is correct **for this failure class** but is scenario-specific: WARN is the
  legitimate level for the `ErrNoAccountAvailable` ("reassignment unavailable") and success ("completed")
  arms. This test does not (and need not) cover those arms.
- The **non-atomic assignment-before-record** behavior it documents is a genuine latent correctness concern
  (a failed durable record leaves a written assignment not rolled back). For task 4.4 (alerting) coverage
  this is acceptable and honestly disclosed; the adjudicator may wish to track the non-atomicity as a
  separate rotation-service item.
- Coverage is unit/integration with synthetic in-memory fixtures (no Postgres); end-to-end durability with
  a real store is out of scope for this test.

## Disposition (reviewer)
- **Technical verdict: PASS** — the test genuinely exercises the real `rotation.Service` + real daemon
  dispatch + real slog/redaction; all four required properties (ERROR alert; no success/`next_account_id`;
  no sentinel/raw-error leak; honest non-atomic observation) are verified against product code and
  reproduced offline (gofmt clean, vet 0, named 20/20, race 20/20, 0 data races).
- **This PASS is NOT whole-task acceptance.** Task 4.4 acceptance (and the unresolved frontend-owner scope
  question) is **out of this reviewer's authority** and is left to **Kiro TL** to adjudicate. Reviewer ≠
  adjudicator.

## Golden Rule check-OUT — 2026-07-18T21:22:28Z
- Files created: this artifact only. Producer files/ledger/state/spec/tasks unchanged (verified
  `git status`: both producer files remain untracked and unmodified). No git stage/commit/push. No
  network/live services/credentials/env-value access. Go runs were offline (`GOPROXY=off`), pinned
  (`go1.26.4`, `GOTOOLCHAIN=local`), cache-only.
- Status: DONE (reviewer report delivered). Adjudication pending Kiro TL.
