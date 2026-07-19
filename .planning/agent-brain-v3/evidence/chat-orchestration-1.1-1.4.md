# Chat orchestration tasks 1.1 and 1.4 — independent acceptance evidence

- Review date: 2026-07-18
- Reviewer: Codex#56#A, independent read-only implementation review
- Verdict: **ACCEPT** for OpenSpec tasks **1.1** and **1.4** only
- Toolchain: Go 1.26.4, `GOTOOLCHAIN=local`, `GOPROXY=off`
- Scope boundary: no product code, OpenSpec, planning register, checkbox, credential, database, Docker, network, provider, or live-service mutation

## Accepted requirements

The accepted scope is limited to the leader identity/instruction protocol and its delegation-only rule:

- `openspec/changes/chat-orchestration-standard/tasks.md:11` defines task 1.1: ordered clarify → OpenSpec → plan → delegate → synthesize instructions with the exact `## Squad Operating Protocol` marker.
- `openspec/changes/chat-orchestration-standard/tasks.md:14` defines task 1.4: the leader does not produce; it delegates and synthesizes.
- `openspec/changes/chat-orchestration-standard/specs/chat-orchestration/spec.md:13-20` requires clarification, documentation, planning, delegation, synthesis, and prohibits a delegation-only leader from producing the work itself.
- `openspec/changes/chat-orchestration-standard/design.md:22-27` places this behavior in leader identity/config instructions, not a new native runtime.

## Implementation anchors

| Invariant | Source anchor | Reviewed result |
|---|---|---|
| Exact task-1.1 marker | `multica-auth-work/server/internal/handler/squad_briefing.go:26` | The production constant begins with the standalone `## Squad Operating Protocol` heading. |
| Ordered task-1.1 protocol | `multica-auth-work/server/internal/handler/squad_briefing.go:28-73` | The summary and numbered responsibilities preserve clarify → OpenSpec gate → plan → delegate → synthesize. |
| Mandatory OpenSpec gate | `multica-auth-work/server/internal/handler/squad_briefing.go:44-50` | From-scratch work without an OpenSpec change cannot proceed. |
| Delegation-only / no production | `multica-auth-work/server/internal/handler/squad_briefing.go:105-112` | The leader cannot implement, edit, run state-changing production commands, fix bugs, or use the former no-suitable-member escape hatch. It escalates instead. |
| No false production claims | `multica-auth-work/server/internal/handler/squad_briefing.go:107-110` | The leader is explicitly forbidden from claiming to the reporter that it performed production work. |
| Synthesis remains allowed and mandatory | `multica-auth-work/server/internal/handler/squad_briefing.go:68-73,105-112` | Synthesis is distinguished from production and remains the leader's responsibility. |
| Briefing reaches issue-bound leader claims | `multica-auth-work/server/internal/handler/daemon.go:1309-1332` | The handler appends the briefing only when the claiming agent is the assigned squad's current leader. |
| Briefing reaches squad quick-create leader claims | `multica-auth-work/server/internal/handler/daemon.go:1684-1707` | The same briefing is appended for the resolved squad leader and the squad identity is retained. |
| Marker reaches daemon identity | `multica-auth-work/server/internal/daemon/daemon.go:3376` | `IsSquadLeader` uses the exact marker. |
| Marker controls leader prompt behavior | `multica-auth-work/server/internal/daemon/prompt.go:157-159` | The same marker selects the squad-leader no-action rule. |
| Runtime workflow respects identity boundaries | `multica-auth-work/server/internal/daemon/execenv/runtime_config.go:684-696` | The workflow requires all work to remain within Agent Identity and explicitly handles delegation-only roles. |

The focused source tests encode the same invariants at `multica-auth-work/server/internal/handler/squad_briefing_test.go:34-182`: exact marker, OpenSpec gate, ordered sequence, regression preservation, delegation-only/no-claim clauses, escape-hatch removal, and allowed synthesis.

## Executed pure protocol verification

The handler package has a database-gated `TestMain`, and `squad_briefing_test.go` mixes pure string tests with database fixture tests. Supplying the whole test file to a `command-line-arguments` test package without `handler_test.go` therefore does not compile because later database tests reference fixture globals. Including `handler_test.go` would restore the early-exit problem. The independent review consequently used the allowed equivalent: a deterministic Go AST/source verifier outside the package harness.

The verifier:

1. parsed the actual production `internal/handler/squad_briefing.go` with `go/parser`;
2. located the `squadOperatingProtocol` constant by AST name;
3. evaluated only its compile-time string-literal `+` expression;
4. normalized whitespace exactly as the focused tests do;
5. executed 24 assertions against the production value:
   - 1 exact marker assertion;
   - 11 required-clause assertions covering coordinator identity, delegation-only, no production, no false production claim, escalation, synthesis, and OpenSpec gating;
   - 2 assertions excluding the old no-suitable-member production escape hatch;
   - 10 presence/order assertions for the five task-1.1 steps.

Command, from `multica-auth-work/server`:

```text
env GOTOOLCHAIN=local GOPROXY=off /home/dataops-lab/go-sdk/bin/go run /tmp/chat14verify/main.go -- internal/handler/squad_briefing.go
```

Executed result:

```text
PASS: 24 deterministic protocol assertions executed against internal/handler/squad_briefing.go
```

This is executed assertion evidence against the production constant. It is not a claim that the handler package's own test functions ran.

## Executed daemon verification

The non-database daemon packages genuinely executed the marker propagation, leader/no-action, direct non-leader, and identity-boundary assertions 20 times:

```text
env GOTOOLCHAIN=local GOPROXY=off /home/dataops-lab/go-sdk/bin/go test ./internal/daemon -run '^(TestSquadLeaderMarkerDetectionExact|TestBuildPromptSquadLeaderNoActionForMemberTrigger|TestBuildPromptSquadLeaderNoActionForAgentTrigger|TestBuildPromptNonSquadLeaderNoRule|TestBuildPromptSquadLeaderNoActionProhibition)$' -count=20
ok github.com/multica-ai/multica/server/internal/daemon 0.046s

env GOTOOLCHAIN=local GOPROXY=off /home/dataops-lab/go-sdk/bin/go test ./internal/daemon/execenv -run '^(TestInjectRuntimeConfigSquadLeaderCommentTriggeredNoAction|TestAssignmentTriggeredProtocolHonorsAgentIdentity)$' -count=20
ok github.com/multica-ai/multica/server/internal/daemon/execenv 0.060s
```

The same focused sets executed under the race detector:

```text
env GOTOOLCHAIN=local GOPROXY=off /home/dataops-lab/go-sdk/bin/go test -race ./internal/daemon -run '^(TestSquadLeaderMarkerDetectionExact|TestBuildPromptSquadLeaderNoActionForMemberTrigger|TestBuildPromptSquadLeaderNoActionForAgentTrigger|TestBuildPromptNonSquadLeaderNoRule|TestBuildPromptSquadLeaderNoActionProhibition)$' -count=1
ok github.com/multica-ai/multica/server/internal/daemon 1.109s

env GOTOOLCHAIN=local GOPROXY=off /home/dataops-lab/go-sdk/bin/go test -race ./internal/daemon/execenv -run '^(TestInjectRuntimeConfigSquadLeaderCommentTriggeredNoAction|TestAssignmentTriggeredProtocolHonorsAgentIdentity)$' -count=1
ok github.com/multica-ai/multica/server/internal/daemon/execenv 1.038s
```

## Handler compile-only limitation

The normal focused handler command was also run:

```text
env GOTOOLCHAIN=local GOPROXY=off /home/dataops-lab/go-sdk/bin/go test ./internal/handler -run '^(TestSquadOperatingProtocolMarkerExact|TestSquadOperatingProtocolMandatoryOpenSpecGate|TestSquadOperatingProtocolDelegationSequence|TestSquadOperatingProtocolNoRegression|TestSquadOperatingProtocolDelegationOnlyInvariant|TestSquadOperatingProtocolSynthesisAllowed)$' -count=1 -v
```

Result:

```text
Skipping tests: database not reachable: failed to connect to `user=multica database=multica`: 127.0.0.1:5432 (localhost): failed SASL auth: FATAL: password authentication failed for user "multica" (SQLSTATE 28P01)
ok github.com/multica-ai/multica/server/internal/handler 0.195s
```

This result proves that the handler package and selected test binary compiled. It is **compile-only evidence**: `internal/handler/handler_test.go:38-54` exits from `TestMain` before `m.Run()` when PostgreSQL is unavailable, so none of the selected handler test functions executed. No database credential was sought or used, and no database or service state was changed.

The database-backed briefing-injection tests are likewise not claimed as executed. Their production call sites were source-reviewed, and the handler package compiled, but end-to-end database claim execution remains outside this evidence.

## Static checks

Executed offline:

```text
env GOTOOLCHAIN=local GOPROXY=off /home/dataops-lab/go-sdk/bin/go vet ./internal/handler ./internal/daemon ./internal/daemon/execenv
```

Result: PASS.

`gofmt -l` over the reviewed handler, daemon, and execenv production/test files returned no paths. `git diff --check` passed.

## Source hashes

SHA-256 hashes were computed only for the reviewed OpenSpec, product source, and test files. No credential, auth, home, provider, or live-service path was read.

| SHA-256 | Source |
|---|---|
| `bebb75317d7ef66c07904576f3a6b6e6d349a8581bc78371d9c8457290ac892b` | `openspec/changes/chat-orchestration-standard/proposal.md` |
| `88f6cfc1a03d3df86a75367f9158f1300288eb9d582f34645433c38ec34e4a40` | `openspec/changes/chat-orchestration-standard/design.md` |
| `4bef5a7f3bd9ef64ecb54ca5bd6b82b88e007011f59d61ee9825bd7fbff3aadd` | `openspec/changes/chat-orchestration-standard/specs/chat-orchestration/spec.md` |
| `a7d19efa305fdfd8a9e4b1c8ca0a306f7fb4339b60ceed3d72987ec2841a00dc` | `openspec/changes/chat-orchestration-standard/tasks.md` |
| `a2998f923852a455782f37d4416bbfb5a74750ea19b2f6d87dc5f56cc262e80a` | `multica-auth-work/server/internal/handler/squad_briefing.go` |
| `3b12615543440f52773d0d1d7bed4277dd6c1b0fc835f7bfd2ee3f12cd823d9c` | `multica-auth-work/server/internal/handler/squad_briefing_test.go` |
| `0e1cbb54c1d733afba473bab1fe42c62bda58aa9af439871267c1c062b3abec6` | `multica-auth-work/server/internal/handler/daemon.go` |
| `096db587b84d6facf60faa5be4514ed651520b93c153dcee4a3a3c5140442104` | `multica-auth-work/server/internal/handler/handler_test.go` |
| `da828dde8edf8cf387fa674f2f84ed1c7dc62f09a473d4f7448b91318577c426` | `multica-auth-work/server/internal/daemon/prompt.go` |
| `50406c891be39a9f645a2e1b957919c43ed879756a77a65c71a6afa11a3029fd` | `multica-auth-work/server/internal/daemon/prompt_test.go` |
| `a1d96a3c8a4edfc9f9c583326c6f74779643473904f13f0a8aab63ea1cd6fe07` | `multica-auth-work/server/internal/daemon/daemon.go` |
| `d60d42c308d9b5839b5c55c2a2c2aa7c83158c31478b09f1d07ecadc3d1a1d04` | `multica-auth-work/server/internal/daemon/daemon_test.go` |
| `b7611bce4e821a0f560ca8f76bf07da095951054f02d2a0fbac10947af642c0d` | `multica-auth-work/server/internal/daemon/execenv/runtime_config.go` |
| `128bb4f889493ac802fd16d03c252d3156f60f4a591be7f661672f51328258c1` | `multica-auth-work/server/internal/daemon/execenv/runtime_config_test.go` |
| `8099867594f18f6d5e5c47fa88581f06ac771e456f203fc0161d09c6584fc0e6` | `multica-auth-work/server/internal/daemon/execenv/execenv.go` |
| `05e37b832ab06cced5758f833579f688fcf249379d9bebf76eb20d0a35197d1b` | `multica-auth-work/server/internal/daemon/execenv/execenv_test.go` |

## Explicit non-claims and preserved gates

- **Task 1.2 is not accepted:** this artifact does not prove or close default workspace setup of a TL/Manager squad with leader and members.
- **Task 1.3 is not accepted:** this artifact does not prove or close default untargeted-chat routing or the direct-agent escape hatch end to end.
- **Verification tasks 2.1, 2.2, and 2.3 are not accepted:** no chat-routing smoke, delegation/synthesis smoke, direct-mention smoke, or deploy-control check-in was executed.
- No task checkbox was edited. Existing checkbox counts, phase counts, PD-01, PD-08, credential/network/live-service STOPs, and all other gates remain unchanged.
- Acceptance is for instruction/identity behavior in tasks 1.1 and 1.4, not a claim of runtime-enforced sandboxing against a malicious model, production readiness, live routing, live delegation, or live synthesis.
- No credentials, auth files, provider data, user-home credential paths, databases, Docker daemons, network endpoints, or live services were accessed or mutated.
