# G4 Runtime Isolation and Adapter Validation

- Owner: Codex 3 — credentialless runtime and adapter validation
- OpenSpec tasks: 8.2 and 8.3
- Recorded: 2026-07-18T03:17:52Z
- Status: COMPLETE for the synthetic/reference-only scope authorized for G4
- Baseline: G3 independently accepted; PD-01 preserved

## Evidence scope

| Evidence | Result | Honest boundary |
|---|---|---|
| EV-G4-02 | Claude Code and Codex trusted-gateway contracts passed in-process synthetic HTTP/SSE protocol fixtures for tool calls, reasoning signals, usage, cancellation, correlation and deterministic errors. | No installed CLI, live OmniRoute, live provider or Multica-daemon dispatch was exercised. |
| EV-G4-03 | Sanitized child environments, controlled task homes, a two-level Linux helper-process tree, command lines and redacted diagnostics passed credentialless isolation checks. | Values were synthetic; process inspection was limited to the test-owned helper PIDs and temporary homes. |
| EV-G4-COD | The controlled Codex Responses provider contract, dedicated environment-name lookup, HTTP/SSE path, correlation headers and auth-file exclusion passed. | This is contract and synthetic transport validation, not live Codex/OmniRoute acceptance. |
| EV-G4-ADP | Claude is ready through the trusted Anthropic Messages profile. Kimi and the OpenAI-compatible GLM/NVIDIA surface returned stable fail-closed gates and produced no child environment. | Native Kimi/GLM/NVIDIA paths remain unaccepted; OpenSpec 5.6 and 5.7 stay open. |
| EV-G4-NIM | The native NIM gateway-required contract returned its deterministic fail-closed gate and produced no child environment. | The legacy direct-NVIDIA backend was not invoked or modified; OpenSpec 5.6 stays open. |
| EV-G4-AGY | The native Antigravity contract returned its deterministic fail-closed gate, produced no child environment and retained explicit, non-automatic Claude/Codex fallback candidates. | No native endpoint support is claimed; OpenSpec 5.8 stays open. |

## Synthetic protocol checks

The test gateway binds only an ephemeral loopback listener. Requests use a synthetic stable value and synthetic model IDs. The fixture validates the controlled authorization projection internally but never logs it.

Claude and Codex each demonstrated:

- a request containing a synthetic tool declaration and a streamed tool-call event;
- a protocol-native reasoning/thinking event;
- non-zero input and output usage;
- task, session and request correlation headers;
- client cancellation reaching the server request context;
- repeatable HTTP status, machine error class and body for the same synthetic failure;
- gateway-aware model/thinking admission from an approved in-memory OmniRoute registry snapshot, with no provider catalog callback or credential lookup.

The fixture does not claim full provider protocol acceptance, continuation affinity, rotation, retry/fallback, account lifecycle or production readiness.

## Isolation checks

The inherited fixture included synthetic provider-variable, cookie and direct-endpoint contamination. Environment construction removed it before launch and applied the trusted gateway profile last.

For both Claude and Codex, the checks proved:

- exactly one controlled synthetic OmniRoute value in each observed helper-process environment;
- no provider-native credential variable, cookie variable or direct-provider endpoint in root or leaf helper processes;
- no credential, endpoint or prompt material in helper command lines;
- no auth-file, cookie or credential path in the real temporary task-home tree;
- controlled Codex configuration contained only the gateway provider contract and environment variable name, never a secret value or `auth.json`;
- formatted environment, sanitization events, helper diagnostics and captured stderr contained no synthetic stable/provider/cookie value or direct-provider endpoint.

The process-tree proof used only `/proc` entries for test-owned helper PIDs. It did not inspect daemon, CLI, provider or user credential processes/files.

## Verification

Passed from `multica-auth-work/server` with the pinned Go toolchain:

- `go test ./internal/daemon/runtimeenv ./pkg/agent`
- `go test ./internal/daemon/runtimeenv -run TestG4 -count=10 -timeout=45s`
- `go test -race ./internal/daemon/runtimeenv -run TestG4 -count=1 -timeout=45s`
- `go test -cover ./internal/daemon/runtimeenv` — 79.9% statement coverage
- `go vet ./internal/daemon/runtimeenv ./pkg/agent`
- production runtimeenv static guard: no filesystem/environment discovery API or process-launch API
- ownership guard: the five owned native adapter source files remained unchanged

## Non-claims and stop conditions

No provider-native credential, auth file, cookie, secret source or live endpoint was read, copied, printed, rewritten, rotated, quarantined or mutated. No reset, stash, revert or discard occurred. No live OmniRoute/provider call, daemon dispatch, production/cutover action, Prodex removal, tier activation or native-adapter acceptance occurred. Central files, `execenv`, `models.go`, gateway, deploy and observability were not edited. The pre-existing ledger credential-file STOP warning was neither inspected nor mutated and remains for its owner to adjudicate.
