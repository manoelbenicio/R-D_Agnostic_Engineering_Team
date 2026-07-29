# A8 — Production Integrity Backend/Config/Deploy Audit

agent: Agy-P0-A8
lane: A8
task: P0-PROD-INTEGRITY-BE
pane: wB:p2
check-in: 2026-07-21T22:38:22Z
status: **DONE — NO_REACHABLE_RESIDUAL**

## 1. Preflight

| Tool | Version / Path | Status |
|---|---|---|
| git | 2.50.1 | ✅ |
| python3 | 3.9.25 | ✅ |
| rg (ripgrep) | 15.2.0 | ✅ |
| openspec | 1.4.1 | ✅ |
| Go 1.26.1 | `/home/ec2-user/goroot/go/bin/go` → go1.26.1 linux/amd64 | ✅ |
| Herdr pane | wB:p2, label Agy-P0-A8, agent_status: blocked | ✅ |

## 2. Scope & Methodology

**Goal:** Prove production reachability for mocks / fake-success / QA routes / placeholders / demo persistence / synthetic defaults in backend/config/deploy, **outside** W1 hotspots, `runtimeenv/`, and `gateway/` ownership.

**Methodology:**
1. Systematic `rg` scans of `multica-auth-work/server/` (non-test `.go` files) for patterns: `mock`, `fake`, `stub`, `placeholder`, `demo`, `dummy`, `sample`, `hardcoded`, `TODO`, `FIXME`, `HACK`, `XXX`, `qa_only`, `test_mode`, `devMode`, `debug`, `bypass`, `skipAuth`, `fake-success`, `fake_result`, `always return true`, `default.*password`, `default.*secret`, `default.*token`, `localhost`, `127.0.0.1`, `example.com`.
2. Systematic scan of deploy/config files: `deploy/helm/`, `deploy/observability/`, `docker-compose*.yml`, `Dockerfile*`, `.env.example`, `Makefile`, `.goreleaser.yml`.
3. HTTP endpoint audit: searched for debug/test/QA/dev/pprof/admin routes registered in production router code.
4. Caller reachability tracing for every finding against production code paths.

**Excluded from scope (per assignment):**
- `internal/daemon/runtimeenv/**` (W3/R1 ownership)
- `internal/daemon/gateway/**` (W2 ownership)
- W1 hotspots: `daemon.go`, `config.go`, `health.go`, `brain/**`, `brain_integration.go`, entrypoints
- `prodex*.go`, `l2_runtime.go` (Phase 3 / HOLD)
- Auth/credential/account/quota/retry/failover internals
- Test files (`*_test.go`)

## 3. Findings — Classified

### 3.1 Items Examined and Cleared (NOT production-reachable)

#### F1: Default JWT secret `"multica-dev-secret-change-in-production"`
- **File:** [`server/internal/auth/jwt.go:14`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/auth/jwt.go#L14)
- **Guard:** `ValidateJWTConfiguration()` called at [`cmd/server/main.go:125`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/cmd/server/main.go#L125) — server **refuses to start** with the default secret when `APP_ENV` is not `dev`/`development`/`test`. Also rejects the known placeholder `"change-me-in-production"` and any secret < 32 bytes.
- **Verdict:** ✅ Not production-reachable. Fail-closed guard enforced at startup.

#### F2: `MULTICA_DEV_VERIFICATION_CODE`
- **Files:** [`handler/auth.go:176-191`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/handler/auth.go#L176-L191), [`cmd/server/main.go:132-138`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/cmd/server/main.go#L132-L138)
- **Guard:** `isDevVerificationCode()` returns `false` when `isProductionEnv()` is true (`APP_ENV=production`). Startup logs explicitly warn "ignored because APP_ENV=production". Uses constant-time comparison.
- **Verdict:** ✅ Not production-reachable. Fail-closed in production.

#### F3: `MULTICA_LOCAL_AUTH_BYPASS`
- **File:** [`middleware/auth.go:38-55`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/middleware/auth.go#L38-L55)
- **Guard:** Defaults to `false`. When set to `true`, also requires `FRONTEND_ORIGIN` to resolve to a loopback address (localhost/127.x). Returns empty string (bypass disabled) for any non-loopback frontend.
- **Verdict:** ✅ Not production-reachable. Legitimate single-user localhost feature with double-gate.

#### F4: `OMNIROUTE_DEV_MODELS_COMPAT=1` in `brain_integration.go`
- **File:** [`daemon/brain_integration.go:227`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/brain_integration.go#L224-L233)
- **Note:** This file is a W1 hotspot (excluded from A8 editing scope). The env var is also defined/used in `gateway/model_projection.go` (W2 scope). Default OFF (env var unset = production path). Only active when operator explicitly sets `OMNIROUTE_DEV_MODELS_COMPAT=1`.
- **Verdict:** ✅ OUT_OF_SCOPE (W1/W2). Documented here for completeness only. Default-off, explicit opt-in.

#### F5: Localhost default DB URLs in backfill/migrate utilities
- **Files:**
  - [`cmd/backfill_codex_usage_cache/main.go:83`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/cmd/backfill_codex_usage_cache/main.go#L81-L84)
  - [`cmd/backfill_task_usage_hourly/main.go:76`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/cmd/backfill_task_usage_hourly/main.go#L74-L77)
  - [`cmd/migrate/main.go:122`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/cmd/migrate/main.go#L120-L123)
- **Pattern:** `dbURL = "postgres://multica:multica@localhost:5432/multica?sslmode=disable"` used when `DATABASE_URL` is unset.
- **Analysis:** These are standalone CLI utilities (backfill scripts, migration tool), **not** the production server binary (`cmd/server/main.go`). The server itself reads `DATABASE_URL` from the environment. The docker-compose.selfhost.yml enforces `POSTGRES_PASSWORD:?` (required). These dev defaults in CLI utilities are standard localhost dev ergonomics — the same pattern used by most Go CLI tools.
- **Verdict:** ✅ Not a production-reachable concern. These binaries are developer/operator tools, not the server runtime. The production server does not use these defaults.

#### F6: `is_demo` field in analytics
- **File:** [`internal/analytics/events.go:86`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/analytics/events.go#L72-L87)
- **Analysis:** A legitimate PostHog analytics segmentation field. Always stamped (including `false`) so dashboards can filter demo data. This is analytics metadata, not a mock/fake-success path.
- **Verdict:** ✅ Legitimate production code. Not a mock.

#### F7: `MULTICA_LARK_HTTP_BASE_URL` override
- **File:** [`cmd/server/router.go:230-238`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/cmd/server/router.go#L229-L238)
- **Analysis:** Optional operator knob to force Lark API base URL (e.g. for CI mocks or single-cloud staging). Defaults to empty (real regional endpoints). Comment-documented.
- **Verdict:** ✅ Legitimate operator configuration. Defaults to production behavior.

#### F8: `scope_authorizer.go` "fake" comment
- **File:** [`cmd/server/scope_authorizer.go:14`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/cmd/server/scope_authorizer.go#L12-L14)
- **Analysis:** Code comment explaining the interface exists so authorizer can be unit-tested "with an in-memory fake (no DB required)." No mock is used in production — this is standard Go testability design.
- **Verdict:** ✅ Comment only. No mock in production path.

#### F9: `cmd_runtime_profile.go` TODO(MUL-3284)
- **File:** [`cmd/multica/cmd_runtime_profile.go:106`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/cmd/multica/cmd_runtime_profile.go#L103-L107)
- **Analysis:** `--fixed-arg` CLI flag intentionally NOT exposed because daemon doesn't yet pass these args. The flag is absent, not a no-op path — users cannot reach it. The column exists in DB but the CLI deliberately omits the flag.
- **Verdict:** ✅ Unreachable feature-gate. No fake-success — the flag simply doesn't exist in the CLI.

### 3.2 Deploy/Config Artifacts — Cleared

| Artifact | Finding |
|---|---|
| `deploy/helm/multica/values.yaml` | `appEnv: production` default. Empty secret refs (`existingSecret: multica-secrets`). No hardcoded secrets. `multica.dev.lan` is a documented placeholder hostname for self-host — does not reach production without operator configuration. |
| `docker-compose.yml` | Dev-only Postgres with `${POSTGRES_PASSWORD:-multica}` — this is the local dev compose, not selfhost. |
| `docker-compose.selfhost.yml` | `POSTGRES_PASSWORD:?` required. `JWT_SECRET:?` required (min 32 bytes). `APP_ENV:-production`. All ports `127.0.0.1` only. Well-hardened. |
| `docker-compose.selfhost.build.yml` | Build-only compose — no runtime secrets. |
| `docker-compose.override.yml` | Minimal dev override. |
| `Dockerfile` / `Dockerfile.web` | Standard multi-stage builds. No embedded secrets or dev flags. |
| `.goreleaser.yml` | Release packaging only. No secrets or mock paths. |
| `docker/entrypoint.sh` | Simple: `./migrate up && exec ./server`. No debug/dev branches. |
| `deploy/observability/.env.example` | Non-secret config with explicit "Replace-with..." instructions in secrets/. `.gitignore` excludes actual secrets. |
| `deploy/observability/secrets/*.example` | Placeholder text `Replace-with-a-strong-Grafana-admin-password` etc. These are `.example` files, not deployed secrets. |

### 3.3 HTTP Endpoint Audit — Cleared

Searched production router code (`cmd/server/router.go`) for debug/test/QA/dev/pprof/admin endpoints. All matches are in `_test.go` files only. **No debug or test-only HTTP endpoints are registered in the production router.**

### 3.4 Mock/Fake Patterns in Non-Test Go Source — Cleared

All `mock`, `fake`, `stub`, `dummy` patterns found in non-test Go source are either:
- Comments explaining testability design (e.g. scope_authorizer.go)
- Code in `_test.go` files (excluded from production binary)
- In `gateway/` or `runtimeenv/` packages (outside A8 scope)

**Zero production-reachable mock/fake-success/stub code was found in A8 scope.**

## 4. Conclusion

**STATUS: NO_REACHABLE_RESIDUAL**

The backend/config/deploy surface outside W1/runtimeenv/gateway ownership contains **zero production-reachable** instances of:
- Mocks
- Fake-success paths
- QA-only routes
- Placeholder data reaching production
- Demo persistence
- Synthetic defaults that bypass production guards

Every dev/test convenience mechanism found (JWT default secret, dev verification code, local auth bypass, localhost DB defaults) is properly guarded by fail-closed checks that prevent activation in production (`APP_ENV=production`).

## 5. W1 Handoff

No corrective changes required. A8 has no edits to hand off to W1.

**Items flagged for awareness (no action by A8):**
- `OMNIROUTE_DEV_MODELS_COMPAT` in `brain_integration.go` (W1 hotspot) and `gateway/model_projection.go` (W2) — default off, explicit opt-in only. W1/W2 may consider adding a startup warning when this is set in production, but this is their ownership decision.

## 6. Evidence

### Searches Executed
```
rg -i --glob '*.go' --glob '!*_test.go' 'mock' server/
rg -i --glob '*.go' --glob '!*_test.go' 'fake' server/
rg -i --glob '*.go' --glob '!*_test.go' 'placeholder|demo[^n]|stub|dummy|sample|hardcoded|qa.only|qa_only|test.mode|test_mode' server/
rg -i --glob '*.go' --glob '!*_test.go' 'fake.?success|fake.?result|always.?return.?true' server/
rg -i --glob '*.go' --glob '!*_test.go' 'TODO|FIXME|HACK|XXX' server/
rg -i --glob '*.go' --glob '!*_test.go' 'localhost|127\.0\.0\.1|example\.com' server/
rg -i --glob '*.go' --glob '!*_test.go' 'devMode|dev.mode|debug.mode|test.only|enableDebug|debugEnabled|skipAuth|skip.auth|bypass' server/
rg -i --glob '*.go' --glob '!*_test.go' 'default.*password|default.*secret|default.*token|default.*key.*=' server/
rg -i --glob '*.go' --glob '!*_test.go' 'HandleFunc|Handle|Get|Post|Put|Delete|Route|Mount).*(debug|test|qa|dev|internal|pprof|admin)' cmd/server/
rg 'ValidateJWTConfiguration' server/ (caller verification)
rg 'DEV_VERIFICATION_CODE' server/ (production guard verification)
rg 'MULTICA_LOCAL_AUTH_BYPASS' server/ (loopback guard verification)
```

### Files Inspected (read-only)
- `server/internal/auth/jwt.go` — JWT secret handling, full file
- `server/internal/handler/auth.go:170-203` — dev verification code
- `server/internal/middleware/auth.go:20-55` — local auth bypass
- `server/internal/daemon/brain_integration.go:215-245` — dev compat flag (W1 scope)
- `server/cmd/server/main.go:125-138` — startup guards
- `server/cmd/server/router.go:220-250` — Lark mock comment
- `server/cmd/server/scope_authorizer.go:1-25` — testability comment
- `server/cmd/multica/cmd_runtime_profile.go:100-113` — TODO no-op
- `server/internal/analytics/events.go:60-100` — is_demo field
- `server/cmd/backfill_codex_usage_cache/main.go:70-100` — localhost default
- `server/cmd/backfill_task_usage_hourly/main.go:65-90` — localhost default
- `server/cmd/migrate/main.go:110-135` — localhost default
- `server/internal/daemon/deploy/*` — deploy package (clean)
- `deploy/helm/multica/values.yaml` — full file
- `docker-compose.yml` — full file
- `docker-compose.selfhost.yml` — full file
- `docker/entrypoint.sh` — full file
- `deploy/observability/.env.example` — full file
- `deploy/observability/secrets/*.example` — all 3 files

### Validation
No source edits produced. Audit is read-only. All findings classified per the five-category scheme from P0_MAIN_BRAIN_EXECUTION.md §3.A.

## 7. Blockers / Limitations

- Go 1.26.1 verified at `/home/ec2-user/goroot/go/bin/go`; available for future focused validation.
- **No source edits in scope:** This phase was read-only by design. No corrections needed.
