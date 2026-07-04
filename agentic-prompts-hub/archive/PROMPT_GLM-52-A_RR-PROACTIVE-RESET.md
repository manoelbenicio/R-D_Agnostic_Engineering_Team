<role>
You are GLM#52#A, engineer. Build the proactive-rotation hook + a GATED reset-claim interface
for the rotation router. NEW file only. "Done" = deterministic proactive decision + no-op
ResetClaimer (claim mechanism NOT confirmed → gated) + tests with fakes, green in container.
</role>

<mandatory_signin_signout priority="0" optional="false">
- BEFORE editing: write /mnt/c/VMs/Projetos/Automonous_Agentic/.deploy-control/GLM-52-A__RR-PROACTIVE-RESET__<START_UTC>.md
  (ABSOLUTE path). Front-matter PLANO (sem bullets, UM arquivo por stream):
  agent: GLM#52#A / stream: RR-PROACTIVE-RESET / started_at: <UTC> / finished_at: / status: IN_PROGRESS
  / files_locked: / build_result: / notes:
- AFTER: same file with finished_at + status:DONE|BLOCKED + build_result. No timestamps+agent = NOT done.
</mandatory_signin_signout>

<lock_discipline>
files_locked (NEW): server/internal/rotation/proactive_reset.go, proactive_reset_test.go
READ-only reuse: proactive.go, warnbanner.go, usage.go, probe_codex.go. Do NOT edit them or
contract.go/service.go/pool.go/daemon.go. New file only.
</lock_discipline>

<context source="/mnt/c/VMs/Projetos/Automonous_Agentic/openspec/changes/rotation-router/design.md §7 — ABSOLUTE, read first; also docs/project/BACKLOG-detection.md">
Proactive = rotate BEFORE the vendor hard-stop, by reading quota (banner/usage panel/ledger).
CLAIM-DE-RESET (Codex "N usage limit resets available") is TUI-only → headless mechanism NOT
confirmed. Therefore the claim is GATED (no-op), NOT implemented. Do NOT invent a claim command.
</context>

<task>
Own types/funcs (do NOT touch contract.go):
  type ProactiveSignal struct { Approaching bool; Source string; ResetsAvailable int }
  func EvaluateProactive(vendor, screenText string, now time.Time) ProactiveSignal
      // reuse WarningDetector/UsageDetector/ParseCodexUsage (read-only) to decide "approaching".
  type ResetClaimer interface { ClaimReset(ctx context.Context, acc Account) (bool, error) }
  type NoopResetClaimer struct{}   // ClaimReset returns (false, nil) — GATED; documents why:
      // "/usage is TUI-only; headless claim mechanism CONFIRMAR CONTRA BINÁRIO (app-server RPC?)"
  func DecideProactive(sig ProactiveSignal, claimer ResetClaimer, acc Account, ctx) Decision
      // if ResetsAvailable>0 → try claimer.ClaimReset FIRST (with no-op it returns false) →
      //   claimed → KEEP account; not claimed → signal ROTATE. If Approaching w/o resets → ROTATE.
  type Decision int  // KEEP, ROTATE, NONE
Deterministic. No secrets. Only design.md §7. NEVER invent a headless claim command.
</task>

<example>
```
// Approaching=true, ResetsAvailable=2, NoopResetClaimer → claim returns false → Decision=ROTATE
//   (documented: real claimer would KEEP; no-op forces ROTATE until headless claim confirmed)
// Approaching=true, ResetsAvailable=0 → Decision=ROTATE
// Approaching=false → Decision=NONE
// fake claimer returning (true,nil) → Decision=KEEP  (proves the KEEP path works)
```
</example>

<verification>
cd /mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work
docker run --rm -v "$PWD":/src -w /src/server golang:1.26-alpine sh -c "go build ./internal/rotation/... && go vet ./internal/rotation/... && go test ./internal/rotation/ -run 'Proactive|Reset|Claim' -v" 2>&1 | tail -15
Tests use FAKES (no real CLI/network). Paste tail. DONE only on green.
</verification>

<persistence>
Finish fully; fix-and-rerun on red; never DONE on red. Do NOT implement a real headless claim —
keep it gated/no-op and document the CONFIRMAR-CONTRA-BINÁRIO note. BLOCKED only on true blocker.
</persistence>
<output>Sign-out: agent GLM#52#A, started_at, finished_at (UTC), status DONE, green tail in build_result.</output>
