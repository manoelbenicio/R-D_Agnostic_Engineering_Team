# C4 — Server auth JWT config test gate (hermetic) evidence

- agent: Codex56#B · lane: C4 · task: C4-AUTH-TEST-GATE · pane: w7:p4 (HERDR_ENV=1)
- lock (exclusive): `multica-auth-work/server/internal/auth/jwt_configuration_test.go`
- OpenSpec: 5.3 (server-wide build/test) — auth JWT configuration gate
- repo HEAD: `a6d50986aee7f1981a323b29c9d90f175b4b6f8b`; file tracked+clean at check-in
- toolchains: go `go1.26.1` (`/home/ec2-user/goroot/go/bin/go`); root disk 100% full → GOCACHE/GOTMPDIR/GOMODCACHE on `/tmp`
- scope honored: Main Brain / auth only; OmniRoute internals not touched (no provider/account/credential/session/rotation/model-mapping probe); no native-runtimes-onboarding. No deploy/inference/secret-read/dep-install/commit/push/reset/OpenSpec-checkbox/out-of-lock edit.

## Root cause (inspected before change)

`internal/auth/jwt.go` `ValidateJWTConfiguration(appEnv, secret)` is a **pure function of its two
arguments** — it does not read `os.Getenv`. The two failing cases in `...AllowsExplicitDevelopmentAndConfiguredProduction`
used `secret: "deployment-owned-secret"` (**23 bytes**), below `minimumProductionJWTSecretBytes = 32`, so
the production rule correctly returned `ErrInsecureJWTConfiguration` while the test expected success →
deterministic FAIL (confirmed identical under clean and polluted ambient env; not actually env-flaky, but
made hermetic per objective).

## Change (test file only; production validation NOT weakened)

1. Configured cases now use `configuredProductionSecret` = a 43-byte deployment-owned secret that genuinely
   satisfies the real production rule (≥32 bytes, not a known-insecure placeholder). This makes the test
   pass by giving it a VALID production secret — it does NOT lower the 32-byte bar (jwt.go untouched).
2. Added `isolateAuthEnv(t)` (`t.Setenv("JWT_SECRET","")`, `t.Setenv("APP_ENV","")`) to both tests →
   provably independent of host/CI environment (hermetic), and future-proof if a code path ever reads env.
3. Strengthened the gate: added a `production too-short non-default` fail-closed case (18-byte non-default
   secret) so the ≥32-byte requirement is explicitly asserted and cannot silently regress.

## Commands + exit codes (GOCACHE/GOTMPDIR on /tmp)

- `gofmt -l internal/auth/jwt_configuration_test.go` → clean, exit 0
- `go vet ./internal/auth/` → exit 0
- `go test ./internal/auth/ -run 'TestValidateJWTConfiguration' -count=1 -v` → both test functions PASS, `ok 0.004s`, exit 0
- Hermeticity proof: `JWT_SECRET=... APP_ENV=production go test ./internal/auth/ -run 'TestValidateJWTConfiguration' -count=1` → `ok 0.018s`, exit 0 (identical result under polluted ambient env)
- `git diff --check` → exit 0
- `git diff --stat` → 1 file changed, 24 insertions(+), 2 deletions(-)

## Non-claims / limitations

- Production validation semantics unchanged: `internal/auth/jwt.go` was READ-ONLY and not modified; the
  32-byte minimum and known-insecure-default rejection remain enforced.
- No live run / inference / deploy / secret read / dependency install (module cache on /tmp only).
- Other auth test files (pat_cache_test.go, cookie/cloudfront env reads) are OUTSIDE this lock and were not
  touched; if 5.3 has env-dependent failures there, they belong to a different owner.
- No OpenSpec checkbox closed; Principal adjudicates.
