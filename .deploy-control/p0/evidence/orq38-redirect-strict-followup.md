# ORQ-38 — redirect strictness follow-up

The peer-review BLOCK on `c536b60` was addressed in a new commit (no amend):
`0ecc6f4e839e024d0cc81cb5c5c33ae0ec4dabd2`.

## Corrections

- `GetJSONExpectedStatus` now clones the configured `http.Client` and sets
  `CheckRedirect` to `http.ErrUseLastResponse`; legacy `GetJSON` behavior is
  unchanged.
- Strict UUID-short/prefix issue resolution now uses
  `fetchIssueCandidatesStrict`, which requires exact HTTP 200 through
  `GetJSONExpectedStatus`.
- Tests prove a 302 `Location` destination is never reached, and list 201 or
  redirect responses fail before any final response is read or printed.

## Focused gates

- `go test ./internal/cli -run '^TestGetJSONExpectedStatus$' -count=1`: PASS.
- `go test ./cmd/multica -run
  '^(TestRequireIssueIdentity|TestRunIssueGetFailClosed|TestResolveIssueRefStrictPrefixRequiresExactListStatus)$'
  -count=1`: PASS.
- `go vet ./internal/cli ./cmd/multica`: PASS.
- `gofmt` and `git diff --check`: PASS.
- Private mode-0700 Go cache was removed after testing.

The change is a four-file subset of the original six-file lock; no unrelated
file, ORQ-26 artifact, board, remote, push, or PR was touched.
