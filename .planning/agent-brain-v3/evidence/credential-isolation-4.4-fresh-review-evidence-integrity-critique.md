# Credential-isolation 4.4 fresh-review evidence-integrity critique

Auditor: Codex-root

Role: independent contract-integrity auditor; not producer, reviewer, or adjudicator

Audit window: 2026-07-18T21:00:36Z–2026-07-18T21:04:43Z

Host: `manoelneto-laptop`, Linux WSL2 x86_64

Repository HEAD observed: `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f`

Adjudication authority: Kiro/Opus-4.8 TL

## Scope and disposition

This audit covers the fresh GLM artifact
`.planning/agent-brain-v3/evidence/credential-isolation-4.4-fresh-review.md`,
its durable producer/reviewer chain, current cited source/tests, OpenSpec task/spec
scope, AB-REQ/EV records, and the static database-offline topology.

No test, Go command, credential, environment value, database, network, live
provider, daemon, or service was executed or accessed. No product, test, spec,
task, checkbox, index, ledger, STATE, or git state was edited. The only writes
were this critique and its Golden Rule check-in/out.

Overall evidence-contract grade: **REJECT**. Current hashes pass 8/8, the
single-run named transcript is durably available through the referenced producer
artifact, reviewer/producer/adjudicator identities are distinct, and the focused
backend alert behavior is well supported. The binding contract still fails because
no GLM reviewer `.deploy-control` check-in/out exists. The x20/race/full results
are summaries rather than raw reviewer transcripts, EV identity has diverged
across artifact/index/ledger, and the two task-scope gaps remain open.

Grades mean:

- **PASS**: directly established by current source or durable records.
- **PARTIAL**: a bounded claim is supported, but the complete field is not.
- **REJECT**: a binding requirement is absent, inconsistent, or unproved.

## Contract-field grades

| Field | Grade | Durable finding |
|---|---|---|
| Artifact identity | PASS | Current SHA-256 is `cdb70e85fab9131c3aff59e52d037a57a8ee080265a51440bde071c526d08d95`, matching the requested hash, `EVIDENCE_INDEX.md:139`, and ledger `:298/:324`. |
| Current source/test manifest | PASS | All 8/8 hashes at artifact `:207-215` match current files exactly; see manifest below. |
| Rule-0 host/tool/source provenance | PARTIAL | Artifact `:28-42` states host, exact Go path/version, repository HEAD, UTC window, environment, and exact commands. However, reviewed files are modified or untracked, so HEAD `b6571299…` does not identify their contents; the 8-file manifest, not the commit, is the effective source provenance. |
| Review execution chronology | PARTIAL | Artifact states a 20:35–20:55Z window (`:36-37`), but current artifact mtime is 20:51:28Z and no transcript/check-out timestamps bracket individual commands. The window is plausible but not independently reconstructable. |
| Reviewer Golden Rule check-in/out | REJECT | No `.deploy-control` file binds `GLM52-auth-QA` or pane `w4:p3` to this 20:35–20:55Z execution. Artifact and later ledger rows are retrospective records, not Rule-3 pre-command check-in proof. |
| Planning-artifact authorship | PARTIAL | Ledger `:252/:300` explicitly assigns a fresh artifact to GLM, but Golden Rule 12 reserves `.planning` authorship to Kiro. The durable record contains delegated intent, not an explicit Rule-12 exception. |
| Reviewer identity | PASS | Artifact `:15` identifies `GLM52-auth-QA`, pane `w4:p3`; ledger `:298/:324` independently preserves that label. |
| Reviewer distinct from producer | PASS | Artifact `:16-26` and ledger identify producer Codex/root and reviewer GLM52-auth-QA without relabeling. |
| Reviewer distinct from adjudicator | PASS | Artifact identifies Kiro/Opus-4.8 as adjudicator (`:20-21`); the reviewer label/pane is GLM52-auth-QA `w4:p3`. Kiro did not perform this review. |
| Exact command text | PASS | Named, x20, race, vet, full-daemon, and gofmt commands are explicit at artifact `:146-195`, with working directory and absolute binaries. |
| “Exact producer reproduction” equivalence | PARTIAL | Fresh commands add `GOTOOLCHAIN=local`; the producer commands at `credential-isolation-reassignment-alerting.md:45-112` did not include it. The test selection is equivalent, but “exactly” at fresh artifact `:141-142` is overstated. |
| Durable named single-run transcript | PASS | Fresh artifact references producer transcript `:49-72`; that durable file contains 11 raw `=== RUN` and 11 raw `--- PASS` records: 6 top-level tests plus 5 subtests, zero FAIL. |
| Durable x20 count | PARTIAL | Fresh artifact `:161-164` asserts 120 top-level PASS and 220 RUN including subtests, but embeds no raw verbose output. If all subtest PASS lines are counted, total PASS records would also be 220; the stated 120 is valid only for a top-level-only filter that is not documented. |
| Race/vet/full-daemon durability | PARTIAL | Exact commands and exit/package summaries are present (`:166-189`), but no raw fresh-review transcript or independent transcript file was located. |
| Build-tag topology | PASS | The focused daemon alert tests are tag-free. `-tags=offline` excludes `internal/rotation/rotation_e2e_test.go` via `//go:build !offline`; staging DB smoke remains excluded because it requires the separate `staging` tag. No alert test is accidentally excluded. |
| Focused test DB/network isolation | PASS | The regex selects six pure daemon tests using in-memory fakes and a `bytes.Buffer`; none reads `DATABASE_URL`, creates a DB pool, or opens a socket (`credential_session_alert_test.go:17-140`, `credential_session_monitor_test.go:14-107`). |
| Full-package database contact prevention | PASS | The only normal daemon DB acceptance helper reads the synthetic DSN then calls `pgxpool.New` (`runtime_isolation_test.go:500-514`). pgx v5.9.2 `pgxpool.New` parses before pool construction (`pgxpool/pool.go:212-218`); `://offline-invalid` is treated as keyword/value and fails for no `=` (`pgconn/config.go:282-290`, `:634-639`). It returns before `NewWithConfig` or `Ping`, so it cannot contact PostgreSQL. |
| Full-package environment-secret isolation | REJECT | pgx calls `parseEnvSettings` before rejecting the malformed DSN (`pgconn/config.go:275-290`), and that function reads `PGHOST`, `PGUSER`, `PGPASSWORD`, passfile/TLS/service variables (`:514-549`). No values were inspected in this audit, but `DATABASE_URL='://offline-invalid'` alone does not prove the original process read no environment secrets. |
| Absolute “no network” claim | PARTIAL | The malformed DSN prevents DB dialing and `GOPROXY=off` prevents module proxy use, but the full daemon suite contains many `httptest.NewServer` loopback tests. Thus “no external/live network” is supported; literal “no network” at artifact `:43-45` is too broad. |
| Module-cache isolation claim | REJECT | Artifact `:42` calls the module cache slot-isolated, but no recorded command sets `GOMODCACHE`, `GOCACHE`, HOME, or another cache root. The inspected pgx module resides in the ordinary `/home/dataops-lab/go/pkg/mod` cache. Offline availability is supported; slot isolation is not. |
| Spec acceptance mapping | PASS | Spec `:82-84` requires alerting without credential overwrite when no account exists. Source/test evidence supports WARN `alert=no_account_available`, bounded attributes, and preserved assignment/auth state. |
| Task text mapping | PARTIAL | Task `tasks.md:28` explicitly names `useSessionMonitor/isExpiringSoon`; backend structured logs cover “registrar/alertar,” but do not deliver the named frontend mechanism. Artifact transparently non-claims it at `:257-259`. |
| AB-REQ-12 primary mapping | PARTIAL | `REQUIREMENTS.md:35` covers credential/quota lifecycle and expired-account handling across 4.x, so it is the closest semantic owner. It is not a dedicated task-4.4 alert requirement, as artifact `:74-78` honestly discloses. |
| AB-REQ-21/38 secondary mapping | PARTIAL | AB-REQ-21 (`REQUIREMENTS.md:49`) supports secret-safe logging and AB-REQ-38 (`:76`) supports operational alerts, but their registered OpenSpec tasks/scenarios are not this sibling task 4.4. They are cross-coverage, not direct acceptance IDs. |
| Proposed EV entry content | PARTIAL | Artifact `:80-88` correctly proposes a PENDING entry and does not claim index authority. Its technical summary inherits the x20 transcript limitation and the two scope gaps. |
| Current EV identity integrity | REJECT | The proposal uses `EV-CREDISO-4.4`; current index uses `EV-CREDISO-4.4-REVIEW` (`EVIDENCE_INDEX.md:139`); ledger `:298` uses `EV-CREDISO-4.4`, while ledger `:324` uses `EV-CREDISO-4.4-FRESH`. Three IDs identify the same fresh artifact, so EV provenance is not canonical. |
| No-secret alert attributes | PASS | Success/error tests inject synthetic sentinels and assert absence (`credential_session_alert_test.go:17-104`); production logging whitelists metadata and bounded error classes (`wakeup.go:330-376`). |
| Success/no-account/error/no-op backend coverage | PASS | The six focused top-level tests cover completion, no-account WARN, generic failure ERROR, DEBUG no-op, forwarding/error preservation, and malformed/unavailable/unrelated rejection. |
| Provider/tenant boundary preservation | PASS | Bridge forwards exact provider/workspace (`credential_session_monitor.go:83-99`); existing reassignment boundary tests are in the manifest. Task 4.4 does not broaden selection. |
| No-account “no overwrite” coverage | PASS | `service.go:97-122` selects before logout and returns on no candidate; the sibling 5.3 synthetic test asserts current assignment and zero auth calls. This is referenced evidence, not newly rerun here. |
| Frontend `useSessionMonitor/isExpiringSoon` root gap | PARTIAL | The omission is honestly and explicitly non-claimed (`:257-259`), so there is no evidence overclaim. It remains an unmet task-scoped delivery or an owner/spec scope decision; backend logs alone cannot close the named frontend mechanism. |
| `RecordRotation` failure-through-dispatch root gap | REJECT | Source returns a recording error before success (`service.go:150-159`) and dispatch would log ERROR, but no synthetic test injects `RecordRotation` failure through the real service/bridge and asserts absence of the completion alert. Current alert tests stub the reassigner directly, and `fakeStore.RecordRotation` cannot fail. Static/happy-path evidence does not prove this failure invariant. |
| Task-level technical completion | REJECT | Backend alerting is strong, but the two root gaps above remain. Artifact appropriately leaves task 4.4 OPEN; its `ACCEPT ... contract-complete` verdict at `:271-280` is too broad under Rule 3 and the unresolved scope/test gaps. |
| Checkbox/adjudication boundary | PASS | Artifact `:269-283`, index `:139`, and ledger `:298-300` keep 4.4 unchecked and reserve adjudication to Kiro TL. This audit changes no checkbox. |

## Current eight-file manifest

| File under `multica-auth-work/server` | Current SHA-256 | Grade |
|---|---|---|
| `internal/daemon/wakeup.go` | `71b89980c9b8cb7c03a81a502da5b9f4bdab4c0c522c117d2147b670e050f730` | PASS |
| `internal/daemon/credential_session_monitor.go` | `936b3e40f19c699078e994740c9ae63fdce5c172516b678d289f05c3cc38d1e2` | PASS |
| `internal/daemon/credential_session_monitor_test.go` | `5fd7005b4209b0c8d5c42ad9b240ad039d7568ad1f895bd6b9463b8e237613e2` | PASS |
| `internal/daemon/credential_session_alert_test.go` | `8e369510e814ff2f5743bf60dc76ee3074110a98da3daa172301aae1fda12bea` | PASS |
| `internal/rotation/service.go` | `f20951a36765d2b6b8576232871f1991697de56d8577377b0226ed5e30130cf0` | PASS |
| `internal/rotation/service_test.go` | `989ae633e1865bc0f1f5a9f7eed5d2541100dbee94960c9e2c15f84346aa8ba1` | PASS |
| `internal/rotation/discovery_reassignment.go` | `c655c9c4f94716d2c93109b1cc4d3c14e62d6fe8dea615dddec3ec93769b2832` | PASS |
| `internal/rotation/discovery_reassignment_test.go` | `d87aa80717e33a3f3768ebb7d698dcd9ace3bd934550317ec64c57f56a6566b2` | PASS |

## Exact read-only audit commands

No Go/test/vet/build command was run. Material checks used:

```text
sha256sum .planning/agent-brain-v3/evidence/credential-isolation-4.4-fresh-review.md multica-auth-work/server/internal/daemon/wakeup.go multica-auth-work/server/internal/daemon/credential_session_monitor.go multica-auth-work/server/internal/daemon/credential_session_monitor_test.go multica-auth-work/server/internal/daemon/credential_session_alert_test.go multica-auth-work/server/internal/rotation/service.go multica-auth-work/server/internal/rotation/service_test.go multica-auth-work/server/internal/rotation/discovery_reassignment.go multica-auth-work/server/internal/rotation/discovery_reassignment_test.go
rg -n 'RecordRotation|record.*fail|reassignment_failed|completed' multica-auth-work/server/internal/daemon/credential_session_alert_test.go multica-auth-work/server/internal/daemon/credential_session_monitor_test.go multica-auth-work/server/internal/rotation multica-auth-work/server/internal/daemon/wakeup.go multica-auth-work/server/internal/daemon/credential_session_monitor.go
rg -n 'DATABASE_URL|TestMain|go:build|pgx|postgres|redis|net.Dial|httptest' multica-auth-work/server/internal/daemon --glob '*_test.go' --glob '*.go'
rg -n -i 'w4:p3|GLM52-auth-QA' .deploy-control --glob '*.md'
rg -n 'EV-CREDISO-4.4|EV-CREDISO-4.4-REVIEW|EV-CREDISO-4.4-FRESH' .planning/agent-brain-v3/EVIDENCE_INDEX.md .planning/agent-brain-v3/AGENT_LEDGER.md .planning/agent-brain-v3/evidence/credential-isolation-4.4-fresh-review.md
rg -n 'useSessionMonitor|isExpiringSoon' openspec/changes/agent-credential-isolation . --glob '!**/.git/**' --glob '!**/node_modules/**'
```

Local pgx v5.9.2 source was read only at
`/home/dataops-lab/go/pkg/mod/github.com/jackc/pgx/v5@v5.9.2`; dependency version
is pinned at `server/go.mod:17` and `go.sum:82-83`. Audit tools: ripgrep 15.1.0
revision `af60c2de9d`, GNU sha256sum coreutils 9.4, git 2.43.0.

## Non-claims

This critique does not assert the summarized reviewer commands failed, inspect any
actual environment value, or contact a DB/network/service. It does not relabel the
reviewer, choose the frontend-vs-daemon scope policy, add/fix an EV entry, mark the
fresh artifact INVALID, or accept/reject the OpenSpec checkbox. Kiro TL adjudicates.
