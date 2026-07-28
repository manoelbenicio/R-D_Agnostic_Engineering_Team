# AGY slot-150 lifecycle containment and human OAuth window

Capture date: 2026-07-27 UTC
Host: ORQ2 (`ip-172-31-30-9.sa-east-1.compute.internal`)
Status: OAuth and task retries are blocked pending owner-present interactive login.

## Cache lifecycle containment

`orq2-agent-cache-lifecycle.timer` remains enabled and active. Its next measured run is `2026-07-28 05:15:51 UTC`.

The associated service has:

- `ReadWritePaths=/tmp /home/ec2-user/.cache /run/orq2-agent-cache-lifecycle`
- `InaccessiblePaths=/home/ec2-user/workspace /home/ec2-user/.agent-cred-homes`
- `ProtectHome=read-only`

The script accepts only `/tmp` and `/home/ec2-user/.cache` as roots and rejects protected tokens including `.agent-cred-homes`, `slots`, `logins`, backups, registries, workspaces and evidence trees.

A dry-run `--audit`, using a lock under `/tmp` because the systemd `RuntimeDirectory` does not exist outside service activation, reported 71 discovered paths and 17 eligible paths. A strict grep found zero references to `/home/ec2-user/.agent-cred-homes`, `registry.json`, or `slot-NNN`; reclaimed bytes were zero. Result: `LIFECYCLE_CREDENTIAL_EXCLUSION_PASS`. The cache lifecycle timer was not disabled or changed.

A distinct system timer, `reap-cred-slots.timer`, invokes `/home/ec2-user/.local/bin/reap-cred-slots.sh` hourly with a 24-hour TTL for inactive login slots. It is not called by the cache lifecycle service. Historical journal metadata records deletion of directories named `slot-151` at `2026-07-26 16:00:47 UTC` and `slot-150` at `2026-07-27 12:11:00 UTC`. This does not establish that the current AGY OAuth artifact was ever materialized in either directory. No timer or service was changed during this audit.

## Slot and terminal audit

Current metadata only, with no credential file read:

- `slot-150`: physical directory, owner `ec2-user:ec2-user`, mode `0700`, current skeleton mtime approximately `2026-07-27 17:58:58 UTC`.
- Physical mode-`0700` children include `home`, `xdg-config`, and `xdg-data`.
- Required AGY artifact `slot-150/home/.gemini/antigravity-cli/antigravity-oauth-token`: absent.
- `slot-151`: absent.
- Filtered structural checks found no `slot-150` or `slot-151` entry in `registry.json` or `multica-assignments.v1.json`.

ORQ1 task metadata contains exactly three terminal failures that name `slot-150`; all failed before `started_at`:

| Issue | Agent | Task | Status |
|---|---|---|---|
| ORQ-37 | Opus-46-B | `6b0928fe-0c69-4a12-a1d6-ba4e070f5b39` | failed |
| ORQ-38 | Agy-P0-A7 | `234e51bc-25ef-49e5-893c-1b40c585d486` | failed |
| ORQ-26 | Agy-P0-A8 | `1ee00841-446e-4e97-ba29-5df701009cb4` | failed |

Only the A7 and A8 rows are in the authorized one-retry scope.

## Exact owner-present human OAuth window

AGY 1.1.7 has no `login` or `auth` subcommand. Launching the interactive CLI initiates the human/browser flow. The prepared private launcher is:

`/home/ec2-user/.local/state/agy-slot-150-human-login.sh`

It is owned by `ec2-user`, mode `0700`, syntax-checked, and has not been executed. Its final process environment uses exactly:

```sh
HOME=/home/ec2-user/.agent-cred-homes/slots/slot-150/home \
XDG_CONFIG_HOME=/home/ec2-user/.agent-cred-homes/slots/slot-150/xdg-config \
XDG_DATA_HOME=/home/ec2-user/.agent-cred-homes/slots/slot-150/xdg-data \
/home/ec2-user/.local/bin/agy
```

No `XDG_STATE_HOME` is set. The launcher has no references to slots 141, 145 or 146 and performs no credential copy or remapping. It refuses to run if the slot paths are symlinks, have unexpected ownership/mode, or the AGY artifact already exists. It must be executed only with the owner present.

After the owner completes OAuth and exits AGY, validation is limited to content-free metadata:

```sh
artifact=/home/ec2-user/.agent-cred-homes/slots/slot-150/home/.gemini/antigravity-cli/antigravity-oauth-token
test -s "$artifact"
test "$(stat -c '%a' "$artifact")" = 600
```

No token content, identity, cookie, fingerprint, or account value may be printed.

## Guarded one-retry procedure

The rerun contract accepts `POST /api/issues/{issue_id}/rerun` with a source `task_id`. It targets the source task's agent even if the issue is currently unassigned and uses a fresh session. Therefore A7/A8 can be retried without reassigning ORQ-38/ORQ-26 or undoing the ownership rollback.

Immediately before mutation:

1. Require successful artifact metadata checks above.
2. Confirm no active owner/file lock conflict for each issue.
3. Confirm no pending/running task for each A7/A8 issue-agent pair.
4. Rerun source task `234e51bc-25ef-49e5-893c-1b40c585d486` exactly once.
5. Rerun source task `1ee00841-446e-4e97-ba29-5df701009cb4` exactly once.
6. Verify admission and non-null `started_at`; monitor each to one terminal result. Never issue a second retry.

Kanban auth-failure/ETA note and final retry outcome remain pending. Any note must use `/note`, avoid task triggers, preserve current ownership, and must not include stale bundle SHA `4d922b25`.

## Kanban worklog

A single authenticated human-only note was added to existing operational card ORQ-16 (`Resolver slots AGY sem sessão ativa`), rather than modifying ownership-conflicted ORQ-26/38:

- comment ID: `67a11d36-1101-4d22-8719-1e27865b3b39`;
- first token: `/note`;
- trigger preview: `agents=[]`;
- ORQ-16 task count: unchanged `1 → 1`;
- ETA recorded: each one-time AGY source-task rerun within 15 minutes after content-free `test -s` and mode `0600` confirmation;
- no assignment, issue status, date, token, stale bundle, or duplicate-agent issue was posted.

## Superseding reconciliation — Gemini selector drop-in

Owner steering superseded the earlier immediate slot-150 OAuth gate. The live drop-in `90-gemini-selector.conf` sets the effective Antigravity allowlist to `141,145,146`; slot-150 remains an uncredentialed skeleton outside the active selector and is future expansion only. No OAuth was initiated.

Independent verification:

- daemon active/running since `2026-07-27 18:15:35 UTC`, `NRestarts=0`;
- websocket connected with six pollers;
- authenticated backend: three ORQ2 runtimes online in each of the two synced workspaces, six total;
- pre-retry active queue: `queued=0 dispatched=0 running=0 waiting_local_directory=0`;
- fresh async model request `75caab7c10d5884e792ed3619e131a08`: `completed`, `supported=true`, exactly 11 models, includes `gemini-3.6-flash-high`;
- independent source evidence `.deploy-control/p0/evidence/gemini-selector-p0-production-repair-20260727.md`: SHA-256 `f679b7b97cbe68f02dedea0313307c32e12936153903adb36da00bc7319a8f32`.

A fresh owner session was created with fixed public email `dataops.cloud.mbf@gmail.com` and password resolved only inside `asm-exec` from the approved full ARN. Login and `/api/me` returned 200; no password, cookie or CSRF value was printed.

Exactly one source-specific retry was issued per AGY task, with no HTTP retry mechanism and no issue reassignment:

| Agent | Source task | New task | Started | Completed | Result |
|---|---|---|---|---|---|
| Agy-P0-A7 | `234e51bc-25ef-49e5-893c-1b40c585d486` | `55a6a702-2d7b-4da3-a3a8-70211c279b21` | `2026-07-27 18:23:33.231156 UTC` | `2026-07-27 18:24:50.937980 UTC` | `completed` |
| Agy-P0-A8 | `1ee00841-446e-4e97-ba29-5df701009cb4` | `bf71c329-6e5c-4a71-9915-51463e0af919` | `2026-07-27 18:23:33.262881 UTC` | `2026-07-27 18:24:41.203294 UTC` | `completed` |

Final state: both agents idle, each issue-agent pair has exactly two task rows (one historical failed, one completed), active queue zero, issue assignee metadata unchanged. No second retry was issued.

Authenticated human-only Kanban publications:

- ORQ-25 GAP A/GAP B exact prepared note marker `AGENT-UPDATE-409-NOTE-V1:4f87b90:097206d2`: comment `0530b739-2ec9-4fa2-817b-e7c0d90b0704`, preview agents zero, marker count one, task count unchanged `3→3`;
- ORQ-16 Gemini/AGY reconciliation: comment `3e467e25-0e63-4027-910b-52c8905d15e6`, preview agents zero, task count unchanged `1→1`.
