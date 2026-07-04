<role>
You are Codex#5.5#A, senior Go engineer. Build latency-based selection SCORING for the rotation
router (pure logic, injectable measurements). NEW file only. Do NOT instrument the daemon here —
real TTFT capture is a separate serial step. "Done" = deterministic scoring + tests with fakes, green.
</role>

<mandatory_signin_signout priority="0" optional="false">
- BEFORE editing: write /mnt/c/VMs/Projetos/Automonous_Agentic/.deploy-control/Codex-5.5-A__RR-LATENCY__<START_UTC>.md
  (ABSOLUTE path; START_UTC=`date -u +%Y%m%dT%H%M%SZ`). Front-matter PLANO (sem bullets, UM arquivo):
  agent: Codex#5.5#A / stream: RR-LATENCY / started_at: <UTC> / finished_at: / status: IN_PROGRESS
  / files_locked: / build_result: / notes:
- AFTER: same file with finished_at + status:DONE|BLOCKED + build_result.
</mandatory_signin_signout>

<lock_discipline>
files_locked (NEW): server/internal/rotation/latency.go, latency_test.go
READ-only: policy.go. Do NOT edit contract.go/service.go/pool.go/daemon.go/execenv. New file only.
</lock_discipline>

<context source="/mnt/c/VMs/Projetos/Automonous_Agentic/openspec/changes/rotation-router/design.md §5 (LATENCY) — ABSOLUTE, read first">
Latency selection = pick fastest vendor/account NOW. Score from TTFT + generation speed, rolling
~1h window weighting recent, OPTIMISTIC score for cold targets (no data) so they still get tried.
This stream is the SCORING ONLY — measurements are injected (a fake in tests). The real TTFT
capture (daemon/execenv) is a SEPARATE serial stream, NOT here.
</context>

<task>
Own types/funcs (do NOT touch contract.go):
  type LatencySample struct { TTFT time.Duration; GenSpeedTokPerSec float64; At time.Time }
  type LatencyStats interface { Recent(target string, now time.Time) []LatencySample }  // injectable
  func Score(target string, stats LatencyStats, now time.Time) float64   // lower=faster; cold=optimistic
      // rolling window ~1h; weight recent > old; if no samples → optimistic (e.g. 0 / best score).
  func PickFastest(targets []string, stats LatencyStats, now time.Time) string
Deterministic given injected stats. No secrets. Only design.md §5. Do NOT measure anything live
(no exec, no daemon) — that's the separate instrumentation stream.
</task>

<example>
```
// target with recent low TTFT scores better (picked) than one with high TTFT
// cold target (no samples) gets optimistic score → gets tried
// old samples (>1h) weigh less than fresh ones
// PickFastest([]) → "" , no panic
```
</example>

<verification>
cd /mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work
docker run --rm -v "$PWD":/src -w /src/server golang:1.26-alpine sh -c "go build ./internal/rotation/... && go vet ./internal/rotation/... && go test ./internal/rotation/ -run 'Latency|Score|Fastest' -v" 2>&1 | tail -15
Paste tail. DONE only on green. Tests use a FAKE LatencyStats — no real timing.
</verification>

<persistence>Finish fully; fix-and-rerun on red; never DONE on red; BLOCKED only on true blocker. Do NOT instrument the daemon here.</persistence>
<output>Sign-out: agent Codex#5.5#A, started_at, finished_at (UTC), status DONE, green tail in build_result.</output>
