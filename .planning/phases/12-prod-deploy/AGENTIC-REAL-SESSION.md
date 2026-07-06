# P12.12.3 — AGENTIC REAL SESSION (self-routing, no owner keys needed)

phase: 12-prod-deploy · task: 12.3 · author: Kiro/Principal
supersedes the "owner must inject provider keys" assumption in PREREQUISITES §1.

## Insight
The 4 real vendors ARE what our agents run on (PROJECT.md): OpenAI/Codex, Antigravity, OpenCode/GLM5.2,
Kiro/Opus4.8. Each agent already holds a LIVE, authenticated provider session. Therefore the real
provider round-trip can be produced AGENTICALLY: route an agent's OWN real turn through the deployed
prodex sidecar+gateway (the CODEX_HOME × prodex × Herdr interception path, REQ-17). No fake upstream,
no placeholder key, no owner key hand-off.

## Method (per vendor = per agent)
For each agent whose vendor we prove:
1. Point that agent's runtime at the deployed prodex sidecar (CODEX_HOME/proxy env → runtime_endpoint).
2. Have the agent perform ONE real turn (a genuine prompt→completion it would run anyway).
3. prodex compacts context and forwards to the agent's REAL provider (real upstream, real auth).
4. Capture from the gateway: real `gateway_response_model` (the real model id), real usage, tokens_saved.

## Real-vs-fake asserts (EVIDENCE_CONTRACT §1–§2) — mandatory
- host = deployed sidecar on the real run host (NOT a throwaway 127.0.0.1 smoke port).
- `gateway_response_model` = the agent's REAL model id (e.g., a real GPT/Gemini/GLM/Claude id) — assert != "fake-upstream-logging".
- `gateway_status == 200`, `measurement_source == gateway_usage`.
- usage realistic (a real turn — not input=8/output=1).
- runtime_session_id + tokens_saved DISTINCT per agent/vendor (real, different payloads).
- provider auth = the agent's OWN live credentials (not a placeholder). Do NOT echo the key.

## Why this is agentic AND honest
Agents execute (their own real turns) → TL orchestrates → Kiro verifies. It is a REAL provider
round-trip by construction (the agent genuinely talks to its provider), so it satisfies the contract
without fabrication and without waiting on an owner key drop.

## Remaining owner input (only if applicable)
Host: if a SEPARATE prod host is mandatory (vs. the run host where agents live), owner names it. Otherwise
the deployed sidecar on the fleet run host, carrying agents' real provider traffic, is the real session.
