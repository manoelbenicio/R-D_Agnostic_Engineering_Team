# Additive test: agent-credential-isolation 5.4 — Cloud PAT non-200 body log redaction

**Producer:** Kiro (producer role). **Does NOT self-accept.** Kiro TL
adjudicates after a distinct independent review.
**Date:** 2026-07-18T18:42:09-03:00 to completion

## Motivation / traceability

This is a parallel, additive test to the Google OAuth non-2xx body logging
coverage already independently reproduced in
`credential-isolation-5.4-codebase-critique.md` (§7). `internal/auth/cloud_pat.go`'s
`fetch()` has the same-shaped risk: a non-200 response from the Multica Cloud
Fleet PAT-verify endpoint is logged with a truncated body snippet under the
`"body"` key —
```go
slog.Warn("cloud_pat: verify returned non-200", "status", resp.StatusCode, "body", snippet)
```
No test previously exercised this call site's log-safety behavior end-to-end
through the real `Verify`/`fetch` path and the production
`redact.SanitizeSlogAttr` hook. This task adds that coverage without
modifying `cloud_pat.go` itself.

## Pre-edit conflict check (Golden Rule)

`git status --short -- multica-auth-work/server/internal/auth/` showed
`jwt.go` (M), `jwt_configuration_test.go`/`recent_auth.go`/`recent_auth_test.go`
(??) already present/dirty — all pre-existing, unrelated native-1.7
password-provisioning work per the earlier-reviewed ownership matrix
(`handler-auth-go-current-diff-ownership-matrix.md`). **`cloud_pat.go` and
`cloud_pat_test.go` were clean** (no local modification) before this task —
zero conflict with the target file or the new disjoint test file.
`FILE_OWNERSHIP.md` has no entry for `internal/auth/**`. `AGENT_LEDGER.md`
has no row showing any agent `IN_PROGRESS` on `cloud_pat.go` or
`internal/auth/**` at check-in time. `cloud_pat.go`'s hash was identical
before and after this task (`98a4aadf...`), confirming it was not edited.

## Provenance (SHA-256)

| File | Hash |
|---|---|
| `server/internal/auth/cloud_pat.go` (read-only, unmodified) | `98a4aadf4dd9a236a388bcdfaa9434d83a779916d25d8afdb53c67a09a7e2778` |
| `server/internal/auth/cloud_pat_log_redaction_test.go` (new) | `1896f90dcd791a9b455f1c940928e7b608ce64f1403b7bcf2986b80a30a55d49` |
| `server/pkg/redact/redact.go` (dependency, unmodified) | `f409ba8a9f3e63618d59c5a8692296f8f7c019c9e558576b8786a058fbf68a5c` |

## Test design

Package: `auth` (same-package test, matching existing convention in
`internal/auth/*_test.go`).

- **`staticRoundTripper`** — a minimal `http.RoundTripper` returning a
  pre-built `*http.Response` for any request. No `httptest.Server`, no
  listener, no socket, no DNS, no real network of any kind — the HTTP
  client's `Transport` field is simply swapped for this synthetic
  implementation, so `CloudPATVerifier.fetch`'s real `v.http.Do(req)` call
  executes exactly as in production, just against an in-process fake.
- **`captureSlogDefault`** — temporarily swaps the process-global
  `slog.Default()` for a `slog.NewJSONHandler` wired to the *actual*
  production `redact.SanitizeSlogAttr` `ReplaceAttr` hook (the same function
  `internal/logger.Init`/`NewLogger` wire in production), writing to an
  in-memory buffer. Returns a `restore` closure; every test calls
  `defer restore()` immediately after capturing, so the prior global default
  is always restored even on test failure. `t.Parallel` is deliberately never
  called, avoiding any race on the shared global.
- Three tests, all invoking the real `CloudPATVerifier.Verify` → `fetch`
  path (not a mock of `fetch` itself):
  1. `TestCloudPATVerifyNon200BodyRedactsAccessTokenAndAPIKeySentinels` — a
     synthetic 400 response body with fake `access_token`/`api_key` fields;
     asserts both sentinels are absent from captured output, a
     `[REDACTED...]` placeholder is present, and the status code (400),
     message (`"cloud_pat: verify returned non-200"`), and a non-secret
     diagnostic reason (`"malformed token"`) remain visible.
  2. `TestCloudPATVerifyNon200BodyRedactsBearerTokenSentinel` — a synthetic
     500 response body with a fake JWT-shaped bearer token embedded in a
     message field; asserts the sentinel is absent and the status code (500)
     remains visible.
  3. `TestCloudPATVerifySafeNon200BodyIsPreservedForDiagnostics` — a
     synthetic 429 response with **no** secret-shaped field at all; asserts
     the safe diagnostic text and status code (429) pass through unaltered,
     proving the redaction assertions in tests 1-2 are not vacuously true.

All token/secret values are explicit, clearly-labeled synthetic sentinels
(`synthetic-access-token-sentinel-...`, `synthetic-api-key-sentinel-...`,
a JWT-shaped string containing the literal substring `SYNTHETIC`). The
token passed to `Verify` itself (`"mcn_synthetic-token-not-real"`) is also a
labeled synthetic value, never a real credential.

## Execution proof (bounded, offline, synthetic-only)

Environment: `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`, pinned local
`/home/dataops-lab/go-sdk/bin/go` (go1.26.4, linux/amd64).

```
gofmt -l internal/auth/cloud_pat_log_redaction_test.go
  => (empty output — file already gofmt-clean)

go build ./internal/auth/...   => exit 0
go vet   ./internal/auth/...   => exit 0 (no findings, test files included)

go test -v -count=1 ./internal/auth/ -run \
  'TestCloudPATVerifyNon200BodyRedactsAccessTokenAndAPIKeySentinels|TestCloudPATVerifyNon200BodyRedactsBearerTokenSentinel|TestCloudPATVerifySafeNon200BodyIsPreservedForDiagnostics'
  => 3 named tests, 3 === RUN, 3 --- PASS, 0 --- FAIL, exit 0, 0.013s

go test -v -count=20 ./internal/auth/ -run '<same 3 tests>'
  => 3 tests x20 = 60 === RUN, 60 --- PASS, 0 --- FAIL, exit 0, 0.085s

go test -race -count=1 ./internal/auth/ -run '<same 3 tests>'
  => ok, exit 0, 1.065s, no data races reported

go test -v ./internal/auth/...   (full package)
  => ok, exit 0, 0.091s — ALL existing tests PASS; three pre-existing tests
     (TestMembershipCache_SetGetInvalidate, TestMembershipCache_TTL,
     TestMembershipCache_IsolatesKeysByUser, TestPATCache_SetGetInvalidate,
     TestPATCache_TTL, TestPATCache_Set_RespectsClampedTTL) --- SKIP with
     "REDIS_TEST_URL not set" — a PRE-EXISTING package gate unrelated to this
     task's new tests, which ran and passed unconditionally in the same
     execution. No DB/Redis was started, connected to, or required by this
     task's new tests.
```

**Actual assertion/run counts:** 3 distinct named tests, each asserting
absence of its specific sentinel(s), presence of a redaction placeholder
(tests 1-2), and presence of non-secret diagnostic context (status code,
message, safe text) in all three tests. 60 total test executions across the
count=20 run, 0 failures, 0 races.

**Package gate identified and reported (as instructed):** `internal/auth`'s
existing `MembershipCache`/`PATCache` tests gate on the `REDIS_TEST_URL`
environment variable and SKIP (not fail, not error) when absent — this is a
pre-existing package convention, not something introduced or triggered by
this task's new tests, and no Redis connection was made by this task's tests
or by running the full package suite in this environment.

## What this test does and does not claim

- **Does** prove, executably and through the real `Verify`/`fetch` code
  path (not a re-implementation or mock of the redaction logic), that a
  non-200 Cloud Fleet response body containing recognizable
  `access_token`/`api_key`/bearer-token-shaped sentinels does not reach
  captured log output when routed through the production
  `redact.SanitizeSlogAttr` hook, while status/diagnostic context remains
  useful.
- **Does not** claim pattern-independent coverage — `redact.Text()`'s
  regex/literal pattern set is fixed (the same documented residual as
  R-5.4-B / the Google OAuth case); a token shape with no recognizable
  pattern could still be missed. This test narrows the *evidence* gap for
  this specific call site; it does not change the underlying mechanism.
- **Does not** modify `cloud_pat.go`, any existing test file, `tasks.md`,
  the shared ledger, `STATE.md`, git index, or any credential/env value.
- **Does not** touch any live Fleet service, listener, socket, DNS, Redis,
  or database. `staticRoundTripper` never performs network I/O.

## Non-claims / recommendation to TL

- Producer does **not** self-accept. Kiro TL adjudicates after a distinct
  independent reviewer reproduces the build/vet/test evidence above.
- Recommend the independent reviewer confirm the `staticRoundTripper`
  approach genuinely never touches the network (e.g. via `strace`/`lsof`
  or by inspecting Go's `net/http` client internals when `Transport` is
  fully substituted) if network-avoidance verification beyond code reading
  is desired — this producer's own verification was limited to code
  inspection (no import of `net`, `net/http/httptest`, or any listener API
  in the new test file) plus the absence of any bind/connect error or
  timeout during execution (all three tests completed in ~0ms each).
