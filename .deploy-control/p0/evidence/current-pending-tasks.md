# Current pending tasks — canonical control table

- **Version:** 6.1
- **Updated:** 2026-07-29T17:13:40Z
- **Scope:** production workspace `20fce817-895d-447b-965a-49f5e279314a` and P0 engineering program
- **Update owner:** General Tech Manager
- **Reconciliation owner for this revision:** Opus48-A under ORQ-67
- **Workspace cohort:** **62 issues** (whole workspace, not a project subset)
- **Cohort split:** 48 issues in project `4b0ef49b-df06-4e83-9a29-8a23b34821d4` (ORQ2) + 14 issues outside it
- **Done:** 35 of 62 (56.5%)
- **Snapshot method:** `multica issue list --limit 500 --output json`; API reported `total=62`, `has_more=false`
- **Prior snapshot preserved:** 60 issues, 34 Done, at `2026-07-29T16:57:17Z` — retained as historical below
- **Live daemon at update:** PID 1291834, uptime 1h55m38s, `cli_version=credential-home-token-only-20260727T102815Z`, `active_task_count=3`
- **Daemon unit:** `multica-daemon-orq2-credential.service` (`systemctl --user`) — `loaded`, `active`/`running`, `NRestarts=0`, `ExecMainStartTimestamp=Wed 2026-07-29 15:17:45 UTC`

This is the canonical human-readable status table. Runtime/API evidence and card notes remain authoritative for individual facts, but every material status, ETA, dependency or owner-decision change must be reflected here with a version increment and changelog entry.

**Volatility warning.** The board is live and moves during a reconciliation run. Three measurements in this run differed: at `16:51:58Z` the workspace held 60 issues with `in_review` 11 and `in_progress` 6; at `16:57:17Z` still 60 issues but `in_review` 14 and `in_progress` 3; at `17:13:40Z` 62 issues with 35 Done. Any count in this document is therefore true **only for its stated UTC timestamp**. A count without a timestamp is not evidence.

**Dispatch policy:** all new executable work starts through Kanban assignee/API and must produce exactly one product task. Herdr is supervision-only and must not launch parallel work. Existing Herdr work may finish without mid-flight redispatch. See `owner-ruling-kanban-only-agent-dispatch.md`.

## Historical workspace snapshot — 2026-07-29T16:57:17Z (preserved, superseded by the 17:13:40Z update)

Whole-workspace status counts, measured, summing to the API `total` of 60:

| Status | Count |
|---|---:|
| done | 34 |
| in_review | 14 |
| in_progress | 3 |
| blocked | 5 |
| todo | 1 |
| backlog | 1 |
| cancelled | 2 |
| **total** | **60** |

### Project ORQ2 — `4b0ef49b-df06-4e83-9a29-8a23b34821d4` (46 issues)

| Status | Count | Cards |
|---|---:|---|
| in_progress | 1 | ORQ-67 |
| in_review | 12 | ORQ-13, ORQ-14, ORQ-23, ORQ-35, ORQ-39, ORQ-50, ORQ-54, ORQ-60, ORQ-62, ORQ-66, ORQ-68, ORQ-69 |
| blocked | 2 | ORQ-37, ORQ-64 |
| todo | 1 | ORQ-63 |
| backlog | 1 | ORQ-53 |
| done | 29 | ORQ-12, ORQ-15, ORQ-16, ORQ-17, ORQ-18, ORQ-19, ORQ-20, ORQ-21, ORQ-22, ORQ-24, ORQ-25, ORQ-26, ORQ-30, ORQ-31, ORQ-33, ORQ-34, ORQ-36, ORQ-46, ORQ-49, ORQ-51, ORQ-52, ORQ-55, ORQ-56, ORQ-57, ORQ-58, ORQ-59, ORQ-61, ORQ-65, ORQ-70 |

### Outside project ORQ2 (14 issues)

These cards are real workspace work and must not be dropped from the cohort just because they carry no ORQ2 `project_id`. Thirteen have `project_id = null` and ORQ-45 belongs to project `abe3c461-c921-4a51-b91a-b08529429145`.

| Status | Count | Cards |
|---|---:|---|
| in_progress | 2 | ORQ-40, ORQ-41 |
| in_review | 2 | ORQ-47, ORQ-48 |
| blocked | 3 | ORQ-42, ORQ-43, ORQ-44 |
| cancelled | 2 | ORQ-32, ORQ-45 |
| done | 5 | ORQ-11, ORQ-27, ORQ-28, ORQ-29, ORQ-38 |

### Correction to the superseded 24-card and 46-card claims

Version 5.0 of this file recorded a **24-card** cohort with 18 Done (75.0%). That cohort no longer exists: the workspace has grown to 62 issues. A later documentation candidate narrowed the claim to the 46 ORQ2 cards only, which silently dropped 14 live cards including the three human-owned blockers ORQ-42, ORQ-43 and ORQ-44. Both framings are superseded. The 75% figure must not be quoted as current completion; the measured figures are **34/60 = 56.7% Done** at `2026-07-29T16:57:17Z` and **35/62 = 56.5% Done** at `2026-07-29T17:13:40Z`.

## Current state update — 2026-07-29T17:13:40Z

The `16:57:17Z` snapshot above is **preserved as historical** and is not edited. This section records
the delta measured after it, by the same method (`multica issue list --limit 500 --output json`).

| Status | 16:57:17Z | 17:13:40Z | Delta |
|---|---:|---:|---:|
| done | 34 | **35** | +1 |
| in_review | 14 | 14 | — |
| in_progress | 3 | **4** | +1 |
| blocked | 5 | 5 | — |
| todo | 1 | 1 | — |
| backlog | 1 | 1 | — |
| cancelled | 2 | 2 | — |
| **total** | **60** | **62** | **+2** |

API reported `total=62`, `has_more=false`. Changes:

- **ORQ-23 is now `done`** (it was `in_review` at the earlier snapshot). This is the rollback safety
  review before live gate 4.5 — the same harness implicated in the daemon regression and the ORQ-64
  exposure. It moves the completion figure to **35/62 = 56.5%**.
- **ORQ-71** created, `in_review`, project ORQ2 — "P0 ORQ2 Root Disk Saturation — safe
  terminal-artifact reclamation".
- **ORQ-72** created, `in_progress`, project ORQ2 — "P0 ORQ2 Emergency Disk Reclaim — five exact
  terminal pnpm stores".

Both new cards concern root-disk saturation on this host, which is consistent with the environment:
the root filesystem measured 98% used with roughly 1.7 GB free at the start of this run.

Measured project split at `17:13:40Z` — **48 in ORQ2**, 14 outside (13 with a null `project_id` plus
ORQ-45 in project `abe3c461-c921-4a51-b91a-b08529429145`):

| Group | Count | done | in_review | in_progress | blocked | todo | backlog | cancelled |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| ORQ2 | 48 | 30 | 12 | 2 | 2 | 1 | 1 | 0 |
| Outside ORQ2 | 14 | 5 | 2 | 2 | 3 | 0 | 0 | 2 |
| **Total** | **62** | **35** | **14** | **4** | **5** | **1** | **1** | **2** |

## Pending work — ordered by priority

| # | Card | State | Current work / next deliverable | Owner decision |
|---:|---|---|---|---|
| 1 | ORQ-64 | **BLOCKED — human** | P0 security incident: a Codex credential was printed by the ORQ-23 harness. Recorded content-free; no secret value is stored in this repository. The harness must run under an isolated `mktemp` test root, with `systemctl` stubbed and stdout/stderr sanitised, before any rerun. | **OWNER ACTION — credential re-authentication** |
| 2 | ORQ-42 | **BLOCKED — human** | Controlled `JWT_SECRET` rotation after ORQ-30. Card metadata records a re-review PASS on `fc77e89` (branch `agent/opus48-a/orq42-secret-tools-clean`) and states clean-branch integration is unblocked, but the card is still raw `blocked`. | **OWNER ACTION — rotation window** |
| 3 | ORQ-44 | **BLOCKED — human** | Lifecycle and rotation of the OmniRoute gateway inference key. | **OWNER ACTION — issuer/origin identity** |
| 4 | ORQ-43 | **BLOCKED — human** | Lifecycle and rotation of the daemon `mdt_` token. | **OWNER ACTION — rotation window** |
| 5 | ORQ-37 | **BLOCKED** | P0 security closure: MCP credential lifecycle and Cedar evidence. The blocker is **infrastructure and access, not a dispatch decision**: the runtime IAM role gets **AccessDenied** on Secrets Manager and SSM, the expected secret **SMA:2773 is absent**, and the scoped **`umask 077` cutover is still pending**. | Grant the runtime role Secrets Manager/SSM access and provision SMA:2773 |
| 6 | ORQ-68 | **IN REVIEW** | Scheduler reconciliation — a terminal failed task must release the agent. Canonical integration commit `15626386da2725af8e8d4ac611754cffe359fe31` exists and is **pending deploy**. Card metadata records that PR creation needs an authenticated `gh`. | None |
| 7 | ORQ-66 | **IN REVIEW** | Durable ORQ2 daemon combining AGY token-only task-home and reasoning admission. Base commit `368a4ba3c5f2653abc3aa7aca6e991a9735ed7a6` on `agent/opus48-a/orq66-token-only` **remains** the head of the delivered work; the follow-on task `64808d3d` was cancelled without delivery. | None |
| 8 | ORQ-63 | **TODO** | Register isolated Codex slot 170 and prove a Kanban canary. `slot-170/codex/auth.json` is now present and non-empty (measured), so the ORQ-66 precondition note that slot-170 had no `codex/auth.json` is stale; no daemon assignment yet points at slot-170. | None |
| 9 | ORQ-48 | **IN REVIEW** | Fleet documentation and Kanban dispatch control. Corrected Wave-3 ledger snapshot is remote commit `da004ca570237b1872047cdf02973a2198f5bfc4`. **This ORQ-67 revision does not modify the ORQ-48 ledger.** | None |
| 10 | ORQ-23 | **DONE at 17:13:40Z** (was `in_review` at the 16:57:17Z snapshot) | Rollback — independent safety review before live gate 4.5. This is the harness whose un-isolated execution caused both the daemon regression and the ORQ-64 exposure. Closing it does **not** clear ORQ-64: the harness isolation preconditions still gate any rerun. | None |
| 11 | ORQ-62 | **IN REVIEW** | OpenSpec full-lineage reconciliation. Accepted content lineage `7618599f29d43e964a485ab12a9932a9fd037e1f` is the base of the present ORQ-67 revision. | None |
| 12 | ORQ-40, ORQ-41, ORQ-47 | **IN PROGRESS / IN REVIEW** | Codex CLI alignment between ORQ1 and ORQ2; decoupling Kanban metadata from paid task execution; durable daily agent-cache lifecycle. All three sit outside project ORQ2 and were absent from the superseded 46-card framing. | None |

## Decisions currently required from the owner

Four cards are blocked on a human and must not be reported as agent-actionable: **ORQ-42**, **ORQ-43**, **ORQ-44** and **ORQ-64**. **ORQ-37** is blocked on infrastructure and access provisioning, not on a dispatch decision.

1. **ORQ-64** — re-authenticate the exposed Codex credential. Until then the ORQ-23 harness must not be rerun outside an isolated test root.
2. **ORQ-42 / ORQ-43** — supply a rotation window for `JWT_SECRET` and for the daemon `mdt_` token.
3. **ORQ-44** — supply the issuer/origin identity for the OmniRoute gateway inference key.
4. **ORQ-37** — grant the runtime IAM role read access to Secrets Manager and SSM, which currently return **AccessDenied**, and provision the absent secret **SMA:2773**. The scoped `umask 077` cutover then remains as the pending implementation step. Earlier control text framed this as "owner decisions on ORQ-34 dispatch"; that framing is **incorrect and is retracted**, because no dispatch decision unblocks an IAM AccessDenied or a secret that does not exist.

Standing owner authorization recorded at `2026-07-28T20:29Z` remains in force: when the exact target, verified backup, rollback and bounded action are technically approved by the General Tech Manager, execution proceeds without re-requesting the same generic approval. A new decision is required only if the target set or risk scope changes. That authorization does **not** cover the four human-blocked cards above, because each needs a credential or window that only the owner holds.

## 2026-07-29 ORQ2 daemon regression and recovery — measured record

### Live artifact identity

The daemon binary currently on the hot path was measured with `sha256sum` at `2026-07-29T16:5xZ`:

- **Live daemon artifact SHA-256:** `88ca4f397ef841edac091f15e77b2ba52754045a471acaa311c280827d2dd1f8`
- Path: `/home/ec2-user/.local/lib/multica/bin/multica-auth-credential-home-v1`, size 15053065 bytes, mode `0700`, mtime 2026-07-29 15:17
- Reported by the running process as `cli_version=credential-home-token-only-20260727T102815Z`

A prior documentation candidate recorded this value as `sha256:88ca4f39` padded to 64 characters with zeros. That padded string is **fabricated** and is retracted here. The 64-character value above is the measured digest.

The pre-recovery artifact is retained beside it as `multica-auth-credential-home-v1.pre-recovery-20260729T151745Z` (21333458 bytes), which is a different size from the live artifact and therefore a different build.

### Health and readiness endpoints — measured from host `orq2`

| Endpoint | Result | Meaning |
|---|---|---|
| `http://127.0.0.1:19514/health` | HTTP 200 | **ORQ2-local** daemon gate; returned PID 1291834, `daemon_id=orq2-credential-runtime-v1` |
| `http://127.0.0.1:18080/readyz` | HTTP 200, `{"status":"ok","checks":{"db":"ok","migrations":"ok"}}` | backend readiness; a **separate** gate from daemon health. Reached through an SSH forward — see below. |
| `http://127.0.0.1:13100/` | HTTP 000 (connection refused) | **not listening on this host** |
| `https://orq1.tail96e2c0.ts.net/` | HTTP 200 | canonical user-facing Kanban URL |

Only `19514/health` is served by a process on ORQ2. The other three describe ORQ1.

### Daemon supervision — the daemon IS systemd-managed, by a user unit

**The canonical unit is `multica-daemon-orq2-credential.service`, and it is a `systemd --user` unit.**
It must be queried with `systemctl --user`, not with system-scope `systemctl`. Measured:

| Property | Value |
|---|---|
| `Id` | `multica-daemon-orq2-credential.service` |
| `Description` | Multica credential-isolated daemon on ORQ2 |
| `LoadState` | `loaded` |
| `ActiveState` | `active` |
| `SubState` | `running` |
| `Type` | `simple` |
| `MainPID` | `1291834` |
| `NRestarts` | `0` |
| `ExecMainStartTimestamp` | `Wed 2026-07-29 15:17:45 UTC` |

Cgroup membership ties the running PID to that exact unit:

`/proc/1291834/cgroup` → `0::/user.slice/user-1000.slice/user@1000.service/app.slice/multica-daemon-orq2-credential.service`

`--foreground` in `ExecStart` is the **correct** invocation for a `Type=simple` unit: systemd requires the
main process to stay in the foreground so it can supervise it. The flag is therefore evidence *of*
proper systemd supervision, not evidence against it.

**Retraction.** Version 6.0 as first committed (`e3ffb4cd22c8d75472ae5077e5c5ca67a4cc8b4c`) claimed
"process supervision is not systemd-unit managed". That claim is **false and is withdrawn**. It came
from querying the wrong unit in the wrong scope: `systemctl show multica-daemon` returns
`LoadState=not-found`, and systemd reports `ActiveState=inactive`/`MainPID=0` as the *default values of
a unit that does not exist*. Those defaults were misread as measurements about the daemon. The
lesson is recorded as a requirement: a unit query is only evidence once `LoadState=loaded` confirms
the unit exists in the scope queried.

### Combined daemon gate — all five signals required

A daemon recovery or restart is verified only when **all** of the following hold together. No single
signal is sufficient.

| # | Signal | Required value | Source |
|---:|---|---|---|
| 1 | `ActiveState` | `active` (with `SubState=running`) | `systemctl --user show multica-daemon-orq2-credential.service` |
| 2 | `NRestarts` | `0` | same unit query |
| 3 | `ExecMainStartTimestamp` | matches the intended start instant | same unit query |
| 4 | cgroup membership | `/proc/<MainPID>/cgroup` names the unit | `/proc/<pid>/cgroup` |
| 5 | daemon health | HTTP 200 on `http://127.0.0.1:19514/health` | `curl` |

Signal 2 is only meaningful in combination with signals 1 and 3: `NRestarts=0` on a unit that never
started, or on a nonexistent unit, proves nothing. Signal 4 is what forbids the error made in the
first version 6.0 commit, because it fails loudly when the queried unit is not the one running the
process.

There is a second active user unit on this host, `multica-orq1-backend-tunnel.service`
(`active`/`running`, `MainPID=3211411`, `NRestarts=0`, `ExecMainStartTimestamp=Sun 2026-07-26 23:47:47 UTC`).
It is not a daemon gate; its role is described under endpoint topology below.

### Endpoint topology — ORQ2 serves only the daemon; the rest is ORQ1

This host is tailnet node `orq2` (`100.110.178.47`); `hostname` is
`ip-172-31-30-9.sa-east-1.compute.internal`. `tailscale serve status` on this host returns
**`No serve config`**. The canonical URL `https://orq1.tail96e2c0.ts.net` answers HTTP 200 and is
served by the tailnet peer `orq1`. Consequently `127.0.0.1:13100` is the internal frontend origin
**on ORQ1**; on ORQ2 nothing listens there.

**`127.0.0.1:18080` on ORQ2 is also ORQ1's backend, not a local one.** The user unit
`multica-orq1-backend-tunnel.service` ("Multica ORQ2 to ORQ1 backend tunnel") runs
`ssh -NT -o BatchMode=yes -o ExitOnForwardFailure=yes -o ServerAliveInterval=15 -o ServerAliveCountMax=3 -L 18080:127.0.0.1:18080 orq1`.
So the `readyz` 200 recorded above is **ORQ1's backend answering through a local forward**, and the
daemon's own `server_url=http://127.0.0.1:18080` resolves to ORQ1 by the same path.

The operational consequence: a `readyz` failure on ORQ2 is ambiguous between an ORQ1 backend fault
and a broken SSH forward, and those have different fixes. The tunnel unit's `ActiveState` must be
checked before an ORQ2-side `readyz` result is attributed to the backend. Describing 18080 as an
"ORQ2 backend" would send an operator to the wrong machine, the same class of error as the 13100
misattribution.

### Cause

The regression was caused by an **unguarded restart that re-exposed the binary already referenced by `ExecStart`** — not by a binary swap and not by a deployment. Recovery therefore restored a known-good artifact and relaunched, rather than reverting a substitution.

### Admission freeze mechanism

The bounded rollback was performed under an admission freeze taken as `LOCK TABLE agent_task_queue IN SHARE MODE`. Earlier notes describing a `pg_advisory_lock` are incorrect and are retracted.

### Credential slot inventory — measured, no credential value read

Measured by artifact presence and non-zero size only, under `/home/ec2-user/.agent-cred-homes/slots`:

| Provider | Slots with a present, non-empty provider artifact |
|---|---|
| AGY / antigravity (`home/.gemini/antigravity-cli/antigravity-oauth-token`) | **162, 163, 168, 169** |
| Codex (`codex/auth.json`) | **152, 170** |
| Kiro (`xdg-data/kiro-cli/*.sqlite3`) | 139, 140, 143, 145, 149, 155, 160, 161 |

The operative Kiro allowlist is **139, 140, 143, 149**. Artifact presence is a superset of allowlist membership: slots 145, 155, 160 and 161 hold Kiro state but are not in the allowlist. Presence on disk is therefore not proof of eligibility, and the two facts must be reported separately.

The affinity ledger `multica-assignments.v1.json` holds 13 `agent_id|provider -> slot` entries and contains **no credential values**. Two antigravity assignments (`16ae5315-...` and `da9201b6-...`) still point to `slot-145`, which holds no AGY token and is not AGY-allowlisted. This confirms the ORQ-66 precondition note about non-allowlisted antigravity assignments and remains open.

### ORQ-64 credential exposure — content-free

Recorded without any secret value, fragment, length or location that would narrow the secret. The exposure happened because the ORQ-23 harness ran without an isolated test root and without output sanitisation. Remediation preconditions before any rerun: isolated `mktemp` test root, stubbed `systemctl`, sanitised stdout/stderr. The card stays `blocked` pending owner re-authentication.

## Referenced commits — every SHA verified with `git rev-parse`

All values below are **commit** SHAs, resolved to full 64-hex form in this repository. They are not artifact digests.

| Label | Full commit SHA | Subject / status |
|---|---|---|
| ORQ-62 accepted content lineage (base of this revision) | `7618599f29d43e964a485ab12a9932a9fd037e1f` | `docs(deploy): restore ORQ-23 rollout and rollback runbook procedures alongside Wave3 OmniRoute rules` |
| ORQ-68 canonical integration — **pending deploy** | `15626386da2725af8e8d4ac611754cffe359fe31` | `fix(server): preserve active tasks during issue review` |
| ORQ-66 delivered base — **remains** | `368a4ba3c5f2653abc3aa7aca6e991a9735ed7a6` | `feat(daemon): wire frozen task identity to the physical credential slot resolver` |
| ORQ-48 corrected snapshot | `da004ca570237b1872047cdf02973a2198f5bfc4` | `docs(orq-48): Wave 3 live recorder ledger correction per GTL review` |
| ORQ-58 production release revision | `112e8dada455b4e7a3400e63728e00e6e3a0aa27` | merge commit; recorded in ORQ-58 card metadata as the release |
| ORQ-58 rollback revision | `82272414c94583e2689bc93735c0978a67e67132` | short form `8227241` as cited by the GTL |
| ORQ-67 candidate — **rejected, do not reuse** | `ea501879df4d0b12700734e018b0457ce102b1f5` | rejected for a padded artifact hash and a stale snapshot |
| ORQ-67 candidate — **superseded** | `e431f0ae156114bb4b32329d77457264e6e95c8d` | superseded by review before `ea501879` |

### ORQ-58 container image digests — reported, not measured here

ORQ-58 card metadata records the release as `112e8dada455b4e7a3400e63728e00e6e3a0aa27`, the candidate image tag `multica-backend:orq58-canonical-112e8da-20260729` and the rollback tag `multica-backend:rollback-before-orq58-922b138-20260729`. The GTL additionally reported the live image digest `sha256:e14f5c35d0640ec4c955efd8d5bbbb6cf219a4d544faafc1fd0f028c5bafcf13` and the rollback image digest `sha256:922b13862036d906a1ad5cde3e1615adad45393edb13896d2b146753b8384ab6`.

Those two image digests are recorded **as reported by the GTL and not independently measured**, because no container runtime is installed on this host: `docker` and `podman` are both absent (`command not found`). They must be re-verified on the host that runs the images before being treated as measured facts. ORQ-58 is `done` and its post-deploy canary card ORQ-70 is `done`.

Note that ORQ-58 card metadata still carries a stale `blocked_reason` and `pipeline_status=candidate_built_waiting_queue_zero` even though the card is `done`. That is an ORQ-58-owned metadata cleanup and is deliberately not changed by ORQ-67.

## Proposed status transitions — for GTL approval, not applied

ORQ-67 is documentation-only and applies no status change to any other card. The following are proposals only:

| Card | Current | Proposed | Reason |
|---|---|---|---|
| ORQ-42 | blocked | keep blocked, annotate | Metadata records a re-review PASS on `fc77e89` and says clean-branch integration is unblocked; the card text and status disagree. Needs a GTL ruling, not a silent flip. |
| ORQ-63 | todo | keep todo, annotate | `slot-170/codex/auth.json` is now present, so the registration precondition is partly met; the Kanban canary is still unproven. |
| ORQ-66 | in_review | keep in_review, annotate | Follow-on task `64808d3d` was cancelled without delivery; base `368a4ba3c5f2653abc3aa7aca6e991a9735ed7a6` stands. The slot-145 antigravity assignments remain a live precondition. |
| ORQ-58 | done | keep done, clean metadata | Stale `blocked_reason` and `pipeline_status` contradict the `done` state. Owned by ORQ-58. |

No card is proposed for `done` in this revision. No duplicate or already-resolved card was found that would justify a `cancelled` transition beyond the two already cancelled (ORQ-32, ORQ-45).

## Appendix A — preserved historical record from version 5.0 (2026-07-28)

The three sections below are reproduced **verbatim** from version 5.0 of this file at commit
`7618599f29d43e964a485ab12a9932a9fd037e1f`. They are historical evidence for the 24-card cohort
and its 18-Done/75% acceptance. They are **superseded** by the measured 60-issue snapshot in the
body of this document and must not be quoted as current state. They are preserved because the
underlying executions, task IDs and evidence digests they record actually happened.

### A.0 — Header block as recorded on 2026-07-28 (superseded)

- **Version:** 5.0
- **Updated:** 2026-07-28T22:07:00Z
- **Scope:** production workspace and P0 engineering program
- **Update owner:** General Tech Manager
- **Kanban cohort:** 24 cards
- **Done:** 18 (75.0%)
- **Target today:** 18 Done (75%)
- **Remaining to target:** 0
- **Live execution:** zero active product tasks at the acceptance snapshot
- **Current columns:** Done 18, In Progress 1, In Review 4, Blocked 1, Todo 0

The 24-card cohort no longer exists and the 75.0% figure is superseded by the measured 60-issue
snapshot in the body of this document. It is preserved because the acceptance it records took place.

### A.1 — Pending work as recorded on 2026-07-28


| # | Card | State | Current work / next deliverable | ETA | Owner decision |
|---:|---|---|---|---:|---|
| 1 | ORQ-26 | **IN REVIEW** | Backend regression fixes are integrated; close only after the production chat-panel browser smoke proves attach/send/stop on the current clean frontend image. | 30–60 min | None |
| 2 | ORQ-39 | **BLOCKED — executable environment** | Static implementation and peer review passed. Rebase the exact seven-file package onto the current base and run the ephemeral Playwright gate locally or on available CI; GitHub billing is not a product blocker. | 45–90 min | None |
| 3 | ORQ-35 | **IN PROGRESS** | PostgreSQL/DATABASE_URL hardening remains. The Codex provider account is now registered, but its refresh token is revoked; work can be reassigned to AGY where provider specificity is unnecessary. | 60–120 min | **OWNER ACTION — refresh Codex slot 152 if Codex execution is required** |
| 4 | ORQ-33 | **IN REVIEW** | JWT_SECRET runbook/tooling requires final independent acceptance and a queue-zero production rotation with tested rollback. | 45–90 min | Rotation window when requested |
| 5 | ORQ-34 | **IN REVIEW** | Secret-safe consumer map is ready; supply exact non-secret secret identity/provider distinction, then execute bounded rotation without exposing value. | 30–60 min after metadata | Exact non-secret identifier |
| 6 | ORQ-37 | **IN REVIEW** | MCP tooling-header custody hardening needs the reviewed installer corrections and scoped cutover; origin rotation still depends on identifying the external issuer. | 60–110 min | Issuer identity for origin rotation |

### A.2 — Owner decisions as recorded on 2026-07-28


No owner decision blocks the already-achieved 75% target. Human actions are now elevated immediately as **OWNER ACTION — IMMEDIATE** with the exact reason, action and estimated duration. Likely next actions for the remaining six cards are:

1. Refresh Codex slot 152 only if ORQ-35 must execute specifically under Codex; ORQ-20 was closed with a successful Kiro `high` canary.
2. Exact non-secret secret identifiers and a rotation window when ORQ-33/34 reach executable readiness.
3. Origin/issuer identity for the two MCP tooling headers before ORQ-37 rotates them; local custody hardening does not wait for that identity.

Standing owner authorization recorded at `2026-07-28T20:29Z`: when the exact target, verified backup, rollback and bounded action are technically approved by the General Tech Manager, execution is authorized without asking the owner for the same generic approval again. A new decision is required only if the target set or risk scope changes.

### A.3 — Live Kanban reconciliation narrative as recorded on 2026-07-28


At `2026-07-28T15:54:35Z`, the exact project `4b0ef49b-df06-4e83-9a29-8a23b34821d4` was read back after a status-only repair:

| Column | Before | After |
|---|---:|---:|
| Blocked | 5 | 5 |
| Done | 5 | 5 |
| In Progress | 0 | 1 |
| In Review | 3 | 2 |
| Todo | 9 | 9 |

ORQ-12 is now raw `in_progress`. Its assignee remains `NULL`, active product-task count remains `0`, and no other card was changed. Evidence: `orq12-in-progress-status-verification.md` (`sha256:90cb69c18c4a67fa29342e759a412094c6101c6722803fd3f619e993e35dea0f`).

After the ORQ-21 delta received independent PASS, ORQ-31 was released from `PAUSED_SEQUENCE` and reconciled from raw `in_review` to `in_progress`. The verified live counts are now: Blocked 5, Done 5, **In Progress 2 (ORQ-12 and ORQ-31)**, In Review 2 and Todo 9. Assignee remains `NULL`, active product-task count remains `0`, and no other card changed. Evidence: `orq31-in-progress-status-reconciliation.md` (`sha256:8df5c227d73f8e98fae1f4ccb2ba33ddb327b4d2b9cf54fd119d15b3ac29edc3`).

ORQ-31 subsequently received independent PASS for its declared Wave-A containment scope and moved to `done`; ORQ-39 moved to `in_review`. Confirmed executions then moved ORQ-15, ORQ-23, ORQ-33, ORQ-34, ORQ-26 and ORQ-36 from `todo` to `in_progress`. Latest verified count: Blocked 5, Done 6, **In Progress 8**, In Review 3, Todo 2. The remaining Todo cards are ORQ-35 and ORQ-37. Active product-task queue remains zero because these were status-only reconciliations, not assignee changes.

ORQ-33 and ORQ-18 then delivered reviewable artifacts and moved to `in_review`; ORQ-37 started and moved to `in_progress`. Latest verified count: Blocked 5, Done 6, **In Progress 7**, In Review 5, **Todo 1 (ORQ-35)**. Active product-task queue remains zero.

ORQ-12 then moved to `in_review` because its execution and gates were complete. External Herdr-owned cards received idempotent `/note` ownership markers, while ORQ-26 retains its real Agy-P0-A8 assignee. Latest verified count: Blocked 5, Done 6, **In Progress 6**, **In Review 6**, Todo 1. Active product-task queue remains zero. Future work must start through product assignee/API with exactly one product task; Herdr may supervise but must not launch a duplicate execution.

After the review wave, ORQ-15 returned to `in_progress` for the B1 correction and ORQ-35 was reassigned through the product API to an eligible Codex-C. Latest verified count: Blocked 5, Done 6, **In Progress 3**, **In Review 10**, **Todo 0**. The active product queue contains exactly one ORQ-35 task in `dispatched`; no Herdr duplicate was launched.

At `2026-07-28T17:25:57Z`, ORQ-12 was assigned natively through Kanban to Opus48-A and moved to `in_progress` in the same update. Exactly one product task, `8b825daf-f675-41a9-845b-c7497bf7e3a6`, was created and reached `running`; no Herdr activation was used. The stale Opus-46-B blocked metadata was removed and the card now carries the exact migration-128 execution contract. The ORQ-35 task was also rechecked and is terminal `failed`, not active, because Codex-C's provider refresh token was revoked. Current project counts read back from the API: Blocked 5, Done 6, **In Progress 3**, **In Review 9**, **Todo 1**.

The first ORQ-12 task completed as a diagnostic instead of writing: it proved that `ReclaimStaleDispatchedTaskForRuntime` filters out covered-provider legacy rows with `credential_account_id IS NULL`, preventing the ORQ-21 fail-closed cancellation path from seeing them. The issue was returned to `in_progress`, the stale no-write interpretation was explicitly superseded, and exactly one new Kanban task (`5601a616-3a14-4219-8409-97c7879d389b`) was created at `2026-07-28T19:07:43Z`. It is now `running`; no Herdr activation or duplicate task was used.

At `2026-07-28T20:32Z`, the blocked column remains **zero**, but execution capacity was reconciled honestly after two pre-start failures. ORQ-12 independent review is running as task `174565eb` on Gemini-3.6-Flash-A and ORQ-14 authoritative-token work is running as `f43e3bc1` on Gemini-3.6-Flash-B. ORQ-13 task `81ff5f41` failed before start on missing Kiro authentication state; ORQ-19 task `c2a35b50` failed before start on Opus-46 individual quota. They were unassigned and returned to Todo instead of being falsely shown as active, with explicit next sequencing to Gemini-A and Gemini-B respectively. The Kiro slot-140 directory skeleton was restored to the required 0700 layout without reading or copying credentials, but its missing `data.sqlite3` was not fabricated. Live project counts are **Done 6, In Progress 3, In Review 12, Todo 3, Blocked 0**; the third Todo is ORQ-35, whose prior Codex-C task failed because its provider refresh token was revoked.

At `2026-07-28T21:49Z`, the AGY credential pool was rebuilt from zero. Four distinct owner-authenticated sessions were assigned automatically to slots `162`, `163`, `168` and `169`; all token files are nonempty `0600` under `0700` slot directories and no token content was read. The daemon allowlist, its local affinity ledger and the production `accounts`/`approved_accounts`/`assignments` metadata now agree. Four product tasks are concurrently `running` through Kanban, each with a distinct frozen `credential_account_id`: ORQ-18/Agy-A7, ORQ-26/Gemini-A, ORQ-14/Agy-A8 and ORQ-21/Gemini-B. Live project counts are **Done 11, In Progress 5, In Review 7, Todo 1, Blocked 0**.

At `2026-07-28T22:07Z`, the verified target was reached: **18 of 24 cards are Done (75.0%)**, with zero active product tasks at the acceptance snapshot. ORQ-14, ORQ-18, ORQ-21 and ORQ-23 completed their acceptance paths; ORQ-15 closed after the four-account AGY pool ran concurrently with unique frozen account IDs; ORQ-36 closed after independent PASS confirmed that `MULTICA_TOKEN` is a per-task `mat_` lifecycle rather than a static secret awaiting manual rotation. ORQ-20 closed after a production Kiro canary ran `thinking_level=high`, started ACP, emitted messages and executed tools without `thinking_not_approved`; all six Codex/Kiro agents read back `high` and AGY agents remain `NULL`. A stale historical note temporarily returned ORQ-20 to blocked after the canary, and the GTM restored the accepted Done state after the task became terminal.

## Version history

| Version | Date | Change |
|---|---|---|
| 6.1 | 2026-07-29 | **Corrected the false supervision claim in 6.0.** The daemon **is** systemd-managed: the canonical unit is `multica-daemon-orq2-credential.service` under `systemctl --user` — `loaded`, `active`/`running`, `Type=simple`, `MainPID=1291834`, `NRestarts=0`, `ExecMainStartTimestamp=Wed 2026-07-29 15:17:45 UTC` — and `/proc/1291834/cgroup` names that exact unit. Version 6.0 queried the nonexistent system-scope unit `multica-daemon` (`LoadState=not-found`) and misread systemd's defaults for a missing unit as measurements; `--foreground` is correct `Type=simple` supervision, not evidence against systemd. Restored the combined five-signal daemon gate (ActiveState + NRestarts + ExecMainStartTimestamp + cgroup membership + health 19514). Corrected the ORQ-37 blocker to IAM Secrets Manager/SSM **AccessDenied**, absent **SMA:2773** and the pending `umask 077` cutover, retracting the "owner decisions on ORQ-34 dispatch" framing. Established that `127.0.0.1:18080` on ORQ2 is ORQ1's backend reached through `multica-orq1-backend-tunnel.service` (`ssh -L 18080:127.0.0.1:18080 orq1`), so only `19514/health` is ORQ2-local. Preserved the `16:57:17Z` 60-issue snapshot as historical and added a measured `17:13:40Z` current state: 62 issues, 35 Done (56.5%), ORQ-23 now `done`, new cards ORQ-71 and ORQ-72. |
| 6.0 | 2026-07-29 | Rebased the cohort on the measured **60-issue** workspace (34 Done, 56.7%) at `2026-07-29T16:57:17Z` with the snapshot method recorded; retracted the padded artifact hash and recorded the measured 64-char daemon digest `88ca4f397ef841edac091f15e77b2ba52754045a471acaa311c280827d2dd1f8`; corrected the freeze mechanism to `LOCK TABLE agent_task_queue IN SHARE MODE`; separated daemon health `19514/health` from backend readiness `18080/readyz`; established that the canonical URL is served by peer `orq1` and that `127.0.0.1:13100` does not listen on ORQ2; restored the 14 non-ORQ2 cards to the cohort including human blockers ORQ-42/43/44; verified every cited commit SHA with `git rev-parse`; marked the ORQ-58 image digests as GTL-reported and unmeasurable on this host. **Superseded in part by 6.1: its supervision section was factually wrong.** |
| 5.0 | 2026-07-28 | Verified and recorded the 75% target: 18/24 Done. Closed ORQ-14/15/18/20/21/23/36, integrated the ORQ-23 rollback fix, recorded the Kiro `high` production canary, and reduced the live pending table to six cards. |
| 4.0 | 2026-07-28 | Rebased the canonical table on the live 24-card Kanban, recorded 11 Done and the seven-card gap to 75%, deployed ORQ-18, rebuilt the four-account AGY pool from zero, and recorded four concurrent Kanban tasks with distinct frozen account IDs. |
| 3.2 | 2026-07-28 | Corrected two pre-start execution failures without cosmetic status: ORQ-13 and ORQ-19 are READY and serially queued behind the two healthy Gemini tasks; blocked remains zero and live counts are 6/3/12/3/0. |
| 3.1 | 2026-07-28 | Eliminated the blocked Kanban column by launching real work for ORQ-12/13/14/19; recorded standing owner authorization, exact task IDs, credential-slot infrastructure findings and current 6/6/11/1/0 board counts. |
| 3.0 | 2026-07-28 | Recorded the real ORQ-12 combined-tree reclaim defect and launched one native correction task with explicit write authority, focused regressions and migration-128 promotion contingent on a green gate. |
| 2.9 | 2026-07-28 | Dispatched ORQ-12 natively to Opus48-A with one running product task and an exact migration-128 execution contract; corrected ORQ-35 from dispatched to terminal auth failure after live verification. |
| 2.8 | 2026-07-28 | Reconciled latest deliveries and ETAs: ORQ-18 and ORQ-36 independent PASS; ORQ-23 and ORQ-37 actionable BLOCKs; ORQ-15 partial build/vet interrupted by provider limit; ORQ-26 diagnosis complete; ORQ-33 review incomplete; ORQ-34 readiness complete. |
| 2.7 | 2026-07-28 | Activated the owner ruling that all new agent work, including independent reviews, is dispatched only through Kanban assignee/API with exactly one product task. Herdr is supervision-only; Kiro-Opus5 remains reserved. |
| 2.6 | 2026-07-28 | Reached Todo zero. Returned ORQ-15 to implementation for the frozen-account/HOME binding defect and started ORQ-35 through the correct product-assignee flow with exactly one dispatched task and no Herdr duplicate. |
| 2.5 | 2026-07-28 | Recorded ORQ-37 Ruling V2: unknown issuer blocks only origin rotation; reversible private custody and scoped `umask 077` implementation continues, with no deletion and no host cutover before review/canary/rollback. |
| 2.4 | 2026-07-28 | Corrected ORQ-37 title and scope to MCP tooling-header lifecycle (Plan C/B7); explicitly separated `mdt_` into ORQ-43 and `mcp_config` regression coverage into its own follow-up. |
| 2.3 | 2026-07-28 | Moved completed ORQ-12 execution to review; recorded idempotent external-owner notes for the temporary Herdr-owned lanes and established the permanent policy of product assignment first with exactly one task. |
| 2.2 | 2026-07-28 | Advanced ORQ-33 and ORQ-18 to review, started ORQ-37, and reduced the live Todo column to one card (ORQ-35). Current board: 7 In Progress, 5 In Review, 6 Done, 5 Blocked. |
| 2.1 | 2026-07-28 | Closed ORQ-31 after independent Wave-A PASS; moved ORQ-39 to review; reconciled six confirmed executions to produce 8 In Progress and only 2 Todo. Corrected ORQ-26 to the actual chat-panel regression card rather than the historical CI-label collision. |
| 2.0 | 2026-07-28 | Resolved ORQ-31 review independence without pausing the card: Codex56-A is limited to factual current-state measurement and Codex56-B owns the independent binary verdict. |
| 1.9 | 2026-07-28 | Recorded ORQ-21 R3 FINAL PASS and its evidence-only commit `42db561`; ORQ-12 combined gate is green and now waits only for Registrar materialization plus the aggregate active-queue-zero integration/deploy gate. |
| 1.8 | 2026-07-28 | Recorded independent PASS for ORQ-21 `feccee4`; released ORQ-31 from sequence hold and reconciled it to verified `in_progress`. Live Kanban now has two cards in progress: ORQ-12 and ORQ-31. |
| 1.7 | 2026-07-28 | Reconciled live Kanban state: ORQ-12 moved from stale `in_review` to verified `in_progress`, with no assignee change or product task; recorded before/after column counts and current dispatches for ORQ-18, ORQ-39 and OPS-LIFECYCLE. |
| 1.6 | 2026-07-28 | Marked ORQ-21 independently PASS for this stage; recorded the narrow ORQ-12 invented-test-schema assumption under active correction; corrected OPS-LIFECYCLE from “external capacity” to transient request-rate throttling with normal plan capacity. |
| 1.5 | 2026-07-28 | Recorded terminal response-stream throttle on the one serial OPS-LIFECYCLE preflight. Cleared Opus48-A assignee, prohibited immediate retry and returned the lane to external-capacity handoff. |
| 1.4 | 2026-07-28 | Assigned exactly one serial Opus48-A task to OPS-LIFECYCLE for read-only preflight; mutation remains NOT_READY. Added max-one-heavy-request, zero-subagent and no-parallel-retry limits. |
| 1.3 | 2026-07-28 | Corrected Opus48-A classification: transient request-rate throttling at 60.4% plan usage, not capacity or credit exhaustion; refreshed the canonical timestamp. |
| 1.2 | 2026-07-28 | Recorded metadata-only ORQ-12 handoff: Opus48-A released, Opus-46-B pinned as pending owner, executable assignee empty and `ACK READY` required before source write/one task. Codex56-B remains ORQ-21 owner. |
| 1.1 | 2026-07-28 | Converted the on-disk report to one continuous table and recorded the explicit ORQ-12 ownership handoff from throttled Opus48-A to Opus-46-B. |
| 1.0 | 2026-07-28 | Initial canonical pending-task table. Corrected prior duplicate ORQ-37 row, restored omitted ORQ-43, incorporated ORQ-21 `aac2227` peer PASS, ORQ-26 full compatibility PASS, ORQ-39 amendment commit and ORQ-42 independent PASS. |
