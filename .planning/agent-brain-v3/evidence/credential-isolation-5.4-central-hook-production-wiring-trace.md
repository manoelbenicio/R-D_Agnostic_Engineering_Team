# Central logging-hook production wiring trace — credential 5.4

**Author:** Kiro/Sonnet, pane `w7:p1` — read-only source trace only.
**Date:** 2026-07-18T19:30:05-03:00
**Adjudication authority:** Kiro TL adjudicates. This document makes no
product/test/shared-planning/spec/task/git/index edit, reads no
credential/env value, and performs no network/DB/live-provider action.

## Golden Rule check-in / check-out

- **Check-IN:** 2026-07-18T19:30:05-03:00 — claimed scope: read-only
  enumeration of every production logger-construction path, direct slog
  handler construction, custom logger/writer type, standard `log`/`fmt`
  sink, and subprocess writer across `multica-auth-work/server`; determine
  which apply `redact.SanitizeSlogAttr`, which apply only `redact.Text`,
  and which apply neither. One output file only.
- **Tooling note:** every search below used the scoped grep tool with
  explicit `include` file-type filters and, where useful, path scoping —
  never an unscoped/broad `grep -r` — per the task's explicit instruction
  and consistent with the immediately prior session's steering correction.
- Excluded (honored): no product/test/shared-doc/spec/task edit; no
  git/index mutation of any kind (not even a read-only `git show` was
  needed for this trace); no credential/env value read; no DB/network/
  live-provider action.
- **Check-OUT:** 2026-07-18T19:55:00-03:00 — DONE; enumeration and
  classification below; no files other than this one were created or
  modified.

## Source hashes (files whose content is cited or classified below)

```
f409ba8a9f3e63618d59c5a8692296f8f7c019c9e558576b8786a058fbf68a5c  server/pkg/redact/redact.go
```
Only `pkg/redact/redact.go` was hashed in this pass (matches every prior
artifact in this session's chain — no drift). `internal/logger/logger.go`
was read directly for classification (see Limitations §3) but not
independently re-hashed in this pass — its content and `ReplaceAttr` wiring
were already hash-verified in multiple earlier artifacts this session.
Other files were read directly for classification, not hashed, since this
task's deliverable is an enumeration/classification, not a hash-integrity
audit (that already exists separately in
`credential-isolation-5.4-redact-core-current-diff-ownership.md` for the
`pkg/redact` files themselves).

## Method

Scoped `rg`-equivalent (the `grep` tool with `include` filters) searches,
each reproducible independently:

1. `slog.New(` / `slog.SetDefault(` — production vs. test split, by
   `include: *.go` then manually excluding `_test.go` matches from the
   result list (the tool does not support a native test-exclusion flag, so
   exclusion was done by inspecting each filename in the result set).
2. `NewTextHandler(` / `NewJSONHandler(` / `tint.NewHandler(` — same split.
3. Standard `log` package: `\blog\.Print|\blog\.Fatal|\blog\.Panic|\blog\.New\(`.
4. Custom writer types: `func \(.*\) Write\(p \[\]byte\)` (exact Go
   `io.Writer` method signature) scoped to `pkg/agent`.
5. Subprocess stderr wiring: `StderrPipe\(\)|cmd\.Stderr\s*=` scoped to
   `pkg/agent`.
6. `redact.Text(` / `redact.SanitizeSlogAttr` / `redact.SanitizeForLog` /
   `redact.InputMap(` call sites, repo-wide, `include: *.go`.
7. Targeted reads (not greps) of every distinct construct found by 1-6 to
   confirm actual behavior, not just presence of a call.

## Enumeration — logger/handler construction

### Production `slog.New(` call sites — exactly 2, both correctly wired

| Site | Handler | ReplaceAttr |
|---|---|---|
| `internal/logger/logger.go:38` (`Init()`) | `tint.NewHandler(os.Stderr, &tint.Options{...})` | `redact.SanitizeSlogAttr` |
| `internal/logger/logger.go:52` (`NewLogger(component)`) | `tint.NewHandler(os.Stderr, &tint.Options{...})` | `redact.SanitizeSlogAttr` |

**Every other `slog.New(`/`slog.NewTextHandler(`/`slog.NewJSONHandler(`
match in the repository is inside a `_test.go` file** (38 files matched the
broad pattern; manual inspection of the file list confirms `logger.go` is
the only non-test file among them — see the raw match list below for full
transparency rather than a bare claim).

Raw match-count transparency (scoped `rg`-equivalent, `include: *.go`,
pattern `slog\.New\(|slog\.SetDefault\(`): **100 total matches across 38
files**; of those 38 files, **exactly 1** (`internal/logger/logger.go`, 2
matches) is not a `_test.go` file. Pattern
`NewTextHandler\(|NewJSONHandler\(|tint\.NewHandler\(`: **112 matches
across 38 files**; same result — only `internal/logger/logger.go` (2
matches) is production.

**Conclusion: there is exactly one production logging entrypoint pair
(`Init`/`NewLogger`), both definitely apply `redact.SanitizeSlogAttr`** via
`ReplaceAttr`. This confirms and re-verifies (independently, via a fresh
scoped search rather than trusting the prior session's citation) the same
finding already established in the earlier 5.4 codebase audit and critique
this session.

## Enumeration — standard `log`/`fmt` sinks

- Pattern `\blog\.Print|\blog\.Fatal|\blog\.Panic|\blog\.New\(` scoped to
  `*.go`: **zero matches** anywhere in `multica-auth-work/server`
  (including tests). The standard library `log` package is not used at all
  in this codebase for diagnostic output — confirmed, not merely repeated
  from a prior artifact.
- `fmt.Print*` production usage was not re-enumerated in this pass (already
  independently traced and categorized in the prior 5.4 remaining-gaps
  closure design this session — CLI operator-terminal output plus the
  already-reviewed `email.go` dev-mode slice). Not re-derived here to avoid
  duplicating completed work, per this task's own "do not duplicate" spirit
  from the adjacent redact-core-ownership task.

## Enumeration — custom logger/writer types (the core of this task)

### `pkg/agent.logWriter` — ONE definition, shared by ALL 13 backend adapters

`type logWriter struct` and `func newLogWriter(...)` are defined **exactly
once**, in `pkg/agent/claude.go` (confirmed: no second definition anywhere
else in `pkg/agent` or elsewhere). Every other backend adapter **calls**
this same shared constructor rather than defining its own writer:

| Adapter file | Call site |
|---|---|
| `claude.go:206` | `newStderrTail(newLogWriter(b.cfg.Logger, "[claude:stderr] "), agentStderrTailBytes)` |
| `codex.go:620` | `newStderrTail(newLogWriter(b.cfg.Logger, "[codex:stderr] "), codexStderrTailBytes)` |
| `antigravity.go:86` | `newStderrTail(newLogWriter(b.cfg.Logger, "[agy:stderr] "), agentStderrTailBytes)` |
| `codebuddy.go:127` | `newStderrTail(newLogWriter(b.cfg.Logger, "[codebuddy:stderr] "), agentStderrTailBytes)` |
| `copilot.go:225` | `newStderrTail(newLogWriter(b.cfg.Logger, "[copilot:stderr] "), agentStderrTailBytes)` |
| `hermes.go:121` | `io.MultiWriter(newLogWriter(b.cfg.Logger, "[hermes:stderr] "), providerErr)` |
| `cline.go:108` | `io.MultiWriter(newLogWriter(b.cfg.Logger, "[cline:stderr] "), providerErr)` |
| `kimi.go:107` | `io.MultiWriter(newLogWriter(b.cfg.Logger, "[kimi:stderr] "), providerErr)` |
| `kiro.go:100` | `io.MultiWriter(newLogWriter(b.cfg.Logger, "[kiro:stderr] "), providerErr)` |
| `qoder.go:132` | `io.MultiWriter(newLogWriter(b.cfg.Logger, "[qoder:stderr] "), providerErr)` |
| `cursor.go:53` | `cmd.Stderr = newLogWriter(b.cfg.Logger, "[cursor:stderr] ")` (direct, no tail wrapper) |
| `gemini.go:64` | `cmd.Stderr = newLogWriter(b.cfg.Logger, "[gemini:stderr] ")` (direct) |
| `openclaw.go:95` | `cmd.Stderr = newLogWriter(b.cfg.Logger, "[openclaw:stderr] ")` (direct) |
| `opencode.go:136` | `cmd.Stderr = newLogWriter(b.cfg.Logger, "[opencode:stderr] ")` (direct) |
| `pi.go:234` | `cmd.Stderr = newLogWriter(b.cfg.Logger, "[pi:stderr] ")` (direct) |

**All 13 adapters share the single `logWriter.Write` implementation.**
Current source (`claude.go:972-985`):
```go
func (w *logWriter) Write(p []byte) (int, error) {
	text := strings.TrimSpace(string(p))
	if text != "" {
		w.logger.Debug(w.prefix + redact.Text(text))
	}
	return len(p), nil
}
```

**This applies `redact.Text` to every byte written by any of the 13
adapters' subprocess stderr**, because the fix made earlier this session
(as producer, in a separate task) touched the one shared definition, not a
per-adapter copy. This is a significant scope correction relative to how
that earlier fix was framed at the time — it was described as fixing
"Claude stderr," but the classification performed in this trace shows it
structurally fixed all 13 adapters simultaneously, since they all call the
same function. **This is stated here as an observed structural fact from
reading the current source, not as a new claim about what that prior task
intended or should be credited for** — the earlier evidence artifact should
be read in light of this finding by whoever adjudicates it, since its own
text did not identify the shared-definition scope.

**Downstream of `redact.Text`, this then also passes through
`logger.Debug(...)`, which routes to the shared production handler with
`ReplaceAttr: redact.SanitizeSlogAttr`.** So this specific writer definitely
applies **both** layers: `redact.Text` explicitly in the writer, and
`redact.SanitizeSlogAttr` again at the handler (redundant-but-safe double
coverage for this one path).

### `pkg/agent.stderrTail` — applies `redact.Text` indirectly via its `inner`, but its own buffered `Tail()` output does NOT

`stderr_tail.go:26-52`:
```go
type stderrTail struct {
	inner io.Writer
	...
}
func (s *stderrTail) Write(p []byte) (int, error) {
	if _, err := s.inner.Write(p); err != nil { return 0, err }
	s.mu.Lock()
	s.buf = append(s.buf, p...)   // <-- raw, unredacted bytes buffered here
	...
}
func (s *stderrTail) Tail() string {
	...
	return strings.TrimSpace(string(s.buf))  // <-- returns the RAW buffer, no redact.Text call
}
```
- `Write` forwards to `s.inner` (which, per the table above, is always a
  `logWriter` for the 8 adapters that wrap `stderrTail` around it) — so the
  **logged** copy is redacted (via `inner.Write`'s own `redact.Text` call).
- **But `s.buf` itself is the raw, pre-redaction byte stream**, and
  `Tail()` returns it unredacted. `Tail()`'s output is then embedded into
  error strings via `withAgentStderr(msg, label, tail string) string {
  return msg + "; " + label + " stderr: " + tail }` (`stderr_tail.go:67`),
  used at 6 confirmed call sites (`codex.go:733/746/774`, `claude.go:360`,
  `antigravity.go:151`, `codebuddy.go:260`, `copilot.go:292`).
- These composed error strings become `finalError`/`Result.Error`, which is
  a plain string field on the backend's result type — **not itself passed
  through `redact.Text` or `redact.SanitizeSlogAttr` inside `pkg/agent`**.

### Downstream consumers of `Result.Error` — mixed coverage, traced across the package boundary

`Result.Error` (containing the potentially-raw `Tail()` content) is
consumed in `internal/daemon/daemon.go`:

| Call site | Sink | Coverage |
|---|---|---|
| `daemon.go:3815` `taskLog.Warn("session resume failed, retrying with fresh session", "error", result.Error)` | slog attr, key `"error"` | `"error"` is not a sensitive key per `IsSensitiveKey`; falls to the `slog.KindString` branch, so `redact.Text` still runs via `SanitizeSlogAttr`. **Covered, pattern-dependent only** (same residual class as prior findings this session). |
| `daemon.go:3853` `"agent_error", result.Error` (attrs) | slog attr, key `"agent_error"` | Same as above — not a sensitive key, falls through to `Text()`. **Covered, pattern-dependent only.** |
| `daemon.go:3923,3949` `comment := result.Error` | fed toward `taskfailure.Classify` and eventually `d.client.FailTask(ctx, task.ID, errMsg, ...)` | This is a client→server call. **Server-side** `TaskService.FailTask` (`internal/service/task.go:1373`) calls `redact.Text(errMsg)` before persisting/broadcasting the comment content (`task.go:1459,1471`, independently confirmed by direct read in this pass). **Covered by `redact.Text`, applied server-side, not inside `pkg/agent` or at the `daemon.go` log line itself.** |
| `daemon.go:2937,3003,3014,3086,3138` `d.client.FailTask(ctx, ..., err.Error()/fallbackErrMsg, ...)` | same `FailTask` path | Same server-side `redact.Text` coverage as above, confirmed by the same `task.go` read. |

**Net finding: `Result.Error`'s raw-stderr-tail content is covered at every
traced sink, but by two different mechanisms depending on the sink** — the
weaker message/attr pattern-scan for the two direct daemon-log lines, and
the stronger explicit `redact.Text(errMsg)` call for the path that reaches
a persisted/broadcast task comment. Neither sink is a full bypass, but
neither is guaranteed-safe against a credential shape outside
`pkg/redact`'s fixed pattern set — consistent with, and now extending
end-to-end, the "pattern-dependent coverage" residual already disclosed
multiple times earlier in this session's review chain.

### `pkg/agent/hermes.go.acpProviderErrorSniffer` — no redaction inside the sniffer itself; depends entirely on its `io.MultiWriter` sibling

`hermes.go:1660-1690` (`Write` method): buffers raw bytes, splits on
newline, and extracts a `terminalMessage()`/`message()` used to build
`sniffer.provider + " provider error: " + acpAgentOutputTerminalRe.FindString(finalOutput)`
(`hermes.go:1775`). **This sniffer's own `Write` never calls `redact.Text`
or any `redact.*` function.** It sits behind `io.MultiWriter(newLogWriter(...),
providerErr)` (`hermes.go:121`, `cline.go:108`, `kimi.go:107`, `kiro.go:100`,
`qoder.go:132` — 5 adapters use this pattern) — meaning the **logged** copy
(via the `logWriter` sibling) is redacted, but the **sniffed** copy that
feeds `sniffer.message()`/`terminalMessage()` into a constructed error
string is not redacted at its source. This error string's ultimate sink
was not traced further in this pass (out of scope: this trace enumerates
construction/wiring, not every consumer of every derived error string) —
**flagged as a limitation**, not resolved.

### `pkg/agent/models.go:907` — `cmd.Stderr = &stderr` (`strings.Builder`), no redaction at all in this path

```go
var stderr strings.Builder
cmd.Stderr = &stderr
stdout, err := cmd.Output()
...
text := string(stdout)
if strings.TrimSpace(text) == "" {
    text = stderr.String()   // raw, unredacted
}
```
This is a distinct, narrower subprocess (`--list-models` discovery, per the
surrounding code read in this pass) that does **not** route through
`logWriter`/`stderrTail`/`redact.*` at all. `text` is returned from this
function; this trace did not follow `text`'s ultimate caller/sink (out of
scope for a construction/wiring enumeration) — **flagged as a limitation**.
This is a genuinely different code path from the main agent-execution
stderr wiring (the `newStderrTail(newLogWriter(...))`/direct-`logWriter`
patterns above) and should not be conflated with it: it is a **complete
bypass of both `redact.SanitizeSlogAttr` and `redact.Text`** at its point of
construction, pending confirmation of what its caller does with `text`.

## Summary classification table

| Path | Applies `redact.SanitizeSlogAttr` | Applies `redact.Text` | Bypasses both |
|---|---|---|---|
| `internal/logger.Init`/`NewLogger` (all production slog output via these) | YES (the hook itself) | Indirectly, via the hook's `Text()` call on string kinds/messages | No |
| `pkg/agent.logWriter.Write` (shared by all 13 backend adapters) | YES (downstream, via `logger.Debug`) | YES (explicit, in the writer itself) | No — double-covered |
| `pkg/agent.stderrTail.Write`'s forward to `inner` | YES/YES (inherits `logWriter`'s coverage) | YES (inherits) | No |
| `pkg/agent.stderrTail.Tail()` / `s.buf` itself | No (not logged directly here) | **No** (raw buffer) | **Yes, at construction** — coverage is deferred entirely to whatever later touches `Result.Error` |
| `daemon.go:3815,3853` (`result.Error` as a slog attr) | YES (attr falls to `Text()` branch) | YES (via `SanitizeSlogAttr`'s internal call) | No, but pattern-dependent only |
| `daemon.go` → `FailTask` → `task.go:1459,1471` (`result.Error`/`errMsg` as a task comment) | N/A (not a slog path) | YES (explicit `redact.Text(errMsg)`, server-side) | No |
| `hermes.go.acpProviderErrorSniffer` (and the 4 other adapters using the same `io.MultiWriter` pattern) | No (sniffer's own Write never calls it) | No (sniffer's own Write never calls it) | **At the sniffer itself, yes** — its sibling `logWriter` in the same `io.MultiWriter` is separately covered, but the sniffed/derived error string is not, and its ultimate sink was not traced (limitation) |
| `pkg/agent/models.go:907` `cmd.Stderr = &stderr` (`--list-models` discovery) | No | No | **Yes, at construction** — narrower, separate code path from main agent execution; caller/sink not traced (limitation) |

## Limitations (disclosed)

1. This trace enumerates **construction and immediate wiring**, not every
   downstream consumer of every derived string. Two genuine bypass-at-
   construction findings (`acpProviderErrorSniffer`'s derived message string;
   `models.go:907`'s discovery-only stderr builder) were identified but
   their ultimate sinks (where the resulting string is finally displayed,
   logged, or returned to a caller outside `pkg/agent`) were not traced —
   flagged explicitly rather than silently assumed safe or unsafe.
2. `fmt.Print*` production call sites were not re-enumerated in this pass;
   they were already independently traced and categorized in this
   session's earlier `credential-isolation-5.4-remaining-absolute-log-safety-gaps.md`
   document. Not repeating that work here per the task's own emphasis on
   not duplicating completed effort.
3. `internal/logger/logger.go` was read directly for classification but not
   independently re-hashed in this pass (its content and the
   `Init`/`NewLogger` `ReplaceAttr` wiring were already hash-verified in
   multiple earlier artifacts this session; re-hashing here would be a
   duplicate check without a specific reason to suspect drift).
4. The `daemon.go` → `FailTask` → `task.go` cross-file trace in this
   document is based on direct reads of both files' relevant line ranges,
   not an executed test — this is a static trace, consistent with the
   task's "read-only" framing, and does not itself constitute proof that
   these code paths behave as read at runtime (no test was run in this
   pass).

## Non-claims

- Does not implement any fix, guard, or change to any file discussed.
- Does not assert the two identified construction-time bypasses
  (`acpProviderErrorSniffer`, `models.go:907`) are exploitable — that
  depends on their untraced downstream sinks, explicitly flagged as a
  limitation above, not resolved here.
- Does not recharacterize or reopen adjudication of the already-completed
  Claude-stderr fix task from earlier this session; it observes, from
  reading current source, that the fix's scope was structurally broader
  (all 13 adapters) than its own evidence artifact stated, and leaves it to
  Kiro TL to decide whether that evidence artifact needs a correction note.
- No product/test/shared-planning/spec/task/git/index/credential/env/
  network/DB/service action was taken.
