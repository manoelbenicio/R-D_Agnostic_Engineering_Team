# EV-CREDISO-5.4-RESIDUAL — INDEPENDENT AUDIT of the remaining absolute log-safety gaps

Independent audit of `credential-isolation-5.4-remaining-absolute-log-safety-gaps.md` against current
source and the exact 5.4 spec.
Reviewer: **Kiro/Opus-4.8 — reviewer session `w8:p2`**. Author of the reviewed document:
**Kiro/Sonnet — session `w7:p1`**. Adjudicator: **Kiro TL — `w3:p3`**.
**Technical audit only — not an acceptance. Does not self-accept.** Kiro TL adjudicates.

> **Independence note.** The reviewed document was authored by a **different model family
> (Kiro/Sonnet)** than this reviewer (**Kiro/Opus-4.8**), and in a separate session with independent
> context. This is *stronger* independence than a same-family review: cross-family reduces (does not
> eliminate) common-mode blind spots. Every finding below was re-derived from source and existing tests
> by this reviewer; bounded offline Go was run against existing tests only (no new test/product code).

## Golden Rule check-IN — 2026-07-18T21:36:00Z
- Mode: READ-ONLY AUDIT. Only file created = this artifact. No product/test/shared/spec/task/git/index
  edits; no credentials/env values; no network/DB/live-provider/service action. Go runs offline
  (`GOPROXY=off`), pinned (`go1.26.4`, `GOTOOLCHAIN=local`), cache-only, against **existing** tests only.

## Provenance / hashes
- git HEAD: `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f` (working tree dirty; multi-stream WIP).
- Reviewed doc SHA-256: `92f8b9b7defa2179fcf270f740ccf0d31002e8b5f23bc0bcb504907fc9656ba3`.
- Exact spec: `openspec/changes/agent-credential-isolation/tasks.md:34` —
  `- [ ] 5.4 Confirmar que nenhum segredo aparece em logs (sanitizeForLog).` The task names
  **`sanitizeForLog`** — i.e. it targets the **structured-log surface**, which is material to the
  scope reading in item 3 below.
- Toolchain: `/home/dataops-lab/go-sdk/bin/go` = `go1.26.4 linux/amd64`.

## Cross-cutting linchpin (independently verified)
Both non-blocking classifications for items 1–2 depend on the production redaction hook actually being
installed. **Verified:**
- `internal/logger/logger.go:36,50` build handlers with `ReplaceAttr: redact.SanitizeSlogAttr` (both
  `Init()` and `NewLogger()`).
- `internal/logger/logger.go:38` `slog.SetDefault(...)`, invoked at `cmd/server/main.go:123` (`logger.Init()`).
  ⇒ **package-level `slog.*` in the server process is redacting.** This is what makes `auth.go:656` and
  `cloud_pat.go:359` (both package-level `slog`) pattern-dependently safe.
- `redact.SanitizeSlogAttr` passes the built-in **`msg`** attribute through `Text()` (msg is not a
  sensitive key ⇒ `KindString` branch), so message-string content is scanned — this is the item-1 backstop.
- `go test ./pkg/redact/ -count=1` → **ok** (offline); `TestRedactCredentialFieldsInJSONBody`
  (redact_test.go:369) exercises `{"access_token":"…","refresh_token":"…","status":"denied"}` generically.

## Item-by-item verdict

### 1. `daemon.go:4477` `taskLog.Info(fmt.Sprintf("tool #%d: %s", n, msg.Tool))` — **CONCUR: non-blocking**
- Verified line at `internal/daemon/daemon.go` (case `agent.MessageToolUse`); `msg.Tool` = `agent.Message.Tool`
  "tool name (ToolUse, ToolResult)" (`pkg/agent/agent.go:~101`). Field semantics = backend/model tool-call
  **name**, a bounded identifier, not user/env/credential content.
- Two controls: (a) safe-identifier by construction; (b) redaction **backstop** — the `fmt.Sprintf` lands
  in the `msg` attr and is scanned by `Text()` via the verified `SetDefault` hook, so even a secret-shaped
  tool name would be masked.
- **Bounded-review limitation (disclosed):** I confirmed the field definition, the sink line, and the
  backstop, and spot-checked the semantics; I did **not** exhaustively re-audit all 13 adapters' `Tool`
  assignments (the reviewed doc enumerates them). Concur with the recommended structured-attr rewrite as
  defense-in-depth, out of scope here (`daemon.go` is a shared hotspot).

### 2. Google OAuth non-2xx body — `internal/handler/auth.go:656` — **CONCUR: non-blocking, test-evidence gap**
- Verified: body read then logged only under `if tokenResp.StatusCode != http.StatusOK` (success bodies
  structurally excluded). `"body"` is **not** in `IsSensitiveKey`, so `SanitizeSlogAttr` → `KindString` →
  `Text(value)` (pattern-dependent). Generic redaction proven by redact_test.go:369.
- **Challenge to the framing (not the verdict):** the durable controls are the **status-guard** + the
  **`Text()` pattern set**. "Google's RFC-6749 error schema never echoes tokens" is a *third-party
  assumption*, not a control — if Google (or a proxy) ever returned a token in a non-standard key/shape,
  redaction would depend entirely on `Text()` matching it (residual **R-5.4-B pattern-dependency**). The
  reviewed doc's "needs direct test evidence" is **correct**: there is no site-level integration test.

### 3. CLI webhook URL — `cmd_autopilot.go` `printWebhookURL` — **CONCUR: out of the spec's stated scope**
- Verified `fmt.Printf("Webhook URL: …")` to operator stdout. **Independently corroborated:** a grep for
  `webhook_url`/`webhook_path`/`WebhookURL` across `internal/**` structured/`log` sinks returns **zero** —
  the webhook URL is **not** written to any daemon/structured log. Given the spec names `sanitizeForLog`
  (the structured-log surface), interactive CLI stdout is outside it. Concur; TL to confirm scope reading.

### 4. "No other unprotected sink missed" — **CHALLENGE: one co-equal sink not enumerated**
- **Finding (material to completeness):** `internal/auth/cloud_pat.go:359`
  `slog.Warn("cloud_pat: verify returned non-200", "status", …, "body", snippet)` is a **second
  external-HTTP-response-body log sink of the identical class as item 2** (Fleet/Cloud PAT verify non-200
  body, capped at 512 bytes, logged under key `"body"`). My targeted grep shows **exactly two** such
  external-body `"body"` slog sinks in the codebase (`auth.go:656` and `cloud_pat.go:359`).
- It is **protected by the same mechanism** (default logger `SetDefault` hook → `Text()`), so it is **not a
  new security exposure** — but the reviewed document's item-4 "closing sweep" concluded "no new sink
  category" and **did not enumerate `cloud_pat.go:359`** as a co-equal instance of the R-5.4-B class. An
  absolute-bar residual inventory should list it explicitly, with the **same test-evidence gap** as item 2.
- **Bounded-review limitation (disclosed):** my sweep targeted the highest-risk external-response-body
  (`"body"`) category and corroborated the webhook finding; I did **not** re-run the prior audit's full
  703/82 callsite enumeration — that remains the prior audit's basis.

## Challenge summary of the non-blocking classifications
| Item | Reviewed verdict | This audit | Basis |
|---|---|---|---|
| 1 daemon tool-name | non-blocking | **CONCUR** | safe identifier + verified `Text()` msg backstop |
| 2 OAuth body | non-blocking (test gap) | **CONCUR**, reframe controls | status-guard + `Text()` are the controls; "Google schema" is an assumption |
| 3 CLI webhook URL | out of scope | **CONCUR**, corroborated | zero server-side log sink for webhook_url; spec = `sanitizeForLog` |
| 4 other sinks | none missed | **CHALLENGE** | `cloud_pat.go:359` = second co-equal external-body sink, unenumerated |

## Recommended exact remaining acceptance gates (after the Claude-stderr fix) — for Kiro TL
1. **Item 2 test evidence** — add a disjoint `internal/handler` test (httptest server, synthetic
   `access_token`-bearing non-200 body, capture via `slog` handler using `redact.SanitizeSlogAttr`),
   asserting the sentinel is absent from the logged `"body"`.
2. **cloud_pat.go:359 (new gate)** — either an analogous bounded disjoint test **or** explicit
   documentation of it under the same R-5.4-B residual with TL sign-off. It must **not** be left
   unenumerated in the closure record.
3. **Redaction-hook invariant** — record/guard the invariant that every logger reaching an
   external/uncontrolled body carries `SanitizeSlogAttr` (verified today for `slog.Default` +
   `internal/logger`); a future logger constructed without it would silently bypass items 1–2/4.
4. **Item 1 (optional, defense-in-depth)** — structured-attr rewrite at `daemon.go:4477`, routed to
   an agent with authorized `daemon.go` scope; non-blocking.
5. **Item 3 scope confirmation** — TL explicitly records CLI operator stdout as outside the
   `sanitizeForLog`/"logs" surface (or opts to harden as a product decision).
6. **Reference** — the Claude-stderr `logWriter` fix is separately reviewed
   (`credential-isolation-5.4-claude-stderr-redaction-independent-review.md`): technical PASS, push
   eligibility **PARTIAL** (mixed-file `claude.go`).
7. **Honest bound on "absolute"** — `Text()` is pattern-based; the truthful closure statement for 5.4 on
   the log surface is "no **known secret shape** routed through a **redacting logger** appears in logs,"
   with R-5.4-B (pattern-dependency) as the acknowledged inherent residual. TL should accept the bar with
   that wording rather than an unqualified "absolute."

## Disposition (reviewer)
- Items 1–3: **concur non-blocking / out-of-scope** as re-verified above (with the item-2 control-framing
  correction). Item 4: **the "no sink missed" claim is incomplete** — `cloud_pat.go:359` must be
  enumerated (same class, same protection, same test-evidence gap). None of this is whole-task acceptance;
  Kiro TL (`w3:p3`) adjudicates, including the item-3 scope reading and the frontend-owner scope tracked
  elsewhere. Reviewer ≠ adjudicator; nothing self-accepted.

## Golden Rule check-OUT — 2026-07-18T21:40:00Z
- Files created: this artifact only. Reviewed doc and all source unchanged; no git stage/commit/push; no
  network/services/credentials/env values. Go runs offline/pinned/cache-only against existing tests.
  Status: DONE (reviewer report). Adjudication pending Kiro TL.
