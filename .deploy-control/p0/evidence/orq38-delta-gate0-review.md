# ORQ-38 — Delta Gate 0 review

**Verdict: BLOCK (fail closed).** Gate 0 was accepted for local differential
review and bounded build/test work, but the required S01–S08 DB-backed run
cannot be proven because no private ephemeral database is available. No
commit or peer-review request is authorized.

## Scope and differential

- Worktree: `/home/ec2-user/workspace/worktrees/gtl-orq38-get-contract`
- Branch: `agent/codex-b/orq38-get-contract`
- Baseline HEAD: `0cb8aeb`
- Merge-base observed locally: `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f`
- The worktree delta is exactly six paths, matching the lock:
  `cmd_id_resolver.go`, `cmd_issue.go`, `internal/cli/client.go`, and the
  three corresponding contract tests (CLI resolver, expected-status client,
  and handler workspace scope).
- `git diff --check` passed. No ORQ-26, remote, PR, board, or unrelated file
  was touched.

## Local checks that are green

Using Go 1.26.1 from `/home/ec2-user/goroot/go/bin/go`, `GOWORK=off`,
`GOFLAGS=-mod=readonly`, `CGO_ENABLED=0`, and private mode-0700 caches (about
223 MB, below the 2 GiB cap):

- `go test ./internal/cli -run '^TestGetJSONExpectedStatus$' -count=1`: PASS
- `go test ./cmd/multica -run '^(TestRequireIssueIdentity|TestRunIssueGetFailClosed)$' -count=1`: PASS
- `go vet ./internal/cli ./cmd/multica`: PASS
- `gofmt -d` over all six locked files: no output

## Required DB gate is unavailable

- Docker is not installed on this host.
- `pg_isready -h 127.0.0.1 -p 5432` reports `no response`.
- No isolated/private PostgreSQL DSN or ephemeral database was prepared.
- The handler `TestMain` exits 0 when its database connection fails; that is a
  documented false-green path, not evidence of S01–S08 execution.

Therefore S01–S08 were **not run** and cannot be marked pass. The zero-skip
gate and baseline-differential “no new failures” gate remain unresolved.

## Exact next action

Prepare an isolated disposable PostgreSQL instance and provide its private
`DATABASE_URL` (never the product database). Then run the S01–S08 matrix with
JSON output and require a nominal `Action=pass` event for every test, no skip,
and a fail-fast TestMain. Compare the resulting failure set with the same
baseline harness. Only if the differential is empty and the six-file lock is
unchanged may a local commit and peer-review request be prepared.

No commit, push, PR, board mutation, ORQ-26 lock change, or product-DB access
was performed.
