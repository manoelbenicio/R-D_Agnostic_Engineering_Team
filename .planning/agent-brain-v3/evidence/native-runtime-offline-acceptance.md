# Native Runtime Offline Acceptance

- Date: 2026-07-18
- Lane: native NIM/Cline runtime verification only
- Working directory: `multica-auth-work/server`
- Toolchain: `/home/dataops-lab/go-sdk/bin/go` (`go version go1.26.4 linux/amd64`)
- Offline controls on every Go verification command: `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`
- Result: PASS for native tasks 1.1-1.4 and 2.1-2.3

## Scope and exclusions

This acceptance covers source-level and synthetic-test verification of the NIM and Cline native packages, NIM credential isolation and rotation detection, model discovery cache/timeout and daemon reporting, config probes, `agent.New`/`SupportedTypes`, and NIM `requiresCredentialIsolation` wiring.

Explicitly excluded: live runtime or provider execution, daemon rebuild/start/restart, web build/start, network access, real credentials or credential homes, live providers, services, smoke testing, UAT, and native onboarding tasks 2.4 through 3.4. No product source, main documentation, OpenSpec artifact, checklist, credential, or other evidence file was changed by this acceptance lane.

## Task acceptance mapping

| Task | Acceptance evidence | Result |
| --- | --- | --- |
| 1.1 NIM backend | NIM SSE/agent loop, tools, usage, default model, required credential/workspace, confinement, symlink rejection, and API-key non-leak tests; repeated and raced | PASS |
| 1.2 NIM isolation/rotation | Per-account NIM credential copy/refresh/fail-closed tests plus daemon and shared rotation detector tests; repeated and raced | PASS |
| 1.3 Cline backend | Native ACP launch/isolation, blocked incompatible arguments, malformed MCP rejection, and tool-name normalization tests; repeated and raced | PASS |
| 1.4 Model discovery | Cache, negative/stale cache, singleflight/cancellation, executable-key isolation, ACP timeout/process-tree cleanup, Cline fallback, NIM static catalog, and daemon result reporting; repeated and raced | PASS |
| 2.1 Config probes | Cline path/model probe and credential-gated NIM inclusion/exclusion tests; repeated and raced | PASS |
| 2.2 `agent.New` and `SupportedTypes` | NIM/Cline factory construction and exact whitelist lockstep tests; repeated and raced | PASS |
| 2.3 NIM credential isolation wiring | Native HTTP runtime version plus case/whitespace-normalized `requiresCredentialIsolation("nim")`; repeated and raced | PASS |

## Exact commands and results

All commands below ran from `multica-auth-work/server`. No command used Docker or allowed module/toolchain network lookup.

### Toolchain identity

```bash
/home/dataops-lab/go-sdk/bin/go version
```

Result: exit 0; `go version go1.26.4 linux/amd64`.

### NIM backend, Cline backend, and agent factory/whitelist

```bash
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go test ./pkg/agent -run '^(TestNIMExecute|TestNIMUsesGLM52AsRuntimeDefault|TestNIMExecuteRequiresCredentialAndWorkspace|TestExecuteNIMTool|TestNIMAPIErrorsAreReturnedWithoutLeakingKey|TestCline|TestNewReturns(Cline|NIM)Backend|TestSupportedTypes)' -count=20
```

Result: exit 0; `ok github.com/multica-ai/multica/server/pkg/agent 0.704s`.

```bash
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go test -race ./pkg/agent -run '^(TestNIMExecute|TestNIMUsesGLM52AsRuntimeDefault|TestNIMExecuteRequiresCredentialAndWorkspace|TestExecuteNIMTool|TestNIMAPIErrorsAreReturnedWithoutLeakingKey|TestCline|TestNewReturns(Cline|NIM)Backend|TestSupportedTypes)' -count=5
```

Result: exit 0; `ok github.com/multica-ai/multica/server/pkg/agent 1.339s`.

### NIM isolation and rotation

```bash
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go test ./internal/daemon/execenv ./internal/rotation -run '^(TestPrepareNimHome|TestNIM)' -count=20
```

Result: exit 0; `internal/daemon/execenv` passed in 0.323s and `internal/rotation` passed in 0.017s.

The daemon-package command below also repeated `TestNIMExhaustionDetectorMatcher` and `TestNIMExhaustionDetectorViaDetect` 20 times.

### Model discovery daemon surface, config probes, and NIM isolation wiring

```bash
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go test ./internal/daemon -run '^(TestNIMExhaustionDetector|TestLoadConfig_(ProbesClineAndNIMCredential|SkipsNIMWithoutCredential)|TestNIMUsesNativeHTTPRuntimeVersion|TestRequiresCredentialIsolationIncludesNIM|TestReportModelListResult_)' -count=20
```

Result: exit 0; `ok github.com/multica-ai/multica/server/internal/daemon 33.810s`.

```bash
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go test ./pkg/agent -run '^(TestCachedDiscovery|TestExecutableDiscoveryKeys|TestDiscoverACPModels(TimeoutReapsProcessTree|ClosesStdinAndWaitsGracefully|ReapsOrphanChild)|TestListModelsClineFallsBackAndAnnotatesThinking|TestNIMStaticModelsDefaultsToGLM52)' -count=20
```

Result: exit 0; `ok github.com/multica-ai/multica/server/pkg/agent 23.259s`.

### Race detector

```bash
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go test -race ./internal/daemon -run '^(TestNIMExhaustionDetector|TestLoadConfig_(ProbesClineAndNIMCredential|SkipsNIMWithoutCredential)|TestNIMUsesNativeHTTPRuntimeVersion|TestRequiresCredentialIsolationIncludesNIM|TestReportModelListResult_)' -count=5
```

Result: exit 0; daemon package passed in 5.630s.

```bash
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go test -race ./internal/daemon/execenv ./internal/rotation -run '^(TestPrepareNimHome|TestNIM)' -count=5
```

Result: exit 0; `internal/daemon/execenv` passed in 1.218s and `internal/rotation` passed in 1.085s.

```bash
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go test -race ./pkg/agent -run '^(TestCachedDiscovery|TestExecutableDiscoveryKeys|TestDiscoverACPModels(TimeoutReapsProcessTree|ClosesStdinAndWaitsGracefully|ReapsOrphanChild)|TestListModelsClineFallsBackAndAnnotatesThinking|TestNIMStaticModelsDefaultsToGLM52)' -count=5
```

Result: exit 0; agent package passed in 6.704s.

No race report or assertion failure occurred.

### Linux vet

```bash
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go vet ./pkg/agent ./internal/daemon/execenv ./internal/rotation ./internal/daemon
```

Result: exit 0; no diagnostics.

### Windows amd64 vet

```bash
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOOS=windows GOARCH=amd64 CGO_ENABLED=0 /home/dataops-lab/go-sdk/bin/go vet ./pkg/agent ./internal/daemon/execenv ./internal/rotation ./internal/daemon
```

Result: exit 0; no diagnostics.

### Windows amd64 test-binary compilation

```bash
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOOS=windows GOARCH=amd64 CGO_ENABLED=0 /home/dataops-lab/go-sdk/bin/go test -c -o /tmp/native-agent-windows.test.exe ./pkg/agent
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOOS=windows GOARCH=amd64 CGO_ENABLED=0 /home/dataops-lab/go-sdk/bin/go test -c -o /tmp/native-execenv-windows.test.exe ./internal/daemon/execenv
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOOS=windows GOARCH=amd64 CGO_ENABLED=0 /home/dataops-lab/go-sdk/bin/go test -c -o /tmp/native-rotation-windows.test.exe ./internal/rotation
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOOS=windows GOARCH=amd64 CGO_ENABLED=0 /home/dataops-lab/go-sdk/bin/go test -c -o /tmp/native-daemon-windows.test.exe ./internal/daemon
```

Result: all four commands exited 0; every output was non-empty. Temporary binaries were removed after verification. This compilation includes the Windows-only fail-closed discovery tests and process-containment implementation.

## Setup failures versus assertion failures

The earlier container-only attempt was blocked during package setup because required modules were not present in that isolated cache while `GOPROXY=off` prohibited lookup. Those were missing-cache setup failures, not assertion failures. In this corrected host-toolchain run, the existing host module/build cache satisfied every import: package setup, compilation, all repeated assertions, race checks, vet checks, and Windows cross-compilation passed. There are no remaining native-task BLOCKs in the accepted scope.

## Source SHA-256 manifest

Hashes were computed from the accepted working tree with `sha256sum` after testing.

| Task(s) | SHA-256 | Source/test path |
| --- | --- | --- |
| 1.1 | `eeb4cd70c2de93a64cf60a9d5f02a4adb2739a9faf23a2dcc1d619dc0e64467f` | `multica-auth-work/server/pkg/agent/nim.go` |
| 1.1 | `3de97d28f75c718addbd3faad833ce60bba39a790f63f8734330e6c3c9991684` | `multica-auth-work/server/pkg/agent/nim_test.go` |
| 1.2 | `e598f503eca6f377c06064283aa6fd43385b7b26168b4fe9a4b1867d251ec93f` | `multica-auth-work/server/internal/daemon/execenv/nim_home.go` |
| 1.2 | `41be73dd400bcda6d5c9cc79acb9030ecb86506b61a3fdbc0516aba52cac0317` | `multica-auth-work/server/internal/daemon/execenv/nim_home_test.go` |
| 1.2 | `796271dad20ecc56e2c4ac8c524fa2fc4a407d628176c125001a3c9393b8c35a` | `multica-auth-work/server/internal/daemon/rotation_detector_nim.go` |
| 1.2 | `36b2264c606309062613135f5ea0edf43fe22abb2ba36ae1d4d7c62da16de893` | `multica-auth-work/server/internal/daemon/rotation_detector_nim_test.go` |
| 1.2 | `be7d576140926d2e1999d5a818d0e68dbdc454fb98829caa9132c89f9fa1405f` | `multica-auth-work/server/internal/rotation/detector_nim.go` |
| 1.2 | `070c8d40d4b566f21371b36fd960c6a6e34b8ac9caae1d0ee1d319ad89e58688` | `multica-auth-work/server/internal/rotation/detector_nim_test.go` |
| 1.3 | `9497ebfccaeb143cef0e08b2ae4f59f5192a40d118d2f68ff208f9ae1322ede0` | `multica-auth-work/server/pkg/agent/cline.go` |
| 1.3 | `d24cc8f4d2ebe5556f1a9e4babdf995c02d43400c6956efd2f341d77bd5c70a9` | `multica-auth-work/server/pkg/agent/cline_test.go` |
| 1.4 | `a6957e3e0b4a05050da6dc198049581d6402103d474185d0912f3360e8a7b313` | `multica-auth-work/server/pkg/agent/models.go` |
| 1.4 | `b1c62961e671f697b32844448c739e6059bccfe5e9c2bc1d6b1e52fb908dad5b` | `multica-auth-work/server/pkg/agent/models_test.go` |
| 1.4 | `75f1cc5d94bd240e955df5a61d34ca412ce9e980277eb66e0cf97495137c4211` | `multica-auth-work/server/pkg/agent/models_process_test.go` |
| 1.4 | `8ff9e9c2ae75d590d4ff75b6bf9d3f1813cde190418def5ee31ee4bb74fb7b7a` | `multica-auth-work/server/pkg/agent/models_windows_test.go` |
| 1.4 | `e92f2c48385d46f06f877398fcacb8e195c1f8ac21864dc9989b35de57c47ba9` | `multica-auth-work/server/pkg/agent/proc_other.go` |
| 1.4 | `7a1601f67bfbbddee65e739f3e4725d8d960ca2ede6e46e5428f2613be69e7cc` | `multica-auth-work/server/pkg/agent/proc_windows.go` |
| 1.4 | `679af9b9f721eb03a5ed74dd87da31d88a53f51e1481a22744980310942cc2c6` | `multica-auth-work/server/pkg/agent/proc_unsupported.go` |
| 1.4 | `6a644d5c4a92490c0311beaa80698a251f9b89a65f7ce8e7d0e87c8330203962` | `multica-auth-work/server/internal/daemon/model_list_report_test.go` |
| 2.1 | `9a8a33f6cc6ad2ff95cb9034d23900a8ca9bdac5b1eb815eb8db979a642189cf` | `multica-auth-work/server/internal/daemon/config.go` |
| 2.1 | `ed0b740520e1b50c50a9731892f9f0b446fd0769e872e08086f532963d89f836` | `multica-auth-work/server/internal/daemon/config_test.go` |
| 2.2 | `84cc33be31e6a4ebcceb93ccb6b408955f74d3fedeb487c6be25da0c4e816ba8` | `multica-auth-work/server/pkg/agent/agent.go` |
| 2.2 | `d94a092e317701746b4937a9d382323830ea7194c0dad8045667df935b14a38f` | `multica-auth-work/server/pkg/agent/agent_test.go` |
| 2.2 | `fe6bdece31eb19b5db976b41061b4b09f5d027badb046fb498628dfc6fc7f8f4` | `multica-auth-work/server/pkg/agent/agent_supported_types_test.go` |
| 1.4, 2.3 | `a1d96a3c8a4edfc9f9c583326c6f74779643473904f13f0a8aab63ea1cd6fe07` | `multica-auth-work/server/internal/daemon/daemon.go` |
| 2.3 | `ae8035fc15ca50cbb24b914b955c99abc103c07acf85ea045f563536a38323f3` | `multica-auth-work/server/internal/daemon/native_runtime_wiring_test.go` |

## Acceptance conclusion

Within the explicitly offline, source/synthetic-test-only scope, native tasks 1.1-1.4 and 2.1-2.3 are accepted as PASS. Tasks 2.4-3.4 remain excluded and require their separately authorized live/build/UI verification lanes.

## Independent reviewer ACCEPT (2026-07-18)

Verdict: **ACCEPT** for the artifact's explicitly offline, source/synthetic-test-only scope. The pre-review artifact SHA-256 was `a899cb0dcc26d71ba1ef58b5e357b17cb5ad2f8d2617ff490109b444836bf496`.

- Recomputed the manifest against the current disk: all 25 listed source/test paths exist and all 25 SHA-256 values match exactly.
- Verified each focused `-run` expression with `go test -list`: every claimed test name exists in the stated package. Re-ran all recorded x20 commands, not merely a sample; package results were PASS (`pkg/agent` backend/factory `0.828s`, `execenv` `0.225s`, `rotation` `0.028s`, `daemon` `15.871s`, and model discovery `21.947s`). Timing differences from the original run are expected and do not change the reproduced assertions.
- Re-ran every recorded race command with `-count=5`: PASS (`pkg/agent` backend/factory `1.306s`, `daemon` `4.834s`, `execenv` `1.123s`, `rotation` `1.077s`, and model discovery `6.618s`).
- Re-ran the exact Linux and Windows `go vet` commands: both PASS with exit 0 and no diagnostics. Re-ran all four exact Windows `go test -c` commands: each produced a non-empty PE32+ x86-64 Windows test executable for the stated package.
- All reruns used `/home/dataops-lab/go-sdk/bin/go` with `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`; no network, credentials, database, Docker daemon, or live service/provider was used. All 25 manifest Go files are `gofmt -l` clean, and `git diff --check` passes.
- The OpenSpec/source mapping is supported for task 1.1 (native NIM backend), 1.2 (NIM isolation and rotation), 1.3 (native Cline ACP), 1.4 (bounded/cached model discovery and daemon reporting), 2.1 (runtime/credential probes), 2.2 (factory and supported-type wiring), and 2.3 (NIM credential isolation wiring). This review makes no claim for tasks 2.4-3.4, runtime rebuild/restart, web/UI build or startup, end-to-end UI population, smoke tests, or live-provider behavior.
