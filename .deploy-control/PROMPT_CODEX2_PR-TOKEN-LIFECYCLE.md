<role>
You are CODEX#2, a senior Go engineer. Your job: prevent "rotate to a dead account" — detect
whether a candidate account's credential is USABLE (not expired/logged-out) BEFORE rotation
selects it. You deliver a NEW file only. "Done" = a deterministic liveness/expiry checker with
tests, green in the container.
</role>

<mandatory_signin_signout priority="0" optional="false">
HARD GATE, non-negotiable.
- BEFORE touching any file: write .deploy-control/CODEX-2__PR-TOKEN-LIFECYCLE__<START_UTC>.md
  (START_UTC=`date -u +%Y%m%dT%H%M%SZ`) with agent+started_at+status:IN_PROGRESS+files_locked.
- AFTER finishing: update SAME file with finished_at + agent name + status + build_result.
- No started_at+finished_at+agent = NOT complete (Opus rejects).
</mandatory_signin_signout>

<lock_discipline>
files_locked (both NEW, no collision):
  - server/internal/rotation/token_lifecycle.go
  - server/internal/rotation/token_lifecycle_test.go
Do NOT edit contract.go, auth_authenticator.go, service.go, or any existing file. New file only.
</lock_discipline>

<context source="verified facts, Opus 2026-07-02 — invent nothing">
- Codex credential = an auth.json in the account's home dir. Real structure (keys, no secrets):
  { "auth_mode": "chatgpt", "OPENAI_API_KEY": <maybe null>, "tokens": {...}, "last_refresh": <RFC3339> }.
- VERIFIED: multiple accounts coexist — each account's auth.json works via its own CODEX_HOME;
  `codex login status` returns "Logged in using ChatGPT" for a valid one, "Not logged in" otherwise.
- PROBLEM this stream solves: rotation currently restores a credential FILE but never checks it
  is still valid; a stale token (e.g. weeks old) may be expired → rotating to it = dead account.
- The Account struct (contract.go, READ-ONLY): has HomeDir, ConfigDir, Vendor, etc.
</context>

<task>
Create server/internal/rotation/token_lifecycle.go with your OWN types (do NOT touch contract.go):
  type CredentialLiveness struct { Usable bool; Reason string; LastRefresh *time.Time; AgeDays int }
Provide TWO layers so it is unit-testable without invoking the real CLI:
- Pure/file layer: `func InspectCodexCredential(authJSONPath string, now time.Time) CredentialLiveness`
  — reads auth.json, parses last_refresh, computes AgeDays, flags Usable=false with Reason when:
  file missing, unparseable, or last_refresh older than a configurable staleness threshold
  (default e.g. 7 days — DOCUMENT the default; expired-token heuristic, not a network call).
  NEVER log token contents/email.
- Port for live check (injectable, so tests use a fake):
  `type LoginStatusChecker interface { Status(ctx, homeDir string) (loggedIn bool, err error) }`
  `func VerifyCodexLogin(ctx, homeDir string, checker LoginStatusChecker) CredentialLiveness`
  — real impl (small adapter) would run `CODEX_HOME=<homeDir> codex login status` and map
  "Logged in using ChatGPT" → true; but the checker is an INTERFACE so unit tests inject a fake.
  Do NOT hardcode exec in a way that tests need the real binary.
- Deterministic. No secrets in logs. Only the verified facts above.
</task>

<example note="expected assertions — show, not just tell">
```
// auth.json last_refresh = 21 days ago, threshold 7d → Usable=false, Reason="stale", AgeDays=21
// auth.json last_refresh = 1h ago → Usable=true, AgeDays=0
// missing file → Usable=false, Reason="missing"
// VerifyCodexLogin with fake checker returning loggedIn=false → Usable=false, Reason="not_logged_in"
// VerifyCodexLogin with fake checker returning loggedIn=true → Usable=true
```
</example>

<verification>
```
cd /mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work
docker run --rm -v "$PWD":/src -w /src/server golang:1.26-alpine \
  sh -c "go build ./internal/rotation/... && go vet ./internal/rotation/... && go test ./internal/rotation/ -run Liveness -v" 2>&1 | tail -20
```
Paste the tail into build_result. DONE only on green build+vet+test. Tests must NOT require the
real codex binary or network (use the injected fake checker + temp auth.json files).
</verification>

<persistence>
Work until fully done — no partial hand-back. If a test fails, fix and re-run before signing out.
Stop early ONLY on a true blocker (status: BLOCKED + real reason). Never DONE on red. Do NOT wire
this into service.go/daemon.go (that integration is a separate serial stream owned later).
</persistence>

<output>
Sign-out check-in MUST contain: agent: CODEX#2, started_at, finished_at (UTC), status: DONE,
green verification tail in build_result.
</output>
