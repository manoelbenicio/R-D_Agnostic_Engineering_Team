<role>
You are Codex#5.5#A, senior Go engineer. Build weighted + window-health load-balancing with
consistent hashing for the rotation router. NEW file only. "Done" = deterministic selection +
tests, green in container.
</role>

<mandatory_signin_signout priority="0" optional="false">
- BEFORE editing: write /mnt/c/VMs/Projetos/Automonous_Agentic/.deploy-control/Codex-5.5-A__RR-LOADBALANCE__<START_UTC>.md
  (ABSOLUTE path; START_UTC=`date -u +%Y%m%dT%H%M%SZ`). Front-matter PLANO (sem bullets):
  agent: Codex#5.5#A / stream: RR-LOADBALANCE / started_at: <UTC> / finished_at: / status: IN_PROGRESS
  / files_locked: / build_result: / notes:  — UM único arquivo por stream.
- AFTER: same file with finished_at + status:DONE|BLOCKED + build_result. No timestamps+agent = NOT done.
</mandatory_signin_signout>

<lock_discipline>
files_locked (NEW): server/internal/rotation/loadbalance.go, loadbalance_test.go
Depends on RR-POLICY (uses RotationPolicy/PolicyItem — already merged, READ-only). Do NOT edit
policy.go/fallback.go/contract.go/service.go/pool.go. New file only.
</lock_discipline>

<context source="/mnt/c/VMs/Projetos/Automonous_Agentic/openspec/changes/rotation-router/design.md §5 — ABSOLUTE path, read first">
Objective is NOT A/B by weight (Requesty) — it is MAX AGGREGATE THROUGHPUT of the subscription
pool: balance the 5h windows so they exhaust together. Consistency via deterministic hashing.
</context>

<task>
Own funcs (do NOT touch contract.go/policy.go):
  func PickWeighted(p RotationPolicy, seed string) PolicyItem   // weight-normalized, deterministic by seed
  func PickConsistent(p RotationPolicy, traceID string) PolicyItem // xxhash(traceID)→same item (session affinity)
  func PickByWindowHealth(items []PolicyItem, health map[string]float64) PolicyItem
      // health = fraction of quota-window REMAINING per accountRef; pick to equalize (spread load
      // toward the healthiest, so windows drain together). Document the rule.
Use a real xxhash (github.com/cespare/xxhash/v2 already in go.mod) OR stdlib fnv if xxhash
unavailable — confirm what's in go.mod first; do NOT add a new dependency without checking.
Deterministic. No secrets. Only design.md §5.
</task>

<example>
```
// PickConsistent(p,"task-42") == PickConsistent(p,"task-42")  (stable, same item)
// distribution over many seeds ~ weights (±tolerance)
// PickByWindowHealth: acct with more window remaining is preferred so windows equalize
// empty policy items → returns zero PolicyItem, no panic
```
</example>

<verification>
cd /mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work
docker run --rm -v "$PWD":/src -w /src/server golang:1.26-alpine sh -c "go build ./internal/rotation/... && go vet ./internal/rotation/... && go test ./internal/rotation/ -run 'LoadBalance|Pick' -v" 2>&1 | tail -15
Paste tail. DONE only on green.
</verification>

<persistence>Finish fully; fix-and-rerun on red; never DONE on red; BLOCKED only on true blocker.</persistence>
<output>Sign-out: agent Codex#5.5#A, started_at, finished_at (UTC), status DONE, green tail in build_result.</output>
