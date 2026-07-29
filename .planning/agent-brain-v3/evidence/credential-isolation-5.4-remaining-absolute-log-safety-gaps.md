# Remaining agent-credential-isolation 5.4 gaps after the Claude-stderr fix

**Reviewer/author:** Kiro/Sonnet, pane `w7:p1` — read-only design/closure inventory only.
Distinct from the Claude-stderr producer session (same underlying assistant identity,
different task role: that session acted as producer; this document is a
read-only follow-on, not a self-review of that fix).
**Date:** 2026-07-18T18:28:09-03:00
**Adjudication authority:** Kiro TL adjudicates after a distinct independent
review. This document does not implement anything, does not self-accept, and
does not touch `daemon.go` (shared hotspot, out of scope by instruction).
**Scope discipline:** read-only synthetic/offline inspection only. No
product/test/spec/tasks/git/index edit. No credential/env value read. No
network/DB/live-provider action. Only this one file was created.

## Requirement re-stated

Task 5.4 / spec "Não vazamento de segredo": the system SHALL NOT log
credential content; the requirement is absolute (no partial-credit framing in
the spec text). This document re-evaluates that absolute bar against the
current bytes on disk, **excluding** `pkg/agent/claude.go`'s `logWriter.Write`
— now fixed and covered by
`credential-isolation-5.4-claude-stderr-redaction-fix.md` — from the list of
unresolved items.

## Method

Read-only re-inspection of the four sites named in the task, plus one more
sweep for any handler/sink not already covered by `redact.*`, cross-checked
against the prior audit (`credential-isolation-5.4-codebase-log-safety-audit.md`)
and critique (`credential-isolation-5.4-codebase-critique.md`). No new grep
methodology invented beyond what those two documents already established as
reproducible (703/82 slog callsite count, production-only `slog.New(`
enumeration, message-string bypass scan) — this document adds targeted
line-level analysis for the four named remaining items and one closing sweep.

---

## 1. `daemon.go:4477` — dynamic tool-name message

```go
taskLog.Info(fmt.Sprintf("tool #%d: %s", n, msg.Tool))
```

**Material secret source vs. safe identifier:** `msg.Tool` is the `Tool
string // tool name (ToolUse, ToolResult)` field of `pkg/agent.Message`
(`agent.go:101`). Every producer of this field across all 13 backend adapters
(`claude.go:419`, `codex.go:1629/1650/1795/1816`, `cursor.go:131/259`,
`gemini.go:127`, `nim.go:265`, `openclaw.go:379`, `opencode.go:302`,
`codebuddy.go:319`, `copilot.go:109`, `pi.go:307`) assigns it from a
backend-controlled tool-name enum/literal (e.g. `"exec_command"`, `"Read"`,
`"file_edit"`, `call.Function.Name` from the model's own tool-call schema) —
never from raw user input, environment content, or credential material. This
is a **safe identifier**, not a material secret source. `n` is a local
`atomic.Int64` counter. Neither operand can carry a credential.

**Existing runtime mitigation:** none needed at this specific site because
there is no secret-shaped input reaching it; the fixed-set nature of tool
names (bounded by each backend's own tool schema) means `Text()`'s
pattern-scan (which does still apply to the message per the critique's
confirmed `ReplaceAttr`/`MessageKey` mechanism) is a backstop, not the primary
control.

**Test/guard coverage:** none dedicated to this line specifically; covered
only incidentally by the general `SanitizeSlogAttr` message-key wrapping
(verified mechanism, not a per-site test).

**Exact smallest disjoint fix, if desired:** replace the `fmt.Sprintf`
message-string form with a structured call —
`taskLog.Info("tool use", "seq", n, "tool", msg.Tool)` — so the tool name
becomes a keyed attr like every other structured field in this codebase,
removing reliance on message-key pattern-scanning for a value that, while
currently safe, has no compile-time guarantee of staying safe if a future
adapter ever assigns `Tool` from less-trusted input. This is a 1-line change
localized entirely inside `daemon.go`.

**Verdict: justified non-blocking residual.** No material secret source
today; recommend the structural fix above as defense-in-depth, not as a
blocking gap. `daemon.go` is explicitly out of scope for this document (shared
hotspot); the fix is recorded here as a recommendation for whoever next
receives authorized `daemon.go` scope, not performed by this document.

---

## 2. Google OAuth non-2xx body logging — `internal/handler/auth.go:656`

```go
slog.Error("google oauth token exchange returned error", "status", tokenResp.StatusCode, "body", string(tokenBody))
```

**Material secret source vs. safe identifier:** `tokenBody` is the raw HTTP
response body from `https://oauth2.googleapis.com/token`, gated by
`tokenResp.StatusCode != http.StatusOK`. This is genuinely external,
uncontrolled content — a **potential** material secret source in principle
(any third-party response body is), but Google's actual OAuth 2.0 error
response schema (RFC 6749 §5.2: `error`, `error_description`,
`error_uri`) never echoes back `access_token`/`refresh_token`/`id_token`
fields on an error path — those only appear in a 200 success body, which
this code path structurally excludes by the status-code guard. This was
verified executably by both the prior audit and this reviewer's critique
(two bounded synthetic tests: a hypothetical token-bearing error body IS
redacted by `Text()`'s pattern set; Google's realistic error schema has no
token field to redact in the first place).

**Existing runtime mitigation:** the `"body"` key is not in `IsSensitiveKey`'s
list, so `SanitizeSlogAttr` falls through to its `slog.KindString` branch,
which calls `Text(value)` — the same regex/literal pattern set applied
everywhere else. This is real coverage, not merely claimed.

**Test/guard coverage:** `pkg/redact/redact_test.go` has
`TestRedactCredentialFieldsInJSONBody`, which exercises the exact
`{"access_token":"...","refresh_token":"...",...}` shape generically (not
Google-specific). No dedicated test exists at the `auth.go:656` call site
itself proving the integration wiring (key `"body"` → `SanitizeSlogAttr` →
`Text()`) is actually exercised end-to-end for this handler.

**Exact smallest disjoint fix, if desired:** a `internal/handler` package test
that constructs an `httptest.Server` returning a synthetic non-200 body with
an `access_token` field, invokes the Google-callback handler function
directly (no real network to Google, no DB required if the handler under
test can be unit-isolated), and asserts the resulting log line — captured via
a local `slog.NewTextHandler(&buf, &slog.HandlerOptions{ReplaceAttr:
redact.SanitizeSlogAttr})` — does not contain the synthetic sentinel. This
would be a small, disjoint, new test file
(e.g. `internal/handler/auth_oauth_log_safety_test.go`), not a source change,
since the existing mechanism already provides coverage; the gap is
**test evidence**, not implementation.

**Verdict: justified non-blocking residual**, with a recommended disjoint
test as the smallest closure step — not a source fix, since the source
already routes through the shared mechanism correctly.

---

## 3. CLI operator-terminal webhook output — `cmd_autopilot.go:579,584`

```go
func printWebhookURL(client *cli.APIClient, trigger map[string]any) {
	if u := strVal(trigger, "webhook_url"); u != "" {
		fmt.Printf("Webhook URL: %s\n", u)
		return
	}
	if path := strVal(trigger, "webhook_path"); path != "" {
		base := strings.TrimRight(client.BaseURL, "/")
		fmt.Printf("Webhook URL: %s%s\n", base, path)
	}
}
```
(Confirmed at `cmd_autopilot.go:576-590`; line numbers 579/584 in the prior
audit's citation are off by roughly 3 from this file's current on-disk
layout — a minor provenance-precision note, content otherwise matches.)

**Material secret source vs. safe identifier:** a webhook URL *may* embed a
bearer-style token in its path or query string (a common webhook-security
pattern), so this is a **potential** material secret source, same class as
item 2. However this is `fmt.Printf` to the operator's own **stdout**, in an
interactively-invoked CLI command the operator ran themselves against their
own account — not a persisted daemon diagnostic log, not broadcast to other
users, and not written to any log file by default. It is the direct,
intended output of a command whose entire purpose is "show me my webhook
URL." This is fundamentally different from `daemon.go`'s always-on structured
logging.

**Existing runtime mitigation:** none, and arguably none is appropriate here
— redacting an operator's own requested output for their own resource would
work against the command's purpose (if it's a webhook secret proxy the
operator will need the exact URL to configure their receiving system).

**Test/guard coverage:** none, and none of the existing `redact.*` test
infrastructure applies to interactive CLI stdout by design (the prior audit
and critique both correctly scoped `redact.*` to the daemon-log surface).

**Verdict: justified non-blocking residual, out of the spec's stated scope.**
The spec text ("nenhum segredo aparece em logs") targets diagnostic/log
surfaces, not an operator's own terminal invoking a command against their own
authenticated session to retrieve their own resource's URL. Recommend no
fix; if the product owner wants this hardened anyway (e.g. because the CLI
output could be piped/redirected into a shared log or CI artifact by the
operator), that would be a product-scope decision, not a 5.4 code defect —
recorded here for visibility only.

---

## 4. Any handler/sink not protected by `redact.*` — closing sweep

Re-ran the categorical enumeration from the original audit
(structured `slog.*`, standard `log.*`, `fmt.Print*`, WebSocket/event
broadcast, agent comment/error surfaces) plus one additional check for any
`Marshal`/`http.ResponseWriter` write path that might leak a struct
containing a credential field without going through `redact.*` or a
redacting carrier type.

- **Standard `log.*` package:** re-confirmed zero Print/Printf/Println/Fatal/Panic
  callsites in server code.
- **`fmt.Print*` full production inventory (excluding claude.go, now fixed,
  and email.go, already independently accepted):** all remaining instances
  are CLI operator-terminal output in `cmd_skill.go`, `cmd_autopilot.go`,
  `cmd_squad.go`, `cmd_agent.go`, `cmd_runtime_profile.go`, `cmd_runtime.go`,
  `cmd_version.go`, `cmd_auth.go:415` (prompt label only, no value — matches
  the original audit's characterization exactly), and `cmd/migrate/main.go`
  (migration status text, no credential-shaped field). All confirmed as
  operator-invoked, self-directed CLI output printing the operator's own
  resource names/IDs/statuses — no distinct new material secret source found
  beyond the already-disclosed webhook-URL case in item 3.
- **WebSocket/event-emitter broadcast:** re-confirmed single ingestion point
  at `internal/handler/daemon.go:2225-2228` (`ReportMessages`), unchanged
  since the original audit, still routing `msg.Content`/`msg.Output` through
  `redact.Text` and `msg.Input` through `redact.InputMap` before persist and
  broadcast.
- **No new `Marshal`-based leak surface found:** credential-bearing carrier
  types (`runtimeenv.StableSecret`, `OpenclawGatewayPin`) were previously
  confirmed to implement `String()/GoString()/Format()`/`MarshalJSON()`
  redaction by construction; no additional struct with a plausible
  credential field was found lacking an equivalent guard in this sweep.

**Verdict: no new unprotected sink category found.** The remaining exposure
surface is fully accounted for by items 1-3 above plus the two residuals
already disclosed in the prior audit/critique (R-5.4-B pattern-dependency in
general, and the now-fixed R-5.4-A instance class for `claude.go`).

---

## Summary table

| Item | Material secret source? | Runtime mitigation exists? | Test/guard coverage? | Smallest fix | Verdict |
|---|---|---|---|---|---|
| `daemon.go:4477` tool-name message | No (safe identifier, backend-controlled enum) | Backstop only (message-key pattern-scan) | None dedicated | 1-line structured-attr rewrite (out of scope here — `daemon.go`) | Non-blocking residual |
| Google OAuth error body (`auth.go:656`) | Potential (external body), but Google's real error schema has no token field | Yes — `"body"` value goes through `Text()` | Generic `pkg/redact` test only, no site-specific test | New disjoint test file exercising the handler directly | Non-blocking residual; test-evidence gap, not a source gap |
| CLI webhook URL print (`cmd_autopilot.go` printWebhookURL) | Potential (URL may embed a token) | None (by design — operator's own terminal, own resource) | None (out of `redact.*`'s intended scope) | None recommended; product-scope question if any | Non-blocking residual, out of spec's stated scope |
| Closing sweep (all other sinks) | — | — | — | — | No new gap found |

## Non-claims / recommendation to TL

- This document implements nothing and self-accepts nothing.
- `daemon.go` was read for citation/verification purposes only, consistent
  with prior read-only reviews in this change; no edit was made or proposed
  to be made by this document's author — it is flagged as a shared hotspot
  requiring separately authorized scope.
- Recommend Kiro TL treat all three items as **non-blocking residuals** for
  the absolute 5.4 requirement, on the basis that none currently exposes a
  demonstrated material secret through a currently-reachable code path, while
  recording the two smallest recommended closure actions (a `daemon.go`
  structured-attr rewrite for item 1, and a new disjoint test for item 2) as
  optional hardening for a future, separately scoped task — not as blockers
  to closing 5.4 on the daemon-log surface specifically.
- Item 3 (CLI operator terminal) is recommended to be explicitly recorded as
  **out of the spec's stated scope** rather than "non-blocking" in the same
  sense as items 1-2, since it targets a different surface (operator's own
  interactive command output) than the "logs" language in the spec appears
  to address; TL should confirm this scope reading rather than have it
  assumed.
