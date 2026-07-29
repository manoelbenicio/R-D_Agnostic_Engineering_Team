# Proposal — ORQ-67 Live Control Reconciliation

## Why

Two prior ORQ-67 documentation candidates were rejected, and the reason in both cases was
**factual**, not structural. `openspec validate --all --strict` passed on the rejected commit
`ea501879df4d0b12700734e018b0457ce102b1f5` while that commit still recorded:

1. A **fabricated artifact digest**. The live daemon hash was written as
   `sha256:88ca4f39` right-padded with zeros to 64 characters. Padding is not measurement. The
   measured digest is `88ca4f397ef841edac091f15e77b2ba52754045a471acaa311c280827d2dd1f8`.
2. A **stale cohort**. It claimed a 28-non-Done board after the board had advanced, and a later
   framing narrowed the cohort to the 46 cards carrying the ORQ2 `project_id`, silently dropping
   14 live workspace cards — including the human-owned blockers ORQ-42, ORQ-43 and ORQ-44.
3. **Unverified mechanism claims**: a `pg_advisory_lock` admission freeze, a binary-swap root
   cause, and a conflated daemon-health/backend-readiness gate.

A first attempt at this change, commit `e3ffb4cd22c8d75472ae5077e5c5ca67a4cc8b4c`, fixed those three
but introduced a **fourth error of the same kind** and was itself factually rejected: it claimed the
daemon was not systemd-managed, having queried the nonexistent system-scope unit `multica-daemon`
(`LoadState=not-found`) and read systemd's defaults for a missing unit — `ActiveState=inactive`,
`MainPID=0` — as measurements. The daemon is in fact supervised by the user unit
`multica-daemon-orq2-credential.service`, `active`/`running`, and `--foreground` is the required form
for a `Type=simple` unit rather than evidence against systemd.

The lesson is the governing requirement of this change: **schema validation cannot certify truth**,
and neither can a well-formed tool output. A padded string can look like a digest and a default field
can look like a state. A control document must carry, for every material number, the measurement
command, the UTC instant, proof that the thing queried actually exists, and the distinction between
what was measured locally and what was reported by someone else.

## What Changes

- **ADDED** a capability spec `kanban-control-reconciliation` that makes measurement provenance a
  hard requirement for control documentation: timestamped snapshots, whole-workspace cohorts,
  unpadded digests, commit-vs-artifact labelling, and explicit separation of measured facts from
  reported facts.
- **MODIFIED** `.deploy-control/p0/evidence/current-pending-tasks.md` from version 5.0 to **6.1**:
  rebased on the measured 60-issue workspace (34 Done, 56.7%) at `2026-07-29T16:57:17Z`, preserved as
  historical, plus a measured current state at `2026-07-29T17:13:40Z` of 62 issues and 35 Done
  (56.5%) with ORQ-23 now `done` and new cards ORQ-71 and ORQ-72. Snapshot method recorded inline.
- **ADDED** the measured live-recovery record: 64-character daemon artifact digest, the
  `LOCK TABLE agent_task_queue IN SHARE MODE` admission freeze, the restart-re-exposure root cause,
  and separate daemon-health and backend-readiness gates.
- **ADDED** the verified supervision record and the **combined five-signal daemon gate**:
  `multica-daemon-orq2-credential.service` under `systemctl --user` with `LoadState=loaded`,
  `ActiveState=active`/`SubState=running`, `NRestarts=0`,
  `ExecMainStartTimestamp=Wed 2026-07-29 15:17:45 UTC`, cgroup membership of `MainPID=1291834`, and
  HTTP 200 on `19514/health`.
- **ADDED** requirement language forbidding the specific error that sank the first commit: a unit
  property is evidence only once `LoadState=loaded` proves the unit exists in the scope queried, and
  `MainPID` must be cross-checked against `/proc/<pid>/cgroup`.
- **ADDED** endpoint attribution by host: the canonical URL is served by tailnet peer `orq1`;
  `127.0.0.1:13100` does not listen on ORQ2; and `127.0.0.1:18080` is ORQ1's backend reached through
  `multica-orq1-backend-tunnel.service` (`ssh -L 18080:127.0.0.1:18080 orq1`), so only
  `19514/health` is ORQ2-local.
- **CORRECTED** the ORQ-37 blocker to IAM Secrets Manager/SSM **AccessDenied**, absent secret
  **SMA:2773** and the pending scoped `umask 077` cutover, retracting the "owner decisions on ORQ-34
  dispatch" framing.
- **ADDED** a content-free ORQ-64 record and preservation of the four human-blocked cards.
- **ADDED** `evidence.md` with the exact commands and their outputs.
- **RETRACTED** the padded digest and the `pg_advisory_lock` claim in writing, so the rejected
  commits are superseded by an explicit correction rather than by omission.
- **PRESERVED** the version 5.0 pending table, owner-decision list and reconciliation narrative
  verbatim in Appendix A, labelled superseded.

## Scope

Documentation only. Files touched: `openspec/changes/orq67-live-control-reconciliation/**` and
`.deploy-control/p0/evidence/current-pending-tasks.md`.

## Non-Goals

- No product code, build, deployment, migration or runtime mutation.
- No change to the ORQ-48 Wave-3 recorder ledger, which is owned by ORQ-48 at commit
  `da004ca570237b1872047cdf02973a2198f5bfc4`.
- No Kanban status transition on any card other than ORQ-67 itself. Transitions are **proposed**
  for GTL approval and left unapplied.
- No agent dispatch and no metadata cleanup on cards owned by other work items.
- No independent verification of container image digests: this host has neither `docker` nor
  `podman`, so those values are recorded as reported and flagged as unmeasured.

## Impact

- Control documentation stops carrying a completion percentage derived from a cohort that no
  longer exists. The measured figures are 34/60 Done at `16:57:17Z` and 35/62 Done at `17:13:40Z`,
  not 18/24.
- Four human-blocked cards regain visibility in the canonical table.
- Every commit SHA quoted in the control table is resolvable with `git rev-parse` in this
  repository, and every digest is either measured here or explicitly marked as unmeasured.
