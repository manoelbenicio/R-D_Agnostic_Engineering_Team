# Credential Isolation Task 4.1 — Provenance Reconciliation

- author: Kiro / Opus-4.8, wave w8:p1 (read-only provenance reconciliation)
- date: 2026-07-18T21:36:00Z
- mode: READ-ONLY. No shared docs/product/test/spec/task/git/index/credential/env/network/service changes. This is the only file created. No substitute/retroactive evidence created.

## Embedded check-in / check-out (recorded here per ownership directive)
- CHECK-IN 2026-07-18T21:32:00Z — Kiro/Opus-4.8 w8:p1 — stream CREDISO-4.1-PROVENANCE-RECONCILIATION — READ-ONLY. Sole writable deliverable is this file.
- CHECK-OUT 2026-07-18T21:36:00Z — DONE. Field verdicts below. Kiro TL adjudicates; root integrates. Not self-accepted; no fabrication.

## Purpose

Determine, for agent-credential-isolation task 4.1 (discovery exhaustion/expiry detector), whether each governance/provenance field can be **truthfully reconstructed without fabrication**, using the clean-room eligibility review (`credential-isolation-4.1-push-eligibility-independent-review.md`) and the ledger/index/task/source of record. Technical completeness is kept strictly separate from governance acceptance.

## Sources inspected (read-only)

- `.planning/agent-brain-v3/evidence/credential-isolation-4.1-push-eligibility-independent-review.md` (clean-room reproduction by Kiro/Opus-4.8; verdicts: technical YES / accepted QUALIFIED-YES / push CONDITIONAL).
- `AGENT_LEDGER.md:196` (the only 4.1 acceptance row), `:198`, `:218` (checkbox adjudication).
- `EVIDENCE_INDEX.md:130` (`EV-CREDISO-4.1`), `:145` (4.2 gap note).
- `openspec/changes/agent-credential-isolation/tasks.md:25` (4.1 `[x]`).
- `evidence/credential-isolation-auto-reassignment.md` (task 4.3; references the detector but names no 4.1 producer).
- Current files: `internal/rotation/detector_discovery.go`, `detector_discovery_test.go` (recomputed hashes; git history: none — untracked).

## Field-by-field reconstructability

| Field | Reconstructable? | Truthful basis | Gap / remediation |
|---|---|---|---|
| **Exact source hash** (`detector_discovery.go`) | ✅ YES | Recomputed `bc61a46c0c700010001a49c9075b87f97e89f802b138e63f90b53193a4b45b55`; matches ledger/matrix/eligibility and the EV-CREDISO-4.2 cross-pin. | none |
| **Exact test hash** (`detector_discovery_test.go`) | ✅ YES | Recomputed `4e8092ff6b0f37a8416b961c89040433682e2d0d654293e192992dd3e1d55a4f`; matches all records. | none |
| **Commands** | ✅ YES | Ledger: `focused ×20 / race / vet / gofmt / diff`. Independently re-run this wave (eligibility review): `go test -run <5 tests> -count=20`, `-race -count=20`, `gofmt -l`, `go vet`, clean-room `go build` + full-package `go test`. | none for commands |
| **Assertion / test count** | ◑ PARTIAL | The **original** EV-4.1 record states only "×20/race/…", **no explicit count**. The **5 named tests / 100 pass (5×20)** count comes from **this wave's reproduction**, not the original acceptance record. | Attribute the count to the w8 reproduction, not retroactively to the original accept. Truthful as reproduction-derived. |
| **Independent reviewer identity** | ◑ PARTIAL | `AGENT_LEDGER:196` records the reviewer only as the generic label **"independent reviewer"** (row id `cred-iso-4.1-accept`), **not a unique agent id**. This wave's corroboration is attributable (Kiro/Opus-4.8). | Smallest remediation: the original reviewer self-identifies in the record, OR cite the named w8 Kiro/Opus-4.8 reproduction as the attributable corroborating reviewer (distinct from the original accept). Do **not** invent a name. |
| **Acceptance authority** | ◑ PARTIAL (ledger-grade) | ACCEPT recorded by the independent reviewer in `AGENT_LEDGER:196` + `EVIDENCE_INDEX:130` (`EV-CREDISO-4.1`, basis = "AGENT_LEDGER row"). This is **ledger-grade, not standalone-artifact-grade**. Final governance authority = **Kiro TL adjudicates + root integrates** (not self-granted). | Elevate via a named `EV-CREDISO-4.1` artifact + Kiro TL adjudication. Also: **checkbox discrepancy** — `tasks.md:25` shows `4.1 [x]` while row 196 says "no checkbox set by TL"; the setter of `[x]` is not attributable to a TL action in the accept record. Owner should reconcile who set the checkbox. |
| **Producer identity** (who authored the detector) | ✗ NO | No record names the producer: ledger 4.1 row names only the *reviewer*; no implementation/START row for the detector; **untracked → no git author**; no dedicated `.deploy-control` producer check-in; task-4.3 evidence references the file but names no 4.1 author. | **Cannot be reconstructed without fabrication.** Smallest legitimate path: the actual producer self-attests in a new named EV-4.1 artifact going forward, **or** an owner waiver explicitly accepts that this pre-existing untracked file has no attributable producer. Do **not** assign a name. |
| **Pre-edit check-in** (Golden Rule 1) | ✗ NO | No pre-edit/START check-in for the detector exists in `.deploy-control` (only an unrelated 2026-07-05 entry matched loosely); no producer lock row in the ledger. | **Apparently absent; cannot be recreated retroactively.** Smallest legitimate path: an **owner waiver** acknowledging the check-in was not recorded for this file. **Do not backdate/fabricate** a check-in. |

## Technical completeness vs governance acceptance (kept separate)

- **Technical completeness: COMPLETE (independently verified).** Both hashes recomputed and matched; the eligibility review reproduced 5 named detector tests ×20 (100 pass), race-clean, gofmt-clean, vet-0, and a **clean-room HEAD+2-file build/test PASS** proving dependency-completeness. Nothing here depends on the missing governance fields.
- **Governance acceptance: THIN / QUALIFIED.** Acceptance exists only as a ledger/index row by an **unnamed** "independent reviewer"; there is **no standalone EV-4.1 artifact**, **no named producer**, and **no pre-edit check-in**. A sibling manifest (`active-accepted-push-candidate-matrix.md`) independently treats this as **HOLD** for exactly these reasons, and my prior integration-manifest review endorsed that HOLD. Ledger-grade acceptance does not meet artifact-grade admission (source manifest + named producer + reviewer separation).

## Net determination

- The **substance** of 4.1 (hashes, commands, technical pass, dependency-completeness) **is truthfully reconstructable and independently verified.**
- The **governance provenance** is **only partially reconstructable**: reviewer identity is generic, acceptance is ledger-grade, and **producer identity + pre-edit check-in are not reconstructable at all** and must not be fabricated.
- Therefore 4.1 is **technically complete but not artifact-grade governance-accepted.** Push-eligibility remains **CONDITIONAL** on Kiro TL authorization + root integration, plus the remediations above — none of which this artifact performs.

## Smallest legitimate remediations (owner/TL choice — not executed here)

1. Producer records a named `EV-CREDISO-4.1` artifact with the two source/test hashes, commands, and the (reproduction-derived) test count — authored prospectively, not backdated.
2. Independent reviewer self-identifies in the record, or the w8 Kiro/Opus-4.8 reproduction is cited as the named corroborating reviewer (distinct from the original accept).
3. Owner waiver for the two irrecoverable process fields (missing producer attribution, missing pre-edit check-in), explicitly acknowledging the gap rather than inventing records.
4. Reconcile the `4.1 [x]` checkbox vs the "no checkbox set by TL" ledger note (who set it, under what authority).

## Explicit non-claims

- Created only this file. No edits to shared docs/STATE/AGENT_LEDGER/EVIDENCE_INDEX/OpenSpec/tasks/product/tests/git index/refs. No `add/restore/commit/push`. No checkbox change.
- Read no credential/env/auth/home values; no DB/network/provider/service calls; the detector `_test` was hashed, not executed here (execution is in the cited eligibility review).
- **No producer name, reviewer identity, check-in, count, or acceptance was fabricated or backdated.** Missing fields are reported as missing with legitimate remediation only.
- This is decision support: technical completeness ≠ governance acceptance. Kiro TL adjudicates; root integrates.
