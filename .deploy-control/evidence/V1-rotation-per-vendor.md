# V1 rotation per vendor empirical evidence

- timestamp_utc: 2026-07-06T00:51:06Z
- milestone: v2.1
- phase: V1
- task: V1.2 rotation per-vendor
- target_sidecar: `http://127.0.0.1:43292`
- secrets_present: false

## Check-in before

- repository: `/mnt/c/VMs/Projects/Automonous_Agentic`
- branch: `main`
- head: `c35bd78`
- initial dirty state before this task:
  - `M CHECKIN_OUT.md`
  - `?? .codex/config.toml`
  - `?? .deploy-control/`
  - `?? scripts/smoke/`
- `prodex-sidecar/` was not edited. This repo has no local `prodex-sidecar/` directory.

## Runtime preflight

Observed listeners before the test:

```text
127.0.0.1:43290 python3 fake upstream
127.0.0.1:43291 prodex gateway
127.0.0.1:43292 prodex-sidecar
```

`GET /healthz` returned HTTP 200:

```json
{"contract_version":"rpp.l2.v1","sidecar":{"commit":"smoke","name":"prodex-sidecar","version":"0.1.0"},"status":"alive"}
```

`GET /readyz` returned HTTP 200 with:

```json
{
  "contract_version": "rpp.l2.v1",
  "status": "ready",
  "checks": [
    {"name": "shared_state_backend", "status": "pass", "details": {"backend_type": "postgres", "connection_status": "ok", "probe": "SELECT 1"}},
    {"name": "kill_switch", "status": "pass"},
    {"name": "runtime_proxy", "status": "pass", "details": {"listen_addr": "127.0.0.1:43291"}},
    {"name": "event_stream", "status": "pass"}
  ]
}
```

## Request shape

For each vendor, the test called `POST /v1/session/start` with this shape:

```json
{
  "contract_version": "rpp.l2.v1",
  "tenant_id": "tenant-v1-rotation",
  "session_id": "v1-rotation-<vendor>-baseline-<ts>",
  "requested_provider": "<vendor>",
  "profile_pool": ["profile-A", "profile-B"],
  "workspace_id": "workspace-v1",
  "task_id": "task-<vendor>"
}
```

Selection was inferred from the `session_started` event returned by:

```text
GET /v1/events/stream?session_id=<session_id>
```

## Baseline results

| Vendor | requested_provider | session/start HTTP | router_owner | runtime_session_id | event_count | selected profile | event provider | redaction |
| --- | --- | ---: | --- | --- | ---: | --- | --- | --- |
| Codex | `codex` | 200 | `rust_l2` | present | 1 | `profile-A` | `codex` | ok |
| Kiro | `kiro` | 200 | `rust_l2` | present | 1 | `profile-A` | `kiro` | ok |
| Antigravity | `antigravity` | 200 | `rust_l2` | present | 1 | `profile-A` | `antigravity` | ok |
| Cline | `cline` | 200 | `rust_l2` | present | 1 | `profile-A` | `cline` | ok |

Empirical result:

- PASS: sidecar accepted `profile_pool` with 2 profiles for all four vendors.
- PASS: sidecar selected a profile for all four vendors, observable via `session_started.profile_id`.
- OBSERVED: selected profile was always the first pool entry, `profile-A`.
- No round-robin or per-vendor alternation was observed in this surface.

## Kill switch attempt

The test then attempted to disable `profile-A` for Codex:

```json
{
  "contract_version": "rpp.l2.v1",
  "request_id": "req-v1-kill-profile-a",
  "tenant_id": "tenant-v1-rotation",
  "scope": {
    "provider": "codex",
    "profile_id": "profile-A"
  },
  "feature": "profile-A",
  "state": "disabled",
  "effective_at": "next_request"
}
```

`POST /v1/killswitch/apply` returned HTTP 200:

```json
{
  "applied": true,
  "contract_version": "rpp.l2.v1",
  "effective_at": "next_request",
  "request_id": "req-v1-kill-profile-a"
}
```

`GET /v1/killswitch/status?tenant_id=tenant-v1-rotation&provider=codex&profile_id=profile-A&feature=profile-A` returned HTTP 200:

```json
{
  "active": true,
  "contract_version": "rpp.l2.v1",
  "feature": "profile-A",
  "profile_id": "profile-A",
  "provider": "codex",
  "session_id": "",
  "tenant_id": "tenant-v1-rotation"
}
```

## Post-kill result

After the active kill switch, the test called `POST /v1/session/start` again for Codex with:

```json
{
  "requested_provider": "codex",
  "profile_pool": ["profile-A", "profile-B"]
}
```

Observed result:

```json
{
  "status": 200,
  "accepted_2_profiles": true,
  "router_owner": "rust_l2",
  "runtime_session_id_present": true,
  "selected_profile_from_event": "profile-A",
  "event_provider": "codex",
  "smart_context_mode": "proxy_rewrite"
}
```

Empirical result:

- PASS: kill switch API accepted `feature=profile-A` and reported it active.
- FAIL/NOT OBSERVED: fallback to `profile-B` was not observed.
- The post-kill Codex session still selected `profile-A`.

## Conclusion

For V1.2 rotation per-vendor on the sidecar currently running at `127.0.0.1:43292`:

- The sidecar accepts `profile_pool` arrays with 2+ profiles.
- The sidecar emits a selected profile in `session_started.profile_id`.
- The selected profile is empirically the first pool entry, `profile-A`, for Codex, Kiro, Antigravity, and Cline.
- A kill switch using `feature=profile-A` can be applied and queried as active, but it does not force fallback to `profile-B` in the observed `POST /v1/session/start` path.
- No `prodex-sidecar/` files were edited.
