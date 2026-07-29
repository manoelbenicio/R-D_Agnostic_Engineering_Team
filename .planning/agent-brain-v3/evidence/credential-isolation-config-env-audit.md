# Credential-isolation config/env audit — tasks 1.1, 1.2, 1.3

## Audit identity and constraints

- Verdict: **BLOCK**.
- Independent grades: **1.1 PARTIAL**, **1.2 PARTIAL**, **1.3 REJECT**.
- Runner: Codex-root, independent source-read-only audit.
- Host: `manoelneto-laptop`, Linux/amd64 (WSL).
- Repository commit: `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f`.
- Toolchain: `/home/dataops-lab/go-sdk/bin/go` → `go version go1.26.4 linux/amd64`.
- Evidence window: `2026-07-18T20:15:52Z` through `2026-07-18T20:22:19Z` (UTC).
- Every Go invocation used `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off DATABASE_URL=`. Tests used `t.TempDir()` and synthetic values. The full `execenv` lane additionally overrode all named provider/config homes to a fresh synthetic directory.
- No credential/auth home, real environment value, token, provider, database, network service, or live process was inspected. No product source, OpenSpec checkbox/file, STATE, ledger, or index was edited.
- Required context read: repository `AGENTS.md` and authoritative `CLAUDE.md`; `.planning/EVIDENCE_CONTRACT.md`; relevant ledger history/current claims; all five apply files under `openspec/changes/agent-credential-isolation/`; existing `credential-isolation-source-contract-map.md`, `credential-isolation-two-account-coexistence.md`, `g2c-runtimeenv-package.md`, and `g4-exactenv-containment.md`.

## Independent grades

| Task | Grade | Reason |
| --- | --- | --- |
| 1.1 — config-dir layout by account/provider | **PARTIAL** | Synthetic copies are isolated for Codex, Kiro, Antigravity, and OpenCode/GLM, but the production assignment path returns only `Account.HomeDir`, ignores `Account.ConfigDir`, omits the tenant/workspace check, and Codex still sources config/session/plugin state from the shared global Codex home. This is not a complete per-account config-dir contract. |
| 1.2 — native provider env injection | **PARTIAL** | Production injects `CODEX_HOME`, Kiro `XDG_DATA_HOME`, Antigravity `HOME`, and OpenCode/GLM XDG roots. That matches the change design's observed binary levers, but not the literal task/proposal matrix (`CLAUDE_CONFIG_DIR`, `GEMINI_CONFIG_DIR`/`CLOUDSDK_CONFIG`, `KIRO_HOME`); Claude is absent and the local custom-env denylist permits those alternate discovery roots after trusted injection. |
| 1.3 — no-assignment global fallback | **REJECT** | Production deliberately fails closed on nil store, no assignment, and empty account home. The only Codex global fallback is an uncalled explicit compatibility helper; non-Codex helper-level “fallback” tests merely prove no isolated output is created. They do not prove a production child retains global behavior. This is the opposite of the task/scenario. Security note: resolving this conflict must not weaken the fail-closed production path; it needs an explicit, narrowly scoped legacy/development policy if the OpenSpec requirement remains desired. |

## Exact production contract matrix

| Provider/task name | Account source/layout | Child env actually emitted | Assignment/fallback | Evidence |
| --- | --- | --- | --- | --- |
| Codex | Task root is `{workspacesRoot}/{workspace}/{shortTask}/codex-home`; `auth.json` is copied from assigned `HomeDir`. However `sessions` is symlinked and `config.json`, `config.toml`, `instructions.md`, and plugin cache are sourced from the shared Codex home. | `CODEX_HOME=<task codex-home>` | Empty assignment fails before preparation. An explicit `prepareLegacySharedCodexHome` exists but has no production caller. | `execenv/execenv.go:235,283-299,637-641`; `execenv/codex_home.go:55-63,104-151,163-165`; `daemon.go:4031-4056,4114-4119`. |
| Claude | No per-account execenv layout and not in `requiresCredentialIsolation`. | No `CLAUDE_CONFIG_DIR` injection in this path. Credentialless gateway uses a controlled `HOME` plus gateway variables, a different contract. | Global/native behavior remains outside tasks 1.1/1.2 implementation. | `daemon.go:4059-4065,4114-4130`; absence confirmed by repository `rg`. |
| Gemini/Antigravity | Antigravity copies `HomeDir/.gemini/antigravity-cli` to `<task>/antigravity-home/.gemini/antigravity-cli`. | `HOME=<task antigravity-home>` | Empty assignment fails in daemon, despite helper returning without creating a home. | `execenv/antigravity_home.go:10-45`; `execenv/execenv.go:312-321,646-649`; `daemon.go:4040-4056,4120-4122`. |
| Kiro | Copies `HomeDir/kiro-cli/data.sqlite3` to `<task>/kiro-data-home/kiro-cli/data.sqlite3`. | `XDG_DATA_HOME=<task kiro-data-home>`; no `KIRO_HOME`. | Empty assignment fails in daemon, despite helper returning without creating a home. | `execenv/kiro_home.go:11-54`; `execenv/execenv.go:301-310,642-645`; `daemon.go:4040-4056,4118-4120`. |
| GLM | Uses OpenCode-compatible copies into `<task>/glm-data-home/opencode` and `<task>/glm-config-home/opencode`. | `XDG_DATA_HOME` plus `XDG_CONFIG_HOME`. | Empty assignment fails in daemon. | `execenv/execenv.go:341-352,661-670`; `execenv/opencode_home.go:10-85`; `daemon.go:4124-4126`. |

The account model and store carry both `HomeDir` and `ConfigDir` (`internal/rotation/contract.go:57-65`, `internal/rotation/store_pg.go:28-49`), but the runtime resolver returns only `account.HomeDir` (`internal/daemon/daemon.go:4045-4056`). It checks vendor equality at `daemon.go:4050-4052`, but never compares `account.TenantID` with `task.WorkspaceID`. Consequently, a corrupted/cross-tenant assignment row can resolve another tenant's home while still passing the provider check.

## Fallback semantics

- The OpenSpec scenario says no assignment preserves the global credential home.
- Production says the opposite: nil store errors at `daemon.go:4031-4033`; `rotation.ErrNoAssignment` errors at `daemon.go:4037-4043`; an empty assigned home errors at `daemon.go:4053-4055`; final injection rejects an empty account home at `daemon.go:4077-4079`.
- `TestVendorCredentialFallbackDoesNotSetIsolatedHomes` (`execenv/vendor_credential_fallback_test.go:5-80`) tests only the lower-level `Prepare`/`Reuse` result when `CredentialAccountHome` is empty. Production never reaches that behavior for a provider in `requiresCredentialIsolation`.
- Codex's `prepareLegacySharedCodexHome` (`execenv/codex_home.go:55-63`) is explicit and tested indirectly through the option at `codex_home_account_test.go:97-119`, but `rg` found no production caller.

## Cross-provider and contamination negatives

What is genuinely proven:

- `TestPrepareCodexHomePerAccountIsolatesAuth`, `TestPrepareKiroHomePerAccountIsolatesDataStore`, and `TestPrepareAntigravityHomePerAccountIsolatesTokenDir` prove same-provider A/B file-copy isolation with synthetic sentinels.
- `TestPrepareGLMUsesOpenCodeCompatibleIsolation` proves GLM's intended OpenCode/XDG mapping.
- `TestCredentialIsolationVendorMatrixCoversExactlySixP0Vendors` proves the current P0 matrix is exactly Codex, Kiro, Antigravity, GLM, Cline, OpenCode and that empty/incomplete task envs fail.
- `TestL2EnabledLocalLaunchFailsClosedOnMissingOrPartialIsolation` proves inherited required keys are deleted before a failed launch; `TestL2EnabledLocalLaunchInjectsCompleteSyntheticIsolation` proves a complete Codex root replaces the inherited one.
- `TestBuildMinimalInheritedRemovesCredentialAndRoutingSurface` and `TestBuildGatewayEnvironmentCodexUsesDedicatedKeyName` prove selected gateway inherited keys are removed and controlled Codex values win.

What is **not** proven and currently has bypasses:

1. No pure named test supplies one provider's prepared environment to another provider and asserts every alternate discovery key is absent. The DB-backed `TestCredentialIsolationPerVendor` is same-provider and was intentionally not run.
2. Local custom env is applied after isolated injection (`daemon.go:3653-3667`). `isBlockedEnvKey` blocks `HOME`, `CODEX_HOME`, and XDG roots, but not `CODEX_CONFIG_DIR`, `CLAUDE_CONFIG_DIR`, `GEMINI_CONFIG_DIR`, `CLOUDSDK_CONFIG`, `KIRO_HOME`, or `KIRO_CONFIG_DIR` (`daemon.go:4860-4866`). Those values can coexist with or redirect a provider away from the intended root depending on CLI/version.
3. Gateway `runtimeenv` denies `CLAUDE_*`, `GEMINI_*`, `KIRO_*`, and `CODEX_*` through provider prefixes (`runtimeenv/policy.go:70-74,92-114`), but `CLOUDSDK_CONFIG` matches neither an exact denial nor a provider prefix and can survive classification. The focused inherited-env test does not include any of the task's config-dir variable names (`runtimeenv/env_test.go:15-39`).
4. Codex account auth is copied, but shared config/session/plugin surfaces remain (`execenv/codex_home.go:114,123-151,163-165`), so “per-account config dir” and cross-account config isolation are not established.
5. The assignment resolver enforces provider equality but not tenant equality (`daemon.go:4050-4056`), and no pure fake-store test covers wrong-tenant or wrong-provider resolution. The only resolver coverage is inside the PostgreSQL-gated aggregate.

## Test/build topology

- `rg -n '^//go:build|^// +build'` over `internal/daemon/execenv/**/*_test.go`, `internal/daemon/runtimeenv/**/*_test.go`, `runtime_isolation_test.go`, and `daemon_test.go` returned no matches. The focused tests are included in normal builds; no broad/offline tag hides them.
- `rg -n 'func TestMain'` over the same packages returned no matches. The named pure tests therefore execute directly and cannot be false-green through a package `TestMain` exit.
- `internal/daemon/runtime_isolation_test.go:497-515` explicitly reads `DATABASE_URL`, attempts PostgreSQL, and skips when unavailable. Its aggregate `TestCredentialIsolationPerVendor` (`:746-771`) was excluded; no DB assertion is claimed.
- Full `runtimeenv` was not run because `gateway_acceptance_test.go:10-49` starts `httptest.NewServer` (loopback network). Full daemon was not run because it contains the PostgreSQL gate and additional network-oriented tests. This is a constraint-driven non-claim, not a pass.
- Full `execenv` was safe after all relevant homes were overridden to synthetic temporary roots. It passed normal and race runs.

## Commands and genuine outputs

Working directory for every Go command: `multica-auth-work/server`.

### Verbose non-zero proof

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off DATABASE_URL= /home/dataops-lab/go-sdk/bin/go test -v -count=1 ./internal/daemon/execenv -run '^(TestPrepareCodexHomePerAccountIsolatesAuth|TestPrepareCodexHomeFailsClosedWhenNoAccount|TestPrepareKiroHomePerAccountIsolatesDataStore|TestPrepareKiroHomeFallbackWhenNoAccount|TestPrepareAntigravityHomePerAccountIsolatesTokenDir|TestPrepareAntigravityHomeFallbackWhenNoAccount|TestPrepareGLMUsesOpenCodeCompatibleIsolation|TestVendorCredentialFallbackDoesNotSetIsolatedHomes)$'
```

Output contained all eight parent `=== RUN` markers, all six fallback subtests, `PASS`, and:
`ok github.com/multica-ai/multica/server/internal/daemon/execenv 0.046s` (exit 0).

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off DATABASE_URL= /home/dataops-lab/go-sdk/bin/go test -v -count=1 ./internal/daemon -run '^(TestCredentialIsolationVendorMatrixCoversExactlySixP0Vendors|TestL2EnabledLocalLaunchFailsClosedOnMissingOrPartialIsolation|TestL2EnabledLocalLaunchInjectsCompleteSyntheticIsolation|TestIsBlockedEnvKey)$'
```

Output contained all four parent `=== RUN` markers, both missing/partial-isolation subtests, all 18 `TestIsBlockedEnvKey` subtests, `PASS`, and:
`ok github.com/multica-ai/multica/server/internal/daemon 0.021s` (exit 0).

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off DATABASE_URL= /home/dataops-lab/go-sdk/bin/go test -v -count=1 ./internal/daemon/runtimeenv -run '^(TestBuildMinimalInheritedRemovesCredentialAndRoutingSurface|TestBuildGatewayEnvironmentCodexUsesDedicatedKeyName)$'
```

Exact terminal result:

```text
=== RUN   TestBuildMinimalInheritedRemovesCredentialAndRoutingSurface
--- PASS: TestBuildMinimalInheritedRemovesCredentialAndRoutingSurface (0.00s)
=== RUN   TestBuildGatewayEnvironmentCodexUsesDedicatedKeyName
--- PASS: TestBuildGatewayEnvironmentCodexUsesDedicatedKeyName (0.00s)
PASS
ok github.com/multica-ai/multica/server/internal/daemon/runtimeenv 0.014s
```

### Repetition and race

The same three regexes were run with `-count=20`; exact results were:

```text
ok github.com/multica-ai/multica/server/internal/daemon/execenv 1.122s
ok github.com/multica-ai/multica/server/internal/daemon 0.121s
ok github.com/multica-ai/multica/server/internal/daemon/runtimeenv 0.089s
```

They were then run with `-race -count=1`; exact results were:

```text
ok github.com/multica-ai/multica/server/internal/daemon/execenv 1.117s
ok github.com/multica-ai/multica/server/internal/daemon 1.084s
ok github.com/multica-ai/multica/server/internal/daemon/runtimeenv 1.048s
```

### Full safe package lane

The first synthetic-HOME full attempt did not execute tests because changing `HOME` moved the Go module cache and offline resolution failed with `module lookup disabled by GOPROXY=off`. It is not counted as evidence. The retry explicitly used the pinned local caches, kept every provider/config home synthetic, and ran:

```sh
tmp=$(mktemp -d); mkdir -p "$tmp/home" "$tmp/codex" "$tmp/claude" "$tmp/gemini" "$tmp/cloudsdk" "$tmp/kiro" "$tmp/xdg-data" "$tmp/xdg-config"; HOME="$tmp/home" CODEX_HOME="$tmp/codex" CODEX_CONFIG_DIR="$tmp/codex" CLAUDE_CONFIG_DIR="$tmp/claude" GEMINI_CONFIG_DIR="$tmp/gemini" CLOUDSDK_CONFIG="$tmp/cloudsdk" KIRO_HOME="$tmp/kiro" KIRO_CONFIG_DIR="$tmp/kiro" XDG_DATA_HOME="$tmp/xdg-data" XDG_CONFIG_HOME="$tmp/xdg-config" GOMODCACHE=/home/dataops-lab/go/pkg/mod GOCACHE=/home/dataops-lab/.cache/go-build GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off DATABASE_URL= /home/dataops-lab/go-sdk/bin/go test -count=1 ./internal/daemon/execenv
```

Result: `ok github.com/multica-ai/multica/server/internal/daemon/execenv 0.614s` (exit 0).

The identical command with `-race -count=1` returned:
`ok github.com/multica-ai/multica/server/internal/daemon/execenv 1.767s` (exit 0).

### Vet, format, and diff

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off DATABASE_URL= /home/dataops-lab/go-sdk/bin/go vet ./internal/daemon/execenv ./internal/daemon/runtimeenv ./internal/daemon
```

Result: exit 0, no output.

`git diff --check` over the audited source set returned exit 0 with no diagnostics. A read-only `gofmt -d` found one pre-existing formatting delta in `internal/rotation/contract.go:42-43` (double spacing before comments); all other audited manifest files produced no formatting output. No source was edited.

## Current source SHA-256 manifest

Hashes are of the shared working tree audited above, not merely `HEAD`.

```text
a1d96a3c8a4edfc9f9c583326c6f74779643473904f13f0a8aab63ea1cd6fe07  internal/daemon/daemon.go
d60d42c308d9b5839b5c55c2a2c2aa7c83158c31478b09f1d07ecadc3d1a1d04  internal/daemon/daemon_test.go
168dc34f17650e3d4f07d324a5272a9dc5839f7b2ed28b5b1e643ef823fa7308  internal/daemon/runtime_isolation_test.go
8099867594f18f6d5e5c47fa88581f06ac771e456f203fc0161d09c6584fc0e6  internal/daemon/execenv/execenv.go
aa3f8bbd82ffb3fe70ed72da037eaae3028af4b558a7b1d0c207ccaeb4cb3fe0  internal/daemon/execenv/codex_home.go
31d759a40ab35780ef875779574fc7d82ace4c5801ae565cc96845a0fe9b6f4a  internal/daemon/execenv/codex_home_account_test.go
f8c616596d8a0f4deafb4eb88a98d45f5be888d2c79f0ae1b1db03b7bb0e083c  internal/daemon/execenv/kiro_home.go
c4a99c9ea3b4fd1d41981c2c06d190ab82ed4e56943e66fad5dbffd2b45c4ce5  internal/daemon/execenv/kiro_home_test.go
9f116ab69d399e275bae47a0f8daf788a579bb7c4dd2c194748dc977de31cb04  internal/daemon/execenv/antigravity_home.go
e7527f8bf70a634c8b26a2afd5c150a1e4ba82ca9737e46ba8266b8a4e51521a  internal/daemon/execenv/antigravity_home_test.go
427bbf7e03f59a7ca7cbd2dd31998ab1c4706f131af4ceb7c26b4ead3aacb4f6  internal/daemon/execenv/opencode_home.go
dc307e2c507018c463aecc2f32963c422bf8838203379e85f416e5f4a2428d89  internal/daemon/execenv/opencode_home_test.go
9e402f8c5583f0288185a4f3a7d3b8fb1376e0da6e22be2f5336a69aaed26c75  internal/daemon/execenv/vendor_credential_fallback_test.go
80d3d990c470ad9e7a21d661d51553bd37690edc13e16c7e95246ebe211df834  internal/daemon/runtimeenv/policy.go
ba3af87afd2f4f2dd05c07c703c73f59e8ee8d12fe07eea8d9f3824190528a31  internal/daemon/runtimeenv/env.go
a41e030c566e29fd9f5941033a7c67d3d3cac379d657018c6ac5f6f7088e4527  internal/daemon/runtimeenv/env_test.go
eef92d45127137f5f339a430490c7b188f5704c9ebf55b545a54fc0c3c85fd4e  internal/rotation/contract.go
e47757784ba22d9c4d749c51028247523a6d8cfd3db002343a503d5fba67edf8  internal/rotation/store_pg.go
```

## Residual platform and acceptance gaps

1. Kiro's `XDG_DATA_HOME` and Antigravity's `HOME` are design-recorded observations, not live validations in this pack; the task/proposal still names different variables.
2. Claude credential-store/config-root behavior remains explicitly unresolved and unimplemented.
3. Gemini/Antigravity login and `HOME` isolation remain unproved against a real binary; no live/provider acceptance is claimed.
4. Windows runtime behavior was not executed; the focused Go helpers are platform-neutral file operations, but path/case/symlink behavior remains a platform gap.
5. No DB-backed assignment/tenant/provider integration ran. The production tenant-boundary omission is therefore a source finding, not an executed DB exploit.
6. No test proves per-account Codex config/session/plugin isolation or blocks every alternate provider discovery root.
7. ExactEnv's separately documented same-UID validation-to-exec TOCTOU remains; this audit did not change or close it.

## Required disposition

Do not check tasks 1.1–1.3. Preserve production fail-closed behavior while reconciling task 1.3. Before acceptance, add tenant-scoped assignment validation, define and consume `ConfigDir` per provider (including Codex config/session semantics), close alternate-root custom/inherited env bypasses, and add pure fake-store/cross-provider tests that run without PostgreSQL.
