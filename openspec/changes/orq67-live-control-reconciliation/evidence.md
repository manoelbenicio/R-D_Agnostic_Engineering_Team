# Evidence — ORQ-67 Live Control Reconciliation

All measurements below were taken on host `orq2` on 2026-07-29 between `16:49Z` and `16:58Z` by
Opus48-A. Every claim in `proposal.md`, `design.md` and
`.deploy-control/p0/evidence/current-pending-tasks.md` traces to an entry here. No credential value,
fragment, length or storage location is recorded.

## E1 — Host identity

```
$ hostname
ip-172-31-30-9.sa-east-1.compute.internal

$ tailscale status --self --peers=false
100.110.178.47  orq2  userid:7650130887043022  linux  -

$ tailscale serve status
No serve config
```

This host is tailnet node `orq2` and publishes no `tailscale serve` mapping. The canonical URL is
therefore served by the peer `orq1`.

## E2 — Board snapshot at 2026-07-29T16:57:17Z (historical, preserved unedited; see E11b for current state)

```
$ date -u +%Y-%m-%dT%H:%M:%SZ
2026-07-29T16:57:17Z

$ multica issue list --limit 500 --output json
api_total 60   returned 60   has_more False
```

Whole-workspace status counts at `2026-07-29T16:57:17Z`:

```
done 34, in_review 14, in_progress 3, blocked 5, todo 1, backlog 1, cancelled 2  => sum 60
```

Split by project:

```
project 4b0ef49b-df06-4e83-9a29-8a23b34821d4 (ORQ2) : 46
project abe3c461-c921-4a51-b91a-b08529429145        :  1   (ORQ-45)
project null                                        : 13
```

ORQ2 cards by status:

```
in_progress (1): ORQ-67
in_review  (12): ORQ-13, ORQ-14, ORQ-23, ORQ-35, ORQ-39, ORQ-50, ORQ-54, ORQ-60, ORQ-62,
                 ORQ-66, ORQ-68, ORQ-69
blocked     (2): ORQ-37, ORQ-64
todo        (1): ORQ-63
backlog     (1): ORQ-53
done       (29): ORQ-12, ORQ-15, ORQ-16, ORQ-17, ORQ-18, ORQ-19, ORQ-20, ORQ-21, ORQ-22,
                 ORQ-24, ORQ-25, ORQ-26, ORQ-30, ORQ-31, ORQ-33, ORQ-34, ORQ-36, ORQ-46,
                 ORQ-49, ORQ-51, ORQ-52, ORQ-55, ORQ-56, ORQ-57, ORQ-58, ORQ-59, ORQ-61,
                 ORQ-65, ORQ-70
```

Cards outside ORQ2 by status:

```
in_progress (2): ORQ-40, ORQ-41
in_review   (2): ORQ-47, ORQ-48
blocked     (3): ORQ-42, ORQ-43, ORQ-44
cancelled   (2): ORQ-32, ORQ-45
done        (5): ORQ-11, ORQ-27, ORQ-28, ORQ-29, ORQ-38
```

### E2.1 — Observed drift within the same run

An earlier snapshot in this same run, at `2026-07-29T16:51:58Z`, returned different counts:

```
2026-07-29T16:51:58Z : done 34, in_review 11, in_progress 6, blocked 5, todo 1, backlog 1, cancelled 2
2026-07-29T16:57:17Z : done 34, in_review 14, in_progress 3, blocked 5, todo 1, backlog 1, cancelled 2
```

`in_review` moved 11 to 14 and `in_progress` moved 6 to 3 in roughly five minutes. Both instants are
recorded. The document quotes the `16:57:17Z` snapshot and states its timestamp inline.

## E3 — Live daemon artifact digest

```
$ ps -o pid,ppid,cmd -p 1291834
    PID    PPID CMD
1291834 2387756 /home/ec2-user/.local/lib/multica/bin/multica-auth-credential-home-v1 daemon start
                --foreground --no-auto-update --server-url http://127.0.0.1:18080
                --daemon-id orq2-credential-runtime-v1 --device-name ORQ2 Credential Runtime

$ readlink -f /proc/1291834/exe
/home/ec2-user/.local/lib/multica/bin/multica-auth-credential-home-v1

$ ls -la /home/ec2-user/.local/lib/multica/bin/multica-auth-credential-home-v1
-rwx------. 1 ec2-user ec2-user 15053065 Jul 29 15:17 .../multica-auth-credential-home-v1

$ sha256sum /home/ec2-user/.local/lib/multica/bin/multica-auth-credential-home-v1
88ca4f397ef841edac091f15e77b2ba52754045a471acaa311c280827d2dd1f8
```

**Measured artifact SHA-256:** `88ca4f397ef841edac091f15e77b2ba52754045a471acaa311c280827d2dd1f8`
(64 hex characters, artifact digest, not a commit).

Retraction: commit `ea501879df4d0b12700734e018b0457ce102b1f5` recorded
`sha256:88ca4f3900000000000000000000000000000000000000000000000000000000`. That value is a
zero-padded fabrication and is withdrawn.

The pre-recovery artifact is retained alongside as
`multica-auth-credential-home-v1.pre-recovery-20260729T151745Z`, size 21333458 bytes — a different
size from the live artifact, confirming a different build.

## E4 — Endpoints

```
$ curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:19514/health      -> 200
$ curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:18080/readyz      -> 200
$ curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:13100/            -> 000  (refused)
$ curl -s -o /dev/null -w '%{http_code}' https://orq1.tail96e2c0.ts.net/    -> 200
```

Backend readiness body:

```
{"status":"ok","checks":{"db":"ok","migrations":"ok"}}
```

Daemon health body, secrets absent by construction:

```
{"status":"running","pid":1291834,"os":"linux","uptime":"1h39m31s",
 "daemon_id":"orq2-credential-runtime-v1","device_name":"ORQ2 Credential Runtime",
 "server_url":"http://127.0.0.1:18080",
 "cli_version":"credential-home-token-only-20260727T102815Z",
 "active_task_count":2,"agents":["codex","kiro","antigravity"], ...}
```

The daemon gate (`19514/health`) and backend readiness (`18080/readyz`) are two separate gates and
are recorded separately. `127.0.0.1:13100` does not listen on this host. Note that of these four,
**only `19514/health` is served by an ORQ2-local process** — see E5b for why `18080` is ORQ1's backend.

## E5 — Daemon supervision: the daemon IS systemd-managed by a user unit

### E5.1 — The correct unit, in the correct scope

```
$ systemctl --user show multica-daemon-orq2-credential.service \
    -p Id -p Description -p LoadState -p ActiveState -p SubState \
    -p Type -p MainPID -p NRestarts -p ExecMainStartTimestamp
Id=multica-daemon-orq2-credential.service
Description=Multica credential-isolated daemon on ORQ2
LoadState=loaded
ActiveState=active
SubState=running
Type=simple
MainPID=1291834
NRestarts=0
ExecMainStartTimestamp=Wed 2026-07-29 15:17:45 UTC

$ systemctl --user is-active multica-daemon-orq2-credential.service
active

$ cat /proc/1291834/cgroup
0::/user.slice/user-1000.slice/user@1000.service/app.slice/multica-daemon-orq2-credential.service
```

`ExecStart` for that unit is:

```
/home/ec2-user/.local/lib/multica/bin/multica-auth-credential-home-v1 daemon start --foreground
  --no-auto-update --server-url http://127.0.0.1:18080
  --daemon-id orq2-credential-runtime-v1 --device-name ORQ2 Credential Runtime
```

`--foreground` is the **required** form for a `Type=simple` unit: systemd supervises the main process
directly and needs it to stay in the foreground. The flag is evidence *of* correct supervision.

### E5.2 — Retraction of the false claim in commit `e3ffb4cd`

The first commit of this change asserted "process supervision is not systemd-unit managed". That is
**false and is withdrawn**. It rested on a query against the wrong unit in the wrong scope:

```
$ systemctl show multica-daemon -p Id -p LoadState -p ActiveState -p MainPID
Id=multica-daemon.service
LoadState=not-found          <-- the unit does not exist in this scope
ActiveState=inactive         <-- systemd default for a missing unit, NOT a measurement
MainPID=0                    <-- systemd default for a missing unit, NOT a measurement
```

`LoadState=not-found` is the field that invalidates the whole result, and it was not read. Every
subsequent conclusion drawn from that output — inactive, dead, unsupervised, `NRestarts=0` meaningless
— was an artefact of querying a nonexistent unit.

This is the same failure mode as the padded digest retracted in E3: a well-formed output that measures
nothing. REQ-06 now requires `LoadState=loaded` before any other unit property counts as evidence, and
requires the `MainPID` to be cross-checked against `/proc/<pid>/cgroup`, which would have caught this
immediately.

### E5.3 — The combined five-signal daemon gate, all measured

| # | Signal | Measured value | Verdict |
|---:|---|---|---|
| 1 | `ActiveState` / `SubState` | `active` / `running` | PASS |
| 2 | `NRestarts` | `0` | PASS |
| 3 | `ExecMainStartTimestamp` | `Wed 2026-07-29 15:17:45 UTC` | PASS — matches the 15:17 recovery relaunch |
| 4 | cgroup membership of `MainPID` | `/proc/1291834/cgroup` names the unit | PASS |
| 5 | daemon health | HTTP 200 on `127.0.0.1:19514/health`, `pid` 1291834 | PASS |

Signal 3 independently corroborates the recovery: the artifact mtime is `Jul 29 15:17` (E3) and the
unit's main process started at `15:17:45 UTC`, so the running process is the recovered artifact rather
than a survivor of the regression.

### E5.4 — Full user-unit inventory

```
$ systemctl --user list-units --all --plain --no-pager | grep -i multica
multica-daemon-orq2-credential.service   loaded active running Multica credential-isolated daemon on ORQ2
multica-orq1-backend-tunnel.service      loaded active running Multica ORQ2 to ORQ1 backend tunnel
```

Both relevant units are user units, which is why the system-scope query found nothing. The
system-scope units seen earlier (`orq2-agent-cache-lifecycle`, `reap-cred-slots`) are unrelated
maintenance timers and do not run the daemon.

## E5b — `127.0.0.1:18080` is ORQ1's backend, reached through an SSH forward

```
$ systemctl --user show multica-orq1-backend-tunnel.service \
    -p Id -p Description -p LoadState -p ActiveState -p SubState \
    -p MainPID -p NRestarts -p ExecMainStartTimestamp
Id=multica-orq1-backend-tunnel.service
Description=Multica ORQ2 to ORQ1 backend tunnel
LoadState=loaded
ActiveState=active
SubState=running
MainPID=3211411
NRestarts=0
ExecMainStartTimestamp=Sun 2026-07-26 23:47:47 UTC

ExecStart: /usr/bin/ssh -NT -o BatchMode=yes -o ExitOnForwardFailure=yes
           -o ServerAliveInterval=15 -o ServerAliveCountMax=3
           -L 18080:127.0.0.1:18080 orq1
```

Therefore the `readyz` 200 recorded in E4 is **ORQ1's backend answering through a local forward**, and
the daemon's own `server_url=http://127.0.0.1:18080` reaches ORQ1 by the same path. Only
`19514/health` is served by an ORQ2-local process.

Operational consequence: a `readyz` failure on ORQ2 is ambiguous between an ORQ1 backend fault and a
broken forward. The tunnel unit's `ActiveState` must be checked before attributing the result. This is
the same misattribution risk as calling `13100` an ORQ2 origin, and REQ-11 now forbids both.

## E6 — Credential slot inventory, presence only

Measured under `/home/ec2-user/.agent-cred-homes/slots` by testing for a present, non-empty
provider artifact. No file content was read.

```
SLOT       AGY   KIRO   CODEX
slot-139   -     yes    -
slot-140   -     yes    -
slot-143   -     yes    -
slot-145   -     yes    -
slot-149   -     yes    -
slot-150   -     -      -
slot-152   -     -      yes
slot-155   -     yes    -
slot-157   -     -      -
slot-159   -     -      -
slot-160   -     yes    -
slot-161   -     yes    -
slot-162   yes   -      -
slot-163   yes   -      -
slot-164   -     -      -
slot-165   -     -      -
slot-166   -     -      -
slot-167   -     -      -
slot-168   yes   -      -
slot-169   yes   -      -
slot-170   -     -      yes
slot-171   -     -      -
slot-172   -     -      -
```

Probe paths: AGY `<slot>/home/.gemini/antigravity-cli/antigravity-oauth-token`, Kiro
`<slot>/xdg-data/kiro-cli/*.sqlite3`, Codex `<slot>/codex/auth.json`.

- **AGY artifact present:** 162, 163, 168, 169 — coincides exactly with the reported AGY allowlist.
- **Codex artifact present:** 152, 170 — coincides exactly with the reported Codex allowlist.
- **Kiro artifact present:** 139, 140, 143, 145, 149, 155, 160, 161 — a **superset** of the
  operative Kiro allowlist 139, 140, 143, 149.

Presence is not eligibility. The two facts are reported separately in the control table.

`slot-170/codex/auth.json` is present and non-empty. The ORQ-66 stored precondition "slot-170 has no
`codex/auth.json`" is therefore stale, which is relevant to ORQ-63.

## E7 — Affinity ledger, keys only

```
$ python3 -c "json.load(open('.../multica-assignments.v1.json'))"
version 1, 13 assignments of the form "<agent_id>|<provider>" -> "slot-NNN"
```

The file contains slot names only and no credential values. Observed mappings by provider:

- antigravity: `slot-145` (x2), `slot-162`, `slot-163`, `slot-168`, `slot-169`
- kiro: `slot-139`, `slot-140` (x2), `slot-149`
- codex: `slot-152` (x3)

Two antigravity assignments point at `slot-145`, which holds no AGY token and is not AGY-allowlisted.
This corroborates the ORQ-66 precondition note and remains open. No assignment yet points at
`slot-170`, consistent with ORQ-63 being `todo`.

## E8 — ORQ-66 cancelled follow-on task

```
$ cd /home/ec2-user/multica_workspaces/20fce817-895d-447b-965a-49f5e279314a
64808d3d: workdir ABSENT
55cfb6b4: workdir PRESENT   (ORQ-48 corrected snapshot run)
92870148: workdir PRESENT   (ORQ-68 run)
e5f7f3d3: workdir PRESENT   (this ORQ-67 run)
```

Task `64808d3d` left no workdir under the daemon workspaces root, consistent with cancellation before
any checkout or delivery. ORQ-66 card metadata records `commit_sha`
`368a4ba3c5f2653abc3aa7aca6e991a9735ed7a6` on branch `agent/opus48-a/orq66-token-only`, which
therefore remains the delivered head.

## E9 — Commit SHA verification

```
$ git rev-parse --verify <sha>^{commit}
7618599f29d43e964a485ab12a9932a9fd037e1f  docs(deploy): restore ORQ-23 rollout and rollback runbook…
15626386da2725af8e8d4ac611754cffe359fe31  fix(server): preserve active tasks during issue review
368a4ba3c5f2653abc3aa7aca6e991a9735ed7a6  feat(daemon): wire frozen task identity to the physical…
da004ca570237b1872047cdf02973a2198f5bfc4  docs(orq-48): Wave 3 live recorder ledger correction per GTL review
112e8dada455b4e7a3400e63728e00e6e3a0aa27  merge commit — ORQ-58 release revision
82272414c94583e2689bc93735c0978a67e67132  short form 8227241 — ORQ-58 rollback revision
ea501879df4d0b12700734e018b0457ce102b1f5  rejected ORQ-67 candidate — do not reuse
e431f0ae156114bb4b32329d77457264e6e95c8d  superseded ORQ-67 candidate
```

All eight resolve in this repository. All are **commit** SHAs, not artifact digests.

### E9.1 — Lineage is not on main

```
$ git merge-base --is-ancestor 7618599f… HEAD(origin/main b6571299…)  -> NO
$ git cat-file -e origin/main:.deploy-control/p0/evidence/current-pending-tasks.md -> does not exist
$ git cat-file -e 7618599f:.deploy-control/p0/evidence/current-pending-tasks.md   -> exists
$ git branch -a --contains 7618599f… -> agent/agy-p0-a7/eb2c0b57
```

The control file exists only on the ORQ-62 lineage, not on `main`. Basing this revision on `main`
would have created the file from nothing and destroyed version 5.0 and its history. The branch was
therefore reset to `7618599f29d43e964a485ab12a9932a9fd037e1f` before editing.

```
$ ls openspec/changes | grep -i recovery
ABSENT
```

The rejected candidate's change directory `live-recovery-reconciliation` is not present in this tree,
so no rejected artifact is amended or reused.

## E10 — Container image digests could not be measured

```
$ docker ps
bash: docker: command not found
$ command -v podman
no podman
```

No container runtime exists on this host. The ORQ-58 image digests
`sha256:e14f5c35d0640ec4c955efd8d5bbbb6cf219a4d544faafc1fd0f028c5bafcf13` (live) and
`sha256:922b13862036d906a1ad5cde3e1615adad45393edb13896d2b146753b8384ab6` (rollback) are recorded
**as reported by the GTL and not independently measured**. Re-verification requires running
`docker image inspect` on the host that runs the images.

ORQ-58 card metadata independently confirms the release revision and the image **tags**:

```
decision: release 112e8dada455b4e7a3400e63728e00e6e3a0aa27;
          candidate multica-backend:orq58-canonical-112e8da-20260729;
          rollback  multica-backend:rollback-before-orq58-922b138-20260729
```

## E11 — Card metadata read at snapshot

Read via `multica issue list --output json`; reproduced because the control table cites it.

```
ORQ-66 in_review  branch=agent/opus48-a/orq66-token-only
                  commit_sha=368a4ba3c5f2653abc3aa7aca6e991a9735ed7a6
                  pipeline_status=waiting_review
                  blocked_reason=cutover preconditions: slot-170 has no codex/auth.json;
                                 2 persisted antigravity assignments point to non-allowlisted slot-145
ORQ-68 in_review  waiting_on=GTL/Kiro review of agent/codex-b/92870148;
                             PR creation requires authenticated gh
ORQ-58 done       decision=release 112e8dada455b4e7a3400e63728e00e6e3a0aa27; …
                  pipeline_status=candidate_built_waiting_queue_zero      <- stale vs done
                  blocked_reason=Approved wrapper failed closed …          <- stale vs done
ORQ-42 blocked    decision=Q-H/Q-I re-review PASS on fc77e89
                           (branch agent/opus48-a/orq42-secret-tools-clean);
                           clean-branch integration unblocked, no correction required
ORQ-37 blocked    blocked_reason=ORQ-34 overlaps ORQ-44; runtime role lacks Secrets Manager/SSM
                                 metadata access and SMA:2773 is absent; …
ORQ-54 in_review  waiting_on=GTL/Kiro re-review of 2aa8f19 …
ORQ-64 blocked    {}   (no metadata; incident detail lives in comments, content-free)
```

Three status/metadata contradictions are recorded and left open for their owning cards: ORQ-42
(`blocked` vs a recorded re-review PASS), ORQ-58 (`done` vs stale `blocked_reason`/`pipeline_status`),
and ORQ-66 (stored slot-170 precondition contradicted by E6).

### E11.1 — ORQ-37's blocker, correctly ordered

The stored `blocked_reason` leads with "ORQ-34 overlaps ORQ-44", which reads as a dispatch decision
awaiting an owner ruling. That ordering is misleading and the earlier framing "owner decisions on
ORQ-34 dispatch" is **retracted**. The operative blockers are infrastructural:

1. The runtime IAM role receives **AccessDenied** on Secrets Manager and SSM.
2. The expected secret **SMA:2773 is absent**.
3. The scoped **`umask 077` cutover is pending**.

No dispatch ruling can resolve an IAM AccessDenied or bring into existence a secret that does not
exist, so items 1 and 2 are hard prerequisites and item 3 is the remaining implementation step. The
ORQ-34/ORQ-44 overlap is a scope note, not the blocker.

## E11b — Post-snapshot board delta

```
$ date -u +%Y-%m-%dT%H:%M:%SZ
2026-07-29T17:13:40Z

$ multica issue list --limit 500 --output json
api_total 62   has_more False
WORKSPACE: done 35, in_review 14, in_progress 4, blocked 5, todo 1, backlog 1, cancelled 2 => sum 62

project split: 4b0ef49b (ORQ2) 48 | abe3c461 1 | null 13
ORQ2   (48): done 30, in_review 12, in_progress 2, blocked 2, todo 1, backlog 1
outside(14): done  5, in_review  2, in_progress 2, blocked 3, cancelled 2
```

Diff against the `16:57:17Z` snapshot in E2, which is preserved unedited as historical:

- `ORQ-23` moved `in_review` to **`done`**; workspace `done` 34 to 35.
- Two new cards, both in project ORQ2: **ORQ-71** `in_review` ("P0 ORQ2 Root Disk Saturation — safe
  terminal-artifact reclamation") and **ORQ-72** `in_progress` ("P0 ORQ2 Emergency Disk Reclaim — five
  exact terminal pnpm stores"); total 60 to 62 and `in_progress` 3 to 4.
- Completion moves from 34/60 (56.7%) to **35/62 (56.5%)**.

Both new cards are consistent with the measured environment: `df -h /` reported the root filesystem at
98% used with roughly 1.7 GB available at the start of this run.

Daemon at this instant: `19514/health` returned `pid 1291834`, `uptime 1h55m38s`,
`active_task_count 3`.

## E12 — Scope containment

The diff for this revision touches only:

- `openspec/changes/orq67-live-control-reconciliation/**` — new change
- `.deploy-control/p0/evidence/current-pending-tasks.md` — version 5.0 to 6.0

No product code, no build output, no deployment manifest, and no ORQ-48 recorder ledger file is
modified. No Kanban status was changed on any card other than ORQ-67 itself, and no agent was
dispatched.
