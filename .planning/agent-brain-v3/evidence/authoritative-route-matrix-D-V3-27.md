# AUTHORITATIVE ROUTE MATRIX — D-V3-27 (Owner Decision 8, FROZEN 2026-07-19)

> Owner-decided, council-unanimous. FROZEN authoritative mapping for live non-prod OmniRoute acceptance.
> **Live non-prod OmniRoute acceptance remains MANDATORY for P0 Main Brain completion.** No task is accepted
> from prose. Selection/fallback is **exclusively OmniRoute-owned and bounded**; **Agent Brain never holds
> provider credentials and never makes provider fallback decisions** (single router; no dual router; no
> direct native credentials). Holds intact: no capacity/9.1, Prodex, cutover, production.

## Frozen route mapping

| # | Route | Provider/runtime path | Status | Acceptance basis |
|---|---|---|---|---|
| 1 | **Antigravity** | Antigravity ↔ OmniRoute | **Already fully tested with OmniRoute → treat as FULLY OPERATIONAL** | **Revalidate provenance/hashes/evidence — NOT redundant reimplementation.** |
| 2 | **Claude** | Claude Code → Anthropic Messages (via OmniRoute) | Accepted route | Independent test: protocol/tools/reasoning/usage/cancel/error |
| 3 | **Codex** | Codex → OpenAI Responses (via OmniRoute) | Accepted route | Independent test: protocol/tools/reasoning/usage/cancel/error |
| 4 | **Kimi** | **Cline (main provider/runtime) → Kimi-K2.7** | Route to test | Independent test via Cline runtime: protocol/tools/reasoning/usage/cancel/error |
| 5 | **GLM52 (primary)** | **Cline → GLM52** | Route to test | Independent test: protocol/tools/reasoning/usage/cancel/error |
| 6 | **GLM52 → NVIDIA (fallback)** | **NVIDIA = GLM52 fallback**; selection/fallback **OmniRoute-owned & bounded** | Route to test | **Explicit primary/fallback evidence for GLM52→NVIDIA**; Agent Brain holds no creds, makes no fallback decision |
| 7 | **Kiro** | **Kiro → Opus48 from AWS** | Route to test | Independent test: protocol/tools/reasoning/usage/cancel/error |

## Evidence required (per applicable route)
- protocol · tools · reasoning · usage · cancel · error — per route above.
- **GLM52→NVIDIA:** explicit primary/fallback evidence (bounded, OmniRoute-owned selection).
- Antigravity: **revalidate existing hashes/evidence** (provenance), do not reimplement.
- Remaining routes (Claude, Codex, Cline→Kimi-K2.7, Cline→GLM52, GLM52→NVIDIA, Kiro→Opus48/AWS):
  **independently tested** (producer ≠ reviewer ≠ adjudicator).

## Invariants
- **No dual router.** **No direct native provider credentials** in Agent Brain (credentialless; OmniRoute owns creds/rotation/fallback).
- **No task accepted solely from prose.**
- Live acceptance is **non-prod only**; does NOT authorize capacity/9.1, Prodex, cutover, or production.

## Dependency / gating note
- **D-V3-25(B):** live-provider tests remain **SECURITY-STOPPED** until the Owner confirms UI invalidation/
  revocation of the exposed key. Therefore live route acceptance under this matrix is **BLOCKED until key
  revocation is confirmed**; offline/synthetic route work (D-V3-25C / D-V3-20) may proceed meanwhile.
- Antigravity provenance revalidation (hashes/evidence) is offline and may proceed now.

## OpenSpec conflict flagged for council correction
- `openspec/changes/build-omniroute-agent-brain/specs/omniroute-agent-routing/spec.md:4` uses generic
  wording "approved **Kimi/GLM/NVIDIA/Antigravity** frontends" — does NOT capture: Kimi via **Cline→Kimi-K2.7**,
  GLM52 via **Cline** with **NVIDIA fallback**, Antigravity **already-operational**, or **Kiro=Opus48/AWS**.
  The acceptance checklist / design / tasks carry the same generic five-vendor framing. **Reported for
  council correction — NOT edited here** (OpenSpec spec edits require council/W8 change process).
