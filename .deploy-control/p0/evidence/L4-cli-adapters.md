# L4 — CLI adapters verification + minimal fixes (evidence)

- agent: Codex56#B · lane: L4 · task: L4-CLI-ADAPTERS · pane: w7:p4 (HERDR_ENV=1)
- ownership (exclusive): `server/pkg/agent/{claude,codex,kimi,nim,antigravity}.go` + their `*_test.go`. EXCLUDES `models.go` (L1).
- repo HEAD at check-in: `a6d50986aee7f1981a323b29c9d90f175b4b6f8b`
- OpenSpec: 5.1 (gofmt), 5.2 (targeted tests); AB-REQ-07,19,21
- toolchains: go `go1.26.1` (`/home/ec2-user/goroot/go/bin/go`), node `v22.23.1`; disk_free 2.6G (90% used)
- constraints honored: no deploy/restart/Docker/systemd; no inference; no secret read/print/hash; no commit/push/merge; no reset/stash/revert/clean; one owner per file; no models.go/OpenSpec/GSD edits.

## Verification performed (argv/log structural redaction + protocol config)

| Adapter | argv/log redaction finding | Result |
|---|---|---|
| claude.go | logs via `safeAgentArgvForLog` (`logAgentCommand`); `sensitiveAgentArgValueFlags` + inline markers redact values | REVIEWED safe; test `TestClaudeCommandLoggingRedactsSensitiveArgv` PASS. Left unmodified. |
| codex.go | redacted command-log projection present; mcp-server command protection maps repeated-extension forms back to protected agent names | test `TestCodexCommandLoggingRedactsSensitiveArgv` PASS; protected-command test fixed (see below) |
| kimi.go | launch argv = `["acp"]` + custom args filtered by `kimiBlockedArgs`; prompt/model/thinking travel via ACP JSON-RPC/stdin/env, NOT argv | REVIEWED safe (no sensitive value in argv); left unmodified |
| nim.go | HTTP adapter (no CLI spawn / no argv launch log) | REVIEWED safe; left unmodified |
| antigravity.go | **DEFECT**: `buildAntigravityArgs` puts `-p <prompt>` (full user/repo content) in argv and line 73 logged it **raw** — sensitive content in logs | FIXED (see below) |

## Changes (all within L4 ownership; models.go untouched; claude.go untouched)

1. `pkg/agent/kimi_test.go` — `gofmt` (removed stray double blank line + added EOF newline). Task 5.1.
2. `pkg/agent/nim_test.go` — `gofmt` (same). Task 5.1.
3. `pkg/agent/codex_test.go` — `TestRenderCodexMcpServersBlockRejectsProtectedAgentCommands` "repeated Windows extensions" case: `/opt/bin/wrapper.cmd.exe` → `/opt/bin/codex.cmd.exe`. The in-progress debranding edit removed `prodex` from the protected set (codex.go, working tree) and changed this case to a non-agent name (`wrapper`), which broke the test's own contract (`RejectsProtectedAgentCommands`). Restored a still-protected agent name so the repeated-extension evasion coverage is preserved. Alternative (if the Principal instead intends a broader "reject any repeated executable extension" rule) is noted below.
4. `pkg/agent/antigravity.go` — added antigravity-local `safeAntigravityArgvForLog` + `antigravitySensitiveValueFlags` (`-p`,`--print`,`--prompt`,`--conversation`); routed the launch log through it (`--model` already covered by the shared sensitive set). Reuses package primitives `redactedAgentArgValue`, `isSensitiveAgentArgValueFlag`, `redactSensitiveInlineArg`. NOTE: the fix is antigravity-local by design — `-p` is `blockedStandalone` in claude.go (no value) but `blockedWithValue` in antigravity, so adding `-p` to the shared `sensitiveAgentArgValueFlags` would mis-redact claude and break `TestClaudeCommandLoggingRedactsSensitiveArgv`.
5. `pkg/agent/antigravity_test.go` — added `TestSafeAntigravityArgvForLogRedactsSensitiveValues` (prompt/model/conversation values redacted; flag shape preserved; real argv unchanged).

## Commands + exit codes

- `gofmt -l <owned files>` → clean (empty) after fixes. exit 0
- `gofmt -w kimi_test.go nim_test.go` → exit 0
- `go vet ./pkg/agent/` → exit 0
- `go test ./pkg/agent/ -run 'TestSafeAntigravityArgvForLogRedactsSensitiveValues|TestClaudeCommandLoggingRedactsSensitiveArgv|TestCodexCommandLoggingRedactsSensitiveArgv' -count=1 -v` → all PASS, exit 0
- `go test ./pkg/agent/ -count=1` (full package) → `ok ... 7.732s`, exit 0
- `git diff --check -- multica-auth-work/server/pkg/agent/` → exit 0 (no whitespace errors)
- `git diff --stat` → 5 files changed, 97 insertions(+), 6 deletions(-)

## Findings routed to manager (NOT in L4 ownership — no edit)

- `gofmt -l` flags `pkg/agent/codebuddy_test.go` and `pkg/agent/cursor_test.go` as unformatted (task 5.1). These adapters are outside L4's `{claude,codex,kimi,nim,antigravity}` scope. Route to their owner (L1/other) for a one-line `gofmt -w`.

## Alternatives / non-claims

- codex protected-command fix assumes the intended contract is name-based protection (supported by allow-test permitting single-extension non-agents and by codex.go having no repeated-extension-count rule). If the Principal intends to reject ANY command with ≥2 stacked executable extensions regardless of name, that requires a codex.go logic change + a distinct error/allow-test — flagged, not implemented.
- No live run, no inference, no deploy. No secret read. models.go and claude.go unmodified.
