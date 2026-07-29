# G4 ExactEnv Physical Containment and R22 Mitigation

- Owner: independent acceptance evidence pack (TL-rerun)
- OpenSpec scope: agent-credential-isolation / Agent Brain v3 — R22 mitigation (broad `os.Environ` inheritance)
- Evidence ID: EV-G4-EXACTENV (physical containment; complements `EV-G4-03` PARTIAL and pins the accepted point-in-time contract plus the residual TOCTOU)
- Recorded: 2026-07-18 (UTC, WSL)
- Status: ACCEPTED point-in-time contract — residual hostile same-UID TOCTOU tracked for OS-level isolation (R22 MITIGATED, not closed)
- Baseline: STATE.md:5 / AGENT_LEDGER.md:188 record owner-asserted "**ExactEnv physical containment ACCEPT** — residual hostile same-UID TOCTOU requires OS-level isolation, so **R22 is MITIGATED, not open**". This pack supplies the missing review-artifact IDs/hashes flagged at AGENT_LEDGER.md:188 ("Unresolved evidence pin ... append the REVIEW-* IDs/hashes when available").
- Constraint: no credential home/file/value or live session inspected; only synthetic temporary paths (`t.TempDir()`) used.

## Provenance

- Host: WSL linux/amd64; toolchain `/home/dataops-lab/go-sdk/bin/go` → `go version go1.26.4 linux/amd64`.
- Env: `GOTOOLCHAIN=local GOPROXY=off` (no network, no module download).
- Workdir: `multica-auth-work/server`.
- Packages: `./internal/daemon/runtimeenv/` (production + tests) and `./internal/daemon/` brain integration consumer.
- No git commit/diff vs HEAD: all target files are untracked (`??`) — see `git status` below. No product/main-planning/OpenSpec/other-evidence file edited.

## Source file hashes (SHA256)

Production sources (`internal/daemon/runtimeenv/`):

| File | SHA256 |
|---|---|
| `env.go` | `ba3af87afd2f4f2dd05c07c703c73f59e8ee8d12fe07eea8d9f3824190528a31` |
| `assert.go` | `2f3100d2d948c14628322149a4c8b9aefb9875659b2c135514a029e926d6f025` |
| `doc.go` | `702e8d1e56f0215ecc992f400043aa2b3c673e6fe1032153321272aadfde0941` |
| `policy.go` | `80d3d990c470ad9e7a21d661d51553bd37690edc13e16c7e95246ebe211df834` |
| `home.go` | `dbf47c330902c2742f7f6933990111c2445a6077399041482a52cc5a96cffcfb` |
| `codex.go` | `4fce4b7af144ca71118b6d33fa03bd23fd71b8162b7c37c044486106271a1759` |
| `adapter.go` | `72e388dc31e695918366ca101434a396c8a5cbe389bdf5b98a187c2cf985d337` |
| `model.go` | `8b5a8ce667b1aa57d8f84474600dd77152c6a8a00c078107c097db7ce08d6e76` |

Test sources (`internal/daemon/runtimeenv/`):

| File | SHA256 |
|---|---|
| `env_test.go` | `a41e030c566e29fd9f5941033a7c67d3d3cac379d657018c6ac5f6f7088e4527` |
| `assert_test.go` | `934df619997135d320c71566b9007276438868a777296447387b28c27313cc4a` |
| `isolation_g4_test.go` | `24ccc85466344c815fc1882dcb30cc56f67dcc0951ea7980894b7bc698dab8d3` |

Brain integration consumer (`internal/daemon/`):

| File | SHA256 |
|---|---|
| `brain_integration.go` | `9d6b59f87111d6a1da2e6bfcac19a3cfbd6aa6a77d4d1990ee0c172fbde1748e` |
| `brain_integration_test.go` | `996cbe09989549de653988d19897eb2e17c5fc3fd1e5aa147ce76e4be31a2d48` |

`g4SyntheticGatewayValue` is defined at `internal/daemon/runtimeenv/gateway_acceptance_test.go:20` (`"synthetic-g4-gateway-value"`); all isolation tests use synthetic values only.

## Exact commands run and results

All commands run from `multica-auth-work/server` with `GOTOOLCHAIN=local GOPROXY=off` and the pinned toolchain `/home/dataops-lab/go-sdk/bin/go` (go1.26.4 linux/amd64).

| # | Command | Result |
|---|---|---|
| 1 | `/home/dataops-lab/go-sdk/bin/gofmt -l internal/daemon/runtimeenv/env.go internal/daemon/runtimeenv/assert.go internal/daemon/runtimeenv/env_test.go internal/daemon/runtimeenv/assert_test.go internal/daemon/runtimeenv/isolation_g4_test.go internal/daemon/runtimeenv/doc.go internal/daemon/brain_integration.go internal/daemon/brain_integration_test.go` | **CLEAN** (no files listed) |
| 2 | `go vet ./internal/daemon/runtimeenv/` | **exit=0** (no diagnostics) |
| 3 | `go test -run 'TestBuildMinimalInheritedRemovesCredentialAndRoutingSurface|TestBuildGatewayEnvironmentRejectsNoncanonicalTrustedHomes|TestControlledDirectoryValidationRejectsSymlinkComponents|TestAssertPreLaunchRejectsHomesOutsideOrEscapingExecutionRoot|TestAssertPreLaunchRejectsPhysicalHomeSubstitution' -count=20 -v ./internal/daemon/runtimeenv/` | **ok**, exit=0; iteration proof: `grep -c '^=== RUN   TestAssertPreLaunchRejectsPhysicalHomeSubstitution$'` = **20** |
| 4 | `go test -run '<same set>' -race -count=1 ./internal/daemon/runtimeenv/` | **ok**, exit=0 (no races) |
| 5 | `go test -count=1 ./internal/daemon/runtimeenv/` (full package) | **ok 0.088s**, exit=0 (confirms `g4SyntheticGatewayValue` resolves and whole package compiles/runs) |
| 6 | `GOOS=windows GOARCH=amd64 go vet ./internal/daemon/runtimeenv/` | **exit=0** |
| 7 | `GOOS=windows GOARCH=amd64 go build ./internal/daemon/runtimeenv/` | **exit=0** (Windows compile clean) |
| 8 | `git status --short -- <target files>` | all `??` (untracked/new); no modified production file |
| 9 | `git diff --check -- <target files>` | no whitespace errors |

Focused-test x20 covered the four containment dimensions the task names:
- **traversal**: `TestBuildGatewayEnvironmentRejectsNoncanonicalTrustedHomes` (3 subtests), `TestAssertPreLaunchRejectsHomesOutsideOrEscapingExecutionRoot` (4 subtests: outside-root + `..` traversal for HOME/CODEX_HOME).
- **symlink**: `TestControlledDirectoryValidationRejectsSymlinkComponents` (4 subtests: execution-root symlink, HOME symlink, HOME redirected component, CODEX_HOME symlink).
- **post-build substitution (TOCTOU proof)**: `TestAssertPreLaunchRejectsPhysicalHomeSubstitution` (2 subtests: claude-code, codex).
- **R22 sanitization (apply-last)**: `TestBuildMinimalInheritedRemovesCredentialAndRoutingSurface`.

## Accepted point-in-time contract (file:line)

### R22 mitigation — minimal inherited env + trusted apply-last

- `env.go:75-97` `BuildMinimalInherited`: iterates inherited env, `ClassifyEnvironmentKey` denies credential/routing keys (`env.go:85-88`), `isSafeInheritedKey` removes gateway-override keys (`env.go:89-92`), removals sorted by key and carry **key name only** (`env.go:65-67` `Removal{Key, Reason}` — no value field). This is the R22 mitigation: broad `os.Environ` is reduced to an allowlist before the child ever sees it.
- `env.go:154-189` `BuildGatewayEnvironment`: merge order is minimal → local (`env.go:169`) → custom (`env.go:170`) → **trusted merged last** (`env.go:178-180`, comment line 176-177 "Trusted entries are merged last by contract and therefore cannot be shadowed"). This is the "trusted apply-last in the Go path, not just shell" requirement from RISKS.md:28.
- `env_test.go:15-39` asserts `PATH/LC_ALL/TERM` survive and `HOME/ANTHROPIC_API_KEY/OPENAI_BASE_URL/KIMI_TOKEN/NVIDIA_API_KEY/AGENT_BRAIN_GATEWAY_BASE_URL/SESSION_COOKIE/HTTP_PROXY` are removed; report exposes no value (`env_test.go:34-38`).
- `env_test.go:63-101` asserts Claude trusted `ANTHROPIC_BASE_URL`/`ANTHROPIC_AUTH_TOKEN`/`HOME` win over inherited untrusted values, internal markers `CLAUDECODE`/`CLAUDE_CODE_SESSION_ID` are removed, and `fmt.Sprintf("%v", environment)` does not expose the secret.

### Physical containment — canonical, no-symlink, exists-as-directory

- `env.go:241-254` `validateControlledDirectory`: absolute, no surrounding whitespace, `filepath.IsAbs`, `filepath.Clean(path)==path` (canonical), not a filesystem root.
- `env.go:262-291` `validatePhysicalControlledDirectory`: resolves relative to volume anchor (`env.go:266-270` rejects `..` escape), then **component-by-component `os.Lstat`** (`env.go:276-285`) rejecting any `os.ModeSymlink` or non-directory, then a final `filepath.EvalSymlinks(path)` cross-check against the original path (`env.go:286-289`) via `sameCanonicalPath`.
- `env.go:293-300` `sameCanonicalPath`: case-insensitive compare on Windows (`env.go:296-298`), exact compare elsewhere — matches the Windows compile gate below.
- `env.go:258-260` `ValidateExecutionRoot` is the exported entry point.

### Pre-launch assertion — provenance, exactly-one secret, controlled roots

- `assert.go:23-72` `AssertPreLaunch`: (i) adapter contract (`assert.go:25-27`), (ii) non-empty entries + secret key + gateway root (`assert.go:28-30`), (iii) `launchRootsAreControlled` (`assert.go:31-33`), (iv) `PATH`+`HOME` present and non-blank (`assert.go:34-39`), (v) every denied key must be a trusted entry (`assert.go:40-52`), (vi) **exactly one** `originTrustedSecret` whose canonical name equals `environment.secretKey` (`assert.go:46-55`), (vii) task-home manifest valid (`assert.go:56-58`), (viii) CLI-specific Codex/Claude contract (`assert.go:59-70`).
- `assert.go:74-87` `launchRootsAreControlled`: re-validates execution root, checks `HOME` is within the trusted `taskHome` via `exactPathWithin`, and for Codex also checks `CODEX_HOME` is within the trusted `codexHome`.
- `assert.go:89-116` `exactPathWithin` + `physicalDirectoryWithin`: requires canonical absolute paths, equality of expected vs actual, and a **second** `filepath.EvalSymlinks` on both root and path before the relative containment check — defends against a symlink swapped between build and assert.

### Brain integration consumption (the wiring under test)

- `brain_integration.go:268` `runtimeenv.ValidateExecutionRoot(env.RootDir)` — first root check.
- `brain_integration.go:272-288` creates `taskHome` under `env.RootDir` with `os.Lstat` symlink/dir rejection (`:273`), `os.Mkdir(..., 0o700)` (`:277`), then `runtimeenv.ValidateExecutionRoot(taskHome)` (`:283`) and `os.Chmod(..., 0o700)` (`:286`).
- `brain_integration.go:299` `runtimeenv.NewStableSecret(value)` — opaque secret (value never read by runtimeenv; `env.go:38-58` redacts on `String`/`GoString`/`Format`).
- `brain_integration.go:303-309` `runtimeenv.BuildGatewayEnvironment` with `Inherited: inherited()` (default `os.Environ`, overridable for tests at `:293-296`).
- `brain_integration.go:332-334` `runtimeenv.AssertPreLaunch` — the gate immediately before launch.

## Residual hostile same-UID TOCTOU requiring OS-level isolation

The test `TestAssertPreLaunchRejectsPhysicalHomeSubstitution` (`assert_test.go:177-218`) is the TOCTOU proof. It:

1. builds the environment against a real controlled dir (`assert_test.go:182-190`),
2. **removes** the controlled directory (`assert_test.go:196` `os.Remove(target)`),
3. **replaces** it with a symlink to a fresh `t.TempDir()` (`assert_test.go:199` `createTestDirectorySymlink`),
4. asserts `AssertPreLaunch` rejects (`assert_test.go:211-214`).

This proves the assertion catches a **pre-existing** substitution. It does **not** close the window between `AssertPreLaunch` returning (`brain_integration.go:332`) and the eventual `exec` syscall in the daemon launch path. A hostile process running as the **same UID** can perform the remove→symlink swap in that window and win the race, because:

- `validatePhysicalControlledDirectory` (`env.go:286`) and `physicalDirectoryWithin` (`assert.go:108-109`) call `filepath.EvalSymlinks` once and return; there is no atomic "validate-then-exec" primitive in Go's `os/exec`.
- The package holds no file descriptor to the validated directory across the assertion→exec boundary (no `open(O_PATH|O_NOFOLLOW)`, no `O_CLOEXEC` pin), so a same-UID attacker can mutate the directory between the check and the use.
- `sameCanonicalPath` (`env.go:293-300`) and the component `os.Lstat` walk (`env.go:276-285`) are point-in-time checks by construction.

This is the precise residual recorded at `AGENT_LEDGER.md:188` ("residual hostile same-UID TOCTOU requires OS-level isolation") and `STATE.md:5` ("Residual ExactEnv same-UID TOCTOU remains tracked for OS-level isolation"). It is **not** closable in-process by this package; it requires OS-level isolation (e.g., a mount namespace / private bind mount, `seccomp`, or a privileged supervisor that opens the directory with `O_PATH|O_NOFOLLOW` and `fexecve`-style exec from the held FD). That follow-up is tracked separately and is out of scope for this evidence pack.

**Net assessment**: R22 (broad `os.Environ` inheritance leaking provider secrets/routes to the child) is **MITIGATED** by the minimal-inherited + trusted-apply-last contract proven above; the physical-containment TOCTOU is a distinct, narrower residual that needs OS isolation. Both facts are consistent with the owner-asserted acceptance at `AGENT_LEDGER.md:188` / `STATE.md:5`.

## Non-claims and stop conditions

No credential home, credential file, secret value, or live session was inspected, listed, hashed, copied, or mutated. All paths in the tests are synthetic `t.TempDir()` values or the synthetic constants `synthetic-omniroute-value` / `synthetic-g4-gateway-value` / `synthetic-provider-native-value` / `synthetic-cookie-value`. No network, database, live OmniRoute, live provider, daemon dispatch, CLI, or service was exercised. No product code, main planning doc (`STATE.md`, `AGENT_LEDGER.md`, `RISKS.md`, `EVIDENCE_INDEX.md`), OpenSpec file, or other evidence file was edited — only this file (`g4-exactenv-containment.md`) was created. No OpenSpec task checkbox was set or changed. The complementary `EVIDENCE_INDEX.md:70` registration of `EV-G4-03` as PARTIAL is unchanged by this pack; this `EV-G4-EXACTENV` pack supplies the pinned hashes/contract referenced as the unresolved evidence pin at `AGENT_LEDGER.md:188` but does not re-grade that index entry.

## Independent Reviewer Section

**STATUS: BLOCK / CORRECTION APPLIED**
- **Hashes**: All production, test, and brain integration file hashes perfectly match the current source tree.
- **Commands & Output**: The original evidence file contained a bash escaping hallucination. The `go test -run` command escaped the regex pipe operator (`\|`) inside single quotes, causing Go's test runner to search for literal `\|` and match zero tests. The original evidence then hallucinated the timing outputs (`0.485s` / `1.067s`) and the `grep -c` match count (`20`).
- **Correction**: The regex pipes were corrected to `|` and the `grep` spacing was fixed (`^=== RUN   Test...`). The corrected commands were independently run on the `/home/dataops-lab/go-sdk/bin/go` toolchain and passed cleanly (x20 iterations confirmed, no races, Windows compile clean).
- **Residual claims**: The hostile same-UID TOCTOU analysis is completely accurate based on the code in `env.go` and `assert.go`. The test `TestAssertPreLaunchRejectsPhysicalHomeSubstitution` indeed tests for a pre-existing substitution, but cannot close the window before the `exec` syscall.

**FINAL INDEPENDENT RE-REVIEW — ACCEPT (Kiro/Opus-4.8 TL, 2026-07-18):** Reproduced on the pinned toolchain `/home/dataops-lab/go-sdk/bin/go` (go1.26.4) with `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`, synthetic `t.TempDir()` paths only, no credential/auth home/file/value inspected. Verified: (a) every `-run` alternation uses real `|` pipes — no literal `\|`; (b) the commands actually execute the named tests — all 5 parent tests RUN and the representative `TestAssertPreLaunchRejectsPhysicalHomeSubstitution` shows exactly 20 `=== RUN` iterations at `-count=20`; (c) focused x20 exit=0, `-race` clean, full package `ok 0.087s` (matches recorded `0.088s` within run variance), Windows vet+build exit=0, gofmt/vet clean; (d) all 13 source SHA256 hashes match the current tree; (e) residual same-UID TOCTOU independently confirmed — `EvalSymlinks` is point-in-time and the package holds no `O_PATH`/`O_NOFOLLOW`/`fexecve` FD-pin, so R22 is MITIGATED (not closed) exactly as stated. No product/OpenSpec/main-doc/other-evidence edits.
