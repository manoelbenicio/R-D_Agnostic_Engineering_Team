# prodex Fork Map

Status: TARGET-MILESTONE MAP. Does not block F0 prodex-as-is rollout.

Pinned source: `github.com/christiandoxa/prodex` tag `0.246.0`, commit
`7750da9b6a5c91a6d429e18e6a4d422cab4bc144`, Apache-2.0. Do not use README as
evidence for this map.

Local architecture truth: `openspec/changes/rotation-parity-polyglot/design.md`
sections 2, 3, 6, and 7, plus
`docs/rotation-parity-polyglot/02_ADR-001-arquitetura.md`.

## Claim Labels

- `verified`: directly backed by pinned prodex docs/repo or local ADR/OpenSpec.
- `inferred`: fork recommendation derived from verified source and local ADR.
- `not-validated`: present as code/docs or desired behavior, but not validated by
  this audit against live accounts, load, or production traffic.

## Primary Source Ledger

- `S1` verified: root `Cargo.toml` declares package `prodex` version `0.246.0`,
  Apache-2.0, repository URL, and the workspace members.
- `S2` verified: `docs/architecture.md` maps command flow, runtime proxy hot
  path, state/persistence, and boundary guard rules.
- `S3` verified: `docs/runtime-policy.md` documents gateway, state backends,
  adaptive routing shadow defaults, runtime proxy contract, and tuning keys.
- `S4` verified: `docs/state-model.md` documents `$PRODEX_HOME/state.json`,
  profile auth isolation, runtime bindings, quota/health, logs, and invariants.
- `S5` verified: `docs/provider-conformance.md` documents the current split
  between provider core and app-side runtime translation, plus the v1 target.
- `S6` verified: `docs/provider-capabilities.md` is generated from provider
  contracts, fixtures, and model catalog and records endpoint capability status.
- `S7` verified: `docs/smart-context.md` documents Smart Context safety,
  budgeting, rollout, telemetry, replay benchmark, and remaining risks.
- `S8` verified: `docs/deployment.md` documents gateway compose deployment,
  persistent paths, file/SQLite/Postgres/Redis state, and single-node limits.
- `S9` verified: `crates/prodex-app/src/runtime_proxy/*` owns live runtime proxy
  orchestration; `crates/prodex-runtime-proxy/*` owns side-effect-free hot-path
  helpers; `crates/prodex-app/src/app_commands/redeem.rs` and
  `crates/prodex-app/src/runtime_proxy/quota/auto_redeem.rs` own redeem paths.

## Crate Map

`verified` from `S1` and `S2`: prodex is one root package plus focused crates.
The table below enumerates every crate under pinned
`/home/dataops-lab/runtime/prodex-src/crates`; no row is an aggregate.

| Crate | Function | Plan Area | Fork Zone | Status |
|---|---|---|---|---|
| `prodex-app` | Application orchestration, command handlers, live runtime proxy wiring, gateway/redeem commands. | P0/P2/P3 | infra | Verified; remain upstream for app shell, wrap runtime modules only. |
| `prodex-app-reports` | App command report helpers. | P0/P8 | infra | Verified; remain upstream. |
| `prodex-audit-log` | Audit log helpers for runtime/security events. | P4/P8 | security | Verified; preserve and map to Multica audit taxonomy. |
| `prodex-bench-support` | Benchmark and check harness helpers. | P6 | infra | Verified; remain upstream as QA support. |
| `prodex-caveman-assets` | Embedded Caveman plugin assets. | P2/P4 | plugin | Gap crate; REQ-32, keep disabled or explicitly scoped before fork use. |
| `prodex-cli` | CLI argument model and parsing helpers. | P0/P3 | infra | Verified; remain upstream for prodex-as-is launch surface. |
| `prodex-codex-config` | Codex config parsing bridge. | P0/P2/P3 | core | Verified; stable config interface point for fork boundary. |
| `prodex-context` | Smart Context audit/compression helpers. | P2/P6 | runtime | Verified; L2 fork candidate for Smart Context policy and validation. |
| `prodex-core` | Core paths and shared primitives. | P0/P2 | core | Verified; preserve as foundation and path authority. |
| `prodex-housekeeping` | Housekeeping helpers. | P0/P8 | infra | Verified; remain upstream. |
| `prodex-mcp-stdio` | Shared stdio framing helpers for prodex MCP servers. | P1/P2/P6 | plugin | Verified MCP crate; map framing contract into conformance tests. |
| `prodex-memory` | Local and Mem0-compatible memory MCP backend. | P2/P4/P6 | plugin | Gap crate; REQ-27/REQ-37b privacy, retention, and redaction decision required. |
| `prodex-presidio` | Presidio runtime configuration helpers. | P4/P6 | security | Security/gap crate; REQ-28 redaction engine integration and tests. |
| `prodex-profile-export` | Profile export envelope crypto helpers. | P2/P4 | security | Security-sensitive; preserve crypto envelope and audit export flows. |
| `prodex-profile-identity` | Profile identity parsing helpers. | P2/P4 | security | Verified; preserve profile identity isolation. |
| `prodex-provider-core` | Provider catalog, adapter contracts, and cost helpers. | P5/P6 | provider | Verified; remain upstream, promote conformance contract before fork gate. |
| `prodex-proxy-config` | Upstream proxy and client configuration helpers. | P2/P3 | runtime | Verified runtime proxy boundary config; L2 sidecar interface input. |
| `prodex-quota` | Pure quota models and rendering helpers. | P2/P5/P9 | runtime | Gap crate; REQ-31, preserve pre-commit quota classification. |
| `prodex-redaction` | Redaction helpers for logs and diagnostics. | P4/P6/P8 | security | Security/gap crate; REQ-28, required for scrubbed evidence/events. |
| `prodex-runtime-anthropic` | Anthropic compatibility translation helpers, including MCP/tool translation surfaces. | P1/P5/P6 | provider | MCP/provider runtime translation; remain upstream with conformance fixtures. |
| `prodex-runtime-broker` | Runtime broker registry, health, and metrics DTOs. | P1/P3/P8 | runtime | Gap crate; REQ-29, L2 fork candidate for health/route/event DTOs. |
| `prodex-runtime-broker-log` | Runtime broker log parsing and cached continuity metrics. | P4/P8 | runtime | Verified; preserve diagnostics-only status and avoid routing from logs. |
| `prodex-runtime-capabilities` | Runtime request compatibility surface detection. | P5/P6 | provider | Verified; remain upstream, gate provider capability claims. |
| `prodex-runtime-claude` | Claude Code runtime launch configuration helpers. | P3/P5 | provider | Verified; remain upstream as provider/runtime launch adapter. |
| `prodex-runtime-cookies` | Runtime proxy cookie relay helpers. | P2/P4 | security | Security/gap crate; REQ-30 auth/session relay audit required. |
| `prodex-runtime-doctor` | Runtime doctor log parsing and diagnostics. | P6/P8 | infra | Verified; remain upstream for diagnostics. |
| `prodex-runtime-gemini` | Side-effect-free Gemini runtime metadata. | P5/P6 | provider | MCP-compatible provider runtime; remain upstream with conformance fixtures. |
| `prodex-runtime-gemini-cli-compat` | Gemini CLI compatibility projection helpers for runtime homes. | P5/P6 | provider | Verified; remain upstream, validate compat behavior. |
| `prodex-runtime-launch` | Runtime launch planning primitives. | P2/P3 | runtime | Verified gateway/launch boundary; L2 fork candidate for sidecar lifecycle. |
| `prodex-runtime-log` | Runtime log path and marker helpers. | P4/P8 | infra | Verified; preserve as diagnostics-only, redact before evidence. |
| `prodex-runtime-metrics` | Runtime metrics rendering. | P8 | infra | Verified; map to Multica observability. |
| `prodex-runtime-policy` | Runtime policy parsing and validation. | P1/P3/P4 | runtime | Verified; L2 fork candidate for `ApplyPolicy` ingestion. |
| `prodex-runtime-proxy` | Runtime proxy boundary primitives and hot-path helpers. | P2/P3/P6 | runtime | Verified; primary L2 fork nucleus, preserve hard affinity and pre-commit routing. |
| `prodex-runtime-quota` | Runtime quota adapter helpers. | P2/P5/P9 | runtime | Gap crate; REQ-31, map quota adapters by provider. |
| `prodex-runtime-state` | Runtime state data structures. | P2/P4 | runtime | Verified; L2 fork candidate, preserve continuation bindings. |
| `prodex-runtime-store` | Runtime store merge and compaction helpers. | P2/P4 | runtime | Verified; L2 fork candidate, keep merge-safe and non-blocking hot path. |
| `prodex-runtime-tuning` | Runtime tuning override and snapshot helpers. | P2/P3 | runtime | Verified; L2 fork candidate for policy/tuning snapshots. |
| `prodex-secret-store` | Secret storage primitives. | P4 | security | Security-sensitive; preserve secret boundary and no-log rule. |
| `prodex-session-store` | Codex session metadata discovery helpers. | P1/P2/P3 | core | Verified; redeem/session affinity intersection, preserve continuation metadata. |
| `prodex-shared-codex-fs` | Shared Codex home file operations. | P2/P4 | security | Verified; preserve profile auth isolation under `$PRODEX_HOME/profiles/<name>`. |
| `prodex-shared-types` | Shared internal data types. | P1/P2 | core | Verified; stable interface point across fork boundary. |
| `prodex-state` | State models and merge helpers. | P2/P4 | runtime | Verified; L2 fork candidate where shared state externalizes to Postgres/Redis. |
| `prodex-terminal-ui` | Terminal rendering helpers. | P0 | ui | Verified; remain upstream, outside runtime decision boundary. |
| `prodex-update-notice` | Update notice and version check helpers. | P0/P8 | infra | Verified; remain upstream, disable network surprise if policy requires. |

## Runtime Boundary Isolation

- Runtime proxy: `prodex-runtime-proxy` and `prodex-proxy-config`, with live
  orchestration currently in `prodex-app/src/runtime_proxy/*`.
- Gateway: `prodex-runtime-launch`, `prodex-proxy-config`, and app gateway
  commands/launch modules.
- Smart Context: `prodex-context`, app `runtime_proxy/smart_context/*`, and
  `prodex-runtime-proxy/src/smart_context/*`.
- State: `prodex-runtime-state`, `prodex-state`, `prodex-runtime-store`, plus
  app runtime persistence.
- Redeem: implicit in `prodex-session-store` session metadata and `prodex-cli`
  command surface, with concrete runtime paths in app redeem and quota modules.

## MCP Contract Mapping

- `prodex-mcp-stdio`: MCP stdio framing; treat as the framing contract input for
  P1/P6 conformance.
- `prodex-runtime-anthropic`: provider translation surface that carries MCP tool
  translation concerns into Anthropic-compatible requests.
- `prodex-runtime-gemini`: provider runtime metadata and compatibility layer for
  Gemini; validate MCP-compatible behavior through P5/P6 provider conformance.
- `prodex-runtime-gemini-cli-compat`: runtime-home compatibility projection for
  Gemini CLI behavior; keep upstream and fixture-backed.

## Security-Sensitive Crates

- `prodex-secret-store`: secret storage primitives; never log secret material.
- `prodex-profile-export`: crypto export envelope; audit import/export paths.
- `prodex-presidio`: PII/redaction runtime configuration; required by REQ-28.
- `prodex-redaction`: log/diagnostic redaction; required for evidence and events.
- `prodex-runtime-cookies`: auth/session cookie relay; REQ-30 audit surface.

## Gap Crates And REQ Links

- `prodex-memory`: Mem0-compatible memory MCP backend; REQ-27/REQ-37b.
- `prodex-presidio` and `prodex-redaction`: native redaction; REQ-28.
- `prodex-runtime-broker` and `prodex-runtime-broker-log`: health, registry,
  metrics, and continuity event surface; REQ-29.
- `prodex-runtime-cookies`: auth/session relay; REQ-30.
- `prodex-quota` and `prodex-runtime-quota`: quota adapters and proactive
  rotation inputs; REQ-31.
- `prodex-caveman-assets`: plugin/RCE-adjacent surface; REQ-32.

## Runtime Areas Isolated

### Runtime proxy

`verified`: `S2` states the launch/proxy path is
`prodex run/caveman/claude -> prodex-app runtime_launch -> prodex-runtime-launch
-> prodex-app runtime_proxy -> prodex-runtime-proxy -> upstream runtime`.
It also states the hot path must preserve hard affinity, rotate only before
commit, pass upstream responses through when upstream exists, and avoid disk I/O
or broad reads in request/stream commit paths.

`inferred`: the fork boundary should keep `prodex-app/src/runtime_proxy` as the
runtime authority and move only Multica-specific ingress/egress adapters around
it. Reimplementing this path in Go or splitting decisions across Go and Rust
would violate the local one-router ADR.

### Gateway

`verified`: `S3` documents `prodex gateway` as a standalone OpenAI-compatible
HTTP gateway. It supports provider presets, bearer auth, virtual keys, route
aliases with fallback/round-robin/least-busy/lowest-cost/lowest-latency/RPM/TPM
strategies, guardrails, observability sinks, admin endpoints, tenant dimensions,
and file/SQLite/Postgres/Redis gateway state. `S8` documents the compose shape
and warns that file and SQLite state are single-node deployment models.

`inferred`: Multica should not make the gateway a hosted central SaaS control
plane. Use the gateway as local L2 runtime/gateway capability and let Go push
desired policy, accounts, budgets, and kill switches through a versioned loopback
sidecar API.

### Smart Context

`verified`: `S7` states control-plane and continuation fields stay exact,
payload segments are classified independently, validation gates rewritten
requests, shadow/canary rollout is native, and fallback preserves exact behavior
when protocol, continuation, or global structure is at risk. Code confirms
shadow/canary decisions in `smart_context/rollout.rs` and original-body
pass-through on disabled, panic, canary-out, invalid JSON, unsupported shape, or
self-check failure in app `runtime_proxy/smart_context.rs` and
`runtime_proxy/smart_context/body.rs`.

`inferred`: Smart Context should remain wholly inside Rust L2. Go can enable,
disable, and observe it, but must not rewrite payloads or re-decide a request
after Rust has accepted runtime authority.

### State

`verified`: `S4` states `$PRODEX_HOME/state.json` owns active profile,
profile metadata, response/session bindings, quota snapshots, and health/backoff
data. Runtime bindings are `previous_response_id -> profile`,
`x-codex-turn-state -> profile`, and `session_id -> profile`. The runtime-state
crate has explicit continuation stores and scheduled-save planning; the
runtime-store crate merges snapshots and continuations instead of replacing
unrelated fields.

`inferred`: local file state remains acceptable for prodex-as-is and single-node
operation. The forked L2 target must externalize shared state to Postgres and/or
Redis for any multi-worker/multi-host deployment and keep file writes out of
request/stream commit paths.

### Redeem

`verified`: manual `prodex redeem <profile>` exists only for OpenAI/Codex
profiles, fetches quota first, prompts when 5h/weekly reset is within one hour
unless `--yes`, and sends a reset-credit consume request. Runtime `--auto-redeem`
exists on run/claude/caveman args and auto-redeem code checks weekly exhaustion,
natural-reset grace, reset-credit availability, profile/provider eligibility,
pool alternatives, and retry state before consuming a credit.

`not-validated`: this audit did not prove real-account effectiveness for
no-credit, with-credit, near-reset, weekly-exhausted, 5h-only, all-exhausted, or
non-OpenAI scenarios. F9 must validate those states empirically.

## fork boundary

This fork boundary is for the future Rust L2 sidecar milestone. It does not
change the current prodex-as-is rollout.

### L2 Fork Candidates

Move or harden inside the Rust L2 sidecar fork:

- `prodex-runtime-proxy`: primary runtime decision/hot-path nucleus.
- `prodex-context`: Smart Context audit/compress policy and exact fallback.
- `prodex-runtime-state`, `prodex-state`, `prodex-runtime-store`: continuation
  bindings, merge-safe state, and Postgres/Redis externalization boundary.
- `prodex-runtime-policy`, `prodex-runtime-tuning`: desired-state policy and
  runtime snapshot controls.
- `prodex-runtime-launch`: sidecar lifecycle, gateway launch, and per-session
  runtime planning.
- `prodex-runtime-broker`, `prodex-runtime-broker-log`: health, registry,
  metrics, continuity, and event DTOs.
- `prodex-proxy-config`: upstream proxy/client config input to the sidecar.

### Remain Upstream

Keep upstream unless a conformance failure forces a narrow fork:

- `prodex-provider-core`;
- `prodex-runtime-anthropic`;
- `prodex-runtime-gemini`;
- `prodex-runtime-gemini-cli-compat`;
- `prodex-runtime-claude`;
- `prodex-terminal-ui`;
- `prodex-app` app shell and user-facing command wiring;
- `prodex-cli`;
- `prodex-app-reports`;
- `prodex-runtime-doctor`;
- `prodex-runtime-metrics`;
- `prodex-runtime-log`;
- `prodex-update-notice`;
- `prodex-housekeeping`;
- `prodex-bench-support`.

### Decision Deferred

Do not enable or fork broadly until security/product scope is settled:

- `prodex-memory`: memory/Mem0 privacy and retention model;
- `prodex-presidio`: native PII runtime config and deployment dependency;
- `prodex-redaction`: redaction policy ownership between Go and Rust;
- `prodex-runtime-cookies`: cookie relay auth/session boundary;
- `prodex-caveman-assets`: plugin/RCE-adjacent Caveman surface;
- `prodex-profile-export`: encrypted profile export/import UX and audit scope.

### Interface Points

- `prodex-shared-types`: stable Rust-side DTO/type interface. Treat as the
  narrowest internal contract for L2 event/state translation.
- `prodex-codex-config`: config bridge between upstream Codex-compatible config
  and Multica desired-state.
- `prodex-session-store`: session metadata and redeem/session-affinity
  intersection.
- `prodex-shared-codex-fs`: filesystem bridge for profile isolation and
  upstream-compatible shared Codex state.

### Invariants To Keep Intact

Keep intact unless tests prove a replacement:

- hard affinity for `previous_response_id`, turn state, and session id;
- rotate-before-commit and never rotate mid-stream after model output starts;
- bounded pre-commit retry/fallback budgets;
- upstream response/status/body/stream pass-through after upstream response;
- Smart Context exact-field safety, shadow/canary rollout, replay corpus, and
  whole-request fallback;
- profile auth isolation under `$PRODEX_HOME/profiles/<name>` and
  upstream-compatible shared Codex state;
- provider capability/conformance fixtures and explicit loss/degraded/rejected
  classification;
- runtime log redaction helpers and structured marker discipline.

### Change Or Wrap For Multica L2

- add a loopback HTTP/gRPC-like JSON sidecar boundary for health, readiness,
  `ApplyPolicy`, `RegisterAccounts`, `StartSession`, `StopSession`, event stream,
  and kill switches;
- use ephemeral high-entropy bearer auth for sidecar calls;
- map prodex runtime logs/audit events into Multica runtime event schemas;
- replace shared gateway/admin/ledger state with Postgres/Redis where the L2 is
  not single-node;
- rebrand product-facing strings and comply with Apache-2.0 attribution;
- close the provider conformance split by moving pure transforms/contracts into
  provider core before using conformance as a product gate;
- add deployment gates for package pin/integrity, SBOM, secrets scanning,
  redaction, rollback to raw Codex, and Smart Context shadow/canary criteria.

### SBOM And Conformance Gaps

- Generate and store SBOM for the pinned prodex source and any fork artifact.
- Verify package pin, commit hash, checksum, Apache-2.0 attribution, and release
  provenance before a forked sidecar gate.
- Promote provider conformance from split app/core behavior into fixture-backed
  contracts before routing non-Codex providers through the sidecar.
- Add MCP framing/translation conformance for `prodex-mcp-stdio`,
  `prodex-runtime-anthropic`, and `prodex-runtime-gemini`.
- Prove redaction, cookie relay, memory retention, Caveman plugin disablement,
  and quota/redeem behavior with scrubbed evidence.

### Do Not Move Into Go

- in-flight profile selection;
- quota fallback and rotate-before-commit;
- Smart Context rewriting;
- continuation binding decisions;
- redeem attempts tied to runtime request fallback.

## Non-Blocking Relationship To F0

`verified` from local ADR/OpenSpec: near-term production uses prodex as-is,
pinned by version and commit, with Multica Go launching and monitoring it.

`inferred`: this fork map is an L2 target milestone. It informs the fork and
hardening backlog but must not block F0 unless it reveals a direct safety issue
in the prodex-as-is rollout controls.

## Gap Crates Analysis

1. Runtime broker (REQ-29): `prodex-runtime-broker`
   - Map health/registry/metrics to HealthCheck, RouteDecisionEvent, RuntimeEventStream
   - Scope: active in AS-IS; broker log is observability surface

2. Memory Mem0 (REQ-27/37b): `prodex-memory`
   - Scope decision: DISABLED by default (privacy/PII risk)
   - If enabled: redaction mandatory, contrato must cover memory events

3. Cookies (REQ-30): `prodex-runtime-cookies`
   - Auth/session relay — security audit required
   - Scope: document surface, flag for P4 security review

4. Quota (REQ-31): `prodex-quota`, `prodex-runtime-quota`
   - Amplify quota_mode per vendor for proactive rotation
   - Scope: map to P5 vendor matrix

5. Caveman (REQ-32): `prodex-caveman-assets`
   - SECURITY: DISABLED by default (RCE/supply-chain risk)
   - If ever used: allowlist + timeout + no external marketplace

## GATE P2

- [x] Fork-map reviewed with all 44 crates
- [x] Invariants traced to crates
- [x] Gap crates scoped with security assessment
- [x] No secrets in documents
