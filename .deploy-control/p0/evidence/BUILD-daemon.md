# BUILD-daemon — focused build + tests (server-build-test lane)

- agent: `Codex56#A` · lane: `server-build-test` · task: `BUILD-DAEMON` · pane: `w7:p3`
- lock: `.deploy-control/p0/evidence/BUILD-daemon.md`
- scope: focused build + vet + tests for `internal/daemon`, `internal/daemon/brain`, `internal/daemon/runtimeenv`
- toolchain: `/home/ec2-user/goroot/go/bin/go` (go1.26.1); caches on `/tmp` (`GOCACHE=/tmp/c3-gocache`, `GOTMPDIR=/tmp/c3-gotmp`, `GOMODCACHE=/tmp/c3-gomod`)
- constraints: **no deploy**, no inference, no secret, no commit/push. ORQ2 repo, read-only source (no edits).

> STATUS: **DONE / VERIFIED GREEN** — build, vet and focused tests for all three packages pass with
> executed evidence. One non-blocking pre-existing `gofmt` finding routed to the daemon owner (not this lane's edits).

## Environment
- cwd: `multica-auth-work/server`; git HEAD `a6d50986…`.
- disk at run: root 4.4G free, /tmp 3.6G free. Module deps fetched into `/tmp` GOMODCACHE (no root pollution).

## Commands, exit codes, results

| Command | Exit | Result |
|---|---:|---|
| `go build ./internal/daemon/...` | 0 | PASS (compiles; deps downloaded to /tmp GOMODCACHE) |
| `go vet ./internal/daemon ./internal/daemon/brain ./internal/daemon/runtimeenv` | 0 | PASS (clean) |
| `go test -count=1 ./internal/daemon` | 0 | `ok` 15.294s |
| `go test -count=1 ./internal/daemon/brain` | 0 | `ok` 0.015s |
| `go test -count=1 ./internal/daemon/runtimeenv` | 0 | `ok` 0.037s |
| `gofmt -l internal/daemon internal/daemon/brain internal/daemon/runtimeenv` | 0 | 6 pre-existing unformatted files listed (see finding) |

Combined test invocation:
`go test -count=1 ./internal/daemon ./internal/daemon/brain ./internal/daemon/runtimeenv` → exit 0, all `ok`.

## Finding (non-blocking; NOT this lane's edits) → route to daemon owner (L1)
`gofmt -l` flags these already-unformatted files in the daemon tree (intentionally-dirty tree; I made
no edits here):
```
internal/daemon/client.go
internal/daemon/coauthor_enabled_test.go
internal/daemon/commitledger/ledger_test.go
internal/daemon/qa_conformance_test.go
internal/daemon/repocache/cache.go
internal/daemon/runtime_profile_test.go
```
These do not affect build/vet/test (all green). Fixing them is the daemon/L1 owner's scope; `runtimeenv`
and `brain` are gofmt-clean.

## Non-claims
- No deploy, no inference, no secret, no source edits (read-only build/test).
- `gofmt` findings are pre-existing and out of this lane's ownership; not fixed here.
- Server-wide `go build ./...` / full `go test ./...` not run (scope = the three named packages).
