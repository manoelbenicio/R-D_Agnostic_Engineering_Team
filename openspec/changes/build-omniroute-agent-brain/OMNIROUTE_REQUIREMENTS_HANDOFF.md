# OmniRoute Requirements Handoff

## Request to the OmniRoute architect

OmniRoute will replace Prodex as the complete hot data plane for Agent Brain. This is broader than pointing model calls at a gateway. OmniRoute must cover provider credentials, subscriptions, model routing, strict account rotation, continuation affinity, token refresh and expiry, quota/reset behavior, 429 circuit breaking, bounded fallback, protocol translation, streaming/tool integrity, Smart Context/token saving, security, observability and capacity.

Please review and answer the linked requirements in writing. For every checklist or parity item, respond **Supported**, **Partially supported**, or **Not supported**, identify the exact OmniRoute version/configuration, and attach reproducible redacted evidence. Do not provide or expose any real API key, OAuth token, cookie, prompt, repository content or provider credential.

## Authoritative document set

1. [Detailed OmniRoute architecture acceptance checklist](./omniroute-architecture-acceptance-checklist.md)
   - Required API/message formats for Claude Code, Codex Responses, OpenAI Chat, Kimi, GLM/NVIDIA and Antigravity.
   - Streaming, reasoning, tools, structured output, continuation and cancellation.
   - Strict round-robin, real-time selection, token expiry, refresh, quota, reset/redeem, 401/403, 429, circuit breakers, retries and fallback.
   - Secret security, readiness, telemetry, failure injection and 20/50/100 capacity evidence.

2. [Prodex-to-OmniRoute feature parity matrix](./prodex-omniroute-feature-parity.md)
   - Every known Prodex capability is assigned to OmniRoute, Agent Brain, or an explicit retirement decision.
   - Includes Smart Context/token-saver requirements SC01-SC10, reset/redeem, hard affinity, pre-commit safety, MCP/tool continuity, quota, state, audit, redaction, broker, memory and special surfaces.
   - Prodex removal is blocked until every required row has evidence or a signed product/security waiver.

3. [AS-IS and TO-BE architecture](./architecture.md)
   - Current real topology: host/WSL daemon, legacy provider credential and rotation paths, existing but not yet enforced OmniRoute container.
   - Target topology: credentialless Agent Brain cold plane and OmniRoute-exclusive hot plane.
   - Responsibility table and end-to-end request/failure sequence.

4. [Formal target design and four-agent delivery topology](./design.md)
   - Neutral daemon extraction, protocol adapters, trusted environment construction, compatibility facade, risks, migration, rollback and parallel file ownership.

5. [Formal capability specifications](./specs/)
   - Normative, testable SHALL/MUST requirements for Agent Brain, OmniRoute routing, credentialless execution, parallel capacity and cutover operations.

6. [Four-Codex implementation plan](./tasks.md)
   - Dependency-aware work split, exclusive ownership, integration order, acceptance, capacity, cutover, legacy deletion and full debranding.

7. [Master OpenSpec/GSD planning, governance and total ETA](./MASTER_PLANNING_AND_GOVERNANCE.md)
   - Source-of-truth hierarchy, historical-change disposition, GSD v3 document set, G0–G8 roadmap, no-orphan controls and total ETA scenarios.

8. [Current Kiro/Opus-4.8 + Codex#56#A leadership handover](./KIRO_OPUS48_CODEX56A_TL_HANDOVER.md)
   - Current phase, role split, pane topology, safety gates and immediate G3 mission.

9. [Historical Claude/GLM-5.2 handover](./CLAUDE_GLM52_TL_HANDOVER.md)
   - Preserved for G0/G1 audit only; superseded on 2026-07-18.

## Non-negotiable confirmations

The architect must explicitly confirm:

- OmniRoute is the exclusive owner of all provider accounts, OAuth/API credentials, subscriptions, health, quota and rotation. Agent Brain holds one scoped OmniRoute key only.
- All approved Claude, Codex, Kimi, GLM/NVIDIA and Antigravity model routes support the exact client protocol and message features stated in the checklist.
- Strict round-robin applies to every new independent logical request and is concurrency-safe. It does not mean the platform is globally limited to one in-flight request.
- If policy permits only one active request per account, OmniRoute/account provisioning can supply enough eligible account slots for the accepted 20/50/100 profile or use a documented bounded queue. This is capacity provisioning, not a change to round-robin semantics.
- Stateful continuations, prompt caches and tool turns preserve required account affinity without making unrelated traffic sticky.
- Expired/revoked credentials, quota exhaustion, 401/403, scoped/global 429, 5xx, timeouts and broken streams follow bounded, observable and safe policies.
- Automatic fallback/replay is pre-commit only; partial output or non-idempotent tool actions are never silently duplicated.
- Smart Context/token saving and reset/redeem have proven OmniRoute replacements or explicit written waivers before Prodex removal.
- All secrets and content are protected; management authorization is separate from the Agent Brain inference key.
- The exact deployment has reproducible protocol, failure and 20/50/100 capacity evidence.

## Response package required from OmniRoute

Please return:

- completed acceptance checklist and model-route matrix;
- completed Prodex parity matrix with evidence links and gaps;
- exact OmniRoute version/image digest and redacted configuration revision;
- API/protocol compatibility statement per model route;
- route/account/concurrency/affinity/retry/circuit/timeout configuration with secrets removed;
- Smart Context/token-saver and reset/redeem design/evidence;
- failure-injection report;
- 20/50/100 capacity report;
- secret storage, redaction, audit and key-rotation evidence;
- account/model hot-management, backup/restore, incident, upgrade and rollback runbooks;
- named architecture/operations owner and escalation path;
- remediation plan and date for every partial or unsupported requirement.

## Go/no-go rule

A successful health check or model completion is not sufficient. Any unresolved gap in authentication, provider-secret isolation, message fidelity, streaming, tools, continuation, safe retry, rotation correctness, required Prodex parity or accepted launch capacity blocks production cutover unless the responsible product and security owners sign an explicit, time-bounded waiver.
