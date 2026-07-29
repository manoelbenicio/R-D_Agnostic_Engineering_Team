# Design — ORQ-67 Live Control Reconciliation

## Problem statement

The failure mode this change addresses is not a missing document. It is a **validated document that
is false**. Commit `ea501879df4d0b12700734e018b0457ce102b1f5` passed
`openspec validate --all --strict` with 5/5 items green while recording a zero-padded artifact
digest and a cohort that had already moved. Structural validation checks that a proposal has a
`## Why`, that tasks exist, and that requirements carry scenarios. It cannot check whether a hash
was measured or invented.

So the design goal is to move truth enforcement out of the validator and into the document's own
shape: make provenance a required field, so that an unmeasured claim is visibly incomplete rather
than invisibly wrong.

## Design decisions

### D1 — Provenance is a required field, not a footnote

Every material number in the control table is accompanied by the command that produced it and the
UTC instant it was produced. Where a value could not be produced locally, the document says so and
names the missing capability. This turns "I could not verify this" from a silent omission into a
visible, reviewable statement.

The concrete consequence in this revision: the two ORQ-58 container image digests are recorded as
**GTL-reported and unmeasured**, because `docker` and `podman` both return `command not found` on
this host. A prior candidate presented equivalent values without qualification.

### D2 — The cohort is the workspace, and the counts must sum to the API total

Cohort selection was the mechanism by which real work disappeared from control documentation.
Filtering to `project_id = 4b0ef49b-...` yields 46 cards and reads as complete, but it drops 14
live cards — 13 with a null `project_id` and ORQ-45 under a different project — including three of
the four human blockers.

The arithmetic check is therefore part of the requirement: per-status counts MUST sum to the
API-reported `total`. At the historical `16:57:17Z` snapshot 34 + 14 + 3 + 5 + 1 + 1 + 2 = 60 against a
reported `total = 60`; at the `17:13:40Z` current state 35 + 14 + 4 + 5 + 1 + 1 + 2 = 62 against a
reported `total = 62`, both with `has_more = false`. A subset can no longer pass as the whole, and each
snapshot carries its own instant rather than being overwritten.

### D3 — Digests are measured, never reconstructed

The rejected commit's `88ca4f39` followed by 56 zeros is the clearest possible illustration: it is
a well-formed 64-character string that is not a hash of anything. The requirement is that a digest
comes from executing a hashing command against a path, and that the path, size and mode are
recorded next to it so the claim is falsifiable by re-running one command.

Retraction is explicit. The padded value is named and withdrawn in the document body, because a
reader comparing revisions must be able to see that the old value was wrong rather than merely
absent.

### D4 — Corrections that contradict prior docs are recorded, not quietly dropped

Measurements in this run contradict standing control text. Each is written up rather than omitted.

- **Supervision — and a correction to this change's own first commit.** The daemon **is**
  systemd-managed. The canonical unit is `multica-daemon-orq2-credential.service` under
  `systemctl --user`: `LoadState=loaded`, `ActiveState=active`, `SubState=running`, `Type=simple`,
  `MainPID=1291834`, `NRestarts=0`, `ExecMainStartTimestamp=Wed 2026-07-29 15:17:45 UTC`, and
  `/proc/1291834/cgroup` names that exact unit.

  The first commit of this change (`e3ffb4cd22c8d75472ae5077e5c5ca67a4cc8b4c`) claimed the opposite.
  That claim was false. It came from querying `multica-daemon` in the **system** scope, which returns
  `LoadState=not-found`; systemd then reports `ActiveState=inactive` and `MainPID=0` as the defaults
  of a unit that does not exist, and those defaults were misread as measurements about the daemon. The
  `--foreground` flag was compounded into the error, when in fact it is the required form for a
  `Type=simple` unit and is therefore evidence *of* supervision.

  This is a sharper instance of the same failure mode as the padded hash: a well-formed output that
  measures nothing. The padded hash was a string that looked like a digest; `ActiveState=inactive` on
  a missing unit is a field that looks like a state. Both pass a shape check. The countermeasure added
  to REQ-06 is to require `LoadState=loaded` before any other unit property counts as evidence, and to
  require cgroup cross-checking, which fails loudly when the queried unit is not the one running the
  process.

  The combined gate is restored accordingly: `ActiveState` plus `NRestarts` plus
  `ExecMainStartTimestamp` plus cgroup membership plus health on `19514`. `NRestarts=0` alone remains
  worthless, for the reason the first commit stumbled over — it is also `0` on a unit that never ran.

- **Topology.** `tailscale serve status` on this host returns `No serve config`, and this host is
  tailnet node `orq2`. The canonical URL answers 200 but is served by peer `orq1`. `127.0.0.1:13100`
  refuses connections here. Further, `127.0.0.1:18080` is **also** ORQ1: the user unit
  `multica-orq1-backend-tunnel.service` runs `ssh -NT -L 18080:127.0.0.1:18080 orq1`, so the `readyz`
  200 is ORQ1's backend answering through a local forward. Only `19514/health` is ORQ2-local. This
  matters operationally because a `readyz` failure is ambiguous between a backend fault and a broken
  forward, and REQ-11 now requires the tunnel unit to be checked before attribution.

- **ORQ-37's blocker.** Card metadata leads with "ORQ-34 overlaps ORQ-44", which reads as a dispatch
  decision awaiting an owner. The operative blocker is infrastructural: the runtime IAM role receives
  **AccessDenied** on Secrets Manager and SSM, the expected secret **SMA:2773 is absent**, and the
  scoped `umask 077` cutover is pending. No dispatch ruling unblocks an AccessDenied or materialises a
  secret that does not exist, so the earlier framing is retracted.

### D5 — Artifact presence is not eligibility

Measuring which credential slots hold a provider artifact gives a superset of the daemon allowlist.
The AGY and Codex measurements happen to coincide exactly with the reported allowlists — AGY
`{162, 163, 168, 169}`, Codex `{152, 170}` — but Kiro does not: eight slots hold Kiro state while
the allowlist is `{139, 140, 143, 149}`.

Reporting the measured set as "the allowlist" would have been a plausible-looking error. The design
records the two as separate facts and states which is which, and notes the one place they interact:
two persisted antigravity assignments still point at `slot-145`, which holds no AGY token.

### D6 — Observation must not mutate

The reconciliation reads the board and proposes transitions in a table for GTL approval. It applies
none of them. This is what allows it to report contradictions honestly: because it is not
responsible for resolving them, it has no incentive to smooth them over.

Three real contradictions are recorded and left open: ORQ-42 is `blocked` while its metadata records
a re-review PASS and says integration is unblocked; ORQ-58 is `done` while still carrying a
`blocked_reason` and `pipeline_status=candidate_built_waiting_queue_zero`; ORQ-66's stored precondition
says slot-170 has no `codex/auth.json`, but that file is now present and non-empty.

### D7 — Superseded history is preserved verbatim

Version 5.0's pending table, owner-decision list and reconciliation narrative are reproduced in
Appendix A, labelled superseded and explicitly excluded from being quoted as current state. The
executions, task IDs and evidence digests recorded there happened; deleting them to make room for a
newer cohort would destroy audit trail to save space.

## Lineage

The revision is based on the accepted ORQ-62 content lineage
`7618599f29d43e964a485ab12a9932a9fd037e1f`, which is where
`.deploy-control/p0/evidence/current-pending-tasks.md` lives. That commit is **not** an ancestor of
`origin/main`; the file does not exist on `main`. Basing on `main` would have silently created the
file from nothing and lost version 5.0 and its history — the specific outcome the GTL instruction
"without losing current docs" guards against.

The rejected candidates `ea501879df4d0b12700734e018b0457ce102b1f5` and
`e431f0ae156114bb4b32329d77457264e6e95c8d` are not in this lineage and are not amended or reused.
Their OpenSpec change directory `live-recovery-reconciliation` is absent from this tree, which is
why the change introduced here uses the distinct identifier `orq67-live-control-reconciliation`.

## Verification approach

| Claim class | How it is verified | Failure handling |
|---|---|---|
| Board counts | `multica issue list --limit 500 --output json`, counts summed against API `total` | drift between measurements is recorded, not hidden |
| Artifact digest | `sha256sum` against the resolved `/proc/<pid>/exe` path | no padding, no truncation; prior padded value retracted |
| Commit SHAs | `git rev-parse --verify <sha>^{commit}` for each cited value | unresolvable SHA would be reported as unverified |
| Endpoints | `curl -o /dev/null -w '%{http_code}'` per endpoint | HTTP 000 recorded as connection refused |
| Supervision | `systemctl --user show multica-daemon-orq2-credential.service` for `LoadState`/`ActiveState`/`SubState`/`Type`/`MainPID`/`NRestarts`/`ExecMainStartTimestamp`, cross-checked against `/proc/<pid>/cgroup` | `LoadState=not-found` is reported as unit-absent-in-scope, never as daemon-unmanaged |
| Loopback attribution | `systemctl --user show multica-orq1-backend-tunnel.service` and its `ExecStart` forward spec | a forwarded port is attributed to the remote host that answers |
| Slot inventory | artifact presence and non-zero size only | no credential content read; presence distinguished from allowlist |
| Image digests | not measurable — no container runtime on host | marked reported-not-measured with the reason |

## Risks

- **Snapshot staleness by design.** The board moved twice during this run. Mitigation is the stated
  timestamp and the recorded drift, not a claim of stability.
- **Slot allowlist is inferred, not read from daemon memory.** The daemon exposes no allowlist
  endpoint. Mitigation is to label the measurement as artifact presence plus the persisted affinity
  ledger, and to state the operative allowlist separately.
- **Unresolved contradictions remain open.** ORQ-42, ORQ-58 and ORQ-66 carry metadata that
  disagrees with their status. Leaving them open is deliberate; each is owned by another card.
