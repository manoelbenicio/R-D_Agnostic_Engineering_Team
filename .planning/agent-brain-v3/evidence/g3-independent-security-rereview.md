# REVIEW-G3-02 — Independent Security Re-review

Overall result: **ACCEPT** for the three original findings. Review was source/evidence read-only; only this artifact was written. No credential, auth, or secret file was inspected. PD-08 remained absolute.

## 1. Hostile CustomArgs before credential/process — ACCEPT

- **Implementation:** `multica-auth-work/server/internal/daemon/config.go:488-503` rejects daemon arguments during gateway-required configuration. `multica-auth-work/server/internal/daemon/daemon.go:3233-3257` rejects daemon/task arguments in the programmatic task-time path; `daemon.go:3275-3305` invokes that gate before Agent Brain admission and its credential source.
- **Test:** `multica-auth-work/server/internal/daemon/brain_integration_test.go:275-304` covers hostile config, config-path, model, base-URL, and daemon overrides; every case asserts `custom_args_not_allowed`, zero credential-callback calls, and no synthetic executable marker.
- **Decision:** Trusted route/config cannot be overridden by CustomArgs in gateway-required mode before credential acquisition or process creation.

## 2. Custom workspace runtime registration/selection/launch — ACCEPT

- **Implementation:** `multica-auth-work/server/internal/daemon/daemon.go:1100-1105` suppresses profile fetch/registration; `daemon.go:1461-1464` suppresses profile refresh; `daemon.go:3233-3257` rejects indexed profile runtimes before admission and selects only the frozen provider entry. Canonical built-in resolution ignores provider-path/profile overrides at `multica-auth-work/server/internal/daemon/config.go:132-175,325-333,477-483`.
- **Tests:** `multica-auth-work/server/internal/daemon/brain_integration_test.go:306-347` proves custom-runtime rejection with zero credential-callback calls and no launch marker, registration/refresh suppression, and canonical resolution despite an untrusted path override.
- **Decision:** Workspace custom runtimes cannot register, select, or launch in gateway-required mode; their credential callback path is unreachable.

## 3. Claude/Codex hostile argv log redaction — ACCEPT

- **Implementation:** `multica-auth-work/server/pkg/agent/claude.go:24-135` projects executable basename and redacted argv; Claude and Codex use it at `claude.go:173-175` and `multica-auth-work/server/pkg/agent/codex.go:547-554`. No raw adapter `agent command` argv log remains.
- **Tests:** `multica-auth-work/server/pkg/agent/claude_test.go:866-920` and `codex_test.go:1878-1928` prove synthetic model/provider/resume/config/settings/home/base-URL/auth/header/key/env/prompt values and executable paths are absent while safe diagnostics remain.
- **Decision:** The reviewed Claude/Codex command logs redact the synthetic hostile argv values and paths covered by the finding.

## Independent verification

Executed in a one-shot Go 1.26 container with the source mounted read-only and ephemeral caches:

`go test ./internal/daemon ./pkg/agent -run 'TestAgentBrain(RejectsAllCustomArgsBeforeCredentialOrLaunch|RejectsCustomRuntimeBeforeCredentialOrLaunch|SuppressesWorkspaceRuntimeProfiles|BuiltInResolutionIgnoresCommandPathOverride)|Test(Claude|Codex)CommandLoggingRedactsSensitiveArgv' -count=1`

Result: **PASS** — `internal/daemon` and `pkg/agent`. Only synthetic values and local test executables were used; no provider/runtime service was contacted.
