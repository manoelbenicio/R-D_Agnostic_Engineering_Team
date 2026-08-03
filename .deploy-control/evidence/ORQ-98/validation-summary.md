# ORQ-98 Focused Validation Evidence

- Evidence captured through: 2026-08-03T04:36:47Z
- Worktree: `/home/ec2-user/worktrees/multica-orq98-gap07-redaction`
- Branch: `recovery/orq98-gap07-redaction`
- Tested base: `18c184218a34bdb4631ef210c7f27ec9cdffa4dd`
- Base tree: `6574d7616010f1d4e5e5534299b5e3e9b214433d`
- Dependency mode: offline (`GOPROXY=off`, `GOSUMDB=off`) using the previously verified
  module cache `/tmp/orq87-go-official/mod` and isolated build/GOPATH caches.
- Credential/config token files accessed: none.

## Acceptance evidence

| Criterion | Command slice | UTC | RC | Result |
|---|---|---:|---:|---|
| Cloud PAT nominal discovery | `go test ./internal/auth -list '^TestCloudPAT(Non200BodyDoesNotReachRawLogAttributes|VerifyNon200BodyRedactsSensitiveFields)$'` | 04:33:43–04:33:45 | 0 | New focused test discovered. |
| Google nominal discovery | `go test ./internal/handler/orq98redaction -list '^TestGoogleLoginNonOK'` | 04:33:55–04:34:07 | 0 | Both focused tests discovered without the parent package's database-gated `TestMain`. |
| Cloud PAT raw-attribute redaction | `go test ./internal/auth -run '^TestCloudPATNon200BodyDoesNotReachRawLogAttributes$' -count=1 -v` | 04:34:16 | 1 | **FAIL:** sentinel reached captured slog output or attributes. |
| Google raw-attribute redaction and safe diagnostic control | `go test ./internal/handler/orq98redaction -run '^TestGoogleLoginNonOK(TokenBodyDoesNotReachLogAttributes|SafeDiagnosticRemainsDiscoverable)$' -count=1 -v` | 04:34:16–04:34:17 | 1 | Redaction assertion **FAIL**; safe diagnostic discovery **PASS**. |
| Cloud PAT race slice | Same Cloud PAT test with `-race` | 04:34:28–04:35:56 | 1 | Same sentinel disclosure failure; no race report preceded it. |
| Google race slice | Same two Google tests with `-race` | 04:34:28–04:36:07 | 1 | Same redaction failure; safe diagnostic **PASS**; no race report preceded it. |
| Focused test compilation | `go test ./internal/auth ./internal/handler/orq98redaction -run '^$' -count=1` | 04:34:28–04:34:32 | 0 | Both test packages compile. |
| Focused vet | `go vet ./internal/auth ./internal/handler/orq98redaction` | 04:34:28–04:34:42 | 0 | PASS. |
| Focused product build | `go build ./internal/auth ./internal/handler` | 04:36:40 | 0 | PASS. |
| Existing sanitized-handler comparison | `go test ./internal/auth -run '^TestCloudPATVerify(Non200BodyRedactsAccessTokenAndAPIKeySentinels|Non200BodyRedactsBearerTokenSentinel|SafeNon200BodyIsPreservedForDiagnostics)$' -count=1 -v` | 04:36:19–04:36:20 | 0 | Three tests PASS when the test handler explicitly installs `redact.SanitizeSlogAttr`. |
| Formatting/diff hygiene | `gofmt` on the two new tests; `git diff --check` | before and after focused runs | 0 | PASS. |

## Contradiction and bounded blocker

The two direct seam tests use a raw JSON slog handler without a test-only replacement hook.
Both prove that their synthetic sentinel reaches emitted slog output/attributes on the recorded
failure paths. The Google safe-diagnostic control passes, proving the handler was exercised and
the assertion is not vacuous.

The existing Cloud PAT redaction tests pass only with a custom handler that explicitly sets
`ReplaceAttr: redact.SanitizeSlogAttr`. At tested base, `internal/logger/logger.go` constructs
`tint.NewHandler` without a visible `ReplaceAttr`/`SanitizeSlogAttr` hook. No product change is
authorized by this auditability-only card, so this result remains **BLOCKED** pending a separately
authorized implementation correction and independent rerun.

No test output records the sentinel value, credential value, or live log content.
