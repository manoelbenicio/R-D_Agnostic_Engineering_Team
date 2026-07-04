# prodex Gap Hardening List

Status: PRE-DEPLOY REQUIRED

## Hardening Before Broad PROD

1. Verify package pin and integrity.
2. Verify no shared SQLite/file state in multi-worker deployment.
3. Verify provider conformance split does not hide unsupported params.
4. Verify Smart Context exact fallback with replay.
5. Verify event redaction on error paths.
6. Verify `redeem` behavior with controlled real accounts.
7. Verify kill switch per tenant/provider/profile.
8. Verify rollback to raw codex path.
9. Verify sidecar readiness checks Postgres and policy.
10. Verify logs do not become required for request success.

## Known Risk Classes

- upstream Codex drift;
- bus-factor/churn;
- provider capability overclaim;
- continuation corruption;
- secret leakage in evidence;
- file lock contention if using file/SQLite state;
- deploy without runbook approval.
