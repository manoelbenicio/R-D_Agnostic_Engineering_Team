# R7 — Task 5.5 Residual Scan Evidence

agent: Agy-P0-A7
lane: R7
task: REC-RESIDUAL-SCAN (5.5)
pane: wB:p1
timestamp: 2026-07-22T11:02:00Z
status: COMPLETE — NO_REACHABLE_RESIDUAL

## Preflight

| Item | Value |
|---|---|
| cwd | `/home/ec2-user/workspace/R-D_Agnostic_Engineering_Team` |
| git HEAD | `a6d5098` |
| git status count | 301 |
| Go | `go1.26.1 linux/amd64` at `/home/ec2-user/goroot/go/bin/go` |
| Node | `v22.23.1` |
| pnpm | `10.28.2` |
| disk_free | 1.9G |

## Scan Methodology

Target paths: startup, config, health, recovery, fallback in production code.
Excluded: `*_test.go`, `*.test.*`, `*.spec.*`, `vendor/`, `testdata/`, `node_modules/`, historical artifacts.

Patterns searched (case-insensitive regex):
1. `mock|fake[^C]|stub[A-Z]|dummy[^_]`
2. `fallback|recover[^y]|default.*model|placeholder`
3. `TODO|FIXME|HACK|XXX|TEMP|WORKAROUND`
4. `hardcoded|hard.coded|sample|demo[^n]|qa.only|dev.only`
5. `mock|fake|stub|placeholder.*data|demo.mode|qa.route|synthetic.*result|hardcoded.*status`

## Scan Scope

| Area | Path | Files Scanned |
|---|---|---|
| Daemon (core) | `server/internal/daemon/` | All `*.go` (excl. `_test.go`) |
| Server cmd | `server/cmd/` | All `*.go` (excl. `_test.go`) |
| Handlers | `server/internal/handler/` | All `*.go` (excl. `_test.go`) |
| Services | `server/internal/service/` | All `*.go` (excl. `_test.go`) |
| Middleware | `server/internal/middleware/` | All `*.go` (excl. `_test.go`) |
| Health endpoint | `server/cmd/server/health.go` | Direct file scan |
| Router | `server/cmd/server/router.go` | Direct file scan |
| Frontend core | `packages/core/` | All `*.ts`, `*.tsx` (excl. test) |
| Frontend views | `packages/views/` | All `*.ts`, `*.tsx` (excl. test) |

## Findings

### Category: mock/fake/stub in production Go code

**Result: ZERO production hits.** Every single `mock`, `fake`, `stub` match is in a `_test.go` file:
- `daemon/auto_update_test.go`: `withStubRelease` — test helper for mocking CLI releases
- `daemon/brain/g2a_test.go`: `fakeReadinessChecker`, `fakePreservedLifecycle`, `fakeTaskExecutor`, `fakeResultSink` — test doubles
- `daemon/config_test.go`: `stageFakeAgent` — test helper for fake binaries
- `server/cmd/server/health_test.go`: `stubReadinessDB`, `stubRow` — test doubles
- `server/cmd/server/scope_authorizer_test.go`: `fakeScopeQuerier` — test double
- `handler/auth_signup_test.go`: `mockDB`, `mockRow` — test doubles
- `handler/file_test.go`: `mockStorage` — test double
- `service/email_test.go`: `fakeSMTPAuthClient` — test double
- `service/task_complete_race_test.go`: `mockRow`, `mockDBTX` — test doubles
- `service/task_notify_test.go`: `stubWakeup` — test double

**Verdict: All confined to test infrastructure. No production mock/fake/stub leakage.**

### Category: router.go "mock" mentions

Two comment-only references at lines 230, 237 of `server/cmd/server/router.go`:
- L230: "CI / integration tests that want to avoid real Lark traffic can point MULTICA_LARK_HTTP_BASE_URL at a mock server"
- L237: "a proxy, a mock for tests, or a single-cloud staging setup"

**Verdict: Comment-only. The code itself uses `os.Getenv("MULTICA_LARK_HTTP_BASE_URL")` — a deployment configuration env var, not a residual mock path. Clean.**

### Category: placeholder in production Go code

Two hits in `server/internal/handler/agent.go`:
- L74: `runtimeConfigGatewayTokenMask` — "the placeholder the API substitutes for any non-empty `runtime_config.gateway.token`"
- L620: "render an explicit 'no access' placeholder instead of a 404"

One hit in `server/internal/handler/agent_env.go`:
- L21: "silently destroy real secrets by saving the masked placeholder"

One hit in `server/internal/service/autopilot.go`:
- L1176/1198: `interpolateTemplate` for `{{date}}` template substitution in issue titles

**Verdict: All are legitimate production patterns (secret masking, template interpolation, access control). No fake-success or demo data.**

### Category: fallback/recovery in production Go code

All hits are legitimate architectural patterns:
- `daemon/brain/release.go`: `FallbackPolicy`, `SameModelAccountFallback`, `CrossModelFallback` — OmniRoute routing policy
- `daemon/brain/identity.go`: "retry, and fallback decisions for a request" — routing comment
- `daemon/brain/recovery.go`: recovery mode gateway outage handling — legitimate fail-closed
- `daemon/config.go`: shell-fallback resolver for CLI path resolution — legitimate
- `daemon/client.go`: `RecoverOrphans` — orphan task recovery endpoint

**Verdict: These are intentional architectural fallback/recovery mechanisms, not fake-success residuals.**

### Category: TODO/FIXME/HACK in production Go code

Hits exist (typical codebase) but none indicate:
- Fake-success paths left behind
- QA-only routes exposed in production
- Placeholder data persisted to storage

### Category: Frontend production code (core + views)

**Result: ZERO production hits.** All `mock/fake/stub/placeholder.*data/demo.mode/qa.route/synthetic.*result/hardcoded.*status` matches are exclusively in `.test.tsx`/`.test.ts` files within `vi.mock()`, `vi.fn()`, test setup code.

### Category: health.go

**Result: ZERO hits.** `server/cmd/server/health.go` has no mock/fake/stub/placeholder/demo/QA patterns in production code.

## Summary

| Category | Production Hits | All in Test Files? |
|---|---|---|
| mock/fake/stub/dummy | 0 | Yes (all in `_test.go`) |
| placeholder | 4 (legit: secret masking, template) | N/A — legitimate patterns |
| fallback/recovery | ~15 (legit: routing policy, path resolution, orphan recovery) | N/A — legitimate architecture |
| demo/QA-only/dev-only | 0 | N/A |
| hardcoded status | 0 | N/A |
| synthetic result | 0 | N/A |

## Conclusion

**NO_REACHABLE_RESIDUAL** across all scanned startup/config/health/recovery/fallback paths.

- Zero production mock/fake/stub leakage
- All fallback patterns are legitimate OmniRoute/shell-resolver/orphan-recovery architecture
- All placeholder patterns are legitimate secret masking and template interpolation
- Zero demo/QA-only/dev-only routes in production handlers or routers
- Frontend production code completely clean
- Health endpoint clean
- All test infrastructure properly confined to `_test.go`/`.test.tsx` files

## Routing

No findings to route — no residuals detected requiring owner action.
