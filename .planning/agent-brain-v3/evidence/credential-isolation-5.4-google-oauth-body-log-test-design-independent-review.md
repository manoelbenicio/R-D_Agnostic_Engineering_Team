# Independent Design Review — credential 5.4 Google OAuth body-log test feasibility

- reviewer: Kiro / Opus-4.8, wave w8:p1 (independent; distinct from design author Kiro/Sonnet w7:p1)
- date: 2026-07-18T21:58:00Z
- mode: READ-ONLY. No product/test/shared/spec/task/git/index/credential/env/network/service changes. This is the only file created.

## Embedded check-in / check-out
- CHECK-IN 2026-07-18T21:53:00Z — Kiro/Opus-4.8 w8:p1 — stream CREDISO-5.4-GOOGLE-OAUTH-LOGTEST-DESIGN-INDEPENDENT-REVIEW — READ-ONLY.
- CHECK-OUT 2026-07-18T21:58:00Z — DONE. Kiro TL adjudicates. Not self-accepted; no implementation.

Reviewed: `credential-isolation-5.4-google-oauth-body-log-test-design.md` (Kiro/Sonnet w7:p1).

## Claim validation (I read the actual source; findings independently confirmed)

| Design claim | Verdict | Source evidence |
|---|---|---|
| No DI seam; both Google calls hardcoded to `http.DefaultClient` | ✅ CONFIRMED | `auth.go:635` `http.PostForm(...)` (package-level → `DefaultClient`); `auth.go:676` `http.DefaultClient.Do(userInfoReq)`. |
| Non-200 body is logged | ✅ CONFIRMED | `auth.go:656` `slog.Error("google oauth token exchange returned error", "status", tokenResp.StatusCode, "body", string(tokenBody))`. |
| No `HTTPClient`/RoundTripper field on `Handler` | ✅ CONFIRMED | No `HTTPClient *http.Client` in `handler.go`; grep shows the field exists only on other types (`cloud_pat.go:188`, `cloudruntime`, `gateway/client.go`, etc.). |
| Global `DefaultClient.Transport` swap risk | ✅ CONFIRMED, with refinement | `github.go:429` also uses `http.DefaultClient.Do` → same-package co-victim of a global swap. **Refinement:** `skill.go`/`agent_template.go` use **local** `&http.Client{}` (injected), so they are **not** affected — the in-package global blast radius is narrower than "any http in the package": it is precisely `github.go:429` + the two `GoogleLogin` calls (+ any future `DefaultClient` use). |
| `TestMain` gates the whole handler package on Postgres | ✅ CONFIRMED | `handler_test.go:38`: `pgxpool.New` then `pool.Ping`; on failure `os.Exit(0)` **before** `m.Run()` → **package-wide zero-test skip**, not per-test. |
| Option B nil-safe `HTTPClient` field matches an existing pattern | ✅ CONFIRMED | `cloud_pat.go:188 HTTPClient *http.Client` (CloudPATVerifier) is the precedent; adding a **struct field** (not a `New(...)` param) is additive and does not break the `New(...)` call sites (`handler_test.go:57`, `router.go`). |

## Independent assessment of Options A and B (against the task's own criteria: avoid DB/network/global races, no broad production coupling)

- **Option A (global `http.DefaultClient.Transport` swap):** zero production edit, but it maximizes exactly the hazard the task says to avoid — a **process-global race** that also captures `github.go:429` and anything the `realtime.Hub` goroutine (started in `TestMain`) touches. It also still lives in the **DB-gated** package, so it **does not execute** without Postgres. Worst fit for the stated criteria.
- **Option B (nil-safe `Handler.HTTPClient`):** clean per-call injection, precedented, additive — but it is a **production edit to a broad, many-caller struct** (broad coupling), and, decisively, **it still does not execute** under the DB-gated `TestMain`. So Option B buys a clean seam but **not executability**; "PASS" evidence would still require a reachable Postgres or a `TestMain` change (further scope).

**Both A and B share a fatal execution problem the design correctly flagged but under-weighted: any test placed in `internal/handler` cannot run offline.** Neither escapes the DB gate.

## Smaller testable seam (the task's core ask) — RECOMMENDED

The load-bearing security property is not "the handler injects a client" but **"a non-200 Google token body cannot leak `access_token`/`refresh_token`/`id_token` into logs."** That redaction is performed by `redact.SanitizeSlogAttr` (`redact.go:117`) wired into the global slog default — **not** at the `auth.go` call site. Therefore the smallest seam avoids the handler HTTP path entirely:

**Seam 1 — redaction-contract test in `pkg/redact` (zero production edit; no DB, no network, no global-http swap, no `Handler` coupling):**
- **File:** `server/pkg/redact/redact_test.go` (existing; add one case) or a new `redact_oauth_body_test.go` in package `redact`.
- **Symbols:** existing `redact.Text` / `redact.SanitizeSlogAttr` (`redact.go:53` regex already covers `access_token|refresh_token|id_token|auth_token`; `redact_test.go:372` already redacts `access_token`+`refresh_token`).
- **Assertion:** feed a Google-shaped non-200 body `{"error":"invalid_grant","access_token":"<SENTINEL>","refresh_token":"<SENTINEL>","id_token":"<SENTINEL>"}` through the **same** sanitizer the global handler uses, and assert the sentinels are `[REDACTED]` while `error`/status remain. This runs **unconditionally offline** — `pkg/redact` has **no** `TestMain`/DB gate (the very reason the `internal/auth` cloud-PAT precedent executed).
- **Highest-value sub-assertion:** the body is logged under a **non-secret attr key `"body"`**, so redaction must come from **value-content** matching, not key-name matching. The test should assert `SanitizeSlogAttr` redacts a secret-bearing **value under a benign key**. If it only redacts by key name, the `"body"` attr would leak — so this test directly probes the real risk. (I did not exhaustively read `SanitizeSlogAttr`'s value-handling; this is the exact behavior the test must pin — flagged, not assumed.)

**Seam 2 — static call-site assertion (no execution needed) to close the wiring gap:**
- A deterministic AST/source check (pattern already used for chat 1.1/1.4) asserting `GoogleLogin`'s non-200 branch logs the body via `slog.Error` (global sanitized sink), **not** via `fmt.*`/`os.Stderr`/a bespoke logger. `auth.go:656` currently satisfies this.

**Together (Seam 1 executed + Seam 2 static) prove the security guarantee** — redaction works for the exact sentinels, and the call site routes through the sanitized global sink — **without** DB, network, global `http` races, or broad `Handler` coupling. Tradeoff (stated honestly): this proves the property compositionally rather than end-to-end through a live non-200 handler response; the end-to-end path remains only Option A/B and remains DB-gated.

## Ownership
- `internal/handler/**` and `pkg/redact/**` are **not** exclusive hotspots in `FILE_OWNERSHIP.md`; no active `IN_PROGRESS` lock on `auth.go`/`handler.go`/`redact.go` in the ledger at review time. The current `auth.go` dirty diff is native-1.7 and does **not** touch the GoogleLogin token-exchange/log lines — so a new redact-layer test is disjoint from it and from the handler file entirely.

## Verdict (design-feasibility)

- **Design correctness: SOUND.** Its decisive findings (no DI seam, global-swap risk, DB-gated `TestMain`, Option B pattern) are all confirmed against source.
- **Design completeness: PARTIAL.** It framed A vs B **both inside the DB-gated handler package**, so both fail the task's own "avoid DB/global races / no broad coupling" bar and neither executes offline. It did not surface the **`pkg/redact` seam**, which is smaller, executes offline, and has zero production coupling.
- **Recommendation to Kiro TL:** prefer **Seam 1 (redact-layer test) + Seam 2 (static call-site assertion)** as the smallest offline-executable proof. Treat **Option B** as an optional follow-up only if true end-to-end handler execution is required — and note it still needs a DB-reachable env or a `TestMain` change to actually run. **Avoid Option A** (global `DefaultClient.Transport` swap) given the explicit "avoid global races" criterion.

## Explicit non-claims
- Created only this file. No product/test/spec/task/shared/git/index edit; no `add/commit/push`; no checkbox change; no implementation.
- Read no credential/env values; no DB/network/provider/service; ran no tests here (feasibility trace only).
- I did **not** exhaustively read `SanitizeSlogAttr`'s value-vs-key redaction path; I specified it as the exact assertion the smaller-seam test must pin, rather than asserting its current behavior.
- Decision support only: this validates feasibility and proposes a seam; it authorizes no production edit and grades no task acceptance. Kiro TL adjudicates.
