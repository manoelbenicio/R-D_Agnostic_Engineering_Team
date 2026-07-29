# Fix: agent-credential-isolation 5.4 — Claude stderr log-writer redaction

**Producer:** Kiro (producer role for this task; distinct from the prior
critique session that identified this finding)
**Date:** 2026-07-18T18:21:07-03:00 to completion
**Does NOT self-accept.** Kiro TL adjudicates after a distinct independent
review.

## Motivation / traceability

This fix addresses one of two concrete findings from
`.deploy-control/evidence/credential-isolation-5.4-codebase-critique.md`
(§6, "Message-string bypass check"): `pkg/agent/claude.go`'s `logWriter.Write`
logged raw, only-whitespace-trimmed Claude CLI subprocess stderr via string
concatenation (`w.logger.Debug(w.prefix + text)`), bypassing the structured
`Attr`-keyed path of `redact.SanitizeSlogAttr` for that content and relying
solely on the message-key wrapping in the `tint`/`slog` handler chain for any
pattern-based coverage.

## Pre-edit conflict check (Golden Rule)

`git status --short` showed `multica-auth-work/server/pkg/agent/claude.go`
already modified (uncommitted) in the working tree *before* this task began.
Investigation (`git diff`) confirmed the pre-existing diff is Codex3's
`G3-security-corrections-adapters` argv-redaction work
(`safeAgentArgvForLog`, `logAgentCommand`, sensitive-flag/marker tables,
changes inside `Execute()`/`buildClaudeArgs()`) — per `AGENT_LEDGER.md` this
reached DONE at `2026-07-18T03:32:36Z` and was later independently reviewed
(EV-G3-SEC-ADAPTERS). **That diff does not touch `logWriter.Write`,
`newLogWriter`, or any line in that function's range** — zero line-range
overlap with this fix's target. `FILE_OWNERSHIP.md` lists
`pkg/agent/{claude,codex,kimi,nim,antigravity}.go` under "Runtime/CLI
security," owned by Codex3 "(com coordenação)." No `AGENT_LEDGER.md` row shows
Codex3 (or anyone) currently `IN_PROGRESS` on `claude.go` at check-in time —
all `IN_PROGRESS` rows for this file are superseded by later `DONE` rows per
the ledger's own reconciliation notes. This is recorded transparently as a
coordination-flagged edit, not a silent override; the pre-existing
uncommitted diff was not stashed, discarded, or reverted.

## Provenance (SHA-256)

| File | Before | After |
|---|---|---|
| `server/pkg/agent/claude.go` | `4ee1e98e0560c1ce0ac3f68999ea3c5807d746632b87d1cf71d949f623408cdc` | `3f9dc4fbdb28eecfcc4886f2a518a07943f8a0493cf7aa3525b745992c6d2f54` |
| `server/pkg/agent/claude_log_writer_redaction_test.go` | (new file) | `81d3e8659125df2fda25c27da16b24d88e1861282255e41f46c33f7cae2ae40a` |
| `server/pkg/redact/redact.go` (dependency, unmodified) | `f409ba8a9f3e63618d59c5a8692296f8f7c019c9e558576b8786a058fbf68a5c` | (unchanged) |

The "before" hash for `claude.go` already includes Codex3's pre-existing
uncommitted argv-redaction diff (see conflict check above); it is **not**
the git-HEAD blob hash.

## Change

```go
func (w *logWriter) Write(p []byte) (int, error) {
	text := strings.TrimSpace(string(p))
	if text != "" {
		// text is raw subprocess stderr and may contain an echoed credential,
		// token, or auth error body from the underlying CLI. Route it through
		// redact.Text before it ever reaches the log message so a leaked
		// secret shape is masked the same way any other logged string is,
		// rather than relying only on the slog ReplaceAttr hook to catch it.
		w.logger.Debug(w.prefix + redact.Text(text))
	}
	// The io.Writer contract is about bytes consumed from p, not bytes
	// written to the log; report the original length regardless of
	// redaction so callers relying on byte-count semantics are unaffected.
	return len(p), nil
}
```

Added `"github.com/multica-ai/multica/server/pkg/redact"` to `claude.go`'s
import block (not previously imported by this file).

Scope discipline: only `claude.go` was edited (the one line inside
`logWriter.Write`, plus the import); `daemon.go` and all pre-existing test
files were left untouched, per the authorized scope.

## Test coverage (new, disjoint file: `claude_log_writer_redaction_test.go`)

Six new named tests, all synthetic-only (no real credential value):

1. `TestLogWriterRedactsAPIKeySentinel` — a fake `OPENAI_API_KEY=sk-proj-...`
   sentinel embedded in a stderr-shaped error line must not appear in
   captured output; a `[REDACTED...]` placeholder must be present.
2. `TestLogWriterRedactsBearerTokenSentinel` — a fake JWT-shaped bearer token
   must not appear in captured output.
3. `TestLogWriterRedactsErrorBodyTokenField` — a fake JSON error body with an
   `"access_token"` field must not leak the field's synthetic value.
4. `TestLogWriterPreservesSafeStderrContent` — an ordinary, non-sensitive
   stderr warning line must survive unaltered (with its prefix), proving the
   fix does not turn stderr capture into a no-op.
5. `TestLogWriterEmptyOrWhitespaceEmitsNothing` — `""`, whitespace-only, and
   newline/tab/CR-only inputs must emit zero log output, matching the
   pre-existing behavior, while still returning `len(p)`.
6. `TestLogWriterReturnedByteCountMatchesInputRegardlessOfRedaction` — a long
   input whose redacted form is drastically shorter must still return
   `len(p)`, proving the `io.Writer` byte-count contract is preserved
   independent of redaction.

## Execution proof (bounded, offline, synthetic-only)

Environment: `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`, pinned local
`/home/dataops-lab/go-sdk/bin/go` (go1.26.4, linux/amd64).

```
gofmt -l pkg/agent/claude.go pkg/agent/claude_log_writer_redaction_test.go
  => (empty output — both files already gofmt-clean)

go build ./pkg/agent/...   => exit 0
go vet   ./pkg/agent/...   => exit 0 (no findings)

go test -v -count=20 ./pkg/agent/ -run \
  'TestLogWriterRedactsAPIKeySentinel|TestLogWriterRedactsBearerTokenSentinel|TestLogWriterRedactsErrorBodyTokenField|TestLogWriterPreservesSafeStderrContent|TestLogWriterEmptyOrWhitespaceEmitsNothing|TestLogWriterReturnedByteCountMatchesInputRegardlessOfRedaction'
  => 6 distinct named tests x20 = 120 === RUN, 120 --- PASS, 0 --- FAIL, exit 0, 0.769s

go test -count=20 -race ./pkg/agent/ -run '<same 6 tests>'
  => ok, exit 0, 6.819s, no data races reported

go test ./pkg/agent/...    => ok, exit 0, 9.058s (full package regression check, no new failures)
```

No test used a real credential, auth home, database, network call, or live
provider. All secret-shaped values are explicitly synthetic (`sk-proj-SYNTHETIC...`,
`...SYNTHETIC...`, `hunter2-synthetic-not-real-...`, etc.) and clearly marked
as such in the test source.

## What this fix does and does not claim

- **Does** ensure raw Claude CLI stderr is pattern-scanned by `redact.Text()`
  before it reaches the log message, closing the specific gap identified in
  the prior critique for this one call site.
- **Does not** claim to make redaction pattern-independent — `redact.Text()`
  is still a fixed regex/literal pattern set (documented residual R-5.4-B in
  the original 5.4 audit and R-5.4-A/critique for message strings); a novel
  vendor-specific credential shape with no recognizable pattern could still
  slip through. This fix narrows the gap for known secret shapes; it is not
  a general-purpose guarantee.
- **Does not** touch `daemon.go`, any existing test file, `tasks.md`, the
  shared ledger, `STATE.md`, git index, or any credential/env value.
- **Does not** perform any network, database, or live-provider action.

## Non-claims / recommendation to TL

- Producer does **not** self-accept. Kiro TL adjudicates after a distinct
  independent reviewer reproduces the build/vet/test evidence above.
- Recommend the independent reviewer additionally re-check `daemon.go:4477`
  (`taskLog.Info(fmt.Sprintf("tool #%d: %s", n, msg.Tool))`), the other
  message-interpolation instance from the prior critique — intentionally
  **out of scope** for this fix per the authorized file list (`daemon.go`
  explicitly excluded), so it remains an open item for a separate, properly
  scoped task.
