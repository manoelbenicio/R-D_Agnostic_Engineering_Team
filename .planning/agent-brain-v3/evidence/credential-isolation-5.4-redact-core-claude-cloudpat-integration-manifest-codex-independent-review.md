# Codex independent review — credential-isolation 5.4 redact-core + Claude + Cloud PAT manifest

## CHECK-IN — 2026-07-18T22:33:28Z

- Reviewer: **Codex56#B**, cross-family and independent of manifest author Kiro/Opus-4.8 `w7:p2`, Kiro-family reviewer Kiro/Opus-4.8 `w8:p1`, Cloud-PAT producer Kiro, Cloud-PAT reviewer GLM52#B, and adjudicator Kiro TL.
- Reviewed manifest: `credential-isolation-5.4-redact-core-claude-cloudpat-integration-manifest.md`; requested prefix `4375f03d...`; actual SHA-256 **`4375f03df3b621439c11da15c5103db90926d7314181add079c918648f2f3376` — PASS**.
- Kiro-family review: `credential-isolation-5.4-redact-core-claude-cloudpat-integration-manifest-independent-review.md`, SHA-256 **`551f67c7945e539a4637e8f2f7f6db9d14c8a7854b5937becb5d3eb0367b0187`**.
- Scope is static/read-only. The sole write is this uniquely named artifact. I did not invoke Git, inspect or mutate repository/index/ref state, create a worktree, run tests, read environment values or authentication/token material, or access a network, DB, provider, or service.
- This is neither task acceptance nor an implementation/integration grant. OpenSpec task 5.4 remains `[ ]` at `openspec/changes/agent-credential-isolation/tasks.md:34`; Kiro TL adjudicates.

## Separate verdicts

| Decision plane | Verdict | Reason |
|---|---|---|
| Five-item composition and pinned bytes | **PASS** | Four current whole-file hashes match; the fifth is a deliberately generated Claude target corroborated by durable clean-room reconstruction. |
| Dependency and exclusion model | **PASS, bounded** | The Claude and Cloud-PAT branches share the redact core but not each other; excluded mixed WIP is not required by the prior clean-room builds. |
| Isolated recipe as literally written | **PARTIAL** | It asks for a five-path cached diff without any staging step and abbreviates the named/race invocations. The Kiro review's “safe as written” conclusion is too strong. |
| Bounded technical reproducibility | **PARTIAL / reproducible after recipe correction** | Existing durable transcripts establish the component tests. A fresh integrator can reproduce the atom, but must explicitly materialize and scope the five paths before executing exact gates. |
| Push authorization / governance | **BLOCKED** | No owner waiver, no Kiro TL authorization, no root integration authorization, no whole-task closure, and this review does not certify the distinct Claude-review gate. |

The technical downgrade is about the manifest's executable procedure, not about a discovered source or test failure. No command in this review supersedes the prior durable test transcripts because no test was rerun under this assignment.

## Five logical items

Paths below are relative to `multica-auth-work/server`.

| # | Item | Current/reconstructed SHA-256 | Finding |
|---|---|---|---|
| 1 | `pkg/redact/redact.go` whole-file overlay | `f409ba8a9f3e63618d59c5a8692296f8f7c019c9e558576b8786a058fbf68a5c` | **PASS**, current disk matches manifest. |
| 2 | `pkg/redact/redact_test.go` whole-file overlay | `5a37941a1c7f1bd7263368a6479104c81300f7981e9f47bfb6b0cd17a602fec9` | **PASS**, current disk matches manifest. |
| 3 | `pkg/agent/claude.go`, generated minimal two-hunk target | `c7922b7bf92826f5b7889c91ef4d710700be602ef69ad0156044ffded9d5ede9` | **PASS by durable reconstruction; not a current-file overlay.** The current mixed-WIP file is `3f9dc4fbdb28eecfcc4886f2a518a07943f8a0493cf7aa3525b745992c6d2f54` and is correctly excluded. |
| 4 | `pkg/agent/claude_log_writer_redaction_test.go` new whole-file overlay | `81d3e8659125df2fda25c27da16b24d88e1861282255e41f46c33f7cae2ae40a` | **PASS**, current disk matches manifest. |
| 5 | `internal/auth/cloud_pat_log_redaction_test.go` new whole-file overlay | `1896f90dcd791a9b455f1c940928e7b608ce64f1403b7bcf2986b80a30a55d49` | **PASS**, current disk matches the manifest's claimed slice identity. |

Thus “five overlays” is accurate only as the manifest's clarified **five logical items**: four whole-file overlays plus one generated patch target. The current Git-state labels (`M`/`??`) and base-ref assertions were not independently rechecked because this assignment expressly prohibits Git/ref access; the Kiro-family review records that separate verification at its lines 20–36.

## Correct Cloud-PAT identity and bounded coverage

The correction is genuine. The new slice is `internal/auth/cloud_pat_log_redaction_test.go`, not the older `internal/auth/cloud_pat_test.go`:

- new log test: `1896f90dcd791a9b455f1c940928e7b608ce64f1403b7bcf2986b80a30a55d49`;
- older distinct test: `2626a8863f469bc38ab393261e888a1a2260ee865a81058ef902de8b75f62e72`, correctly excluded;
- unchanged production sink `internal/auth/cloud_pat.go`: `98a4aadf4dd9a236a388bcdfaa9434d83a779916d25d8afdb53c67a09a7e2778`, with the non-200 body log at `cloud_pat.go:349-360`.

The new file contains exactly three named tests at `cloud_pat_log_redaction_test.go:65`, `:118`, and `:150`. They call the real verifier with an in-process custom `RoundTripper` (`:29-46`), install the actual `redact.SanitizeSlogAttr` hook (`:48-62`), use synthetic sentinels, and preserve safe status/diagnostic content. Its imports contain no listener or `httptest.Server`; the supplied `http.Client.Transport` returns synchronously, so the test design cannot fall through to the default network transport. No `t.Parallel()` occurs anywhere in `internal/auth/*_test.go` under the static search used here, and each test defers restoration of the process-global logger (`:85-87`, `:132-134`, `:166-168`).

Durable evidence is correctly linked:

- producer evidence `credential-isolation-5.4-cloud-pat-body-log-test.md`: `99b50ea57e70a4eb872ea52e06fe19b027bb1bd489c49ef1e2fb3033cd414b17`;
- GLM52#B review: `8ecb3a3666ae9582da626bdff7fb17147ce0126511a43643e0ebdc4d200da8c4`.

That review records 3/3 named passes, 60/60 at `-count=20`, focused race success, build/vet success, and a full `internal/auth` pass with 13 pre-existing Redis-gated skips. Those are durable prior results, not executions by Codex56#B in this review.

## Generated minimal Claude patch and dependencies

The generated target is narrowly defined by two hunks against the recorded base blob: add the `pkg/redact` import and change only the stderr emission from `w.prefix + text` to `w.prefix + redact.Text(text)`, while retaining whitespace suppression and `return len(p), nil`. The exact diff is preserved at `credential-isolation-5.4-redact-core-plus-claude-clean-room-atomic-review.md:44-60`.

That atomic review has SHA-256 `2f25c316546570c8deb3f9b544944b19ab15d4acd85a4f726f4eebfab3daac0b`; its independent review is `129025ccba78547223365797e1a61ae95aa2857b4895a8d468432819d99824e4`. It records the generated target hash, clean build/vet/full package runs, 60 redact passes, 120 Claude passes, and focused race success. It also establishes that the minimal Claude slice needs the current redact core but not the argv/environment/model WIP.

Dependency graph:

```text
redact.go ──> generated Claude patch ──> Claude logWriter test
    │
    └───────> unchanged Cloud-PAT sink ──> Cloud-PAT log test
redact_test.go verifies the shared core; neither branch imports the other.
```

The Cloud-PAT test imports `pkg/redact` directly (`cloud_pat_log_redaction_test.go:12`) and exercises the unchanged callsite. The Claude test intentionally constructs a plain handler without `ReplaceAttr` (`claude_log_writer_redaction_test.go:17-19`), proving the explicit pre-log `redact.Text` placement rather than accidentally passing through the central hook.

## Exclusions

The manifest's exclusion boundary is coherent:

- exclude the whole current mixed-WIP `pkg/agent/claude.go`; regenerate only the two-hunk target;
- exclude unchanged `internal/auth/cloud_pat.go` and the distinct committed `cloud_pat_test.go` from the overlay set while retaining them in the pristine base;
- exclude `internal/handler/auth.go` and the listed argv/environment/model/JWT/recent-auth work;
- do not infer that the five-item slice closes other log sinks or all of task 5.4.

No excluded file is required by the prior clean-room Claude build. Static inspection also shows no Claude import in File 5. The manifest therefore does not hide a cross-branch source dependency.

## Isolated commands: exact defect and required correction

Manifest lines 84–101 are a sensible gate inventory but not a complete executable transcript:

1. Step 2 says to overlay/regenerate files, but never stages them. Step 3 then requires `git diff --cached --name-only` to equal five paths. Without an explicit exact-path staging operation in the isolated worktree/index, that command cannot show the two new untracked test files and need not show any of the five paths. The Kiro-family review's “safe as written” statement at its lines 52–54 is therefore **PARTIAL**, not PASS.
2. “Named ... `-count=20`” and “`-race` clean” state expected counts but do not spell out the full regular expressions, package paths, count, and race invocation. The prior durable evidence supplies them, but a self-contained integration recipe should copy the exact names and specify whether Cloud-PAT race is focused `-count=1` (the actual reviewed proof) rather than imply a broader or x20 race run.
3. “Secret gate” is a review criterion, not a deterministic command. It can validate that the intended test literals are labeled synthetic; it cannot prove that arbitrary unknown secret shapes are absent from all output.

An authorized root integrator should, inside the separate worktree only, first materialize the four pinned files and generated target, then verify the **worktree/untracked** path set, explicitly add only those five exact paths to that isolated index, verify the cached path set, and run the manifest's pinned offline build/vet/full/focused/race gates with the exact regexes from the durable transcripts. This describes the correction; it is not permission to stage or integrate anything.

## Central-hook limitation: effect on conclusions

The central hook is real where installed: `internal/logger/logger.go:30-38` and `:44-52` wire `redact.SanitizeSlogAttr`, and the production server installs it at `cmd/server/main.go:122-123`. Therefore the Cloud-PAT server callsite is consistent with the production hook topology, while its test explicitly installs the same function. The Claude test does not depend on that topology because it proves pre-log redaction through a plain handler.

The limitation remains material outside this bounded atom. Current static evidence shows `cmd/multica/cmd_id_resolver.go:42` uses package-level `slog.Warn` while the CLI entrypoint lacks `logger.Init`; and values under benign keys/messages still receive fixed-pattern rather than content-independent protection. The Codex residual review recording these facts is SHA-256 `45a64b19427866cab2cc3b3178aa4ce7f1c5da629bbeccbdbd2c145151581490`.

Consequences:

- **Bounded technical slice:** unchanged. The five-item atom still reproducibly covers its named redact/Claude/Cloud-PAT shapes once the recipe defect is corrected.
- **Absolute or codebase-wide claim:** not established. Central-hook omissions and pattern-dependent sanitization prevent “no secret appears in logs” from following from this atom.
- **Governance/push:** remains blocked, and the central-hook gap reinforces the codebase-wide 5.4 gate. A bounded slice waiver cannot silently become whole-task acceptance. This review does not close the manifest's distinct cross-family Claude-review gate merely by inspecting its patch.

## Commands actually executed by Codex56#B

All were static and read-only from repository root; failures from an initial wrong `server/...` path probe produced no repository mutation and were corrected to `multica-auth-work/server/...`.

```text
sha256sum <manifest> <Kiro review> <current five-item files and cited evidence>
rg --files -g 'AGENTS.md' -g '!node_modules' -g '!vendor'
cat multica-auth-work/AGENTS.md
cat multica-auth-work/CLAUDE.md
nl -ba / sed -n on the manifest, reviews, source, tests, task, and durable evidence
rg -n 't\.Parallel\(' multica-auth-work/server/internal/auth --glob '*_test.go'
  => no matches
rg -n 'logger\.Init|SanitizeSlogAttr|slog\.(Debug|Info|Warn|Error)' <bounded source roots>
rg -n '5\.4|nenhum segredo|sanitizeForLog' <OpenSpec/evidence roots>
date -u +%Y-%m-%dT%H:%M:%SZ
```

No Go test/build/vet/gofmt command was run; their results above are explicitly attributed to hashed durable evidence.

## CHECK-OUT — 2026-07-18T22:35:52Z

Review complete. Corrected Cloud-PAT identity/hash **PASS**; five-item bytes/composition **PASS**; minimal Claude generation and dependency/exclusion model **PASS by current hashes plus durable clean-room evidence**; isolated recipe **PARTIAL** because its cached-diff and exact-command steps are incomplete; bounded technical reproducibility **PARTIAL until that procedure is corrected and rerun by an authorized integrator**; push authorization **BLOCKED**. Task 5.4 remains OPEN. No acceptance, EV award, waiver, checkbox, or integration authorization is issued.
