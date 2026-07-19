# Credential-isolation 4.3 and 5.3 evidence-integrity critique

Reviewer: Codex-root

Role: independent contract-integrity auditor; not producer or adjudicator

Audit window: 2026-07-18T20:41:31Z–2026-07-18T20:45:10Z

Host: `manoelneto-laptop`, Linux WSL2 x86_64

Repository HEAD observed: `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f`

Disposition authority: Kiro remains the sole adjudicator

## Scope and conclusion

This was a read-only integrity audit of the completed review sections in:

- `evidence/credential-isolation-auto-reassignment.md` (task 4.3);
- `evidence/credential-isolation-task-5.3-automatic-rotation.md` (task 5.3).

No test, provider, credential, authentication home, environment value, database,
network, or live service was executed or inspected. No product, test, spec, task,
OpenSpec checkbox, STATE, ledger, evidence index, identity label, git stage, commit,
or push was changed. The only writes were this critique and its Golden Rule
check-in/out.

Contract-integrity result: **REJECT for 4.3 review; REJECT for 5.3 review**, without
overriding the recorded technical reproductions and without adjudicating either
OpenSpec task. The decisive contract failures are missing Rule-0 provenance,
missing reviewer execution check-ins, absent AB-REQ mappings, and review EV IDs
recorded in the ledger but absent from the evidence index. Task 4.3 also has a
currently stale 11/13 source manifest. Task 5.3 retains a genuine current 14/14
manifest and a genuine embedded non-zero single-run transcript, but those positives
do not cure the other mandatory contract failures or its disclosed producer process
exception.

Grades mean:

- **PASS**: directly supported by the durable files inspected.
- **PENDING**: plausible, but the durable record is insufficient to verify it.
- **REJECT**: a binding field is absent, mismatched, or contradicted by the durable
  record. This is an audit grade, not Kiro's task adjudication.

## Binding criteria

`EVIDENCE_CONTRACT.md:11-14` requires every evidence file to state exact command,
host, binary version plus commit, UTC timestamp, and runner. Lines 29-31 require a
`.deploy-control` check-in before any command. Lines 33-35 require a plan task ID.
`EVIDENCE_INDEX.md:3-6` defines the EV-to-AB-REQ/acceptance registry. The OpenSpec
tasks remain unchecked at `openspec/changes/agent-credential-isolation/tasks.md:27`
and `:33`.

## Field grades — task 4.3 review

| Evidence field | Grade | Durable finding |
|---|---|---|
| Review artifact identity | PASS | Current SHA-256 is `6184aa3703b390fdba16c1ac1c4cfbabfcbd3b7ca18bb30e0ed6b3ca436c4848`, exactly matching `AGENT_LEDGER.md:291`. |
| Current source/test manifest | REJECT | 11/13 cited hashes match current files. `credential_session_monitor.go` is now `936b3e40f19c699078e994740c9ae63fdce5c172516b678d289f05c3cc38d1e2`, not the cited `a77d2f70…0478e` at artifact `:151`; `wakeup.go` is now `71b89980c9b8cb7c03a81a502da5b9f4bdab4c0c522c117d2147b670e050f730`, not `d11d1ac4…3d3b` at `:153`. The later 5.3 review transparently attributes both changes to 4.4 at `:218-222`, so this is staleness, not evidence of tampering. |
| Producer command text | PASS | Focused x20, race, and vet command lines are present at artifact `:111`, `:124`, and `:137`, with package/exit summaries at `:114-140`. |
| Reviewer exact commands | REJECT | The reviewer states environment/toolchain at `:205-209`, but does not record the exact verbose command that allegedly produced 480 PASS lines, and does not restate exact reviewer invocations with the claimed `GOSUMDB=off`. Rule 0 requires exact commands, not reconstruction. |
| Durable non-zero test transcript | REJECT | The artifact contains zero raw `=== RUN` lines and zero raw `--- PASS` lines. The `480 PASS / 0 FAIL` statement at `:213-219` is a narrative count with no durable verbose transcript. Package `ok` summaries do not prove a non-zero selected-test count. |
| Race/vet result durability | PENDING | Exit/package summaries exist at `:220-222`, but no raw reviewer transcript or external transcript file was located. The results may be genuine; the durable record cannot independently establish their execution. |
| Plan/task mapping | PASS | The artifact explicitly scopes task 4.3 at `:7-10`, and the ledger maps it at `AGENT_LEDGER.md:291`. |
| Contract citation integrity | REJECT | Artifact `:199-201` attributes TL countersign to “EVIDENCE_CONTRACT rule 6/7.” The binding file ends at Rule 6; Rule 6 is log scrubbing, and Rule 7 does not exist. The task-level Kiro boundary is supported elsewhere in the ledger, but this specific contract citation is false. |
| AB-REQ mapping | REJECT | No AB-REQ appears in the artifact, `REQUIREMENTS.md`, or `TRACEABILITY.md` for this change/task. No semantic AB-REQ was inferred or invented. |
| EV registration | REJECT | `AGENT_LEDGER.md:291` records `EV-CREDISO-4.3-REVIEW`, but `EVIDENCE_INDEX.md:135` still records only `EV-CREDISO-4.3-INTERIM` with the old pre-review artifact hash; the review EV ID is absent from the index. |
| Producer provenance | REJECT | The artifact supplies only a date at `:3`; it does not identify producer, host, source commit, exact UTC execution timestamp, or binary commit. The ledger identifies `Codex#56#A worker (w4)` retrospectively at `AGENT_LEDGER.md:222`, which does not complete the artifact's Rule-0 fields. |
| Reviewer provenance | REJECT | Reviewer identity and Go version are present at `:194-209`, but host, source/binary commit, and exact UTC execution timestamp are absent. The planning artifacts are currently untracked, so repository history cannot supply those missing fields. |
| Pre-execution check-in | REJECT | No 4.3 producer/reviewer or `GLM52-auth-QA` `.deploy-control` check-in was located. The artifact does not disclose this missing reviewer check-in. A later ledger row is not the Rule-3 pre-command check-in. |
| Producer/reviewer/adjudicator separation | PASS | Durable ledger chain identifies producer `Codex#56#A worker (w4)` (`AGENT_LEDGER.md:222`), reviewer `GLM52-auth-QA` (`:291`), and Kiro as adjudicator (`:265-269`). No identity was relabeled. |
| Technical non-claims/blockers | PASS | The artifact expressly limits itself to a bounded synthetic slice (`:164-192`) and records five blockers. The review preserves task 4.3 as unchecked at `:290-298`; Kiro's later cross-check confirms the bounded scope at `AGENT_LEDGER.md:267`. |
| Process-exception disclosure | REJECT | Technical blockers are disclosed, but the absent Rule-3 reviewer check-in and incomplete Rule-0 provenance are not disclosed as process exceptions. |

## Field grades — task 5.3 review

| Evidence field | Grade | Durable finding |
|---|---|---|
| Review artifact identity | PASS | Current SHA-256 is `aece8372e620e6dbf572b9dce70e4abedc675f2bd614b84c10abccfae20367b7`, exactly matching `AGENT_LEDGER.md:292`. |
| Current source/test manifest | PASS | All 14 cited hashes at artifact `:131-145` match current files, including task test SHA `9a849e508c54353110d011737ff7a659909af604c9adc60e82384f331bf724b1`. |
| Producer command text | PASS | Exact verbose, x20, race, and vet commands are present at artifact `:66`, `:90`, `:105`, and `:120`. |
| Reviewer exact commands | REJECT | Reviewer environment and abbreviated command descriptions appear at `:194-211`, but the exact reviewer invocations are not recorded as executed commands. Rule 0 does not permit reconstructing them from the producer section. |
| Durable named-test execution | PASS | Embedded output at `:71-80` contains three `=== RUN` and three `--- PASS` lines: one top-level test and two named subtests. This is genuine non-zero proof for the single run. |
| Durable x20 test count | PENDING | The x20 section has package `ok` summaries at `:93-98`, and the review repeats timings at `:207-208`, but neither contains verbose RUN/PASS lines or a durable count. It does not independently prove that all five selected top-level tests executed 20 times. |
| Race/vet result durability | PENDING | Package/exit summaries exist at `:108-123` and `:209-211`; no raw reviewer transcript or external transcript file was located. |
| Plan/task mapping | PASS | The artifact explicitly scopes task 5.3 at `:7-11`, and the ledger maps it at `AGENT_LEDGER.md:292`. |
| AB-REQ mapping | REJECT | No AB-REQ appears in the artifact, `REQUIREMENTS.md`, or `TRACEABILITY.md` for this change/task. No mapping was inferred. |
| EV registration | REJECT | `AGENT_LEDGER.md:292` records `EV-CREDISO-5.3-REVIEW`; that ID has no entry in `EVIDENCE_INDEX.md`. |
| Producer provenance | REJECT | The artifact provides a date but no producer identity, host, source/binary commit, or exact UTC execution timestamp. `AGENT_LEDGER.md:226` identifies producer `Codex56#D` retrospectively and explicitly records the missing pre-edit check-in. |
| Reviewer provenance | REJECT | Reviewer identity and Go version are present at `:183-198`, but host, source/binary commit, exact UTC execution timestamp, and exact reviewer commands are absent. |
| Producer pre-execution check-in | REJECT | The missing pre-edit check-in is expressly admitted at artifact `:13-20` and ledger `:226-228`; later reproduction cannot retroactively cure Rule 3. |
| Reviewer pre-execution check-in | REJECT | No `GLM52-auth-QA` `.deploy-control` check-in was located for the reviewer commands, and this separate reviewer exception is not disclosed in the review section. |
| Producer/reviewer/adjudicator separation | PASS | Durable ledger chain identifies producer `Codex56#D` (`AGENT_LEDGER.md:226`), reviewer `GLM52-auth-QA` (`:292`), and Kiro as adjudicator (`:263`, `:268`). No identity was relabeled. |
| Producer process-exception disclosure | PASS | Artifact `:13-20` clearly says the pre-edit check-in is missing and retrospective evidence does not cure it; reviewer reiterates this at `:252-262`. Kiro's ruling at `AGENT_LEDGER.md:268` preserves the exception and keeps 5.3 open. |
| Technical scope/non-claims | PASS | Synthetic-only scope and exclusions are explicit at artifact `:148-181`; the review keeps task 5.3 unchecked at `:264-270`. |

## Cross-artifact provenance and transcript findings

### Current scoped source/test manifest comparison

| Current file under `multica-auth-work/server` | Current SHA-256 | 4.3 citation | 5.3 citation |
|---|---|---|---|
| `internal/daemon/credential_rotation_task53_test.go` | `9a849e508c54353110d011737ff7a659909af604c9adc60e82384f331bf724b1` | N/A | PASS |
| `internal/daemon/credential_session_discovery_producer.go` | `4e240c09af1653fd2ecdbcd0763f8c05d3eb3f57ee8b9c339045edfa32a9ce6c` | PASS | PASS |
| `internal/daemon/credential_session_discovery_producer_test.go` | `818e69d4fe04057503b776716ae0704a6b27b7fc7b2b875e8f507b9a7b99491a` | PASS | PASS |
| `internal/daemon/credential_session_monitor.go` | `936b3e40f19c699078e994740c9ae63fdce5c172516b678d289f05c3cc38d1e2` | REJECT (`a77d2f70…0478e` cited) | PASS |
| `internal/daemon/credential_session_monitor_test.go` | `5fd7005b4209b0c8d5c42ad9b240ad039d7568ad1f895bd6b9463b8e237613e2` | PASS | PASS |
| `internal/daemon/wakeup.go` | `71b89980c9b8cb7c03a81a502da5b9f4bdab4c0c522c117d2147b670e050f730` | REJECT (`d11d1ac4…3d3b` cited) | PASS |
| `internal/rotation/detector_discovery.go` | `bc61a46c0c700010001a49c9075b87f97e89f802b138e63f90b53193a4b45b55` | PASS | PASS |
| `internal/rotation/detector_discovery_test.go` | `4e8092ff6b0f37a8416b961c89040433682e2d0d654293e192992dd3e1d55a4f` | PASS | PASS |
| `internal/rotation/discovery_reassignment.go` | `c655c9c4f94716d2c93109b1cc4d3c14e62d6fe8dea615dddec3ec93769b2832` | PASS | PASS |
| `internal/rotation/discovery_reassignment_test.go` | `d87aa80717e33a3f3768ebb7d698dcd9ace3bd934550317ec64c57f56a6566b2` | PASS | PASS |
| `internal/rotation/pool.go` | `0c4c453c730f5b9d49c605e2206e1501b778f32f40b20b142d86e70cdb8fb3fc` | PASS | PASS |
| `internal/rotation/pool_test.go` | `da401ef882af6fe06bb923494f4393b685c45dca01e9b0707127bda16a87f005` | PASS | PASS |
| `internal/rotation/service.go` | `f20951a36765d2b6b8576232871f1991697de56d8577377b0226ed5e30130cf0` | PASS | PASS |
| `internal/rotation/service_test.go` | `989ae633e1865bc0f1f5a9f7eed5d2541100dbee94960c9e2c15f84346aa8ba1` | PASS | PASS |

The review artifact hashes are durably corroborated by the ledger. However, all
four inspected planning records—the two artifacts, ledger, and evidence index—are
currently untracked according to `git status --short`. Therefore no git commit can
be used as provenance for their content or timestamps. Filesystem mtimes are not a
substitute for Rule-0 UTC execution timestamps.

No external durable transcript or relevant GLM reviewer check-in was found by:

```text
rg --files .deploy-control | rg -i '(crediso|cred.iso|glm52-auth|automatic.rotation|reassignment)'
rg -n '480|GLM52-auth-QA|cred-iso-4.3-review|cred-iso-5.3-review|0.042s|0.028s|1.041s|1.039s' .deploy-control .planning --glob '*.md'
```

The only durable raw named-test transcript found is the embedded 5.3 single-run
output. Ledger summaries such as `AGENT_LEDGER.md:258`, `:291`, and `:292` are
corroborating assertions, not raw command transcripts.

## Exact read-only audit commands

The audit used only these command classes; no Go test, vet, build, provider, or
service command was run:

```text
sha256sum .planning/agent-brain-v3/evidence/credential-isolation-auto-reassignment.md .planning/agent-brain-v3/evidence/credential-isolation-task-5.3-automatic-rotation.md
sha256sum multica-auth-work/server/internal/daemon/credential_rotation_task53_test.go multica-auth-work/server/internal/daemon/credential_session_discovery_producer.go multica-auth-work/server/internal/daemon/credential_session_discovery_producer_test.go multica-auth-work/server/internal/daemon/credential_session_monitor.go multica-auth-work/server/internal/daemon/credential_session_monitor_test.go multica-auth-work/server/internal/daemon/wakeup.go multica-auth-work/server/internal/rotation/detector_discovery.go multica-auth-work/server/internal/rotation/detector_discovery_test.go multica-auth-work/server/internal/rotation/discovery_reassignment.go multica-auth-work/server/internal/rotation/discovery_reassignment_test.go multica-auth-work/server/internal/rotation/pool.go multica-auth-work/server/internal/rotation/pool_test.go multica-auth-work/server/internal/rotation/service.go multica-auth-work/server/internal/rotation/service_test.go
rg -n 'EV-CREDISO-4.3-REVIEW|EV-CREDISO-5.3-REVIEW' .planning/agent-brain-v3/EVIDENCE_INDEX.md .planning/agent-brain-v3/REQUIREMENTS.md .planning/agent-brain-v3/TRACEABILITY.md
rg -n 'agent-credential-isolation|credential isolation|reassign|automatic rotation|rotação automática' .planning/agent-brain-v3/REQUIREMENTS.md .planning/agent-brain-v3/TRACEABILITY.md
rg -c '^=== RUN|^--- PASS|^[[:space:]]+--- PASS' .planning/agent-brain-v3/evidence/credential-isolation-auto-reassignment.md .planning/agent-brain-v3/evidence/credential-isolation-task-5.3-automatic-rotation.md
git status --short -- .planning/agent-brain-v3/evidence/credential-isolation-auto-reassignment.md .planning/agent-brain-v3/evidence/credential-isolation-task-5.3-automatic-rotation.md .planning/agent-brain-v3/EVIDENCE_INDEX.md .planning/agent-brain-v3/AGENT_LEDGER.md
```

Tool provenance: `ripgrep 15.1.0 (rev af60c2de9d)`, GNU `sha256sum` coreutils
9.4, git 2.43.0. Audit commands ran under repository HEAD
`b6571299b00c8e388abefe7ef9dcbcf8ac715d7f`; the worktree was already dirty and
the inspected planning files were untracked.

## Required disposition boundary

This critique does not mark either artifact invalid, alter an EV entry, accept or
reject an OpenSpec task, or prescribe a retroactive identity/provenance record.
Kiro must adjudicate whether remediation requires fresh check-in-backed reviewer
reproduction, contract-only re-documentation where independently supportable, or
continued OPEN status. Historical identities and the 5.3 process exception must
remain unchanged.
