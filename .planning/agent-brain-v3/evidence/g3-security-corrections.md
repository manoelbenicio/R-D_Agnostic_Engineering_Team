# G3 security corrections

Status: implemented and locally verified for the default-off development slice. G3 is not re-accepted; independent reviewer pane pB must re-review these corrections.

## EV-G3-SEC-ARGS — untrusted argument rejection

- Gateway-required mode has no custom-argument allowlist. It rejects every daemon-level and task-level custom argument before Agent Brain admission, gateway credential acquisition, execution-environment preparation, backend construction, or process launch.
- Configuration loading rejects non-empty daemon argument settings in gateway-required mode without including argument contents in the error.
- The task-time gate repeats the check for programmatic callers that bypass configuration loading.
- Table-driven regressions cover attempted config, model, base-URL, and daemon model overrides. Each proves the synthetic credential callback remains unused and a synthetic executable launch marker is absent.

Implementation: `multica-auth-work/server/internal/daemon/config.go`, `multica-auth-work/server/internal/daemon/daemon.go`, and `multica-auth-work/server/internal/daemon/brain_integration_test.go`.

## EV-G3-SEC-RUNTIME — immutable built-in runtime selection

- Gateway-required configuration maps `CLIKind` only to the canonical built-in `claude` or `codex` command and stores the resolved absolute path. `MULTICA_*_PATH` and per-machine workspace-profile command overrides do not participate in that mapping.
- Workspace custom runtime profiles are neither fetched/registered during initial registration nor refreshed during workspace synchronization in gateway-required mode.
- Task-time selection rejects every runtime with a workspace profile ID before Agent Brain admission or gateway credential acquisition, rejects a claimed provider that disagrees with the frozen `CLIKind`, and selects the executable only from the built-in provider entry.
- Regressions prove profile registration/refresh suppression, canonical resolution ignoring an untrusted path override, and rejection of a synthetic custom executable before credential access or process launch.

Implementation: `multica-auth-work/server/internal/daemon/config.go`, `multica-auth-work/server/internal/daemon/daemon.go`, and `multica-auth-work/server/internal/daemon/brain_integration_test.go`.

## Verification

- Focused four-regression security test: pass.
- Focused Agent Brain and credentialless Codex tests: pass.
- Focused security race test: pass.
- Focused daemon/execenv/brain/gateway/runtimeenv/deploy/observability/command package matrix: pass.
- Full Go suite: pass.
- Full Go vet: pass.
- Existing synthetic credential-isolation harness: pass.
- Formatting and `git diff --check`: pass.

Go verification used a one-shot Go 1.26 build container. Source was mounted read-only for tests/vet; the focused and vet runs were network-disabled. The full suite downloaded missing public Go modules into the existing build cache, then passed. No runtime/provider service was contacted.

## Safety and limitations

- PD-01 dirty baseline was preserved. No reset, stash, revert, discard, deletion, or unrelated rewrite occurred.
- PD-08 remained active. No credential/auth/secret file was read, copied, rewritten, rotated, quarantined, or mutated; only synthetic/reference-only test values were used.
- No gateway/runtimeenv/deploy/observability or `pkg/agent` adapter file was edited.
- No live OmniRoute/provider call, Multica task dispatch, production action, cutover, Prodex removal, or capacity-tier change occurred.
- This evidence establishes local correction coverage only. It does not replace independent security review or authorize G3 re-acceptance.
