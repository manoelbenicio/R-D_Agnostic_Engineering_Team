# Clean-room atomic review — chat orchestration tasks 1.1 and 1.4

## Golden Rule provenance

- **Runner/reviewer:** Codex56#B, independent clean-room verifier. This identity is distinct from prior acceptance reviewer Codex#56#A. Kiro TL adjudicates; root integrates.
- **Check-IN:** START was communicated in the assignment thread before any tool action: create one `/tmp` clean room from committed `HEAD`, overlay exactly three candidate files, run only offline/synthetic checks, remove that exact directory, and write only this artifact. The exact START UTC was not captured by the shell and is intentionally not invented; this is a provenance limitation. The first captured execution timestamp was `2026-07-18T21:35:15Z`.
- **Check-OUT:** `2026-07-18T21:35:53Z`, DONE. `/tmp/chat14-cleanroom.D2iHYJ` was removed with `find /tmp/chat14-cleanroom.D2iHYJ -depth -delete`; an immediate existence check returned `CLEANROOM_REMOVAL=PASS`.
- **Repository boundary:** no repository product, test, shared planning, OpenSpec, task, spec, git index, or git ref was changed. No credential, environment value, database, network, provider, or service was accessed. The only retained write is this artifact.

## Verdict

**Technical verdict: ACCEPT. Push-readiness verdict: READY-CANDIDATE for the exact three-file atom, subject to Kiro TL adjudication and root integration.**

The candidate compiles and its focused daemon behavior genuinely executes against committed-HEAD dependencies. All 24 deterministic AST assertions pass against the overlaid production protocol constant. The handler test package compiles, contains all six focused handler tests, and passes vet, but the handler tests were deliberately not executed because package `TestMain` attempts PostgreSQL before `m.Run()` and can exit success with zero tests. This review does not convert that compile-only boundary into runtime evidence.

The exact accepted atom is:

| SHA-256 | Path |
|---|---|
| `50406c891be39a9f645a2e1b957919c43ed879756a77a65c71a6afa11a3029fd` | `multica-auth-work/server/internal/daemon/prompt_test.go` |
| `a2998f923852a455782f37d4416bbfb5a74750ea19b2f6d87dc5f56cc262e80a` | `multica-auth-work/server/internal/handler/squad_briefing.go` |
| `3b12615543440f52773d0d1d7bed4277dd6c1b0fc835f7bfd2ee3f12cd823d9c` | `multica-auth-work/server/internal/handler/squad_briefing_test.go` |

Canonical path-ordered three-file manifest SHA-256: `f7d7a2ef786a87d4a9aa6b351663f247886bdf99f2db3e9264fb406424629a32`, exactly matching READY-1 in `active-accepted-push-candidate-matrix.md`.

## Inputs and clean-room construction

| Input | Current SHA-256/value |
|---|---|
| committed `HEAD` | `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f` |
| pinned Go | `/home/dataops-lab/go-sdk/bin/go`, `go version go1.26.4 linux/amd64` |
| candidate matrix | `b61cb4f90d9432234419557638782470c868cf5b38c76ba71e43219dff76c830` |
| accepted evidence | `c7064375c36e797e6989231b532441af6fbd47371420e6f22612f403a945d473` (`chat-orchestration-1.1-1.4.md`) |

Exact materialization commands, from repository root:

```sh
tmp=$(mktemp -d /tmp/chat14-cleanroom.XXXXXX)
mkdir -p "$tmp/repo" "$tmp/head-three" "$tmp/verifier"
git archive --format=tar HEAD | tar -xf - -C "$tmp/repo"
cp "$tmp/repo/multica-auth-work/server/internal/daemon/prompt_test.go" "$tmp/head-three/prompt_test.go"
cp "$tmp/repo/multica-auth-work/server/internal/handler/squad_briefing.go" "$tmp/head-three/squad_briefing.go"
cp "$tmp/repo/multica-auth-work/server/internal/handler/squad_briefing_test.go" "$tmp/head-three/squad_briefing_test.go"
cp multica-auth-work/server/internal/daemon/prompt_test.go "$tmp/repo/multica-auth-work/server/internal/daemon/prompt_test.go"
cp multica-auth-work/server/internal/handler/squad_briefing.go "$tmp/repo/multica-auth-work/server/internal/handler/squad_briefing.go"
cp multica-auth-work/server/internal/handler/squad_briefing_test.go "$tmp/repo/multica-auth-work/server/internal/handler/squad_briefing_test.go"
```

The accepted evidence file was read and hash-checked but was not overlaid into the build tree because it is outside build scope.

## Targeted diff and formatting

For each overlaid file:

```sh
git diff --no-index --check "$tmp/head-three/<file>" "$tmp/repo/<candidate-path>"
```

Each returned `rc=1` because the no-index inputs differ, with `diagnostics=0`; therefore there was no whitespace error. Diff statistics were:

- `prompt_test.go`: 44 insertions;
- `squad_briefing.go`: 54 insertions, 18 deletions;
- `squad_briefing_test.go`: 152 insertions.

Formatting command:

```sh
/home/dataops-lab/go-sdk/bin/gofmt -l \
  "$tmp/repo/multica-auth-work/server/internal/daemon/prompt_test.go" \
  "$tmp/repo/multica-auth-work/server/internal/handler/squad_briefing.go" \
  "$tmp/repo/multica-auth-work/server/internal/handler/squad_briefing_test.go"
```

Output: empty; all three files are gofmt-clean.

## Deterministic AST assertions

A standalone stdlib verifier parsed the actual overlaid `squad_briefing.go`, located `squadOperatingProtocol` by AST name, and evaluated only compile-time `+` concatenations of string literals. It executed exactly:

- 1 exact standalone marker assertion;
- 11 required identity/OpenSpec/delegation-only clauses;
- 2 forbidden former escape-hatch clauses;
- 5 required-step presence assertions;
- 4 adjacent ordering assertions;
- 1 complete strict-order assertion.

Final commands:

```sh
/home/dataops-lab/go-sdk/bin/gofmt -w "$tmp/verifier/main.go"
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off \
  /home/dataops-lab/go-sdk/bin/go build -o "$tmp/verifier/chat14verify" "$tmp/verifier/main.go"
"$tmp/verifier/chat14verify" \
  "$tmp/repo/multica-auth-work/server/internal/handler/squad_briefing.go" \
  | tee "$tmp/ast.txt"
```

Actual result: `24 ASSERT PASS`, `0 ASSERT FAIL`, final line `PASS: 24 deterministic AST assertions`. Verifier source SHA-256 was `a85acc2aaa8c7d2b3b6cddcf085a5f05f465748af5d569259e44904afb7645f3`; successful transcript SHA-256 was `7b4eb129abdc80df3fafb923f91e02e4ed19c5b99ca6c1799fc17617fd541c92` before clean-room removal.

Verifier correction disclosure: three preflight invocations executed zero assertions—(1) `go run main.go <reviewed.go>` was rejected because Go treated both files as source from different directories; (2) `go run main.go -- <reviewed.go>` reached the verifier usage guard because this toolchain preserved `--` in `os.Args`; (3) the first compiled verifier assumed one `BasicLit` and panicked because the production constant is a compile-time string concatenation. The verifier was corrected to recursively accept only AST string literals joined by `token.ADD`, then rebuilt and produced the genuine 24/24 result above. None of these preflight failures changed or exercised the candidate.

## Genuine daemon execution

Working directory: `$tmp/repo/multica-auth-work/server`.

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off \
  /home/dataops-lab/go-sdk/bin/go test -v -count=20 ./internal/daemon \
  -run '^(TestSquadLeaderMarkerDetectionExact|TestBuildPromptSquadLeaderNoActionForMemberTrigger|TestBuildPromptSquadLeaderNoActionForAgentTrigger|TestBuildPromptNonSquadLeaderNoRule|TestBuildPromptSquadLeaderNoActionProhibition)$'
```

Actual output summary: `100 === RUN`, `100 --- PASS`, `0 FAIL`, `0 SKIP`; package result `ok .../internal/daemon 0.041s`. Transcript SHA-256: `fabc44607adbf681ab1595166c34f35d07e73335749535436a7996bf09efd478`.

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off \
  /home/dataops-lab/go-sdk/bin/go test -race -v -count=1 ./internal/daemon \
  -run '^(TestSquadLeaderMarkerDetectionExact|TestBuildPromptSquadLeaderNoActionForMemberTrigger|TestBuildPromptSquadLeaderNoActionForAgentTrigger|TestBuildPromptNonSquadLeaderNoRule|TestBuildPromptSquadLeaderNoActionProhibition)$'
```

Actual output summary: `5 === RUN`, `5 --- PASS`, `0 FAIL`, `0 SKIP`; race-clean package result `ok .../internal/daemon 1.067s`. Transcript SHA-256: `bda44745e08ea12c8ef53b74d8f9d97f2117278432af7ea43468076c51152c88`.

## Handler compile boundary and false-green prevention

The handler package was never executed. In committed HEAD, `internal/handler/handler_test.go:38-54` reads `DATABASE_URL`, substitutes a localhost PostgreSQL DSN when empty, constructs/pings a pool, and calls `os.Exit(0)` before `m.Run()` when unavailable. Running a focused `go test` would therefore either contact a prohibited service or report a false-green zero-test success.

Instead, the exact commands were:

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off \
  /home/dataops-lab/go-sdk/bin/go test -c -o "$tmp/handler.test" ./internal/handler
/home/dataops-lab/go-sdk/bin/go tool nm "$tmp/handler.test" \
  | rg 'TestSquadOperatingProtocol(MarkerExact|MandatoryOpenSpecGate|DelegationSequence|NoRegression|DelegationOnlyInvariant|SynthesisAllowed)$'
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off \
  /home/dataops-lab/go-sdk/bin/go vet ./internal/handler
```

Actual result: test-binary compile exit 0; exactly 6 focused test symbols present; vet exit 0. Symbol transcript SHA-256: `c56d182dd5514f2511ecd7025ae0c54e33748f13b7845bf4acd4fd4d70194682`. Handler runtime assertion count is truthfully **zero**.

## Dependency completeness and excluded chat files

The three-file candidate compiled entirely against committed `HEAD`; no other dirty-worktree product/test file was copied. The excluded chat task-1.2/1.3 files in the clean room matched committed HEAD exactly:

| Excluded file | HEAD and clean-room SHA-256 | Match |
|---|---|---|
| `internal/handler/agent.go` | `f24b77d9e1faf9738827964d68831091baff024fd63b1f6ddc69ef8b4db1d98a` | yes |
| `internal/handler/chat.go` | `5d02e22d53c607e03941e9193c5758c7481053034f6da8869d0f4ad6b937d79b` | yes |
| `internal/handler/chat_test.go` | `d75db3e08c274579364ea67d697ff4f6930e9af6b8ac20f54ef975c5877b0658` | yes |
| `internal/handler/workspace.go` | `8587dbe0079b86799d69f947a55805a1888b3cb715666ad2ef038ec94ea0995d` | yes |

This proves the compile, vet, daemon tests, and AST assertions did not rely on the current excluded changes in those files. The production change is confined to the protocol constant; the other two atom files are tests. Existing committed code already carries the briefing into handler claims and detects its exact marker in prompt generation, which the overlaid daemon test exercises.

## Limitations and non-claims

- No handler test function ran; all six are compile/symbol evidence only because of the DB-gated `TestMain`.
- No PostgreSQL-backed roster construction, HTTP handler, chat routing, workspace setup, WebSocket, daemon process, UI, provider, or end-to-end smoke ran.
- Full daemon/handler suites were not run; only the named deterministic daemon set, handler compilation/vet, and AST contract were required for this atomic review.
- Tasks 1.2/1.3 and smokes 2.1–2.3 are not included or graded. This review does not change the accepted state of 1.1/1.4 or self-authorize a push.
- Transcript files and verifier were intentionally destroyed with the exact clean-room directory after their hashes/counts were recorded. The durable evidence is this artifact.

Kiro TL adjudicates technical admission; root alone integrates the candidate.
