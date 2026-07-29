# Feasibility design: additive Google OAuth non-200 body log test

**Author:** Kiro/Sonnet, pane `w7:p1` — read-only feasibility trace only.
**Date:** 2026-07-18T18:47:59-03:00
**Does not implement.** No product/test/shared/spec/task/git/index file was
edited or created other than this one design document.
**Adjudication authority:** Kiro TL adjudicates whether to authorize the
minimal production seam this design identifies as required, or to accept a
narrower/alternate approach instead.

## Target site

`internal/handler/auth.go`, inside `GoogleLogin` (func starts at the
`GoogleLoginRequest`/`googleTokenResponse` block, handler body from roughly
line 609 onward in the current file):

```go
// line 635
tokenResp, err := http.PostForm("https://oauth2.googleapis.com/token", url.Values{
    "code":          {req.Code},
    "client_id":     {clientID},
    "client_secret": {clientSecret},
    "redirect_uri":  {redirectURI},
    "grant_type":    {"authorization_code"},
})
...
// line 656
if tokenResp.StatusCode != http.StatusOK {
    slog.Error("google oauth token exchange returned error", "status", tokenResp.StatusCode, "body", string(tokenBody))
    writeError(w, http.StatusBadRequest, "failed to exchange code with Google")
    return
}
```

This is the same class of finding already independently reproduced for
`internal/auth/cloud_pat.go`'s `fetch()` (see
`credential-isolation-5.4-cloud-pat-body-log-test.md`) and analyzed in the
5.4 remaining-gaps closure design (§2). This document is the read-only
feasibility trace for doing the equivalent additive test at this
Google-specific site, as requested.

## Seam inspection — decisive finding

**`http.PostForm` is a package-level `net/http` function.** Per its Go
standard-library implementation, it always issues the request through
`http.DefaultClient` — there is no parameter, no `Handler` field, and no
call-site override available. The second Google call
(`http.DefaultClient.Do(userInfoReq)`, line 676) explicitly names
`http.DefaultClient` too.

Read the full `Handler` struct (`internal/handler/handler.go:98-...`,
~50 fields spanning `Queries`, `DB`, `Hub`, `Bus`, various `*Service`,
`*Cache`, `WebhookRateLimiter`, `CloudRuntime`, `LarkAPIClient`, etc.).
**There is no `HTTPClient`, `http.RoundTripper`, or any Google-specific
transport field anywhere in `Handler`.** The struct's own doc comments show
the established pattern for this kind of seam elsewhere in the same file —
e.g. `LarkAPIClient` is explicitly documented as swappable in tests
(`"tests that need a no-op behaviour can swap in lark.NewStubAPIClient(...)
directly"`) — but no equivalent exists for the Google OAuth path. This is a
structural absence, not an oversight in this design's search: a repo-wide
grep confirms only two `net/http` call sites in this file
(`http.PostForm`, `http.DefaultClient.Do`), both hardcoded, and a third,
unrelated `http.DefaultClient.Do` in `github.go:429` (a different
integration, same package) that would share any global `Transport` swap.

**Conclusion: the current code does NOT permit dependency injection for this
call site.** A test cannot substitute a custom `http.RoundTripper` or
`http.Client` without either (a) mutating the process-global
`http.DefaultClient.Transport` for the duration of the test, or (b) adding a
new field to `Handler` (a production code change).

## Option A — global `http.DefaultClient.Transport` swap (no production edit, but real risk)

A test could, in principle, do:
```go
prior := http.DefaultClient.Transport
http.DefaultClient.Transport = &staticRoundTripper{...}
defer func() { http.DefaultClient.Transport = prior }()
```
This requires **zero production code changes** — it is legal Go and would
make `http.PostForm`/`http.DefaultClient.Do` route through the synthetic
transport for the call under test.

**Risks, identified rather than dismissed:**
- `http.DefaultClient` is a **process-wide global**. Any other test running
  concurrently in the same test binary (same package, `go test` runs one
  package's tests in one process) that happens to hit `github.go:429`'s
  `http.DefaultClient.Do` — or any future code added to this package using
  the same default client — would be redirected to the synthetic transport
  too, or would race on the `Transport` field assignment itself if `t.Parallel`
  is used anywhere in the package. `handler_test.go` does not call
  `t.Parallel()` in the code reviewed for this design (not exhaustively
  verified for every test file in the package, since that would require
  reading the whole package's several thousand lines of tests — flagged as
  a limitation of this feasibility trace, not asserted as false).
- The mitigation (immediate `defer` restore, no `t.Parallel` on this test)
  is the same discipline already used successfully in the cloud-PAT test
  (`captureSlogDefault`'s `slog.SetDefault`/restore pattern) — but a global
  `http.Client` field swap is a strictly larger blast radius than a global
  `slog` default swap, because *any* code path in the process using
  `http.DefaultClient` during the test's execution window is affected, not
  just log output.
- Because `internal/handler`'s `TestMain` gates the entire package on a live
  Postgres connection (see below), this package's test binary, when it runs
  at all, runs serially by default unless individual tests opt into
  `t.Parallel()` — reducing but not eliminating the race risk (a background
  goroutine spawned by another test, or the `realtime.Hub`'s `go hub.Run()`
  started in `TestMain`, could still be mid-flight during the swap window).

**Assessment: technically possible, not recommended without an explicit
owner sign-off on the shared-global risk.** This is a "prefer custom
RoundTripper/injected client" case where the current code only offers the
global-swap variant, not the injected-client variant the task asked to
prefer.

## Option B — minimal production seam (recommended, but is a code change)

Add a `HTTPClient *http.Client` field to `Handler` (nil-safe: `GoogleLogin`
would do `client := h.HTTPClient; if client == nil { client = http.DefaultClient }`,
matching the existing `CloudPATVerifier` pattern in `internal/auth/cloud_pat.go`
almost exactly) and change `http.PostForm(...)` to
`client.PostForm(...)`/`client.Do(...)`. This is the same pattern already
proven safe and already independently reviewed for `CloudPATVerifierConfig.HTTPClient`
and for `LarkAPIClient`'s stub-swap convention in this very file.

**This is a production code edit** — explicitly out of scope for this
read-only design document, and likely out of scope for an "additive test
only" task in general, since `HTTPClient *http.Client` on `Handler` is a
shared struct that other tests/call sites also touch (`New(...)` constructor
call sites across `cmd/server/router.go` and every existing `handler_test.go`
fixture would need to keep compiling, though adding a new nil-safe field is
additive and should not break them). Recorded here as the structurally
correct fix, for Kiro TL to authorize as a separate, properly-scoped task if
desired — not performed by this document.

## Package-gate blocker (independent of the seam question)

`internal/handler/handler_test.go`'s `TestMain` connects to a real Postgres
instance (`DATABASE_URL` env var, or a hardcoded
`postgres://multica:multica@localhost:5432/multica?sslmode=disable`
fallback) and calls `pool.Ping(ctx)`. **If the DB is unreachable, `TestMain`
calls `os.Exit(0)` before `m.Run()` — this skips every test in the package
process-wide, not per-test.** This differs categorically from
`internal/auth`'s `REDIS_TEST_URL`-gated tests (which `SKIP` individually
while the rest of the package still runs) — `internal/handler`'s gate is a
package-level exit, consistent with what the prior credential-isolation and
chat-orchestration reviews in this session chain already found ("handler
package compiled only: DB-gated TestMain exited before handler tests ran").

**Consequence for this task:** even a perfectly-designed additive test in
`internal/handler` would not execute in an environment without a reachable
Postgres instance — it would compile, but `go test` would report `exit
status 0` with **zero tests actually run**, which must be reported
transparently as "compiled, not executed" rather than claimed as a passing
test run. This is a structural constraint of the package, not something a
new test file can route around without also modifying `TestMain` (out of
scope) or being placed in a different package.

## Smallest additive test design (assuming Option A or B is authorized)

If a seam is authorized (either A's global-swap or B's field-injection),
the smallest test would be:

- **File:** `internal/handler/auth_google_oauth_log_redaction_test.go` (new,
  same package `handler`, matching the existing single-package-test
  convention already used for `cloud_pat_log_redaction_test.go` in `auth`).
- **Symbols:**
  - A `staticRoundTripper` type, structurally identical to the one already
    written and reviewed for the cloud-PAT test (`statusCode int; body
    string`), reused here or duplicated (package boundary means it cannot be
    imported directly from `internal/auth` without exporting it, which
    would itself be a small production-adjacent change — likely simplest
    to duplicate the ~10-line type locally).
  - A `captureSlogDefault(buf *bytes.Buffer) (restore func())` helper,
    identical in shape to the one already written for the cloud-PAT test,
    wiring `redact.SanitizeSlogAttr` into a `slog.NewJSONHandler`.
  - One test function, e.g. `TestGoogleLoginNon200TokenBodyRedactsSentinels`,
    that:
    1. Swaps the seam (Option A: `http.DefaultClient.Transport`; Option B:
       constructs a `Handler` with `HTTPClient` set to a client using
       `staticRoundTripper`).
    2. Issues an HTTP request to the router's `POST /auth/google` (or calls
       `testHandler.GoogleLogin` directly via `httptest.NewRecorder()` +
       `httptest.NewRequest`, bypassing full router wiring — the smaller,
       more disjoint option since `GoogleLogin` doesn't touch the DB before
       reaching the token-exchange call, based on reading the function body
       above: the only DB-touching code is further down, in the
       find-or-create-user path, which the non-200 branch returns from
       *before* reaching).
    3. Sets `GOOGLE_CLIENT_ID`/`GOOGLE_CLIENT_SECRET` env vars via
       `t.Setenv` to non-empty synthetic placeholder strings (not real
       credentials — needed only to pass the `clientID == "" ||
       clientSecret == ""` guard so the handler reaches the token-exchange
       call at all).
    4. Asserts the captured log buffer does not contain a synthetic
       `access_token`/`refresh_token` sentinel embedded in the fake
       non-200 Google response body, while the status code and a
       non-secret diagnostic reason remain visible — mirroring the
       assertion shape already used and reviewed in the cloud-PAT test.
  - `t.Parallel` deliberately not called, for the same shared-global reasons
    as Option A/the cloud-PAT test.

- **Package gate to report:** as above — this test would compile but not
  execute without a reachable Postgres instance, because it lives in a
  package whose `TestMain` gates the entire binary on a DB ping. Unlike the
  `internal/auth` cloud-PAT test (no `TestMain`, no DB gate, ran
  unconditionally), this one's actual PASS/FAIL could only be demonstrated
  in an environment with `DATABASE_URL` reachable or the local Postgres
  default running — a materially different feasibility profile than the
  cloud-PAT precedent this task is modeled on.

## Disjoint ownership check

- `FILE_OWNERSHIP.md` has no entry for `internal/handler/**` under any
  agent's owned-hotspot table (same finding as the prior `auth.go` ownership
  matrix).
- `AGENT_LEDGER.md` has no row showing any agent currently `IN_PROGRESS` on
  `internal/handler/auth.go` or `internal/handler/handler.go` at the time of
  this design.
- `git status --short -- multica-auth-work/server/internal/handler/` was
  read-only inspected (not shown as a full listing here to avoid
  duplicating the already-produced `handler-auth-go-current-diff-ownership-matrix.md`,
  which already partitions this exact file's current dirty diff by hunk and
  producer). That matrix found the current `auth.go` diff belongs entirely
  to native-1.7 password provisioning and does **not** touch the
  Google-OAuth-logging lines (656) or the `GoogleLogin` token-exchange call
  (635) at all — meaning a hypothetical new test file here would be
  disjoint from that existing diff's hunks, consistent with "additive test
  only, no existing-file edits."
- If Option B (the `HTTPClient` field) were ever authorized, it would touch
  `handler.go`'s `Handler` struct definition — a file with broad,
  many-caller usage; that edit's ownership/conflict profile was not traced
  in this design (out of scope: this document does not implement Option B
  and was not asked to trace ownership for a hypothetical production edit).

## Global slog restoration / race risk (for whichever option is chosen)

Identical in shape to the cloud-PAT test's already-reviewed
`captureSlogDefault` pattern: swap `slog.Default()` for a
`redact.SanitizeSlogAttr`-wired handler, `defer restore()` immediately, no
`t.Parallel`. No new risk beyond what that precedent already established,
**except** that Option A's `http.DefaultClient.Transport` swap stacks an
*additional* global-state risk (the HTTP client, not just the logger) on
top of the already-accepted logger-swap risk — this compounding is the main
new consideration this design surfaces beyond the cloud-PAT precedent.

## Recommendation to Kiro TL

1. **The current code does not permit clean dependency injection** for the
   Google OAuth token-exchange call — confirmed by reading the actual
   `Handler` struct and the two hardcoded `net/http` call sites.
2. Two paths exist: **Option A** (global `http.DefaultClient.Transport` swap,
   zero production edit, real but bounded/precedented shared-global risk) or
   **Option B** (add a nil-safe `HTTPClient` field to `Handler`, matching
   the existing `CloudPATVerifierConfig`/`LarkAPIClient` seam conventions,
   a small production edit requiring separate authorization).
3. **Independent of the seam choice, `internal/handler`'s package-level
   `TestMain` DB gate means any new test here would only executably run
   with a reachable Postgres instance** — a materially different
   feasibility profile than the `internal/auth` cloud-PAT precedent, which
   had no such gate. This should be weighed before authorizing implementation:
   the test can be written and will compile, but "actual assertion" evidence
   (not just "compiled, not executed") requires a DB-reachable environment,
   consistent with what prior reviews in this session chain already
   disclosed for this package.
4. No implementation was performed by this document. If TL authorizes
   proceeding, recommend Option A for a strictly test-only change (smallest
   blast radius consistent with "no existing-file edits"), with the
   compounding global-state risk explicitly accepted by TL rather than
   assumed away, and with the DB-gate caveat reported honestly in whatever
   evidence artifact follows rather than claimed as an executed pass if the
   local environment lacks Postgres.
