# P7 Kill-Switch Test Evidence

timestamp_utc: 2026-07-05T05:19:41Z
task: 7.1 Kill-switch por tenant/provider/profile
status: PASS
secrets_present: false

## Scope

Validated against a real local `prodex-sidecar` process bound to an ephemeral
loopback port. The bearer token was test-only and is not recorded.

## Commands

```bash
cargo build --release
bash scripts/smoke/p7-kill-switch-exercise.sh
bash scripts/smoke/kill-switch-smoke.sh --dry-run --feature smart_context
```

## Results

- `cargo build --release`: PASS.
- `p7-kill-switch-exercise.sh`: PASS.
- `kill-switch-smoke.sh --dry-run --feature smart_context`: PASS.

## Assertions Covered

- Tenant-scoped `smart_context` kill switch can be disabled and re-enabled.
- Provider-scoped `gateway` kill switch blocks a new `codex` session, then
  permits a new session after re-enable.
- Profile-scoped `runtime_proxy` kill switch blocks a new session for the
  target profile, then permits a new session after re-enable.
- Tenant-scoped `auto_redeem` kill switch can be disabled and re-enabled.
- Session responses after re-enable preserve `router_owner=rust_l2`.

## Regression Tests

```bash
cargo test
/home/dataops-lab/.cache/codex-go/go/bin/go test ./internal/l2runtime ./internal/daemon
```

Results:

- `cargo test`: PASS.
- `go test ./internal/l2runtime`: PASS.
- `go test ./internal/daemon`: PASS.

## Evidence Hygiene

- No production tenant id, profile id, bearer token, database URL, Redis URL, or
  credential path was recorded.
- The exercise used synthetic ids: `tenant-p7-smoke`, `codex`, and
  `codex-smoke-main`.
