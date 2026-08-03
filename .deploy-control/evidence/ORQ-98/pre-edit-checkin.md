# ORQ-98 Pre-Edit Check-in

- Timestamp (UTC): 2026-08-03T04:29:46Z
- Card: ORQ-98 / ORQ-78 GAP-07
- Card authority SHA-256: `1f0c54bda37eaa14b46c4ce286debb81860fdc2f6583a0e730fcc435e62043ce`
- Branch: `recovery/orq98-gap07-redaction`
- Worktree: `/home/ec2-user/worktrees/multica-orq98-gap07-redaction`
- Base commit: `18c184218a34bdb4631ef210c7f27ec9cdffa4dd`
- Base tree: `6574d7616010f1d4e5e5534299b5e3e9b214433d`
- Initial status: clean

## Exact path lease

- New focused handler test:
  `multica-auth-work/server/internal/handler/orq98redaction/auth_log_redaction_test.go`
- New focused Cloud PAT test:
  `multica-auth-work/server/internal/auth/cloud_pat_site_log_redaction_test.go`
- ORQ-98 evidence only under `.deploy-control/evidence/ORQ-98/`.
- Durable checkout report: `/home/ec2-user/recovery-checkouts/ORQ-98.md`.

Product files `auth.go` and `cloud_pat.go` are read-only under this test-only card.
ORQ-97 and ORQ-99 own disjoint outcomes and paths.

## Baseline seam identity

- `server/internal/handler/auth.go:656`: logs the non-OK Google token response body as
  structured attribute `body`.
  SHA-256: `d69877a9dcabec59726717628c103b2b1ab0c9a4c7673d14a80c8360b7a259e0`.
- `server/internal/auth/cloud_pat.go:359`: logs the non-200 Fleet response snippet as
  structured attribute `body`.
  SHA-256: `98a4aadf4dd9a236a388bcdfaa9434d83a779916d25d8afdb53c67a09a7e2778`.

The cited seams exist at the recorded lines. Existing
`cloud_pat_log_redaction_test.go` installs `redact.SanitizeSlogAttr` in its test handler;
ORQ-98 adds direct site tests that fail if a sentinel reaches the emitted output/attributes
without relying on a test-only sanitizer installation.

The Google test lives in a test-only subpackage so the parent handler package's database-gated
`TestMain` cannot skip this focused test and produce a false-green result.
