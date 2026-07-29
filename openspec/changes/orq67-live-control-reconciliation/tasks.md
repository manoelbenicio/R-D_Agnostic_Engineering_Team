# Tasks

> Owner: Opus48-A under ORQ-67. Documentation only. No product code, no deployment, no dispatch,
> no Kanban status change on any card other than ORQ-67 itself.

## Phase 0 — Lineage

- [x] 0.1 Verify `7618599f29d43e964a485ab12a9932a9fd037e1f` exists and carries
  `.deploy-control/p0/evidence/current-pending-tasks.md`
- [x] 0.2 Confirm the lineage commit is **not** an ancestor of `origin/main`, and that the control
  file is absent on `main`, so basing on `main` would have destroyed version 5.0
- [x] 0.3 Base the working branch on the lineage commit
- [x] 0.4 Confirm the rejected change directory `live-recovery-reconciliation` is absent from the
  tree, so no rejected artifact is amended or reused

## Phase 1 — Live measurement

- [x] 1.1 Snapshot the board with `multica issue list --limit 500 --output json`; record the UTC
  instant, the API `total` and `has_more`
- [x] 1.2 Confirm the workspace total is **60** and that per-status counts sum to it
- [x] 1.3 Split the cohort into 46 ORQ2 cards and 14 cards outside ORQ2, listing every identifier
- [x] 1.4 Take a second snapshot and record the observed drift rather than discarding it
- [x] 1.5 Resolve the running daemon binary via `/proc/1291834/exe` and measure its SHA-256
- [x] 1.6 Confirm the measured digest is
  `88ca4f397ef841edac091f15e77b2ba52754045a471acaa311c280827d2dd1f8` at full 64-hex width
- [x] 1.7 Measure `127.0.0.1:19514/health`, `127.0.0.1:18080/readyz`, `127.0.0.1:13100/` and
  `https://orq1.tail96e2c0.ts.net/` and record each HTTP status
- [x] 1.8 Measure supervision with `systemctl --user show multica-daemon-orq2-credential.service`,
  confirm `LoadState=loaded` before reading any other property, and cross-check `MainPID` against
  `/proc/1291834/cgroup`
- [x] 1.8b Record that the system-scope query `systemctl show multica-daemon` returns
  `LoadState=not-found`, and that its `ActiveState=inactive`/`MainPID=0` are systemd defaults for a
  missing unit rather than measurements — the error that sank the first commit
- [x] 1.8c Identify `multica-orq1-backend-tunnel.service` and its `-L 18080:127.0.0.1:18080 orq1`
  forward, so `18080/readyz` is attributed to ORQ1
- [x] 1.9 Measure credential-slot artifact presence without reading any credential value
- [x] 1.10 Read the affinity ledger key set and confirm it contains no credential values
- [x] 1.11 Confirm no workdir exists for the cancelled ORQ-66 follow-on task `64808d3d`
- [x] 1.12 Attempt container-image verification; record the absence of `docker` and `podman`

## Phase 2 — SHA verification

- [x] 2.1 Resolve every cited commit with `git rev-parse --verify <sha>^{commit}`
- [x] 2.2 Confirm ORQ-68 canonical integration `15626386da2725af8e8d4ac611754cffe359fe31`
- [x] 2.3 Confirm ORQ-66 base `368a4ba3c5f2653abc3aa7aca6e991a9735ed7a6` remains the delivered head
- [x] 2.4 Confirm ORQ-48 corrected snapshot `da004ca570237b1872047cdf02973a2198f5bfc4`
- [x] 2.5 Expand the short rollback reference `8227241` to
  `82272414c94583e2689bc93735c0978a67e67132`
- [x] 2.6 Label every value as commit or artifact; never as both

## Phase 3 — Document

- [x] 3.1 Raise `current-pending-tasks.md` to version 6.0 with the measured 60-issue cohort
- [x] 3.2 Record 34/60 Done (56.7%) and mark the 18/24 (75%) figure superseded
- [x] 3.3 Retract the padded digest `88ca4f39` + zeros explicitly in writing
- [x] 3.4 Retract the `pg_advisory_lock` claim and record
  `LOCK TABLE agent_task_queue IN SHARE MODE`
- [x] 3.5 Record the root cause as an unguarded restart re-exposing the binary already referenced by
  `ExecStart`, not a binary swap
- [x] 3.6 Record the verified supervision facts for `multica-daemon-orq2-credential.service` and
  restore the combined five-signal daemon gate (ActiveState + NRestarts + ExecMainStartTimestamp +
  cgroup membership + health 19514)
- [x] 3.6b Retract, in writing, the first commit's false claim that the daemon is not systemd-managed
- [x] 3.6c Correct the ORQ-37 blocker to IAM Secrets Manager/SSM AccessDenied, absent SMA:2773 and the
  pending scoped `umask 077` cutover, retracting the ORQ-34-dispatch framing
- [x] 3.6d Preserve the `16:57:17Z` 60-issue snapshot as historical and add the measured `17:13:40Z`
  current state, including ORQ-23 now `done` and new cards ORQ-71/ORQ-72
- [x] 3.7 Record the endpoint topology by host: canonical URL served by peer `orq1`;
  `127.0.0.1:13100` does not listen on ORQ2; `127.0.0.1:18080` is ORQ1's backend via the tunnel unit;
  only `19514/health` is ORQ2-local
- [x] 3.8 Record ORQ-64 content-free, with remediation preconditions
- [x] 3.9 Preserve ORQ-42, ORQ-43, ORQ-44 and ORQ-64 as human-blocked in the pending table
- [x] 3.10 Record ORQ-68 pending deploy and the ORQ-66 cancelled follow-on
- [x] 3.11 Mark the ORQ-58 image digests reported-not-measured with the reason
- [x] 3.12 Preserve the version 5.0 table, owner decisions and narrative verbatim in Appendix A
- [x] 3.13 Record proposed status transitions as proposals only, applying none
- [x] 3.14 Confirm the ORQ-48 recorder ledger is untouched

## Phase 4 — Validation

- [x] 4.1 `openspec validate --all --strict --no-interactive` passes with the new change included
- [x] 4.2 `git diff --check` clean; no trailing whitespace and no Markdown hard breaks introduced
- [x] 4.3 Secret-pattern scan over the diff returns no credential value
- [x] 4.4 Confirm the diff touches only `openspec/**` and the single control evidence file
- [x] 4.5 Commit, push, and report the exact remote SHA from `git ls-remote`
