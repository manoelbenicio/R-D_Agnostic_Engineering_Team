# Persist-Prodex runtime 1.3 evidence-integrity critique

Auditor: Codex-root

Role: independent contract-integrity auditor; not producer, reviewer, or adjudicator

Audit window: 2026-07-18T20:52:13Z–2026-07-18T20:56:08Z

Host: `manoelneto-laptop`, Linux WSL2 x86_64

Repository HEAD observed: `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f`

Adjudication authority: `Kiro#Opus48-TL` remains the adjudicator

## Scope and disposition

This audit covers only:

- `.deploy-control/evidence/persist-prodex-runtime-1.3-review.md`;
- `.deploy-control/Kiro__REVIEW-PP-1.3__20260718T174500Z.md`;
- current source/test files cited by that package;
- durable planning records needed to verify identity and AB-REQ/EV provenance;
- the exact current OpenSpec task 1.3 text.

No Go command, test, build, vet, database, network, live service, credential,
authentication home, or environment value was executed or inspected. No product,
test, spec, task, checkbox, evidence index, ledger, STATE, or git state was edited.
The only writes were this critique and its Golden Rule check-in/out.

Overall evidence-contract grade: **REJECT**. The three cited current hashes match,
the narrow loader behavior is visible in source, and producer/reviewer separation
is supported. The package is not contract-complete because it lacks exact command
provenance and raw non-zero transcripts, uses an unregistered AB-REQ/EV pair,
contains incoherent UTC filename/check-out chronology, and cannot establish that
reviewer `Kiro w7:p1` is identity-distinct from adjudicator `Kiro#Opus48-TL`.
The task remains unchecked; this critique performs no acceptance action.

Grades mean:

- **PASS**: directly established by durable records inspected.
- **PENDING**: plausible but not independently establishable from the durable record.
- **REJECT**: a binding field is absent, inconsistent, unregistered, or broader
  than the durable proof.

## Exact OpenSpec boundary

Current `openspec/changes/persist-prodex-runtime-integration/tasks.md:5` is:

> `- [ ] 1.3 Add MULTICA_PRODEX_REQUIRED startup enforcement so a required Prodex/L2 configuration cannot silently downgrade`

The review quotes the words accurately at artifact `:16-17`, but incorrectly says
they are at current line 15 (`:14`). The checkbox is still `[ ]`.

The related spec says that when `MULTICA_PRODEX_REQUIRED` is enabled and durable
configuration cannot be loaded or resolves Prodex/L2 disabled, startup fails closed
with a redacted configuration error
(`specs/prodex-runtime-continuity/spec.md:10-12`). Task 2.1 separately owns the
durable launcher/source, so this audit does not require task 1.3 to prove a live
restart.

## Contract-field grades

| Contract field | Grade | Durable finding |
|---|---|---|
| Review artifact identity | PASS | Current SHA-256 is `1783f3a49eecefc7cca049734234dedfc845e64a45b1db25711ac5a93a2b9c9f`, matching `AGENT_LEDGER.md:279`. |
| Check-in file integrity | PASS | Current check-in SHA-256 is `d816df2131cb7fab373055f0059380844f03762c791979a2bc4b53b2de804fd3`; the check-in path is exactly the one named in the assignment and locked at check-in `:10-12`. |
| Cited source/test hashes | PASS | All 3/3 cited hashes at artifact `:35-39` match current disk exactly; see manifest below. |
| Manifest scope for task-level startup claim | REJECT | The manifest omits `config.go`, where loader errors propagate and required+L2-disabled is rejected (`config.go:458-475`), and `daemon.go`, where runtime startup errors are returned (`daemon.go:819-821`). The three-file manifest supports loader-level claims, not the entire startup-enforcement claim. |
| Exact producer/reviewer commands | REJECT | Artifact `:59-71` names an environment and pinned Go path, but its commands use bare `go`, omit the working directory, omit the environment prefixes, and report a time range rather than exact invocation records. The check-in repeats the abbreviated commands at `:15-21`. Rule 0 requires exact commands, not reconstruction. |
| Non-zero named-test transcript | REJECT | There are zero raw `=== RUN` and zero raw `--- PASS` lines. Artifact `:69` is a narrative “60/60” total; `:70-71` contains only `PASS` and an `ok` summary. The check-in and ledger repeat summaries, not a raw durable transcript. |
| Race result | PENDING | Artifact `:66-74`, check-in `:19-21`, and ledger `:279` consistently assert race-clean exit 0, but no durable raw race transcript is referenced or embedded. |
| Build/vet result | PENDING | Artifact `:63-64` and check-in `:17-18` record exit 0 summaries, but no raw output transcript or exact working directory is durable. |
| Host provenance | REJECT | Neither review artifact nor check-in records the execution host. This audit's host observation cannot retroactively supply it. |
| Binary version and commit | REJECT | Go `1.26.4 linux/amd64` and a path are recorded at artifact `:59-60`; no Go binary commit/build identity or reviewed source commit is recorded. Current repository HEAD cannot be retroactively attributed to the run. |
| Execution timestamp | REJECT | Artifact timestamp is a document time (`:5`) and the run is only “2026-07-18T17:5x” (`:31`). The check-in filename says `20260718T174500Z`, while content says start `17:45-03:00` (`:7`), which equals `20:45Z`; the filename is therefore mislabeled by three hours. Both files' current mtimes precede their claimed `17:52-03:00` finish, so filesystem metadata cannot repair the chronology. |
| Pre-execution check-in existence | PENDING | A task-specific check-in exists and claims a 17:45-03 start before the fuzzy 17:5x run. Because its current DONE content/mtime predates its stated 17:52-03 finish and no creation history is durable, Rule-3 ordering cannot be proven. |
| Plan/task traceability | PASS | Check-in `:14` maps directly to current OpenSpec task 1.3; artifact `:6` and `:12-21` preserve the same scope and unchecked state. |
| In-artifact AB-REQ/EV semantics | PASS | Artifact `:23-31` explicitly maps the task wording to `AB-REQ-PP-1.3` and the package to `EV-PP-1.3-KIRO`; ledger `:279` repeats that pair. |
| Formal AB-REQ registration | REJECT | `AB-REQ-PP-1.3` is absent from `REQUIREMENTS.md` and `TRACEABILITY.md`. It appears only in this artifact and the later ledger row, so it is an ad hoc label rather than a registered requirement ID. |
| Formal EV registration | REJECT | `EV-PP-1.3-KIRO` is absent from `EVIDENCE_INDEX.md`; the index retains only the broader contract-incomplete `EV-PP-1.1-1.3-REVIEW` at `EVIDENCE_INDEX.md:138`. |
| Producer identity | PASS | Ledger `:243` identifies producer `Opus48#A`; artifact `:3-4` and `:47-53` preserve that attribution rather than relabeling it. |
| Reviewer identity | PENDING | Artifact/check-in identify only “Kiro.” The later ledger row `:279` supplies `Kiro w7:p1`, but neither original file durably binds the run to pane `w7:p1`. |
| Reviewer distinct from producer | PASS | Durable ledger `:279` records `Kiro w7:p1` reviewer and `Opus48#A` producer; these are distinct identities. |
| Reviewer distinct from adjudicator | REJECT | STATE `:6` and `HERDR_TRANSPORT.md:24` bind adjudicator `Kiro#Opus48-TL` to `w3:p3`; ledger `:279-281` expressly records the collision: reviewer `Kiro w7:p1` versus adjudicator `Kiro w3:p3` is not demonstrably identity-distinct. No owner ruling establishes independence. |
| Scope/non-claims | PASS | Artifact `:105-111` explicitly limits restart evidence to static code plus unit tests and disclaims a live restart. Artifact `:113-124` excludes DB/network/credentials/git/checkbox actions. |
| Task-level technical ACCEPT support | REJECT | Two selected tests prove required-but-Prodex-disabled and required+missing-tenant loader failures (`prodex_runtime_integration_test.go:165-196`). The third is a non-required fallback control (`:198-214`), not a third fail-closed case. No selected test invokes `LoadConfig`, covers required+`MULTICA_L2_ENABLED` disabled at `config.go:473-475`, verifies startup error propagation, or checks redacted error behavior. The narrow loader results may be genuine, but artifact `:128-134` overstates them as complete task-1.3 technical satisfaction. |
| Checkbox/adjudication boundary | PASS | Artifact `:136-143`, check-in `:5` and `:26-29`, and ledger `:279-281` all keep 1.3 OPEN and reserve adjudication. No checkbox was changed by this audit. |

## Current hash manifest

| File | Cited SHA-256 | Current SHA-256 | Grade |
|---|---|---|---|
| `server/internal/daemon/prodex_runtime_integration_test.go` | `312fef692eeca53b84413f429b2cb19136974f7326a133105379d18263bdf78e` | `312fef692eeca53b84413f429b2cb19136974f7326a133105379d18263bdf78e` | PASS |
| `server/internal/daemon/l2_runtime.go` | `a54fb79dcb55e4f5928261a4a7cb200dd6377c57c528168f0983053e561139de` | `a54fb79dcb55e4f5928261a4a7cb200dd6377c57c528168f0983053e561139de` | PASS |
| `server/internal/daemon/prodex.go` | `82035719e5a3b3a1472af973412f431e0c14ad11e04939885141893187ee38f7` | `82035719e5a3b3a1472af973412f431e0c14ad11e04939885141893187ee38f7` | PASS |

Additional current startup-path files omitted from the reviewer manifest were read
only to assess scope: `config.go` SHA-256
`9a8a33f6cc6ad2ff95cb9034d23900a8ca9bdac5b1eb815eb8db979a642189cf`;
`daemon.go` SHA-256
`a1d96a3c8a4edfc9f9c583326c6f74779643473904f13f0a8aab63ea1cd6fe07`.
Their hashes are audit observations, not retroactive reviewer provenance.

## Exact read-only audit commands

No Go command was run. Material verification used:

```text
sha256sum .deploy-control/evidence/persist-prodex-runtime-1.3-review.md .deploy-control/Kiro__REVIEW-PP-1.3__20260718T174500Z.md multica-auth-work/server/internal/daemon/prodex_runtime_integration_test.go multica-auth-work/server/internal/daemon/l2_runtime.go multica-auth-work/server/internal/daemon/prodex.go
sha256sum multica-auth-work/server/internal/daemon/config.go multica-auth-work/server/internal/daemon/daemon.go
rg -n 'EV-PP-1.3-KIRO|AB-REQ-PP-1.3|persist-prodex-runtime-1.3-review|REVIEW-PP-1.3' .planning/agent-brain-v3/EVIDENCE_INDEX.md .planning/agent-brain-v3/REQUIREMENTS.md .planning/agent-brain-v3/TRACEABILITY.md .planning/agent-brain-v3/AGENT_LEDGER.md .deploy-control --glob '*.md'
rg -n -i 'w7:p1|w7p1' .planning .deploy-control openspec --glob '*.md'
rg -n '^=== RUN|^--- PASS|^[[:space:]]+--- PASS|^FAIL|^PASS$|^ok[[:space:]]' .deploy-control/evidence/persist-prodex-runtime-1.3-review.md .deploy-control/Kiro__REVIEW-PP-1.3__20260718T174500Z.md
rg -n 'loadProdexLaunchConfig\(|loadL2RuntimeConfig\(|MULTICA_PRODEX_REQUIRED|startL2Runtime' multica-auth-work/server/internal/daemon --glob '*.go'
git status --short -- .deploy-control/evidence/persist-prodex-runtime-1.3-review.md .deploy-control/Kiro__REVIEW-PP-1.3__20260718T174500Z.md multica-auth-work/server/internal/daemon/prodex_runtime_integration_test.go multica-auth-work/server/internal/daemon/l2_runtime.go multica-auth-work/server/internal/daemon/prodex.go multica-auth-work/server/internal/daemon/config.go multica-auth-work/server/internal/daemon/daemon.go
```

Audit tool provenance: ripgrep 15.1.0 revision `af60c2de9d`, GNU sha256sum
coreutils 9.4, git 2.43.0. The worktree was already dirty; both reviewer-package
files and the cited test were untracked, while the cited product files were modified.
No git history was used to invent source provenance.

## Non-claims and adjudication boundary

This critique does not assert the summarized x20/race/build/vet results are false;
it finds that their durable transcript and Rule-0 provenance are insufficient. It
does not mark the original artifact INVALID, add an index entry, register an
AB-REQ, relabel either Kiro identity, or accept/reject the OpenSpec checkbox.
Kiro remains responsible for adjudication, subject to the reviewer/adjudicator
separation problem already recorded at `AGENT_LEDGER.md:279-281`.
