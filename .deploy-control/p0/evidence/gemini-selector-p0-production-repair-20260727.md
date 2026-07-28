# P0 Gemini model selector production repair

- Executor/owner: Kiro, exclusive writer for this bounded P0 lane
- UTC execution window: 2026-07-27 17:55–18:18
- Production topology: Multica backend/frontend/Postgres on ORQ1; credential-isolated daemon on ORQ2
- Result: **PASS** for the authenticated selector/API path for Antigravity model `gemini-3.6-flash-high`

## Scope and secret safety

The ORQ-17 bootstrap-owner credential was consumed only by `asm-exec` using the metadata-confirmed full Secrets Manager ARN and `AWSCURRENT`. No `GetSecretValue`/`BatchGetSecretValue` call, SMA loopback access, plaintext credential output, token/cookie output, container environment dump, or persistent auth artifact occurred. Cookies, CSRF data, and API bodies were held in mode-0600 child-only temporary directories and removed by traps.

The worktree already contained extensive unrelated modified/untracked files. This repair did not edit product source and did not touch or revert those files.

## Static chain

1. The selected online runtime is the source of the model dropdown catalog.
2. `POST /api/runtimes/{runtimeId}/models` creates a pending request.
3. ORQ2 receives it through heartbeat, calls `agent.ListModelsWithHome`, and reports the result.
4. For Antigravity, `resolveCredentialModelDiscoveryHomes` validates every allowlisted account home before `agy models` runs.
5. The UI polls the request, caches by runtime, groups returned model rows, and offers each returned ID. It does not have a global Gemini filter.

Native Gemini runtime support also exists, but it is not the acceptance path here. The target `gemini-3.6-flash-high` is an Antigravity catalog model with reasoning tier embedded in the model ID.

## Before state and root cause

- ORQ1 services healthy: backend `18080=200`, frontend `13100=200`, Postgres healthy.
- Eight runtime rows; six online rows from `orq2-credential-runtime-v1` (Antigravity, Codex, Kiro across two workspaces).
- The public ORQ2 Antigravity runtime was online and heartbeating.
- Authenticated login, `/api/me`, workspace list, and runtime list returned HTTP 200.
- Kiro async discovery completed with 19 models and Codex with 11, proving the generic endpoint/heartbeat/report/poll chain.
- Antigravity async discovery failed with zero models.
- Exact error: credential isolation required the Antigravity artifact in `slot-150`, but the OAuth artifact was absent.
- Effective ORQ2 AGY allowlist: `141,145,146,150`.
- Metadata-only validation: slots 141/145/146 each had a non-empty regular mode-0600 OAuth artifact; slot 150 was missing/invalid.
- Because discovery validates the whole allowlist before trying a home, invalid slot 150 prevented all three valid homes from reaching `agy models`.

A separate native Gemini observation was not causal: ORQ2 intentionally has `MULTICA_GEMINI_PATH=/run/multica-disabled/gemini`, the target is absent, and no native `gemini` executable was found. No fake native runtime/profile was created.

## Safety gates and correction

Two consecutive pre-restart DB reads, three seconds apart, both returned:

```text
queued=0 dispatched=0 running=0 waiting_local_directory=0 total=0
```

The original user unit file was not edited. One non-secret late systemd user drop-in was atomically created:

```text
/home/ec2-user/.config/systemd/user/multica-daemon-orq2-credential.service.d/90-gemini-selector.conf
SHA-256 214e5ce19cd46173c3b106baa1bb98fa601f7206783695999a339ae4279a7a04
```

Effective change:

```text
MULTICA_CREDENTIAL_SLOT_ALLOWLIST_ANTIGRAVITY=141,145,146,150
-> MULTICA_CREDENTIAL_SLOT_ALLOWLIST_ANTIGRAVITY=141,145,146
```

Only `multica-daemon-orq2-credential.service` was restarted. PID changed `3417665 -> 3993320`; post-restart health was 200, `ActiveState=active`, `SubState=running`, `NRestarts=0`. The same six runtime IDs were re-registered and remained online.

Duplicate prevention is inherent: no DB/runtime/profile row was inserted manually; daemon registration used the existing built-in `(workspace_id, daemon_id, provider)` upsert identity. `runtime_profile` remained empty.

## Authenticated production proof

Using workspace `orq2-dev` and runtime `405b751d-e831-4da3-8fd5-bb3744c49334`:

```text
AUTH_LOGIN=200
GET /api/runtimes/=200
Antigravity runtime: online, public
POST /api/runtimes/{id}/models=200
terminal request status=completed
supported=true
catalog count=11
gemini-3.6-flash-high count=1
```

Returned Gemini rows:

- `gemini-3.6-flash-high`
- `gemini-3.6-flash-medium`
- `gemini-3.6-flash-low`
- `gemini-3.5-flash-high`
- `gemini-3.5-flash-medium`
- `gemini-3.5-flash-low`
- `gemini-3.1-pro-high`
- `gemini-3.1-pro-low`

Authenticated agent read also proved both `Agy-P0-A7` and `Agy-P0-A8` persist `model=gemini-3.6-flash-high`, use the same public Antigravity runtime, are not archived, and keep `thinking_level` empty as required for Antigravity's model-embedded tier.

This is sufficient UI-path proof under the owner ruling that manual browser validation is not required when the authenticated selector API is proven. Static UI inspection confirms the dropdown presents returned runtime models without removing Gemini IDs.

## Post-state and rollback

Final DB gate:

```text
queued=0 dispatched=0 running=0 waiting_local_directory=0 total=0
six ORQ2 runtime rows online, heartbeat age about 14 seconds
runtime_profile total=0, Gemini profiles=0
```

Private rollback set on ORQ2:

```text
/home/ec2-user/.local/state/gemini-selector-20260727T181535Z/
manifest.txt  mode 0600
rollback.sh   mode 0700
```

Rollback removes only `90-gemini-selector.conf`, runs `systemctl --user daemon-reload`, restarts only the daemon, and requires service active plus daemon health 200. It restores the prior effective allowlist from the untouched original unit. Rollback was not needed.

## Residual notes

- `slot-150` remains an empty/uncredentialed isolation skeleton and is no longer in the active AGY allowlist. No token was copied, moved, linked, or read.
- Restoring four-account AGY capacity requires a distinct owner-controlled OAuth login that materializes a valid slot-150 artifact, followed by reverting this drop-in. It is not required for the repaired three-account selector path.
- Antigravity model discovery before the fix produced an explicit failed request; after the fix it completed with the expected 11-model catalog.

## Read-only closeout verification — 2026-07-28

Closeout was performed without authentication secrets, model/task retry, daemon restart, service/configuration mutation, OAuth access, or allowlist change. PostgreSQL proof ran inside `BEGIN READ ONLY` at `2026-07-28T13:14:37.305Z`. Result payload contents were deliberately not selected; only presence, JSON type, and byte length were measured.

### Retry terminal proof

| Agent | Retry task | Status | Started UTC | Completed UTC | Result proof | Error | Failure reason |
|---|---|---|---|---|---|---|---|
| Agy-P0-A7 | `55a6a702-2d7b-4da3-a3a8-70211c279b21` | `completed` | `2026-07-27T18:23:33.231Z` | `2026-07-27T18:24:50.937Z` | non-null JSON object, 1913 bytes | `NULL` | `NULL` |
| Agy-P0-A8 | `bf71c329-6e5c-4a71-9915-51463e0af919` | `completed` | `2026-07-27T18:23:33.262Z` | `2026-07-27T18:24:41.203Z` | non-null JSON object, 2608 bytes | `NULL` | `NULL` |

Cardinality gate: exactly two requested rows, `completed=2`, `with_result=2`, `with_error=0`, `with_failure_reason=0`.

### Agent, model, runtime, and queue proof

- `Agy-P0-A7`: `status=idle`, `visibility=workspace`, `archived=false`, `active_tasks=0`, runtime `405b751d-e831-4da3-8fd5-bb3744c49334`, model `gemini-3.6-flash-high`, empty `thinking_level`.
- `Agy-P0-A8`: same idle/visible/unarchived/zero-active state, same runtime and target model, empty `thinking_level`.
- Cardinality: two requested agent rows, `idle=2`, `visible=2`, target model persisted on both, active tasks zero.
- Public runtime `405b751d-e831-4da3-8fd5-bb3744c49334`: `provider=antigravity`, `status=online`, `visibility=public`, daemon `orq2-credential-runtime-v1`, DB heartbeat age 5 seconds at the snapshot.
- DB model anchor: four active agents persist `gemini-3.6-flash-high`; A7/A8 both bind the same public online Antigravity runtime.
- Global active queue: `queued=0`, `dispatched=0`, `running=0`, `waiting_local_directory=0`, total zero.

The runtime model-list lifecycle is held in the server's short-lived model-list store rather than a PostgreSQL catalog table. Therefore the durable DB closeout proof is the persisted target model on A7/A8 plus the online public runtime binding; the authenticated catalog proof earlier in this canonical record remains the authoritative selector proof: status `completed`, `supported=true`, catalog count 11, and exactly one `gemini-3.6-flash-high`. No fresh catalog request was issued because closeout explicitly prohibited retry/secret use.

### Daemon and configuration preservation

Read-only ORQ2 inspection found the original repaired process unchanged: PID `3993320`, `ActiveState=active`, `SubState=running`, `NRestarts=0`. The effective non-secret allowlist remains `MULTICA_CREDENTIAL_SLOT_ALLOWLIST_ANTIGRAVITY=141,145,146`. The daemon's actual loopback health endpoint `127.0.0.1:19514/health` returned HTTP 200. Drop-in, rollback set, and slot-150 state were not modified.

Kanban closeout uses existing ORQ-16 (`fb025de9-cbd2-4dd1-86f7-6c720ef2ddd9`) with first-token `/note` and idempotency marker `AGY-CLOSEOUT-V1:55a6a702:bf71c329`; marker precheck was zero and ORQ-16 task count was one.
