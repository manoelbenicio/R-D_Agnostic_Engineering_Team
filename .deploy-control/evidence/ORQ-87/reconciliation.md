# ORQ-87 Candidate-Bound Native Runtime Reconciliation

- Card: `ORQ-87 / REC-NATIVE-CLAIMS-01`
- Captured: 2026-08-02 UTC
- Branch: `recovery/orq87-native-claims`
- Baseline commit: `c7f14ee13fa6357b4c67a8b1230fecc9bcd7c81e`
- Baseline tree: `c73cd8dbc56ec3a31c4a300d166a62816b7fac34`
- Boundary: offline source, native/backend tests, compile/vet, and web build validation only
- Excluded: live smoke, UAT, production/runtime mutation, credentials, network, OmniRouter, and architecture changes

## Reconciliation result

The recovered `19/19` claim is not evidence-supported. The baseline resolves to:

- 5 bounded-direct: `0.1`, `1.1`, `1.2`, `1.3`, `2.2`.
- 5 reopen: `1.4`, `1.5`, `1.7`, `2.1`, `2.3`.
- 9 contradicted completion claims, now open: `1.6`, `2.4a`, `2.4b`, `2.5a`, `2.5b`, `3.1`, `3.2`, `3.3`, `3.4`.

`openspec/changes/native-runtimes-onboarding/reconciliation/SOURCE_PARITY_EVIDENCE.md`
and `openspec/changes/native-runtimes-onboarding/reconciliation/SECOND_CHANGE_REVIEW.md`
were absent from the baseline and were not fabricated by ORQ-87.

## Acceptance matrix

| Criterion | Candidate-bound result | Evidence |
| --- | --- | --- |
| 0.1 | BOUNDED-DIRECT: owner choice is recorded; no runtime claim | proposal/tasks blobs in `candidate-source-manifest.tsv` |
| 1.1 | BOUNDED-DIRECT: current NIM implementation/tests executed | `native-agent.log`, exit 0 |
| 1.2 | BOUNDED-DIRECT from exact candidate bytes plus prior bounded evidence; current daemon-side rerun had setup failure | manifest; `native-daemon.log` |
| 1.3 | BOUNDED-DIRECT: current Cline ACP implementation/tests executed | `native-agent.log`, exit 0 |
| 1.4 | REOPEN: current agent discovery tests passed; daemon surface did not compile offline | `native-agent.log`; `native-daemon.log` |
| 1.5 | REOPEN: current web assertions/typecheck did not execute | `web-tests.log`; `web-typecheck.log` |
| 1.6 | CONTRADICTED/OPEN: historical evidence records production build as blocked; current build did not execute | `web-build.log`; historical 1.6 evidence |
| 1.7 | REOPEN: current auth suite stopped during missing-module setup | `auth-offline.log` |
| 2.1 | REOPEN: current config/daemon tests stopped during missing-module setup | `native-daemon.log` |
| 2.2 | BOUNDED-DIRECT: current factory/SupportedTypes tests executed | `native-agent.log`, exit 0 |
| 2.3 | REOPEN: current daemon wiring tests stopped during missing-module setup | `native-daemon.log` |
| 2.4a | CONTRADICTED/OPEN: current build/vet stopped during missing-module setup | `backend-build.log`; `backend-vet.log` |
| 2.4b | CONTRADICTED/OPEN: restart/rollout explicitly not performed | `tasks.md` |
| 2.5a | CONTRADICTED/OPEN: production web build did not execute | `web-build.log` |
| 2.5b | CONTRADICTED/OPEN: local web startup explicitly not performed | `tasks.md` |
| 3.1 | CONTRADICTED/OPEN: only the focused agent package passed; remaining slices are setup failures | all command logs |
| 3.2 | CONTRADICTED/OPEN: deployed smoke explicitly not performed | `tasks.md` |
| 3.3 | CONTRADICTED/OPEN: deployed UAT explicitly not performed | `tasks.md` |
| 3.4 | CONTRADICTED/OPEN: both cited reconciliation artifacts are absent; no independent second review occurred | this record; `tasks.md` |

## Commands and real outcomes

Every log records the exact shell argument vector, working directory, UTC start/end, output,
and exit code.

| Evidence | Exit | Interpretation |
| --- | ---: | --- |
| `native-agent.log` | 0 | Current NIM, Cline, model-discovery, factory, and supported-type assertions passed in `pkg/agent`. |
| `native-daemon.log` | 1 | Setup failure: required Go modules absent while `GOPROXY=off`; no assertions executed. |
| `auth-offline.log` | 1 | Setup failure: required Go modules absent while `GOPROXY=off`; no auth assertions executed. |
| `backend-vet.log` | 1 | Setup failure for missing offline Go modules; no clean vet claim. |
| `backend-build.log` | 1 | Setup failure for missing offline Go modules; no binary build claim. |
| `web-toolchain.log` | 1 | Pinned pnpm 10.28.2 absent; Corepack network disabled. |
| `web-tests.log` | 1 | Same toolchain setup failure; no test assertions executed. |
| `web-typecheck.log` | 1 | Same toolchain setup failure; no typecheck executed. |
| `web-build.log` | 1 | Same toolchain setup failure; no Next.js build executed. |
| `openspec-validate.log` | 0 | Strict validation of the reconciled OpenSpec change passed. |

The `go: downloading` lines in failed Go logs describe attempted module resolution; with
`GOPROXY=off`, no network retrieval occurred. Web invocations used
`COREPACK_ENABLE_NETWORK=0`; no pnpm download occurred. Setup failures are preserved and are
not represented as product assertion failures or passes.

## Hash manifests

- `candidate-identity.tsv`: exact baseline commit/tree/branch and generation time.
- `candidate-source-manifest.tsv`: 64 baseline source, test, and OpenSpec paths; each row records the exact Git blob OID and SHA-256 without exposing file contents.
- `changed-file-map.tsv`: every ORQ-87 repository change mapped to its acceptance/evidence purpose.
- `captured-artifacts.sha256`: SHA-256 values for the command logs, reconciliation record, and candidate manifests; the checksum file excludes itself.

## Residual reopen boundary

1. Supply the pinned Go modules in an approved offline cache, then rerun daemon, auth, vet, and server/CLI build commands unchanged.
2. Supply pnpm 10.28.2 and the lockfile-resolved dependency tree without mutating credentials, then run the scoped web tests, typechecks, and production build.
3. Independently review the resulting candidate-bound evidence before closing 3.4; ORQ-87 cannot self-create a `SECOND_CHANGE_REVIEW`.
4. Live smoke, UAT, restart, rollout, and production validation remain expressly outside this card.

This artifact reconciles evidence only. It does not mark the Kanban card complete or authorize deployment, runtime mutation, credentials, smoke, or UAT.
