# REG — PD-08 credential-source wiring regression

- agent: **Opus48#C** · lane **regression** · task **REG-CREDSOURCE** · pane `w8:p1`
- as-of (UTC): `2026-07-22T22:06Z` · HEAD `a6d50986aee7f1981a323b29c9d90f175b4b6f8b`
- lock: `multica-auth-work/server/internal/daemon/credsource_regression_test.go` (NEW, added) + this evidence file.
- check-in: `.deploy-control/p0/checkins/Opus48-C__REG-CREDSOURCE__20260722T220447Z.json`
- go: `/home/ec2-user/goroot/go/bin/go` (go1.26.1); GOCACHE/GOTMPDIR under `/tmp`.

## 0. Result

**DONE — regression added and green.** New focused test file asserts the PD-08 credential-source wiring intent of the `cmd_daemon.go` branch, verified at the `daemon`/`brain_integration` level.

## 1. What the regression asserts (PD-08)

Source of the branch (`multica-auth-work/server/cmd/multica/cmd_daemon.go:438-444`):
```go
if cfg.AgentBrain.DevelopmentEnabled && cfg.AgentBrain.Neutral.Gateway.Required {
    d = daemon.NewWithAgentBrainDependencies(cfg, logger, daemon.AgentBrainDependencies{
        CredentialSource: daemon.FileCredentialSource{},
        HTTPClient:       &http.Client{Timeout: 10 * time.Second},
    })
} else {
    d = daemon.New(cfg, logger) // no AgentBrainDependencies -> nil CredentialSource
}
```
Gate that PD-08 relies on (`internal/daemon/brain_integration.go:197`): `if r.dependencies.CredentialSource == nil { ... "credential_source_unavailable" }`. `FileCredentialSource` is the production `gateway.CredentialSource` (`credential_file_source.go:40`, `var _ gateway.CredentialSource = FileCredentialSource{}`).

New tests (`internal/daemon/credsource_regression_test.go`, package `daemon`):
| Test | Asserts |
|---|---|
| `TestPD08CredentialSourceWiredOnlyForDevGatewaySlice` | across all 4 (DevelopmentEnabled × Gateway.Required) combos, a non-nil `FileCredentialSource` is wired **iff both are true** (else nil); wired value is exactly `daemon.FileCredentialSource` |
| `TestPD08AdmissionFailsClosedWithoutCredentialSource` | "else" branch consequence: nil `CredentialSource` → `admitTask` fails closed with deterministic class `credential_source_unavailable` (never admits) |
| `TestPD08AdmissionPassesCredentialGateWithFileCredentialSource` | dev+gateway branch consequence: `FileCredentialSource{}` clears the PD-08 nil gate — admission is never `credential_source_unavailable` (may fail at a later gate, which is correct) |

The predicate mirror (`credentialSourceForBranch`) documents `cmd_daemon.go`'s intent; the two `admitTask` tests bind the assertion to real daemon behavior (not a tautology) via the existing `newAgentBrainRuntime`/`admitTask`/`syntheticGatewayTask`/`newSyntheticGateway` harness.

## 2. Evidence — exact commands + exit codes

Run from `multica-auth-work/server`, `GOROOT=/home/ec2-user/goroot/go`, `GOCACHE=/tmp/l5-gocache`, `GOTMPDIR=/tmp`:
```text
gofmt -l internal/daemon/credsource_regression_test.go        -> (empty; clean) ; exit 0
go vet ./internal/daemon/                                     -> clean ; exit 0
go test ./internal/daemon/ -run 'TestPD08' -count=1 -v        -> PASS 3/3 ; ok 0.006s ; exit 0
    --- PASS: TestPD08CredentialSourceWiredOnlyForDevGatewaySlice
    --- PASS: TestPD08AdmissionFailsClosedWithoutCredentialSource
    --- PASS: TestPD08AdmissionPassesCredentialGateWithFileCredentialSource
go test ./internal/daemon/ -count=1  (full package regression) -> ok 15.174s ; exit 0
git diff --check -- .../credsource_regression_test.go         -> PASS ; exit 0
```

## 3. Scope / non-claims
- Added ONE new test file (`credsource_regression_test.go`, untracked `??`). **No product code edited** — the ` M cmd_daemon.go` / ` M brain_integration.go` shown by `git status` are pre-existing modifications from prior P0 lanes, NOT this lane.
- `-race` not run (cgo/gcc absent); the assertions are deterministic.
- No OmniRoute internals: `newSyntheticGateway` is an in-process httptest stub; `FileCredentialSource` reads no real secret in this hermetic test (no secret read/print/hash).
- No install; `go.mod`/`go.sum` unchanged. No live run/inference. OpenSpec checkbox not mutated.
- The positive test intentionally asserts only the ABSENCE of `credential_source_unavailable` (a wired source clears the PD-08 nil gate); it does not assert full admission success, because the file-backed reader has no real secret file in a hermetic run.

## 4. Status
- STATUS: DONE.
- DELIVERED: `internal/daemon/credsource_regression_test.go` (3 tests) asserting PD-08 credential-source wiring (dev+gateway → non-nil FileCredentialSource; else nil → fail-closed) at the daemon/brain_integration level; all green + full-package regression green.
- FILES: created `credsource_regression_test.go` + this evidence. No product change.
