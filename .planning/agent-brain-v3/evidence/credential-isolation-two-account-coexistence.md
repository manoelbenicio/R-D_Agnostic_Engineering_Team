# Agent credential isolation task 5.2: independent KEEP-CHECKED validation

Validated on 2026-07-18 from repository source and synthetic, temporary test
directories only. No real credential, auth home, session home, credential file,
token, environment secret, network service, live session, or database was read or
used.

## Verdict

**KEEP-CHECKED.** Task 5.2 is supported by deterministic source-only evidence that
two accounts of the same provider can be prepared concurrently with distinct
task-local credential/config paths and sentinel content that does not cross account
boundaries. The final daemon launch gate removes inherited provider keys before
injecting a complete task-isolated set, and the no-assignment fallback does not
invent isolated homes.

This is not a claim that the PostgreSQL-backed aggregate acceptance test ran. See
"Explicit non-claim" below.

## Exact synthetic-test source anchors

All paths in this section are relative to `multica-auth-work/server`.

| Coverage | Exact source anchor | What the synthetic test proves |
| --- | --- | --- |
| Codex, two accounts | `internal/daemon/execenv/codex_home_account_test.go:13` | Separate temporary account dirs and distinct synthetic `auth.json` sentinels are prepared into separate task homes; lines 58-78 prove regular-file copying and that refreshing A does not change B. |
| Codex, missing assignment | `internal/daemon/execenv/codex_home_account_test.go:83` | Credential-bearing preparation fails closed before creating task state when no account home is supplied. |
| Codex, compatibility fallback | `internal/daemon/execenv/codex_home_account_test.go:97` | The legacy shared seed is available only through the explicit compatibility flag and is copied, not symlinked. |
| Codex, credentialless gateway | `internal/daemon/execenv/codex_home_account_test.go:124` | A synthetic inherited shared fixture is not seeded into the task-local gateway home. |
| Kiro, two accounts | `internal/daemon/execenv/kiro_home_test.go:9` | Distinct synthetic account stores are copied to distinct `XDG_DATA_HOME` trees; lines 25-43 prove no symlink and no A-to-B mutation. |
| Kiro fallback | `internal/daemon/execenv/kiro_home_test.go:46` | No account assignment creates no credential copy. |
| Antigravity, two accounts | `internal/daemon/execenv/antigravity_home_test.go:9` | Distinct synthetic token trees are copied to distinct task `HOME` trees; lines 25-43 prove no symlink and no A-to-B mutation. |
| Antigravity fallback | `internal/daemon/execenv/antigravity_home_test.go:46` | No account assignment creates no credential copy. |
| Cline, two accounts | `internal/daemon/execenv/cline_home_test.go:9` | Separate account directories and sentinel provider files produce distinct task data dirs; lines 43-79 verify exact Cline env mapping, sandbox path isolation, and no A-to-B/source mutation. |
| OpenCode, two accounts | `internal/daemon/execenv/opencode_home_test.go:9` | Separate account data/config trees and A/B sentinels remain distinct; lines 45-79 verify exact XDG mapping and no mutation of B or A's source when A's task copy changes. |
| GLM provider boundary | `internal/daemon/execenv/opencode_home_test.go:82` | GLM deliberately uses the OpenCode-compatible isolated data/config layout and exact `XDG_DATA_HOME`/`XDG_CONFIG_HOME` mapping. |
| NIM, two accounts | `internal/daemon/execenv/nim_home_test.go:9` | Two synthetic API-key files yield non-overlapping copied paths with mode `0600`; lines 37-48 verify account-specific env values and no source overlap. NIM is extra execenv coverage, not one of the six P0 aggregate vendors. |
| No-assignment vendor fallback | `internal/daemon/execenv/vendor_credential_fallback_test.go:5` | Prepare and Reuse with an empty account home do not set isolated homes for Kiro, Antigravity, Cline, OpenCode, GLM, or NIM. |
| Exact P0 provider matrix | `internal/daemon/runtime_isolation_test.go:495` and `:773` | The P0 matrix is exactly Codex, Kiro, Antigravity, GLM, Cline, and OpenCode; every provider must be isolation-required and reject empty or incomplete isolation. |
| Missing/partial isolation | `internal/daemon/runtime_isolation_test.go:793` | Synthetic inherited Codex and Cline roots are deleted when complete task isolation cannot be built, including with L2 enabled. |
| Complete replacement | `internal/daemon/runtime_isolation_test.go:846` | A synthetic inherited `CODEX_HOME` is replaced by the task-local home while an unrelated env key remains unchanged. |

The provider-native env mapping is implemented at
`internal/daemon/execenv/execenv.go:633-679`: Codex uses `CODEX_HOME`, Kiro uses
`XDG_DATA_HOME`, Antigravity uses `HOME`, Cline uses its data/sandbox keys,
OpenCode and GLM use both XDG roots, and NIM uses `NVIDIA_API_KEY`.

The daemon boundary is explicit:

- `internal/daemon/daemon.go:4059-4065` identifies all isolation-required
  providers.
- `internal/daemon/daemon.go:4068-4090` rejects an empty account home, missing
  execution environment, or incomplete provider-native mapping.
- `internal/daemon/daemon.go:4101-4103` deletes every inherited provider key
  before validation/injection.
- `internal/daemon/daemon.go:4104-4110` injects only the prepared isolated map.
- `internal/daemon/daemon.go:4114-4127` defines the exact required key set at
  each provider boundary.

## Commands and results

Working directory for every Go command:
`multica-auth-work/server`. Toolchain:
`/home/dataops-lab/go-sdk/bin/go`; environment:
`GOTOOLCHAIN=local GOPROXY=off`.

### Thirteen execenv tests, repeated 20 times

```sh
GOTOOLCHAIN=local GOPROXY=off /home/dataops-lab/go-sdk/bin/go test ./internal/daemon/execenv -count=20 -run '^(TestPrepareCodexHomePerAccountIsolatesAuth|TestPrepareCodexHomeFailsClosedWhenNoAccount|TestPrepareCodexHomeLegacySharedSeedRequiresExplicitOptIn|TestPrepareCredentiallessGatewayCodexHomeNeverSeedsSharedAuth|TestPrepareKiroHomePerAccountIsolatesDataStore|TestPrepareKiroHomeFallbackWhenNoAccount|TestPrepareAntigravityHomePerAccountIsolatesTokenDir|TestPrepareAntigravityHomeFallbackWhenNoAccount|TestPrepareClineHomePerAccountIsolatesDataDir|TestPrepareOpenCodeHomePerAccountIsolatesDataAndConfig|TestPrepareGLMUsesOpenCodeCompatibleIsolation|TestPrepareNimHomePerAccountCopiesAPIKey|TestVendorCredentialFallbackDoesNotSetIsolatedHomes)$'
```

Result: exit 0; `ok github.com/multica-ai/multica/server/internal/daemon/execenv 0.885s`.

### Three daemon tests, repeated 20 times

```sh
GOTOOLCHAIN=local GOPROXY=off /home/dataops-lab/go-sdk/bin/go test ./internal/daemon -count=20 -run '^(TestCredentialIsolationVendorMatrixCoversExactlySixP0Vendors|TestL2EnabledLocalLaunchFailsClosedOnMissingOrPartialIsolation|TestL2EnabledLocalLaunchInjectsCompleteSyntheticIsolation)$'
```

Result: exit 0; `ok github.com/multica-ai/multica/server/internal/daemon 0.065s`.

### Race detector

```sh
GOTOOLCHAIN=local GOPROXY=off /home/dataops-lab/go-sdk/bin/go test -race ./internal/daemon/execenv -run '^(TestPrepareCodexHomePerAccountIsolatesAuth|TestPrepareCodexHomeFailsClosedWhenNoAccount|TestPrepareCodexHomeLegacySharedSeedRequiresExplicitOptIn|TestPrepareCredentiallessGatewayCodexHomeNeverSeedsSharedAuth|TestPrepareKiroHomePerAccountIsolatesDataStore|TestPrepareKiroHomeFallbackWhenNoAccount|TestPrepareAntigravityHomePerAccountIsolatesTokenDir|TestPrepareAntigravityHomeFallbackWhenNoAccount|TestPrepareClineHomePerAccountIsolatesDataDir|TestPrepareOpenCodeHomePerAccountIsolatesDataAndConfig|TestPrepareGLMUsesOpenCodeCompatibleIsolation|TestPrepareNimHomePerAccountCopiesAPIKey|TestVendorCredentialFallbackDoesNotSetIsolatedHomes)$'
GOTOOLCHAIN=local GOPROXY=off /home/dataops-lab/go-sdk/bin/go test -race ./internal/daemon -run '^(TestCredentialIsolationVendorMatrixCoversExactlySixP0Vendors|TestL2EnabledLocalLaunchFailsClosedOnMissingOrPartialIsolation|TestL2EnabledLocalLaunchInjectsCompleteSyntheticIsolation)$'
```

Results: both exit 0; execenv `ok` in 1.150s and daemon `ok` in 1.083s.

### Vet, format, and diff checks

```sh
GOTOOLCHAIN=local GOPROXY=off /home/dataops-lab/go-sdk/bin/go vet ./internal/daemon/execenv ./internal/daemon
/home/dataops-lab/go-sdk/bin/gofmt -d internal/daemon/daemon.go internal/daemon/runtime_isolation_test.go internal/daemon/execenv/execenv.go internal/daemon/execenv/codex_home.go internal/daemon/execenv/codex_home_account_test.go internal/daemon/execenv/kiro_home.go internal/daemon/execenv/kiro_home_test.go internal/daemon/execenv/antigravity_home.go internal/daemon/execenv/antigravity_home_test.go internal/daemon/execenv/cline_home.go internal/daemon/execenv/cline_home_test.go internal/daemon/execenv/opencode_home.go internal/daemon/execenv/opencode_home_test.go internal/daemon/execenv/nim_home.go internal/daemon/execenv/nim_home_test.go internal/daemon/execenv/vendor_credential_fallback_test.go
git diff --no-index --check -- /dev/null .planning/agent-brain-v3/evidence/credential-isolation-two-account-coexistence.md
```

Results: vet exited 0 with no output; `gofmt -d` exited 0 with no output. The
Because the artifact is new and untracked, the no-index `git diff --check`
examined it against `/dev/null`: it produced no whitespace diagnostics and the
expected status 1 indicating that content differs from `/dev/null`.

## Source SHA-256 manifest

Hashes were computed only over the repository source/test paths listed below.
No external home, credential, auth, session, token, environment-secret, or live
service path was enumerated, read, copied, or hashed.

```text
a1d96a3c8a4edfc9f9c583326c6f74779643473904f13f0a8aab63ea1cd6fe07  internal/daemon/daemon.go
168dc34f17650e3d4f07d324a5272a9dc5839f7b2ed28b5b1e643ef823fa7308  internal/daemon/runtime_isolation_test.go
8099867594f18f6d5e5c47fa88581f06ac771e456f203fc0161d09c6584fc0e6  internal/daemon/execenv/execenv.go
aa3f8bbd82ffb3fe70ed72da037eaae3028af4b558a7b1d0c207ccaeb4cb3fe0  internal/daemon/execenv/codex_home.go
31d759a40ab35780ef875779574fc7d82ace4c5801ae565cc96845a0fe9b6f4a  internal/daemon/execenv/codex_home_account_test.go
f8c616596d8a0f4deafb4eb88a98d45f5be888d2c79f0ae1b1db03b7bb0e083c  internal/daemon/execenv/kiro_home.go
c4a99c9ea3b4fd1d41981c2c06d190ab82ed4e56943e66fad5dbffd2b45c4ce5  internal/daemon/execenv/kiro_home_test.go
9f116ab69d399e275bae47a0f8daf788a579bb7c4dd2c194748dc977de31cb04  internal/daemon/execenv/antigravity_home.go
e7527f8bf70a634c8b26a2afd5c150a1e4ba82ca9737e46ba8266b8a4e51521a  internal/daemon/execenv/antigravity_home_test.go
eed1a62351db6b3f3d65e556b98021a0057a599ff3387bf2af6c00b2f828c3bd  internal/daemon/execenv/cline_home.go
2d59decd07b70ad9ec10455a913b376e62929fe55a52fdfa11cc788bac17f181  internal/daemon/execenv/cline_home_test.go
427bbf7e03f59a7ca7cbd2dd31998ab1c4706f131af4ceb7c26b4ead3aacb4f6  internal/daemon/execenv/opencode_home.go
dc307e2c507018c463aecc2f32963c422bf8838203379e85f416e5f4a2428d89  internal/daemon/execenv/opencode_home_test.go
e598f503eca6f377c06064283aa6fd43385b7b26168b4fe9a4b1867d251ec93f  internal/daemon/execenv/nim_home.go
41be73dd400bcda6d5c9cc79acb9030ecb86506b61a3fdbc0516aba52cac0317  internal/daemon/execenv/nim_home_test.go
9e402f8c5583f0288185a4f3a7d3b8fb1376e0da6e22be2f5336a69aaed26c75  internal/daemon/execenv/vendor_credential_fallback_test.go
```

The manifest was revalidated with `sha256sum -c` against these repository paths
after the artifact was written.

## Explicit non-claim

`internal/daemon/runtime_isolation_test.go:750` defines
`TestCredentialIsolationPerVendor`, whose setup at lines 497-505 requires
`DATABASE_URL` and PostgreSQL. It was deliberately not executed: this validation
used no database, network, live service, or environment secret. Therefore this
artifact does **not** claim that the DB-backed aggregate passed, ran, compiled a
database interaction, or exercised PostgreSQL rows. Its conclusions are limited
to the genuinely executed source-only synthetic tests and static source anchors
listed above.

No product file, main documentation, OpenSpec artifact/checklist, or other evidence
file was edited for this validation.
