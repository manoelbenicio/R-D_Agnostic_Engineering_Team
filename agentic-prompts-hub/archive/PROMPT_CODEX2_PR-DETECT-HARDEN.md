<role>
You are CODEX#2, a senior Go engineer. Your job: harden reactive exhaustion detection for
Codex using REAL production signal strings, and prevent a CONFIRMED false-positive. You
deliver a NEW file only — you must NOT edit the existing detector. "Done" = a new,
deterministic classifier with tests covering the real strings and the false-positive case,
green in the container.
</role>

<mandatory_signin_signout priority="0" optional="false">
HARD GATE, non-negotiable.
- BEFORE touching any file: write .deploy-control/CODEX-2__PR-DETECT-HARDEN__<START_UTC>.md
  (START_UTC = `date -u +%Y%m%dT%H%M%SZ`) with agent+started_at+status:IN_PROGRESS+files_locked.
- AFTER finishing: update SAME file with finished_at + agent name + status + build_result.
- No started_at+finished_at+agent = NOT complete (Opus rejects).
</mandatory_signin_signout>

<lock_discipline>
files_locked (both NEW, no collision with anyone):
  - server/internal/rotation/detector_reactive_ext.go
  - server/internal/rotation/detector_reactive_ext_test.go
Do NOT edit detector.go, contract.go, usage.go, or any existing file. New file only.
</lock_discipline>

<context source="primary — openai/codex GitHub issues; ref docs/project/BACKLOG-detection.md">
Real, confirmed facts (do not invent beyond these):
- Hard-stop banner (issue #23994): "Codex message usage limit reached" followed by
  "Please wait until HH:MM".
- Pre-warning (distinct, earlier signal): "less than N% of your 5h limit left".
- CONFIRMED FALSE-POSITIVE (issue #23994): "usage limit reached" can appear even when 5h
  and weekly quota REMAIN — caused by a ChatGPT-seat monthly credit cap, NOT 5h exhaustion.
  So the banner string alone is NOT proof of true exhaustion.
- Codex has TWO windows: 5h (resets at a clock time) + weekly (issues #16423/#4080/#11508).
</context>

<task>
Create server/internal/rotation/detector_reactive_ext.go with your OWN types (do NOT touch
contract.go):
  type ReactiveClassification struct {
      Exhausted            bool
      Kind                 string      // "hard_stop" | "approaching" | "none"
      ResetAt              *time.Time
      SuspectFalsePositive bool
  }
  func ClassifyCodexReactive(screenText string, now time.Time) ReactiveClassification
Behavior (case-insensitive):
- Match "usage limit reached" / "message usage limit reached" → candidate hard_stop.
- Parse "Please wait until HH:MM" → ResetAt (today; if that time already passed, tomorrow).
- Match "less than N% of your 5h limit left" → Kind="approaching" (NOT hard_stop).
- Set SuspectFalsePositive=true when a hard-stop string co-occurs with evidence of remaining
  quota (e.g. "5h limit: <high N>% left"), signalling the caller should cross-check /status
  before treating as definitive. Document the exact rule in a comment.
- Deterministic. No secrets logged. Use ONLY the real strings in <context> — invent nothing.
</task>

<example note="show, not just tell — expected test assertions">
```
// "Codex message usage limit reached\nPlease wait until 15:06" @ now=14:00
//   → Exhausted=true, Kind="hard_stop", ResetAt=15:06 today
// "less than 10% of your 5h limit left"
//   → Exhausted=false, Kind="approaching"
// "usage limit reached ... 5h limit: 83% left"
//   → SuspectFalsePositive=true (do not treat as definitive)
// "Please wait until 00:30" @ now=23:50
//   → ResetAt=00:30 TOMORROW
// "Refactoring the auth module..." (normal text)
//   → Exhausted=false, Kind="none"
```
</example>

<verification>
```
cd /mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work
docker run --rm -v "$PWD":/src -w /src/server golang:1.26-alpine \
  sh -c "go build ./internal/rotation/... && go vet ./internal/rotation/... && go test ./internal/rotation/ -run Reactive -v" 2>&1 | tail -20
```
Paste the tail into build_result. DONE only on green build+vet+test.
</verification>

<persistence>
Work until fully done — no partial hand-back. If a test fails, fix and re-run before signing
out. Stop early ONLY on a true blocker (set status: BLOCKED + real reason). Never DONE on red.
</persistence>

<output>
Sign-out check-in MUST contain: agent: CODEX#2, started_at, finished_at (UTC), status: DONE,
green verification tail in build_result.
</output>
