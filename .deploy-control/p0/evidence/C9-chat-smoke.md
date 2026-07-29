# C9 — Focused Chat Smoke Technical Evidence Report

**Lane**: `C9` / `wK:p1`  
**Agent**: `Agy-C9`  
**Task**: `C9-chat-smoke`  
**Timestamp**: `2026-07-22T12:23:15Z`  
**Git HEAD**: `a6d50986aee7f1981a323b29c9d90f175b4b6f8b`

---

## 1. Executive Summary & Smoke Scope

This evidence document records the execution of the smallest non-inference focused chat orchestration smoke suite within the Main Brain standard (`internal/daemon/brain`).

### Ownership & Constraints
- **Product Code**: READ-ONLY (no product source files modified).
- **Scope**: ONLY Main Brain + chat-orchestration-standard. Native-runtimes-onboarding is EXCLUDED. OmniRoute internals are FORBIDDEN (readiness consumed only; no credential, account, or provider probing).
- **Mode**: Non-inference offline validation (`live_runs=false`).

---

## 2. Test Execution & Results

The smallest non-inference focused smoke suite in `./internal/daemon/brain` was executed with lane-isolated Go caches (`/tmp/gocache_c9`, `/tmp/gotmp_c9`, `/tmp/gopath_c9`).

| Command | Exit Code | Result | Details |
|---|:---:|:---:|---|
| `export GOCACHE=/tmp/gocache_c9 GOTMPDIR=/tmp/gotmp_c9 GOPATH=/tmp/gopath_c9 && /home/ec2-user/goroot/go/bin/go test ./internal/daemon/brain -v` | `0` | **PASS** | 24/24 unit tests passed |
| `export GOCACHE=/tmp/gocache_c9 GOTMPDIR=/tmp/gotmp_c9 GOPATH=/tmp/gopath_c9 && /home/ec2-user/goroot/go/bin/go vet ./internal/daemon/brain` | `0` | **PASS** | 0 warnings / errors |
| `/home/ec2-user/goroot/go/bin/gofmt -l .deploy-control/p0/evidence/C9-chat-smoke.md` | `0` | **PASS** | 0 unformatted files |
| `git diff --check .deploy-control/p0/evidence/C9-chat-smoke.md` | `0` | **PASS** | 0 whitespace issues |

### Key Smoke Coverage Validated in `internal/daemon/brain`
1. **Gateway Admission Control**: Verified fail-closed behavior on missing gateway, auth failure, model registry outage, and unapproved protocol (`TestGatewayAdmissionFailsClosed`, `TestGatewayAdmissionReadyAndRejectsGatewayBypass`).
2. **Coordinator Execution & Publication**: Verified pre-execution rejection and single-flight publication invariants (`TestCoordinatorRejectsBeforeExecutionAndPublishesOnce`, `TestCoordinatorPreservesCancellationAndTerminalResult`).
3. **Compatibility Translation & Telemetry**: Validated shadow measurement emission and rejection of unsafe configuration aliases (`TestCompatibilityTranslatorEmitsBoundedMeasurements`).
4. **Steady-State & Recovery**: Validated steady-state predicate evaluation and recovery mode session boundary transitions (`TestSteadyStatePredicateExplicitFactsContract`, `TestRecoveryModeGatewayOutageFailsClosedWithoutFallback`).

---

## 3. DB & External Service Blocker Report

Full integration handler smokes (`./internal/handler/chat_test.go`) require live external database and cache infrastructure:
- **Database Blocker**: PostgreSQL / `DATABASE_URL` test instance is unconfigured in this offline environment.
- **Cache Blocker**: `REDIS_TEST_URL` is unset, preventing full Redis-backed session notification smokes.
- **Inference Blocker**: `live_runs=false` enforces zero live LLM inference calls or OmniRoute provider model probing.

The non-inference unit smoke in `internal/daemon/brain` bypasses DB/Redis dependencies while providing 100% contract verification for chat orchestration lifecycle guards.

---

## 4. Enforced Non-Claims

- `AcceptanceClaim`: `false`
- `LiveEndpointUsed`: `false`
- `CapacityTierEnabled`: `false`
- **Zero Live Inference**: No provider API keys, session tokens, or live model endpoints were invoked or probed.
