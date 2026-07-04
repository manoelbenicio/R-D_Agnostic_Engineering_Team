<role>
You are CODEX#2, senior Go engineer. Build the fallback retry/backoff mechanics + error
classification for the rotation router. NEW file only. "Done" = deterministic functions + tests, green.
</role>

<mandatory_signin_signout priority="0" optional="false">
- BEFORE editing: write /mnt/c/VMs/Projetos/Automonous_Agentic/.deploy-control/CODEX-2__RR-FALLBACK__<START_UTC>.md
  (ABSOLUTE path; START_UTC=`date -u +%Y%m%dT%H%M%SZ`) with agent+started_at+status:IN_PROGRESS+files_locked.
- AFTER: same file with finished_at + agent + status:DONE|BLOCKED + build_result.
- No started_at+finished_at+agent = NOT complete.
</mandatory_signin_signout>

<lock_discipline>
files_locked (NEW): server/internal/rotation/fallback.go, fallback_test.go
Do NOT edit detector.go/contract.go/service.go/pool.go. New file only (may READ detector.go).
</lock_discipline>

<context source="openspec/changes/rotation-router/design.md §3 — read first; real params from Requesty public doc">
backoff: 500ms→1s→2s→4s (cap 4s). jitter: ±10%. retries per item: 0–10.
RETRY on: 429 / timeout / 503 (transient). FAILOVER_NOW on: 401/400/auth (non-retryable).
</context>

<task>
Own types (do NOT touch contract.go):
  type RetryDecision int  // RETRY, FAILOVER_NOW
  func NextBackoff(attempt int) time.Duration          // 500ms,1s,2s,4s, cap 4s
  func Jitter(d time.Duration) time.Duration           // ±10%, deterministic-testable (inject rand or clamp)
  func ClassifyError(err error, httpStatus int) RetryDecision
  type RetryPlan struct { MaxRetries int }
  func (rp RetryPlan) ShouldRetry(attempt int) bool     // attempt < MaxRetries
Deterministic. For Jitter testability, allow an injectable source or expose the ±10% bounds so
tests assert range. No secrets in logs. Invent nothing beyond design.md.
</task>

<example>
```
// NextBackoff(0)=500ms, (1)=1s, (2)=2s, (3)=4s, (4)=4s(cap)
// Jitter(1s) within [900ms,1100ms]
// ClassifyError(nil,429)=RETRY ; ClassifyError(nil,401)=FAILOVER_NOW ; (nil,503)=RETRY ; (nil,400)=FAILOVER_NOW
// RetryPlan{2}.ShouldRetry(0)=true, ShouldRetry(2)=false
```
</example>

<verification>
cd /mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work
docker run --rm -v "$PWD":/src -w /src/server golang:1.26-alpine sh -c "go build ./internal/rotation/... && go vet ./internal/rotation/... && go test ./internal/rotation/ -run Fallback -v" 2>&1 | tail -15
Paste tail. DONE only on green.
</verification>

<persistence>
Finish fully; fix-and-rerun on red; never DONE on red; BLOCKED only on true blocker.
</persistence>

<output>Sign-out: agent CODEX#2, started_at, finished_at (UTC), status DONE, green tail in build_result.</output>
