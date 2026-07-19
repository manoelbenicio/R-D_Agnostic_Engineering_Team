# Native task 1.7 auth/password provisioning security evidence

Evidence date: 2026-07-18. This is an implementation evidence record for independent review. It does not accept the change, mark an OpenSpec task complete, or attest to unrelated dirty-worktree changes.

## Changed-file scope and SHA256 manifest

The security correction and its offline test-topology correction are limited to the following 17 files. The hashes are of the current bytes at evidence collection time. Concurrent/out-of-scope worktree changes are deliberately excluded.

```text
3f9b95b76f2683bb0a91a9d6a7bc6db939dfb3af3dd43d10977f27a756db0512  multica-auth-work/.env.example
6059f7e20ece7485016e2546ef977fadde925569e4b5ed1d862d0f3cace27de9  multica-auth-work/server/cmd/multica/cmd_user.go
80aa70ae912912b6233880dea2cbcc669ab9414ab8112bcb81a1441ee6dc8a3f  multica-auth-work/server/cmd/multica/cmd_user_password_test.go
5aa5cc4268474e8b79ada549ce908df04b071d2fda4aed90a5460c577d423bb6  multica-auth-work/server/cmd/server/main.go
5c6492bfd64347d48bb13749ac3f1b38ef84b4275fd4d20b1fc44e0f1cdb5a74  multica-auth-work/server/cmd/server/router.go
bbb5fa1ca1bf24f94756906512a5717e7a2783113be0c91ba941c700cf8822fd  multica-auth-work/server/internal/auth/jwt.go
9df59d84abfbb5e44a8f1f00571fdc9b47119a15bcf6ce532a01f400bc00fdf5  multica-auth-work/server/internal/auth/jwt_configuration_test.go
e800814c59e5ea55295d6b8c2209bf57fe776595e5d1e541ff511f4f892db94b  multica-auth-work/server/internal/auth/recent_auth.go
ecbc885334affbcf20cadc2c7b73a80d6f77fd570f3ea14676e51f2c942fdf90  multica-auth-work/server/internal/auth/recent_auth_test.go
d69877a9dcabec59726717628c103b2b1ab0c9a4c7673d14a80c8360b7a259e0  multica-auth-work/server/internal/handler/auth.go
3c8f75b5ac2e9a4e2ca83b228285c3ff21d2ced484aab3b29f71cb7b67d70857  multica-auth-work/server/internal/handler/auth_provider.go
4871a86311316e4da83c6fb56e97249da1bfff5714f1c1b4e101380510055e64  multica-auth-work/server/internal/handler/passwordtest/provision_test.go
10d75ff5a2d7db032eab78a307a86ba0157c2725ee72965494c2bc1f571eae6a  multica-auth-work/server/internal/middleware/auth.go
e76cd669222074125561e4e57eac2c737418d7c15b1c6deffa4b2215f2c5b124  multica-auth-work/server/internal/middleware/auth_test.go
14c3ee447ef5f397100100fb086157b538ba36e42a6c17bc340348bb10711808  multica-auth-work/server/internal/middleware/ratelimit.go
43418c0ec0652bbc7e60102d196f3de155cb6734d38643bc14123d9f55944084  multica-auth-work/server/internal/middleware/ratelimit_test.go
97bd6dee369edc88e602f83dd4c6d70c9f83d1b4594f1dcc29d2ef111e52c298  multica-auth-work/server/internal/rotation/rotation_e2e_test.go
```

Nine pure `cmd/server` tests were restored to the normal untagged topology during review; because those restorations leave their bytes at the repository baseline, they are topology evidence below rather than changed-file manifest entries.

## Threat model and production behavior

- Predictable or absent JWT signing material: production-like startup validates before database initialization and rejects an empty secret or the repository's known development default. Only explicit `APP_ENV=dev`, `development`, or `test` permits that development behavior (`server/internal/auth/jwt.go:35-47`; startup call at `server/cmd/server/main.go:122-128`).
- Password brute force during Redis absence/failure: the middleware uses a mutex-protected local fixed-window fallback for both nil Redis and Redis errors. It caps attacker-selected keys at 10,000, reclaims expired entries, and rejects new keys while a full set remains live (`server/internal/middleware/ratelimit.go:17-75,124-160`).
- Password takeover using a stolen ordinary bearer token: an update requires either current-password verification against the authenticated human's identity or a verified `auth_time` no older than five minutes (`server/internal/handler/auth.go:513-575`). An ordinary `iat` is not accepted. `/api/cli-token` deliberately issues the ordinary JWT without `auth_time` (`server/internal/handler/auth.go:205-227,763-785`), and middleware only transfers a verified signed `auth_time` into private server context (`server/internal/middleware/auth.go:310-324`; `server/internal/auth/recent_auth.go:8-24`).
- Weak/pathological password inputs: the server requires valid UTF-8, at least 12 Unicode characters, and no more than bcrypt's 72-byte input limit before hashing or writing (`server/internal/handler/auth_provider.go:16-30,56-71,87-104`).
- Interface and disclosure constraints: `AuthProvider` remains provider-neutral for a future Firebase adapter (`server/internal/handler/auth_provider.go:32-46`); the update endpoint remains human-authenticated and emits generic authentication/update failures. Passwords and tokens are not included in the added log messages.

This model addresses single-process memory exhaustion and fail-open brute-force behavior, but it does not claim distributed enforcement or token revocation; those are residual blockers below.

## Build-tag and offline topology

`//go:build !offline` is confined to database-backed/integration tests, so these tests remain included in ordinary untagged CI and are excluded only from the explicit offline run:

```text
server/cmd/server/activity_listeners_test.go
server/cmd/server/autopilot_failure_monitor_test.go
server/cmd/server/autopilot_listeners_test.go
server/cmd/server/comment_attachment_integration_test.go
server/cmd/server/comment_edit_mention_integration_test.go
server/cmd/server/comment_trigger_integration_test.go
server/cmd/server/integration_test.go
server/cmd/server/notification_listeners_test.go
server/cmd/server/quick_create_subscriber_test.go
server/cmd/server/rerun_session_test.go
server/cmd/server/runtime_sweeper_race_test.go
server/cmd/server/runtime_sweeper_test.go
server/cmd/server/subscriber_listeners_test.go
server/cmd/server/workspace_scope_guard_test.go
server/internal/rotation/rotation_e2e_test.go
```

`server/cmd/server/auth_routes_test.go` and the pure `dbstats`, `health_realtime`, `health`, `listeners_frame`, `listeners_scope`, `metrics`, `runtime_sweeper_filter`, `scope_authorizer`, and `trusted_proxies` tests are untagged and therefore execute in the offline run and normal CI.

The existing `server/internal/handler` package `TestMain` can skip database-dependent tests when PostgreSQL is unavailable. Therefore, its package-level `ok` line is not used as proof for this task. The task's backend assertions are in the separate `server/internal/handler/passwordtest` package and use deterministic fakes/mocks; router assertions are the untagged `server/cmd/server/auth_routes_test.go`. The focused command below names and executes these tests with an invalid database URL. No external database, Redis, service, network endpoint, or credentials were used. Some repository tests use in-process `httptest` loopback servers; those are not live services.

## Genuine offline execution evidence

Working directory for all Go commands: `multica-auth-work/server`. The fixed Go binary was `/home/dataops-lab/.cache/codex-go/go/bin/go`. `GOPROXY=off` prevented dependency-network access, and `DATABASE_URL=://offline-invalid` is intentionally unparsable.

### Focused auth/password tests

Command:

```sh
env APP_ENV=test DATABASE_URL='://offline-invalid' GOPROXY=off /home/dataops-lab/.cache/codex-go/go/bin/go test -tags=offline -count=1 ./internal/auth ./internal/middleware ./internal/handler/passwordtest ./cmd/multica ./cmd/server -run 'Test(ValidateJWTConfiguration|HasRecentAuthentication|RateLimit_NilRedis|RateLimit_RedisErrorUsesLocalFallback|BoundedLocalRateLimiter|Auth_VerifiedJWT|Auth_IssuedAt|PostgresPasswordCredentialStore|UpdatePassword|PasswordLoginMints|PasswordAuthRoutes|PasswordUpdateRoute|ReadPassword|UserPasswordUpdate|RunUserPasswordUpdate|PasswordPrompt)'
```

Exact output and status:

```text
ok  	github.com/multica-ai/multica/server/internal/auth	0.013s
ok  	github.com/multica-ai/multica/server/internal/middleware	0.013s
ok  	github.com/multica-ai/multica/server/internal/handler/passwordtest	0.331s
ok  	github.com/multica-ai/multica/server/cmd/multica	0.061s
ok  	github.com/multica-ai/multica/server/cmd/server	0.171s
exit status 0
```

### Focused race detector

Command:

```sh
env APP_ENV=test DATABASE_URL='://offline-invalid' GOPROXY=off /home/dataops-lab/.cache/codex-go/go/bin/go test -tags=offline -race -count=1 ./internal/auth ./internal/middleware ./internal/handler/passwordtest ./cmd/multica ./cmd/server -run 'Test(ValidateJWTConfiguration|HasRecentAuthentication|RateLimit_NilRedis|RateLimit_RedisErrorUsesLocalFallback|BoundedLocalRateLimiter|Auth_VerifiedJWT|Auth_IssuedAt|PostgresPasswordCredentialStore|UpdatePassword|PasswordLoginMints|PasswordAuthRoutes|PasswordUpdateRoute|ReadPassword|UserPasswordUpdate|RunUserPasswordUpdate|PasswordPrompt)'
```

Exact output and status:

```text
ok  	github.com/multica-ai/multica/server/internal/auth	1.031s
ok  	github.com/multica-ai/multica/server/internal/middleware	1.055s
ok  	github.com/multica-ai/multica/server/internal/handler/passwordtest	4.718s
ok  	github.com/multica-ai/multica/server/cmd/multica	1.084s
ok  	github.com/multica-ai/multica/server/cmd/server	2.387s
exit status 0
```

### Current full offline test

Command:

```sh
env APP_ENV=test DATABASE_URL='://offline-invalid' GOPROXY=off /home/dataops-lab/.cache/codex-go/go/bin/go test -tags=offline -count=1 ./...
```

Exact output and status:

```text
ok  	github.com/multica-ai/multica/server/cmd/backfill_codex_usage_cache	0.032s
?   	github.com/multica-ai/multica/server/cmd/backfill_task_usage_hourly	[no test files]
ok  	github.com/multica-ai/multica/server/cmd/migrate	0.034s
ok  	github.com/multica-ai/multica/server/cmd/multica	1.215s
ok  	github.com/multica-ai/multica/server/cmd/server	0.294s
ok  	github.com/multica-ai/multica/server/internal/agenttmpl	0.036s
ok  	github.com/multica-ai/multica/server/internal/analytics	10.036s
ok  	github.com/multica-ai/multica/server/internal/auth	0.277s
ok  	github.com/multica-ai/multica/server/internal/cli	1.115s
ok  	github.com/multica-ai/multica/server/internal/cloudruntime	0.036s
ok  	github.com/multica-ai/multica/server/internal/daemon	23.937s
ok  	github.com/multica-ai/multica/server/internal/daemon/brain	0.029s
ok  	github.com/multica-ai/multica/server/internal/daemon/deploy	0.043s
ok  	github.com/multica-ai/multica/server/internal/daemon/execenv	11.706s
ok  	github.com/multica-ai/multica/server/internal/daemon/gateway	0.314s
ok  	github.com/multica-ai/multica/server/internal/daemon/observability	0.617s
ok  	github.com/multica-ai/multica/server/internal/daemon/repocache	10.785s
ok  	github.com/multica-ai/multica/server/internal/daemon/runtimeenv	0.540s
ok  	github.com/multica-ai/multica/server/internal/daemonws	0.489s
ok  	github.com/multica-ai/multica/server/internal/events	0.066s
ok  	github.com/multica-ai/multica/server/internal/handler	0.131s
ok  	github.com/multica-ai/multica/server/internal/handler/passwordtest	0.385s
ok  	github.com/multica-ai/multica/server/internal/integrations/lark	3.489s
?   	github.com/multica-ai/multica/server/internal/issueguard	[no test files]
?   	github.com/multica-ai/multica/server/internal/issueposition	[no test files]
ok  	github.com/multica-ai/multica/server/internal/l2runtime	0.050s
?   	github.com/multica-ai/multica/server/internal/logger	[no test files]
ok  	github.com/multica-ai/multica/server/internal/metrics	0.683s
ok  	github.com/multica-ai/multica/server/internal/middleware	0.380s
?   	github.com/multica-ai/multica/server/internal/migrations	[no test files]
ok  	github.com/multica-ai/multica/server/internal/realtime	0.432s
ok  	github.com/multica-ai/multica/server/internal/rotation	0.064s
ok  	github.com/multica-ai/multica/server/internal/scheduler	0.027s
ok  	github.com/multica-ai/multica/server/internal/service	0.092s
ok  	github.com/multica-ai/multica/server/internal/skill	0.035s
ok  	github.com/multica-ai/multica/server/internal/storage	0.084s
ok  	github.com/multica-ai/multica/server/internal/taskusagebackfill	0.018s
ok  	github.com/multica-ai/multica/server/internal/util	0.022s
ok  	github.com/multica-ai/multica/server/internal/util/secretbox	0.013s
ok  	github.com/multica-ai/multica/server/pkg/agent	12.301s
?   	github.com/multica-ai/multica/server/pkg/db/generated	[no test files]
?   	github.com/multica-ai/multica/server/pkg/protocol	[no test files]
--- FAIL: TestSanitizeForLog (0.00s)
    redact_test.go:262: query secret not redacted: mysecretvalue
FAIL
FAIL	github.com/multica-ai/multica/server/pkg/redact	0.025s
ok  	github.com/multica-ai/multica/server/pkg/taskfailure	0.016s
FAIL
exit status 1
```

This current full-tree failure is outside the 17-file task scope. `server/pkg/redact/redact.go` and `redact_test.go` were modified concurrently during evidence collection (filesystem mtimes 15:28:44 and 15:29:22 -03:00); this agent did not modify them. The auth/password packages passed in the same run. This record does not convert the full-tree failure into a pass.

### Current full offline race test

Command:

```sh
env APP_ENV=test DATABASE_URL='://offline-invalid' GOPROXY=off /home/dataops-lab/.cache/codex-go/go/bin/go test -tags=offline -race -count=1 ./...
```

Exact output and status:

```text
ok  	github.com/multica-ai/multica/server/cmd/backfill_codex_usage_cache	1.077s
?   	github.com/multica-ai/multica/server/cmd/backfill_task_usage_hourly	[no test files]
ok  	github.com/multica-ai/multica/server/cmd/migrate	1.048s
ok  	github.com/multica-ai/multica/server/cmd/multica	1.845s
ok  	github.com/multica-ai/multica/server/cmd/server	3.900s
ok  	github.com/multica-ai/multica/server/internal/agenttmpl	1.122s
ok  	github.com/multica-ai/multica/server/internal/analytics	16.113s
ok  	github.com/multica-ai/multica/server/internal/auth	1.881s
ok  	github.com/multica-ai/multica/server/internal/cli	2.629s
ok  	github.com/multica-ai/multica/server/internal/cloudruntime	1.085s
ok  	github.com/multica-ai/multica/server/internal/daemon	23.102s
ok  	github.com/multica-ai/multica/server/internal/daemon/brain	1.102s
ok  	github.com/multica-ai/multica/server/internal/daemon/deploy	1.119s
ok  	github.com/multica-ai/multica/server/internal/daemon/execenv	2.853s
ok  	github.com/multica-ai/multica/server/internal/daemon/gateway	2.045s
ok  	github.com/multica-ai/multica/server/internal/daemon/observability	3.909s
ok  	github.com/multica-ai/multica/server/internal/daemon/repocache	2.834s
ok  	github.com/multica-ai/multica/server/internal/daemon/runtimeenv	5.538s
ok  	github.com/multica-ai/multica/server/internal/daemonws	1.569s
ok  	github.com/multica-ai/multica/server/internal/events	1.073s
ok  	github.com/multica-ai/multica/server/internal/handler	2.837s
ok  	github.com/multica-ai/multica/server/internal/handler/passwordtest	5.751s
ok  	github.com/multica-ai/multica/server/internal/integrations/lark	4.787s
?   	github.com/multica-ai/multica/server/internal/issueguard	[no test files]
?   	github.com/multica-ai/multica/server/internal/issueposition	[no test files]
ok  	github.com/multica-ai/multica/server/internal/l2runtime	1.130s
?   	github.com/multica-ai/multica/server/internal/logger	[no test files]
ok  	github.com/multica-ai/multica/server/internal/metrics	3.565s
ok  	github.com/multica-ai/multica/server/internal/middleware	1.792s
?   	github.com/multica-ai/multica/server/internal/migrations	[no test files]
ok  	github.com/multica-ai/multica/server/internal/realtime	1.563s
ok  	github.com/multica-ai/multica/server/internal/rotation	1.259s
ok  	github.com/multica-ai/multica/server/internal/scheduler	1.093s
ok  	github.com/multica-ai/multica/server/internal/service	1.188s
ok  	github.com/multica-ai/multica/server/internal/skill	1.068s
ok  	github.com/multica-ai/multica/server/internal/storage	1.160s
ok  	github.com/multica-ai/multica/server/internal/taskusagebackfill	1.098s
ok  	github.com/multica-ai/multica/server/internal/util	1.073s
ok  	github.com/multica-ai/multica/server/internal/util/secretbox	1.044s
ok  	github.com/multica-ai/multica/server/pkg/agent	10.711s
?   	github.com/multica-ai/multica/server/pkg/db/generated	[no test files]
?   	github.com/multica-ai/multica/server/pkg/protocol	[no test files]
--- FAIL: TestSanitizeForLog (0.02s)
    redact_test.go:262: query secret not redacted: mysecretvalue
FAIL
FAIL	github.com/multica-ai/multica/server/pkg/redact	0.056s
ok  	github.com/multica-ai/multica/server/pkg/taskfailure	1.067s
FAIL
exit status 1
```

The focused race command above is the exact task-scope race proof. The full race command is currently red for the same concurrent, out-of-scope `pkg/redact` test failure and is not represented as passing.

### Full offline vet

Command:

```sh
env APP_ENV=test DATABASE_URL='://offline-invalid' GOPROXY=off /home/dataops-lab/.cache/codex-go/go/bin/go vet -tags=offline ./...
```

Exact output and status:

```text
<no output>
exit status 0
```

## Production missing-JWT startup probe

Command:

```sh
env GOPROXY=off /home/dataops-lab/.cache/codex-go/go/bin/go build -o /tmp/multica-server-security-check ./cmd/server
env APP_ENV=production JWT_SECRET= DATABASE_URL='://must-not-be-reached' /tmp/multica-server-security-check
```

Exact runtime output and status:

```text
15:33:01.356 ERR refusing to start with insecure JWT configuration error="JWT_SECRET must be set to a non-development value outside explicit development or test mode"
exit status 1
```

The build emitted no output. The deliberately invalid database URL was not parsed or contacted: the process exited at the JWT validation call before database initialization.

## Residual blockers

Exactly four known residual blockers remain:

1. **CLI bootstrap:** the current CLI password-update request supplies `new_password` through an ordinary PAT/bearer flow, but that token is intentionally not recent-auth proof and the CLI does not yet collect `current_password`. First-password bootstrap therefore still needs a narrowly scoped authenticated bootstrap design (for example, a recent interactive Google/password session or a separate one-time mechanism) rather than weakening the server check.
2. **Mobile legacy endpoints:** the mobile client still references the removed/legacy verification-code endpoints (`/auth/send-code` and `/auth/verify-code`). Mobile auth/store/UI migration remains outside this native backend correction.
3. **Per-process fallback:** the bounded fallback is process-local. During Redis outage, a multi-replica deployment multiplies the effective allowance by replica count and does not share counters. The fallback prevents fail-open behavior and unbounded per-process key growth, but is not a distributed limiter.
4. **Token revocation:** existing stateless JWTs and PATs are not revoked automatically when a password changes. A stolen still-valid token remains usable until expiry/revocation through existing mechanisms; password-change-triggered session/token invalidation remains future work.

## Review boundary

No product, OpenSpec checkbox, `STATE`, ledger, or evidence index was modified while creating this artifact. No credentials, database, Redis, external network endpoint, or live service was accessed. This artifact intentionally makes no ACCEPT decision and requires independent review.

## Independent reviewer ACCEPT (2026-07-18)

Verdict: **ACCEPT** for the auth task-1.7 security-correction scope (implementation + offline verification). Reproduced with `/home/dataops-lab/go-sdk/bin/go` (`go1.26.4`), `GOTOOLCHAIN=local GOPROXY=off`, `APP_ENV=test DATABASE_URL='://offline-invalid'`; no database, Redis, network, credentials, or live service used.

- **17-file manifest:** all 17 SHA256 match the current disk bytes exactly.
- **Focused auth/password tests:** `internal/auth`, `internal/middleware`, `internal/handler/passwordtest`, `cmd/multica`, `cmd/server` all `ok` (exit 0) under `-tags=offline`.
- **Focused race:** all five packages `ok` (exit 0); `-v` confirms the named tests genuinely execute — **25 distinct named parent tests ran** (incl. `TestValidateJWTConfigurationFailsClosedOutsideExplicitDevelopment`, `TestBoundedLocalRateLimiterConcurrentLimit`, `TestBoundedLocalRateLimiterRejectsNewKeysAtCapacityAndReclaimsExpired`, `TestHasRecentAuthentication`, `TestPasswordAuthRoutes`, `TestPostgresPasswordCredentialStoreProvisionPasswordValidatesBeforeWrite`) — not a zero-match.
- **Vet:** `-tags=offline` vet on the scoped packages exit 0.
- **Production missing-JWT startup probe:** `go build ./cmd/server`, then `APP_ENV=production JWT_SECRET= DATABASE_URL='://must-not-be-reached'` → exit 1 with "refusing to start with insecure JWT configuration"; the invalid DB URL was never reached (fail-closed before DB init).
- **Bounded local limiter / password proof+policy / build-tag topology:** exercised by the executed tests above; `//go:build !offline` is confined to DB/integration tests, so the auth suite executes in the offline run and normal CI.
- **Out-of-scope failure distinguished:** the full `./...` red is isolated to `pkg/redact` (`TestSanitizeForLog`, redact_test.go:262), independently reproduced; `pkg/redact` is **not** in the 17-file manifest and was modified concurrently — it does not affect auth-1.7 correctness.
- **Residuals unchanged:** the four documented residual blockers (CLI bootstrap, mobile legacy endpoints, per-process fallback, token revocation) remain future work — scope limits, not correctness failures, and honestly disclosed.

No product/OpenSpec/STATE/ledger/index edit and no checkbox change were made by this review; only this reviewer section was appended. Independent-review verdict = **ACCEPT**.
