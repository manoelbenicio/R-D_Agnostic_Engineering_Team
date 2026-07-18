# G3 Independent Security Review

Status: **CHANGES REQUIRED**. Strictly read-only source/evidence review; no credentials or secret files inspected. PD-08 remained absolute.

## 1. HIGH — Gateway CLI arguments override trusted routing after validation

- **Anchors:** `multica-auth-work/server/internal/daemon/daemon.go:3657-3663,3700-3709`; `multica-auth-work/server/internal/daemon/runtimeenv/assert.go:12-20,47-59`; `multica-auth-work/server/pkg/agent/codex.go:87-104,130-138`; `multica-auth-work/server/pkg/agent/claude.go:578-599`; `multica-auth-work/server/pkg/agent/claude_test.go:451-469`; `multica-auth-work/server/pkg/agent/codex_test.go:2027-2035`.
- **Defect:** Gateway pre-launch validation covers environment and generated config, but final daemon/task arguments are appended afterward. Codex permits last-wins provider/base-URL/env-key configuration; Claude permits a later model override. Existing tests preserve these overrides.
- **Exploit boundary:** Requires gateway-required development mode, an injected credential source, and control of daemon-wide or task agent arguments. The ordinary PD-08 command path remains fail-closed without that source.
- **Required fix:** In gateway-required mode, reject custom/default arguments or enforce a strict allowlist that denies routing, provider, credential, settings/home, model, and resume overrides; validate final argv before launch.
- **Re-review:** Prove hostile Codex provider/base-URL/env-key overrides and Claude model/settings/resume overrides fail before credential acquisition or process creation; prove trusted route/model remain final.

## 2. HIGH — Custom runtime executable can receive the gateway credential

- **Anchors:** `multica-auth-work/server/internal/daemon/daemon.go:1007-1018,1082-1165,3225-3239,3636-3640`; `multica-auth-work/server/internal/daemon/brain_integration.go:171-180,237-305`; `multica-auth-work/server/internal/daemon/brain_integration_test.go:93-99`.
- **Defect:** Gateway mode still registers arbitrary workspace custom-runtime executables. A task can replace the built-in Claude/Codex path before admission; admission validates the declared provider/CLI kind, then launches the replacement with the trusted credential-bearing environment. The G3 smoke launches a test helper directly and does not cover daemon executable selection.
- **Exploit boundary:** Requires gateway-required development mode, an injected credential source, and a selectable custom profile declaring `claude` or `codex`. PD-08 currently prevents the ordinary command path from supplying the credential.
- **Required fix:** Suppress custom-runtime registration in gateway-required mode and reject custom runtime tasks before readiness/credential access; resolve executables only from an immutable accepted-CLI registry.
- **Re-review:** Prove custom profiles cannot register or launch in gateway-required mode, cannot reach the credential callback, and built-in Claude/Codex paths still launch through the accepted registry.

## 3. MEDIUM — Final CLI argv can disclose credential values or auth paths in logs

- **Anchors:** `multica-auth-work/server/pkg/agent/claude.go:35-61`; `multica-auth-work/server/pkg/agent/codex.go:518-554`; `.planning/agent-brain-v3/evidence/g3-serial-integration.md:28-32`.
- **Defect:** Both adapters log complete final argv. Because custom/default arguments survive, inline authentication configuration, endpoint values, or auth-file paths can enter daemon logs, contradicting the G3 safe-diagnostics claim.
- **Exploit boundary:** Requires control of CLI arguments and access to daemon logs; it does not require reading an existing credential or secret file.
- **Required fix:** Log only an allowlisted command shape or flag names with values redacted in gateway-required mode; never log routing/auth values or paths.
- **Re-review:** Capture gateway-mode logs from synthetic hostile arguments and prove no credential value, endpoint value, auth path, or unredacted config payload appears while safe command diagnostics remain.

## Independent re-review gate

- All three fixes have focused regression tests covering the final daemon launch path, not only environment/config helpers.
- The full applicable suite, race tests, vet, credential-isolation harness, and synthetic G3 smoke pass without weakening PD-08.
- No provider-native credential/auth-file/direct-endpoint flow, unsupported adapter, or raw secret/path logging remains reachable in gateway-required mode.
