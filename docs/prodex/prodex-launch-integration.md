# prodex Launch Integration

Status: F0-PREP SPEC. LIVE launch is F0-GATED. Documentation only; no deploy.

Scope: concrete Multica Go L4 launch/integration plan for running pinned
prodex AS-IS in place of raw `codex` for approved F0 sessions.

Pinned prodex target:

```text
package: @christiandoxa/prodex
version: 0.246.0
requested alias: v0.246.0
commit: 7750da9b6a5c91a6d429e18e6a4d422cab4bc144
license: Apache-2.0
```

`docs/prodex/prodex-pin-integrity.md` notes that the official tag observed at
the audited commit is `0.246.0`; `v0.246.0` remains `a validar` unless upstream
release evidence confirms that alias.

## Source Basis

Official prodex docs used for prodex behavior:

- official `docs/architecture.md`: `prodex run`, runtime launch/proxy flow,
  profile/session/shared Codex filesystem areas, runtime log and redaction
  crates, hot-path invariants.
- official `docs/state-model.md`: `$PRODEX_HOME/state.json`, profile auth
  isolation under `$PRODEX_HOME/profiles/<name>`, managed `CODEX_HOME` paths,
  response/session bindings, runtime logs.
- official `docs/runtime-policy.md`: `PRODEX_HOME`, policy/env precedence,
  runtime log env, Smart Context rollout env, gateway state backend and
  observability settings.
- official `README.md`: daily `prodex s`, `prodex run`, profile commands,
  multi-account/profile routing, profile credential isolation, and supported
  provider launch paths.

Local Multica cross-references:

- `docs/prodex/prodex-l2-facade.md`: target fork facade; exact endpoints are
  not prodex AS-IS and remain `a validar`.
- `docs/prodex/prodex-pin-integrity.md`: pin, integrity, SBOM, attestation, and
  launch block conditions.
- `docs/deploy/l2-sidecar-deploy-plan.md`: F0 deploy gate, ext4/profile
  invariants, state backend, startup sequence.
- `docs/deploy/prod-rollout-runbook.md`: owner approval gate, pre-checks,
  initial runtime settings, rollout and success criteria.
- `docs/deploy/rollback-runbook.md`: rollback to raw Codex and rollback
  evidence.
- `docs/go-integration/sidecar-lifecycle.md`, `policy-push.md`, and
  `event-ingest.md`: Go lifecycle, desired-state, and event-ingest boundaries.

## Claim Labels

- `confirmed`: directly documented in official pinned prodex docs.
- `Multica F0 wiring`: required by local Multica docs or dispatch; not an
  official prodex AS-IS contract.
- `a validar`: not confirmed as exact prodex AS-IS behavior or endpoint shape;
  must be verified before live F0 launch.

## F0 Launch Boundary

F0 uses prodex AS-IS as the launched runtime binary. It does not claim the
target L2 facade endpoints from `docs/prodex/prodex-l2-facade.md` are already
available.

Rules:

1. Multica Go starts and supervises the prodex process.
2. Multica Go supplies environment, profile references, desired defaults, and
   rollback controls.
3. prodex/Rust owns runtime proxy, profile/session affinity, pre-commit
   selection/fallback, Smart Context mechanics, and guarded redeem behavior.
4. Go must not invoke legacy Go routing for a session already owned by prodex.
5. Runtime events/logs are observability and evidence only; they must not
   re-route a committed request.
6. Live launch is F0-GATED until owner approval is recorded.

## Launch Commands

Primary OpenAI/Codex path:

```bash
prodex run --profile <prodex-profile> -- <codex args>
```

Official basis:

- `prodex run` is the normal runtime launch path.
- Official architecture maps `prodex run` through runtime launch and runtime
  proxy before upstream Codex.
- Official README documents OpenAI/Codex profile-backed routing and
  `prodex run`.

Accepted shorthand:

```bash
prodex --profile <prodex-profile> -- <codex args>
```

Status: `confirmed` that the CLI rewrites default `prodex <args>` to
`prodex run <args>`; Multica should still prefer the explicit `prodex run` form
for audit clarity.

Super/provider-bridge path:

```bash
prodex s
prodex s --provider <provider>
prodex s gemini
```

Official basis:

- official README documents `prodex s` as the daily Super/provider launch path
  and provider bridge entry.

F0 stance:

- Use `prodex run --profile <profile>` for raw Codex replacement unless owner
  explicitly approves `prodex s` for a provider-bridge/Super path.
- `prodex s` in automated Multica F0 launch is `a validar` for non-interactive
  behavior, prompt handling, and exact argument passthrough.

## Environment Mapping

All secret values must come from approved secret boundaries and must not be
printed in logs, traces, evidence, shell history, screenshots, or check-ins.

| Multica input / policy | prodex or child env | Required value | Status |
|---|---|---|---|
| approved prodex binary | `PATH` or absolute executable | pinned prodex binary only | Multica F0 wiring |
| `MULTICA_PRODEX_VERSION` | evidence/env only | `0.246.0` | Multica F0 wiring |
| `MULTICA_PRODEX_COMMIT` | evidence/env only | `7750da9b6a5c91a6d429e18e6a4d422cab4bc144` | Multica F0 wiring |
| profile/account root | `PRODEX_HOME` | per approved tenant/profile-pool root on ext4 or approved Linux fs | `confirmed` for prodex root; per-Multica placement is F0 wiring |
| account isolation | `CODEX_HOME` | per-account/per-task Codex home on ext4 or approved Linux fs | `confirmed` as managed/shared Codex surface; Multica per-account mapping is F0 wiring |
| optional shared Codex root | `PRODEX_SHARED_CODEX_HOME` | only when intentionally overriding native `~/.codex` shared state | `confirmed`; default should be unset |
| runtime logs | `PRODEX_RUNTIME_LOG_DIR` | scrubbed writable log dir on approved Linux fs | `confirmed` |
| runtime log format | `PRODEX_RUNTIME_LOG_FORMAT` | `json` preferred for F0 evidence | `confirmed` |
| Smart Context shadow | `PRODEX_SMART_CONTEXT_SHADOW` | `1` for initial F0 | `confirmed` |
| Smart Context canary | `PRODEX_SMART_CONTEXT_CANARY_PERCENT` | `0` for initial F0 | `confirmed` |
| auto-redeem | prodex launch args / local policy | disabled unless separately approved | `confirmed` that auto-redeem is opt-in via `--auto-redeem`; exact env disable is `a validar` |
| kill-switch default | `PRODEX_KILL_SWITCH_DEFAULT_ON` | `1` | Multica F0 wiring; exact prodex AS-IS env behavior `a validar` |
| gateway Postgres | policy `gateway.state.postgres_url_env` plus named secret env | owner-approved secret-manager env name | `confirmed` for gateway state; F0 use depends on gateway path |
| Redis gateway state | policy `gateway.state.redis_url_env` plus named secret env | owner-approved secret-manager env name if used | `confirmed` for gateway state; F0 use depends on gateway path |
| no proxy interception | `NO_PROXY` / `no_proxy` | include `127.0.0.1`, `localhost`, `::1` | `confirmed` for local broker connection behavior |

Notes:

- Official prodex docs say policy values come from `$PRODEX_HOME/policy.toml`,
  with env overrides where supported.
- `PRODEX_HOME` owns `state.json`, profile metadata, runtime bindings, quota
  snapshots, and profile paths.
- Prodex profile auth is isolated under `$PRODEX_HOME/profiles/<name>`.
- Shared Codex state defaults to native `~/.codex` unless
  `PRODEX_SHARED_CODEX_HOME` is set.
- Multica F0 requires both `PRODEX_HOME` and every effective `CODEX_HOME` to be
  on ext4 or an approved Linux filesystem, not 9p/`/mnt/c`.

## Filesystem Layout

Recommended F0 layout:

```text
<approved-ext4-root>/prodex/<tenant-or-workspace>/
  policy.toml
  state.json
  profiles/
    <profile-a>/
    <profile-b>/
  runtime-logs/

<approved-ext4-root>/codex/<tenant-or-workspace>/<profile-or-account>/
  auth.json
  config.toml
  sessions/
  history.jsonl
```

Status:

- `confirmed`: prodex uses `$PRODEX_HOME/state.json` and
  `$PRODEX_HOME/profiles/<name>`.
- `confirmed`: Codex-owned shared files include `history.jsonl`, `sessions/`,
  `config.toml`, `environments.toml`, plugins, skills, prompts, and memory.
- `Multica F0 wiring`: the exact ext4 root naming and tenant/workspace
  partitioning.

Required checks before launch:

```text
PRODEX_HOME filesystem is ext4 or approved Linux fs
effective CODEX_HOME filesystem is ext4 or approved Linux fs
no path resolves under /mnt/c, /mnt/wsl, /mnt/9p, drvfs, 9p, or CIFS
credential directories mode 0700
credential files mode 0600
owner uid/gid matches daemon runtime user
profile homes resolve inside approved managed roots
```

Any failure blocks F0 launch.

## Profile-Pool Wiring

F0 profile pool input from Go:

```text
tenant_id
workspace_id
policy_id
approved_profiles[]:
  profile_id
  prodex_profile_name
  provider
  codex_home_ref
  prodex_home_ref
  enabled
  priority
```

Mapping:

1. Go resolves approved profiles from Multica state.
2. Go rejects disabled, missing, non-ext4, unsafe-permission, or unpinned
   profile entries before launch.
3. For a single-profile F0 session, Go launches:

   ```bash
   PRODEX_HOME=<approved-prodex-home> \
   CODEX_HOME=<approved-codex-home> \
   PRODEX_SMART_CONTEXT_SHADOW=1 \
   PRODEX_SMART_CONTEXT_CANARY_PERCENT=0 \
   PRODEX_KILL_SWITCH_DEFAULT_ON=1 \
   prodex run --profile <prodex_profile_name> -- <codex args>
   ```

4. For a pool, Go must expose only approved prodex profile names under the
   selected `PRODEX_HOME`; prodex performs quota-aware selection/rotation before
   commit.
5. Go records `runtime_router_owner=rust_l2` or the F0 AS-IS equivalent before
   the controlled session is admitted.

Status:

- `confirmed`: prodex supports multiple profiles, profile-backed routing, quota
  preflight/auto-rotation for eligible OpenAI/Codex profiles, and hard
  continuation affinity.
- `confirmed`: prodex profile commands include profile list/add/import/login/use
  paths.
- `a validar`: exact automated non-interactive enrollment/registration of
  Multica-approved profiles into prodex AS-IS. Do not send raw OAuth tokens,
  cookies, API keys, or `auth.json` through a facade.
- `a validar`: exact F0 evidence field name for `runtime_router_owner` when
  launching AS-IS without target facade endpoints.

## Ordered F0 Launch Sequence

All steps are F0-GATED. Execute only after owner approval from
`docs/deploy/prod-rollout-runbook.md` is recorded.

1. Freeze runtime config for the launch window.
2. Verify owner approval record includes prodex version, commit, artifact hash,
   rollback command reference, kill-switch command reference, and accepted risk.
3. Verify prodex binary pin and integrity per
   `docs/prodex/prodex-pin-integrity.md`.
4. Resolve prodex executable from approved absolute path or PATH entry; reject
   unpinned `latest`.
5. Verify `prodex --version` reports `0.246.0` or approved equivalent evidence.
6. Verify `git/source/artifact` evidence ties to commit
   `7750da9b6a5c91a6d429e18e6a4d422cab4bc144`.
7. Validate `PRODEX_HOME`, profile directories, and `CODEX_HOME` roots are on
   ext4/approved Linux fs with safe permissions.
8. Write or verify `$PRODEX_HOME/policy.toml` for F0 defaults:
   Smart Context shadow, canary 0, auto-redeem disabled, redaction/logging
   enabled, gateway state backend approved if gateway is used.
9. Construct the approved profile pool from references only.
10. Build launch env without printing secret values.
11. Start prodex through the approved command:

    ```bash
    prodex run --profile <prodex_profile_name> -- <codex args>
    ```

    or, only with owner-approved provider/Super path:

    ```bash
    prodex s [--provider <provider>]
    ```

12. Confirm process liveness through process supervision and runtime log path.
13. Run `prodex doctor --runtime --json` or equivalent diagnostic command if it
    is safe and scrubbed.
14. Confirm runtime log path under `PRODEX_RUNTIME_LOG_DIR` or policy
    `runtime.log_dir`.
15. Start one controlled owner-approved smoke session.
16. Confirm Go legacy runtime routing is suppressed for the prodex-owned
    session.
17. Confirm Smart Context remains shadow-only and canary 0.
18. Confirm kill switch can disable Smart Context before the next request.
19. Confirm event/log evidence is scrubbed and durable.
20. Mark F0 smoke complete or rollback.

`a validar`: exact health/readiness endpoints from
`docs/prodex/prodex-l2-facade.md` are target-fork endpoints, not confirmed
prodex AS-IS. F0 AS-IS readiness may use process supervision, runtime logs,
`prodex doctor --runtime --json`, and controlled smoke evidence until the facade
exists.

## Rollback To Raw Codex

Rollback goal: new sessions launch through raw Codex or the previous approved
runtime path while preserving profile isolation, audit evidence, logs, and
Postgres state.

Trigger rollback on any condition from `docs/deploy/rollback-runbook.md`,
including:

- secret leakage;
- `PRODEX_HOME` or `CODEX_HOME` on 9p/shared mount;
- unsafe credential permissions;
- profile switch fail-open;
- continuation affinity failure;
- Smart Context integrity failure without exact fallback;
- kill switch unavailable;
- prodex health/readiness failure;
- event/audit ingest failure;
- owner/Opus request.

Rollback sequence:

1. Declare rollback and record trigger, timestamp, owner/operator, and deploy id.
2. Freeze new prodex sessions in Multica admission.
3. Apply kill switches for Smart Context, gateway, auto-redeem, and provider
   bridge where available.
4. Stop or drain prodex-backed sessions per rollback runbook.
5. Restore raw Codex launch configuration:

   ```bash
   CODEX_HOME=<approved-codex-home> codex <codex args>
   ```

6. Remove prodex-specific launch env from new sessions:
   `PRODEX_HOME`, `PRODEX_SMART_CONTEXT_SHADOW`,
   `PRODEX_SMART_CONTEXT_CANARY_PERCENT`, `PRODEX_KILL_SWITCH_DEFAULT_ON`, and
   prodex gateway state envs unless still needed for evidence-only tools.
7. Restart or reload the minimum required Go daemon component.
8. Start a controlled raw Codex smoke session.
9. Confirm no new session selects prodex runtime routing.
10. Preserve prodex logs and scrubbed evidence; do not delete audit rows or
    credential material.
11. Notify owner and Opus with a scrubbed summary.

Success criteria:

- new sessions launch through raw Codex or previous approved path;
- no new prodex runtime events appear for new sessions after rollback boundary;
- Go daemon is healthy;
- audit rows remain durable;
- redaction smoke still passes;
- ext4/profile permission invariants still pass.

## Block Conditions

Block F0 launch if any is true:

- owner approval is absent;
- prodex version or commit evidence does not match the pin;
- artifact hash, attestation, SBOM, or dependency audit evidence is missing;
- prodex executable resolves to an unapproved path;
- `PRODEX_HOME` or effective `CODEX_HOME` is on 9p/shared host mount;
- credential permissions are unsafe;
- profile-pool entries are missing, disabled, or outside approved roots;
- Smart Context cannot be forced to shadow/canary 0;
- kill-switch default-on behavior cannot be verified;
- raw Codex rollback path is not ready;
- event/log evidence cannot be scrubbed;
- Go would still run legacy routing for prodex-owned sessions.

## Non-Goals

- Do not implement target L2 facade endpoints in this document.
- Do not add Go runtime routing, fallback, or Smart Context rewriting.
- Do not run `prodex redeem` or enable auto-redeem in F0.
- Do not migrate credentials or copy raw auth material into evidence.
- Do not deploy from this document alone.
