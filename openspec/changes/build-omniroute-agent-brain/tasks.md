## 0. Governance and GSD Rebaseline

- [ ] 0.1 [Product owner] Approve the OpenSpec/GSD source hierarchy, G0–G8 roadmap, total ETA range and preservation of RPP/Prodex v2.1 as historical evidence.
- [x] 0.2 [Product owner] Confirm Kiro/Principal as the `.planning/` author or explicitly change the current GSD authorship rule before any Codex agent writes GSD artifacts.
- [x] 0.3 [Kiro/Principal or newly authorized planning owner] Create the Agent Brain v3 GSD `PROJECT`, `REQUIREMENTS`, `ROADMAP`, `STATE`, `DECISIONS`, `RISKS` and phase plans from the approved OpenSpec artifacts.
- [x] 0.4 [Planning owner] Create bidirectional `TRACEABILITY`, `COMPONENT_REGISTER`, `INTERFACE_REGISTER`, `REMOVAL_REGISTER`, `FILE_OWNERSHIP` and `EVIDENCE_INDEX` documents.
- [x] 0.5 [Planning owner] Assign a formal disposition to every active/historical OpenSpec change and block concurrent execution of superseded Prodex/router plans.
- [x] 0.6 [Codex 1 + planning owner] Audit for orphan requirements, scenarios, components, interfaces, tasks, owners, evidence and removal decisions; close every orphan before Wave 0.
- [ ] 0.7 [Product owner] Approve the GSD v3 baseline and only then mark implementation authorization for Waves 0–3.

## 1. Architecture and Supplier Gate

- [x] 1.1 [Codex 1] Freeze the working `Agent Brain` terminology, cold/hot ownership boundary, `CLIKind`/`RouteModel`/`RouterOwner` contract, and compatibility-surface inventory.
- [x] 1.2 [Codex 1] Record the existing dirty worktree and assign exclusive ownership for `daemon.go`, `config.go`, `health.go`, command entrypoints, `execenv`, CLI adapters, gateway modules, and deployment files.
- [ ] 1.3 [Codex 4] Obtain a completed OmniRoute architecture checklist with version/image digest and evidence for every protocol, rotation, expiry, quota, 429, streaming, security, observability, and capacity item.
- [ ] 1.4 [Codex 4] Obtain a completed Prodex feature-parity matrix, including SC01-SC10 Smart Context, reset/redeem, special crate surfaces, approved waivers, owners, restrictions, and dates.
- [x] 1.5 [Codex 4] Complete the per-model route matrix for Claude, Codex/OpenAI, Kimi, GLM, NVIDIA and Antigravity with exact API format, account pool, tools, reasoning, context, rotation/affinity and fallback.
- [x] 1.6 [Codex 1] Approve the initial supported model set, fallback chains, stable-key scope, runtime endpoint topology, capacity target, and cutover blockers.

## 2. Wave 0 — Contract and Merge Boundary Freeze

- [x] 2.1 [Codex 1] Define neutral task/gateway/runtime interfaces and data types in new files without changing the active daemon path.
- [x] 2.2 [Codex 1] Define neutral configuration names, legacy alias precedence, gateway-required mode, secret-file reference, readiness policy, and 20/50/100 task-tier schema.
- [x] 2.3 [Codex 1] Define the compatibility facade for the current daemon API, task token, runtime router-owner value, environment variables, stored configuration, CLI command, and runtime brief.
- [x] 2.4 [Codex 1] Publish the frozen interfaces and file ownership to Codex 2–4; require all shared-entrypoint changes to be integrated by Codex 1 only.

## 3. Wave 1A — Neutral Brain Foundation (Codex 1)

- [x] 3.1 Create the neutral coordinator/task-executor/runtime-registry package around existing lifecycle interfaces without moving provider credential logic.
- [x] 3.2 Add neutral task fields for `CLIKind`, `RouteModel`, `RouterOwner`, task/session/request correlation and approved route policy.
- [x] 3.3 Add compatibility translations from supported legacy task/config fields into the neutral contract and emit measurable legacy-use events.
- [x] 3.4 Add gateway-required admission/readiness states and fail-closed task statuses without enabling the new execution path yet.
- [x] 3.5 Preserve current workspace, repository/worktree, cancellation, watchdog, context/skills, stream batching, recovery and terminal-result behavior behind the neutral interfaces.

## 4. Wave 1B — OmniRoute Gateway Package (Codex 2)

- [x] 4.1 Create an OmniRoute client with redacted authentication, configurable host/container base URL, bounded timeouts, cancellation and request/session correlation.
- [x] 4.2 Implement separate liveness/readiness checks and authenticated `/v1/models` retrieval with deterministic error classification.
- [x] 4.3 Implement and cache the versioned model/capability registry with explicit protocol, tools, reasoning, streaming, context and structured-output validation.
- [x] 4.4 Define trusted runtime profiles for Anthropic Messages, OpenAI Responses, OpenAI Chat and the documented Antigravity-compatible route.
- [x] 4.5 Define route-policy types for strict independent-request round-robin, continuation affinity, retry deadline, same-model fallback, approved cross-model fallback, circuit behavior and Smart Context flags.
- [x] 4.6 Parse safe OmniRoute telemetry headers/events for actual model/route, pseudonymous connection, retries, fallback, quota/circuit state and usage without content or secrets.
- [x] 4.7 Add protocol fixtures/contracts for Anthropic Messages/SSE, Responses/SSE and Chat Completions/SSE using synthetic credentials and content only.

## 5. Wave 1C — Credentialless Runtime and CLI Adapters (Codex 3)

- [x] 5.1 Implement a minimal inherited-environment builder that removes provider keys, OAuth/cookie variables, direct-provider base URLs and unsafe gateway overrides.
- [x] 5.2 Expand custom-environment validation to deny provider credentials and routing/auth variables in gateway-required mode, then apply trusted gateway configuration last.
- [x] 5.3 Remove provider-auth copying from per-task homes while preserving sandbox, config, skills, session and workspace isolation.
- [x] 5.4 Generate a controlled Codex custom-provider configuration for OmniRoute Responses API, stable-key environment lookup, HTTP/SSE transport and correlation headers without `auth.json`.
- [x] 5.5 Implement Claude Code's trusted OmniRoute root URL/token environment and ensure internal Claude markers do not leak or override gateway policy.
- [ ] 5.6 Implement the approved OpenAI-compatible Kimi/GLM/NVIDIA adapter and convert or replace the native NIM direct-NVIDIA credential path.
  - G2C no-secret contract delivered as `EV-G2C-06`; execution remains fail-closed pending an accepted adapter contract.
- [ ] 5.7 Implement the accepted Kimi provider-registry path or the documented Claude/Codex frontend fallback for Kimi routes.
  - G2C no-secret contract delivered as `EV-G2C-07`; native registry execution remains fail-closed pending acceptance evidence.
- [ ] 5.8 Implement native Agy endpoint configuration only if the installed version proves support; otherwise enforce the Claude/Codex `agy/...` model fallback and disable the direct native path.
  - G2C no-secret contract delivered as `EV-G2C-08`; native execution remains fail-closed and fallback selection is never automatic.
- [x] 5.9 Make model/thinking validation gateway-aware so approved OmniRoute IDs are accepted without provider-native catalog or credential lookup.
- [x] 5.10 Add a pre-launch assertion that the child environment/config contains only the stable OmniRoute secret and approved local task data.

## 6. Wave 1D — Deployment, Security, Observability and Capacity Assets (Codex 4)

- [x] 6.1 Define a Linux permission-restricted OmniRoute service secret derived operationally from the existing host source without copying its value into the repository, image, logs or screenshots.
- [x] 6.2 Add host/WSL and future container endpoint configuration, reachability prerequisites and service start/recreate instructions without hard-coding Docker DNS for the host daemon.
- [x] 6.3 Define structured redacted events/metrics and correlation for admission, gateway readiness, selection, affinity, refresh, quota, 401/403, 429/circuit, retry/fallback, cancellation, usage and overload.
- [x] 6.4 Define dashboards and alerts for no eligible accounts, auth refresh failure, 401/403/429/5xx spikes, circuit state, queue growth, resource pressure, latency and error SLOs.
- [x] 6.5 Build a synthetic capacity/failure acceptance harness specification for 20/50/100 tasks, protocol mix, streaming, tools, prompt/output sizes, cancellation and account distribution.
- [x] 6.6 Define backup/restore, account/route hot-change, key rotation, upgrade, rollback, incident classification and escalation runbooks.
- [x] 6.7 Define staged feature flags, canary cohorts, rollback triggers and evidence locations for every protocol/provider/capacity gate.

## 7. Wave 2 — Sole-Owner Core Integration (Codex 1)

- [x] 7.1 Review and integrate the gateway, runtime/CLI and operations streams against the frozen contracts; reject hidden provider credentials or duplicate routing decisions.
- [x] 7.2 Wire neutral/gateway configuration and aliases through the central daemon/config/command entrypoints as their sole editor.
- [x] 7.3 Replace task-time credential-account resolution with `CLIKind` plus `RouteModel` and trusted gateway profile resolution in gateway-required mode.
- [x] 7.4 Apply sanitized runtime environment and controlled per-CLI configuration after custom settings and before process launch.
- [x] 7.5 Gate admission/launch on OmniRoute readiness, stable-key authentication and selected model/protocol capability.
- [x] 7.6 Set `RouterOwner=omniroute` and emit common task/session/request correlation through launch, result, error and cancellation paths.
- [x] 7.7 Disable Prodex/L2 startup, legacy Go rotation/retry, provider credential-home preparation and native provider-account selection for gateway-required tasks.
- [x] 7.8 Keep legacy behavior isolated behind an explicit default-off migration flag only while legacy tasks drain; prevent any task from having two router owners.
- [x] 7.9 Expose neutral health/readiness/config diagnostics with secrets and raw content redacted.
- [x] 7.10 Produce a first runnable vertical slice using one approved Claude or Codex model route without enabling broad production admission.

## 8. Wave 3 — Protocol, Security and Failure Acceptance

- [ ] 8.1 [Codex 2 + Codex 4] Verify authenticated models/capabilities and one non-streaming and streaming completion for every approved protocol family and model route.
- [ ] 8.2 [Codex 3 + Codex 4] Verify Claude, Codex, Kimi, GLM/NVIDIA and Antigravity accepted paths with tools, reasoning, cancellation, usage and deterministic errors.
  - G4 acceptance is synthetic/reference-only: Claude and Codex trusted-gateway paths passed; Kimi/GLM/NVIDIA/NIM/Agy remained deterministic fail-closed contracts. Native tasks 5.6–5.8 remain open.
- [x] 8.3 [Codex 3] Verify child environments, task homes, process trees, logs and diagnostics contain no provider-native credentials, auth files or direct-provider endpoints.
  - EV-G4-03 covers synthetic child environments, real temporary controlled homes, a two-level Linux helper-process tree and redacted diagnostics; it does not claim live daemon/CLI/provider isolation.
- [ ] 8.4 [Codex 2 + Codex 4] Demonstrate strict concurrent round-robin for independent requests and correct affinity for Responses continuation, prompt cache and tool turns.
- [ ] 8.5 [Codex 2 + Codex 4] Demonstrate expired access token, revoked refresh token, quota exhaustion, 401, 403, account-scoped 429, provider-global 429, 5xx, timeout and malformed upstream handling.
- [ ] 8.6 [Codex 2 + Codex 4] Demonstrate safe retry before first output, no replay after partial output/tool action, request deduplication and prompt cancellation slot release.
- [ ] 8.7 [Codex 2 + Codex 4] Demonstrate account add/remove/quarantine/re-entry and OmniRoute restart/config rollback during active load.
- [x] 8.8 [Codex 4] Record evidence against every OmniRoute checklist and Prodex parity ID; stop the cutover for unsupported blocker rows without an approved waiver.

## 8-OBS. Wave 3-OBS — End-to-End Observability Stop-Gate (G4-OBS)

> BLOCKING GATE (owner decision D-V3-17). Metadata-only across all eight hops; independent
> reviewer ≠ producer ≠ adjudicator. G4-OBS must PASS before any capacity tier (§9) or cutover
> (§10). No secrets, prompts, tool payloads, repository content, opaque reasoning, cookies, keys,
> account emails, or connection strings in any span/label/log. Owning lanes per FILE_OWNERSHIP
> (W1–W8). Traces to AB-REQ-39/40/41 and the `end-to-end-observability` spec.

- [ ] OBS-1 [W5] Freeze the eight-hop correlation ID schema and propagation contract (metadata-only): `request_id`, `queue_msg_id`, `task_id`, `session_id`, `launch_id`, `proc_id`, `omni_request_id`, `result_id`, `delivery_id`; join keys, header/metadata carriers, `contract_version`, and the `secrets_present=false` invariant. Evidence `EV-OBS-01`.
- [ ] OBS-2 [W6] Emit the ingress-API span (hop 1): control-API request received with method/route, pseudonymous principal, status, latency; `request_id → task_id`; no request/response bodies. Evidence `EV-OBS-02`.
- [ ] OBS-3 [W7] Emit the DB-queue span (hop 2): enqueue/dequeue timestamps, queue depth, wait time; `queue_msg_id ↔ task_id`; no task payload content. Evidence `EV-OBS-03`.
- [ ] OBS-4 [W1] Emit the daemon admission/lifecycle span (hop 3): admission decision, readiness-gate result, `CLIKind`/`RouteModel` labels, fail-closed classification; `task_id`/`session_id`/`launch_id`. Evidence `EV-OBS-04`.
- [ ] OBS-5 [W3] Emit the CLI-process span (hop 4): launch/exit, exit code, cancellation, structurally-redacted argv shape (never values); `launch_id`/`proc_id`; reuses R21 structural argv redaction. Evidence `EV-OBS-05`.
- [ ] OBS-6 [W2] Emit the OmniRoute/provider span (hop 5) from safe telemetry only (extends task 4.6): actual route/model, pseudonymous account/connection, selection reason, retries/fallback, quota/circuit state, safe usage; `request_id ↔ omni_request_id`. Evidence `EV-OBS-06`.
- [ ] OBS-7 [W7] Emit the terminal-persistence span (hop 6): persist latency, byte/token counts, terminal status; `task_id`/`result_id`; no result content. Evidence `EV-OBS-07`.
- [ ] OBS-8 [W6] Emit the WS/UI-delivery span (hop 7): delivery latency, backpressure/drops, reconnects; `session_id`/`delivery_id`; no delivered payload content. Evidence `EV-OBS-08`.
- [ ] OBS-9 [W5] Assemble the end-to-end trace (hop 8): join every hop on correlation IDs, detect gaps/orphans, and prove one continuous trace per synthetic task across all eight hops. Evidence `EV-OBS-09`.
- [ ] OBS-10 [W5 + W4] Run the structural (not pattern-only) secret/content leakage scan across every span, label and log for all hops and prove leak-clean; any leak = STOP. Evidence `EV-OBS-10`.
- [ ] OBS-11 [W4] Deliver dashboards + alerts (per-hop latency, error classification, drop/gap) and the consolidated G4-OBS acceptance bundle with independent-review sign-off; declare G4-OBS PASS only when OBS-1..OBS-10 are each accepted, OBS-9 shows a continuous trace for every synthetic task, and OBS-10 is clean. Evidence `EV-OBS-11`.

## 9. Wave 4 — Capacity Tiers and Operational Readiness

> Entry gate: G4-OBS (OBS-1..OBS-11) PASS is mandatory before 9.1 runs (D-V3-17).

- [ ] 9.1 [Codex 4] Run and report the approved 20-task model/stream/tool profile with latency, errors, queue, selection fairness, CPU, memory, sockets, retries and fallback.
- [ ] 9.2 [Codex 1] Enable the 20-task tier only if its acceptance thresholds pass and admission/overload/cancellation counters reconcile.
- [ ] 9.3 [Codex 4] Run the equivalent 50-task sustained and recovery profile; document required account/route/host tuning.
- [ ] 9.4 [Codex 1] Enable the 50-task tier only after evidence passes; otherwise enforce 20 and record remediation.
- [ ] 9.5 [Codex 4] Run the equivalent 100-task sustained and recovery profile with bounded overload beyond the tier.
- [ ] 9.6 [Codex 1] Enable the 100-task tier only after evidence passes; otherwise enforce the highest proven tier without changing round-robin semantics.
- [ ] 9.7 [Codex 4] Complete dashboards, alerts, backup/restore, secret rotation, incident, upgrade/rollback and named-owner operational sign-off.

## 10. Wave 5 — Default Cutover and Legacy Removal

- [ ] 10.1 [Codex 1] Enable gateway-required mode by default for new tasks after protocol, security, failure and launch-capacity gates pass.
- [ ] 10.2 [Codex 4] Observe controlled/default development cohorts for the approved period and confirm no direct-provider traffic, dual router ownership or secret-policy violation; do not claim production readiness.
- [ ] 10.3 [Codex 1] Drain or stop legacy tasks and remove the temporary legacy execution flag after rollback no longer requires it.
- [ ] 10.4 [Codex 1 / W1] Reconcile Prodex/L2 to a default-OFF, mutually-exclusive, operator-gated **cold platform recovery mode** in the final Kanban lane (do NOT delete): keep startup/facade/profile/filesystem code present but disabled by default; guarantee it is never per-request, never automatic, and never simultaneously hot with OmniRoute; wire it to the platform recovery-mode state machine (AB-REQ-41) at the single runtime-authority select point. Deletion is explicitly out of scope per D-V3-16.
- [ ] 10.5 [Codex 1] Delete legacy Go rotation state/retry/account-selection code and provider-auth home preparation after zero-use evidence.
- [ ] 10.6 [Codex 3] Delete obsolete provider credential copy/restore implementations and direct NIM/provider-key paths; retain only credentialless state isolation.
- [ ] 10.7 [Codex 4] Reconcile documentation, deployment, threat model, runbooks and evidence after removal and prove rollback uses only accepted Agent Brain/OmniRoute versions.

## 11. Wave 6 — Complete Debranding

- [ ] 11.1 [Codex 1] Inventory every remaining Multica/Prodex name by public API, CLI, environment, path, package/type, storage, metric/log and user-visible surface.
- [ ] 11.2 [Codex 1] Introduce final product names for the binary, module/package boundaries, configuration and new contracts without breaking active compatibility consumers.
- [ ] 11.3 [Codex 2] Rename remaining gateway-facing legacy identifiers and router-owner values while retaining read compatibility for stored historical records.
- [ ] 11.4 [Codex 3] Rename task-home/runtime-brief/CLI-facing paths and variables with deterministic migration of required local state.
- [ ] 11.5 [Codex 4] Migrate deployment, dashboards, alerts, documentation, runbooks and operator procedures to final names.
- [ ] 11.6 [Codex 1] Remove each compatibility alias only after telemetry shows zero use, its consumer migration is complete, and the removal is included in release notes.
- [ ] 11.7 [Codex 1 + Codex 4] Complete final architecture, parity, security, capacity and operational sign-off with no active Multica/Prodex runtime dependency.
