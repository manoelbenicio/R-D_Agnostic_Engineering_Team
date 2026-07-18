# EV-G1-OPS-PREP — Prodex-to-OmniRoute parity matrix

- Prepared by: Codex 4 — Operations/Parity/Evidence
- Baseline: OmniRoute `3.8.48`; image digest `NOT PROVIDED`
- Scope: required P01–P34 and SC01–SC10 rows
- Rule: no Prodex removal until every blocker has accepted evidence or an explicit time-bounded product/security waiver

Status meanings: **Supported** requires accepted row-level evidence on the exact deployment/routes. **Partial** means related code/configuration exists but required behavioral proof is incomplete. **Not-supported** means the deployed target lacks the required behavior/evidence or an explicit architecture contradiction exists. No waiver is approved by the reviewed documents.

## P01–P34 core parity

| ID | Feature | Status | Target owner | Acceptance evidence: available / required | Gap remediation plan | Waiver flag |
|---|---|---|---|---|---|---|
| P01 | Profile/account isolation | Partial | OmniRoute | Available: OmniRoute owns account state. Required: storage review and concurrent cross-account isolation proof | Prove provider auth/state never reaches Agent Brain or crosses pseudonymous accounts | None approved; blocker |
| P02 | Export, secret store, backup | Partial | OmniRoute | Available: AES-256-GCM component identified. Required: key management, redacted backup/restore, access-control proof | Pin state revision; write and execute encrypted backup/restore runbook | None approved; blocker |
| P03 | Stable scoped gateway credential | Partial | OmniRoute | Available: authentication required; missing credential 401, authorized request 200. Required: route scope and rotate/revoke demo | Scope credential to approved inference routes; separate management; prove rotation/revocation | None approved; blocker |
| P04 | Pre-commit eligible-account selection | Partial | OmniRoute | Available: selector/combo code and single-process atomicity. Required: correlated traces and concurrency distribution | Add safe selection telemetry and run tier profiles | None approved; blocker |
| P05 | Strict independent-request RR | Partial | OmniRoute | Available: logical-request unit and limit-one RR documented; four sequential Agy successes. Required: concurrent deterministic sequence/fairness | Run simultaneous independent-request tests per strict route; retain Kimi exception explicitly | None approved; blocker |
| P06 | Hard continuation affinity | Partial | OmniRoute | Available: Responses state policy and affinity-pin component. Required: Responses/tool/cache origin-account proof | Prove `preserve` origin pin or stateless materialization while unrelated traffic rotates | None approved; blocker |
| P07 | Rotate/retry before commit only | Partial | OmniRoute | Available: normative pre-commit policy. Required: before/after-first-output and tool-action failure injection | Implement/verify commit boundary and deduplication; surface partial failure without replay | None approved; blocker |
| P08 | Bounded retry/fallback budget | Partial | OmniRoute | Available: cooldown/backoff/breaker components. Required: effective attempts/deadlines/cancel results | Publish redacted policy and inject timeout/reset/5xx/429/cancel cases | None approved; blocker |
| P09 | Quota adapters | Partial | OmniRoute | Available: quota cooldown and persisted limit time. Required: provider quota-source map and exhaustion test | Map headers/APIs/counters and distinguish context overflow; test reset/re-entry | None approved; blocker |
| P10 | Guarded redeem/reset claim | Partial | OmniRoute | Available: Codex-only reset-credit component identified. Required: provider matrix, idempotency/grace/pool/post-check demos | Restrict to proven Codex routes; implement missing policy/tests or formally defer other providers | None approved; product/security waiver required for parity loss |
| P11 | OAuth refresh and expiry | Partial | OmniRoute | Available: refresh path referenced. Required: proactive clock-skew and single-flight expired/revoked tests | Add/verify per-account single-flight lock and quarantine-only fallback | None approved; blocker |
| P12 | 401/403 classification | Partial | OmniRoute | Available: error-classifier architecture. Required: injected classes and resulting account/model state | Publish taxonomy; bound refresh/re-auth to one policy attempt | None approved; blocker |
| P13 | Rate-limit backoff/circuit | Partial | OmniRoute | Available: scoped classifier, jitter, cooldowns, breaker/half-open, persisted limit time. Required: account/global 429 demos | Run failure injection including all-accounts throttled and half-open recovery | None approved; blocker |
| P14 | Provider fallback/adaptive routing | Partial | OmniRoute | Available: operational Kimi→dedicated→NVIDIA chain reported. Required: versioned approved policy, actual-model telemetry, capability equivalence | Approve exact ordered chains; same-model first; reject capability downgrade | None approved; blocker |
| P15 | Anthropic translation | Partial | OmniRoute | Available: Messages/SSE route and translators. Required: exact-route blocks/tools/thinking/errors/usage/SSE fixtures | Execute conformance on every approved Claude/Agy-via-Anthropic model | None approved; blocker |
| P16 | OpenAI Responses for Codex | Partial | OmniRoute | Available: Responses route and state policy. Required: exact-model stream/function/reasoning/continuation/error fixtures | Complete Codex custom-provider conformance over HTTP/SSE | None approved; blocker |
| P17 | Gemini/Antigravity compatibility | Partial | OmniRoute | Available: direct endpoint and four-account 200 sequence for one Agy model. Required: exact schema/auth, all models, or Claude/Codex fallback proof | Prove native override or accepted compatible frontend per exact route | None approved; blocker |
| P18 | Kimi/GLM/NVIDIA adapters | Partial | OmniRoute + Agent Brain adapter | Available: Chat route and combos. Required: exact-model protocol and installed CLI configuration proof | Codex 3 proves native Kimi registry or compatible frontend; convert NIM to gateway-only | None approved; blocker |
| P19 | Capability discovery | Partial | OmniRoute | Available: models endpoint/connectivity. Required: versioned protocol/context/tools/reasoning/structure/stream/availability registry | Publish complete registry; reject unsupported fields/models deterministically | None approved; blocker |
| P20 | Tool/MCP continuation integrity | Partial | OmniRoute for protocol; Agent Brain for local MCP | Available: protocol routes/translators. Required: parallel-tool and multi-turn continuation fixtures | Prove IDs/schemas/deltas/results/order and fail closed on unsafe account switch | None approved; blocker |
| P21 | Streaming commit integrity | Partial | OmniRoute | Available: SSE route implementations. Required: event order, heartbeat, backpressure, usage, cancel, broken stream, restart | Add bounded buffers/deadlines and run long/slow/cancel/failure tests | None approved; blocker |
| P22 | Hot-path nonblocking I/O | Partial | OmniRoute | Available: single-process selector architecture. Required: code/latency/resource evidence under load | Audit blocking DB/state operations and measure selection latency/resource use | None approved; blocker |
| P23 | Runtime state/store | Not-supported | OmniRoute | Available: single-node SQLite volume and in-memory cursor. Required: accepted state topology, consistency, backup/restore, restart recovery | Choose single-instance with drain/rollback or shared backend; reverify strict rotation | None approved; explicit cutover blocker |
| P24 | Runtime broker/registry | Partial | OmniRoute | Available: route/account data and health concepts. Required: safe registry API and account changes under load | Expose redacted eligibility/health; prove hot add/remove without stale Brain routing | None approved; blocker |
| P25 | Versioned atomic runtime policy | Not-supported | OmniRoute; Agent Brain selects approved policy ID | No versioned atomic apply/rollback/idempotency proof supplied | Define schema/revision validation, atomic activation, stale rejection, rollback | None approved; blocker |
| P26 | Scoped kill switches | Not-supported | OmniRoute; Agent Brain also stops tasks | Compression controls exist, but tenant/key/route/provider/model/account feature switch demonstrations are absent | Implement or inventory switches; prove next-request effect and audit under load | None approved; blocker |
| P27 | Health/readiness | Not-supported | OmniRoute | Readiness fail-closed is an explicit outstanding gate | Separate liveness/readiness; inject config/state/auth/route dependency failures | None approved; blocker |
| P28 | Runtime events/route decisions | Not-supported | OmniRoute | Desired fields are designed; no accepted schema-versioned redacted sample/dedup proof | Publish event schema and correlation headers; test redaction and idempotency | None approved; blocker |
| P29 | Metrics/logs/audit | Partial | OmniRoute; Agent Brain aggregates | Metrics concepts/components identified. Required: catalog, samples, retention, alerts | Produce bounded telemetry catalog and dashboards/alerts with safe pseudonymous dimensions | None approved; blocker |
| P30 | Redaction and PII protection | Partial | OmniRoute | Encryption component and redaction requirement identified. Required: nested credential/error fixtures and diagnostic-content PII policy | Add default content-off policy, recursive redaction fixtures, audited opt-in diagnostics | None approved; blocker |
| P31 | Runtime cookies | Not-supported | OmniRoute | No provider cookie-route inventory or isolation/redaction proof supplied | Inventory routes; isolate/encrypt cookies or mark affected routes unavailable | None approved; blocker |
| P32 | Request idempotency | Not-supported | OmniRoute + Agent Brain request IDs | No ambiguous-timeout inference/tool deduplication proof or config conflict behavior | Define idempotency key/state window; test duplicate billing/tool prevention and config replay | None approved; blocker |
| P33 | Capacity and overload | Partial | OmniRoute inference; Agent Brain admission | Available: layered concurrency and bounded queue controls. Required: 20/50/100 reports | Build harness; accept tier 20 first; keep 50/100 disabled until evidence | None approved; lower-tier restriction authorized, not a waiver |
| P34 | Actual model/cost/usage | Partial | OmniRoute provides; Agent Brain aggregates | Route/account/usage concepts exist. Required: actual-model headers/events, normalized reconciliation, pricing revision | Publish safe telemetry and cost-source version; reconcile sample without account identity | None approved; blocker |

## SC01–SC10 Smart Context/token-saver parity

The architect identified a compression subsystem with multiple modes/engines plus fidelity/risk/pipeline/cache/memo/evaluation components. That is architectural evidence only. No row has the required deterministic fixture, shadow/canary evidence, exact whole-request fallback proof, or live kill-switch demonstration.

| ID | Behavior | Status | Target owner | Acceptance evidence: available / required | Gap remediation plan | Waiver flag |
|---|---|---|---|---|---|---|
| SC01 | Segment classification | Partial | OmniRoute | Available: compression engines. Required: deterministic system/control/continuation/tool/user/repo/appendix corpus | Publish classifier design and golden fixtures | None approved; product+security waiver required to defer |
| SC02 | Exact protocol preservation | Partial | OmniRoute | Available: fidelity/risk gates. Required: byte/semantic role/order/control/tool/JSON fixtures | Add protocol-specific invariants and exact-mode tests | None approved; product+security waiver required to defer |
| SC03 | Structural validation | Partial | OmniRoute | Available: pipeline guards/self-check concepts. Required: negative malformed/tool/reference/allocation fixtures | Fail optimization closed and dispatch original on any invalid structure | None approved; product+security waiver required to defer |
| SC04 | Whole-request exact fallback | Partial | OmniRoute | Available: fallback architecture claimed. Required: fault injection proving original or exact-equivalent body | Preserve immutable original payload and test panic/unsupported/self-check failures | None approved; product+security waiver required to defer |
| SC05 | Shadow mode | Partial | OmniRoute | Compression preview/evaluation surfaces identified. Required: always-original dispatch with redacted comparison telemetry | Implement/configure shadow cohort and prove no rewritten upstream body | None approved; product+security waiver required to defer |
| SC06 | Canary rollout | Partial | OmniRoute | Modes exist. Required: deterministic percentage/route distribution and unchanged canary-out body | Add versioned cohort policy, stable assignment, rollback trigger | None approved; product+security waiver required to defer |
| SC07 | Live-mode safety | Partial | OmniRoute | Fidelity/risk gates identified. Required: approved fixture/replay benchmark before rewritten dispatch | Require all capability/structure checks and exact fallback on uncertainty | None approved; product+security waiver required to defer |
| SC08 | Continuation/cache integrity | Partial | OmniRoute | Cache-aware/affinity components identified. Required: multi-turn/cache/tool proof | Bind optimization decision to continuation ownership and preserve tool/cache relationships | None approved; product+security waiver required to defer |
| SC09 | Savings/quality telemetry | Partial | OmniRoute | Evaluation/memo components identified. Required: redacted original/optimized token and decision/fallback samples | Emit schema-versioned metrics without content or opaque reasoning | None approved; product+security waiver required to defer |
| SC10 | Immediate kill switch | Partial | OmniRoute | Compression modes include off. Required: live next-request disable proof without restart | Expose audited policy switch; verify original traffic continues immediately | None approved; product+security waiver required to defer |

## Owner and disposition summary

- OmniRoute architect/operator owns every hot-path remediation and must supply immutable version/configuration plus redacted proof.
- Codex 2 and Codex 4 jointly own protocol, rotation, failure, and capacity evidence collection in Waves 3–4.
- Codex 3 owns credentialless CLI/adaptor proof; Codex 1 is the sole integrator and approves supported routes/policies.
- Product and Security must sign any time-bounded waiver. No waiver is present in the reviewed artifacts.
- B01–B08 cold-plane retention and R01–R05 retirement decisions remain governed by the source parity document; they were not requested as rows in this deliverable.

## G1 parity disposition

All 44 requested rows are classified. No row is accepted as full parity on the exact deployed image because the image digest is missing and live row-level evidence is incomplete. P23, P25–P28, P31, and P32 are not supported as accepted; the other rows are partial. Prodex removal remains NO-GO until the blocker evidence is accepted or a permitted waiver is signed.
