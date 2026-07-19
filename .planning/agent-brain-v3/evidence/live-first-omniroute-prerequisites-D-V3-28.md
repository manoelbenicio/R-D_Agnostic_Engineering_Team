# LIVE-FIRST — local OmniRoute / startup / credential prerequisites (D-V3-28; NO VALUES)

> Per Owner directive 2026-07-19. Reports exact prerequisites to run live non-prod integration/acceptance.
> **No secret values here** — Owner injects keys directly into OmniRoute; agents never read/print/copy values.

## OmniRoute endpoint / reachability
- **Current (dev):** host loopback `127.0.0.1:20128`.
- **Future (containerized):** container DNS `http://omniroute:20128` (do NOT hard-code Docker DNS for the host daemon).
- Reachability precheck before admission; **fail-closed** if OmniRoute is unreachable (liveness/readiness separated).
- Readiness = authenticated `/v1/models` retrieval succeeds.

## Credential prerequisites (values NEVER displayed; Owner-injected)
- **Single stable OmniRoute API key**, injected **by the Owner directly into OmniRoute** (not by any agent).
- Agent Brain child env var **NAME only** = `AGENT_BRAIN_OMNIROUTE_API_KEY` (name referenced; value never read/printed/copied; never falls back to `OPENAI_API_KEY`).
- Linux permission-restricted **secret-file reference** (path/ownership/mode/rotation are reference-only; no value read/copied/hashed into telemetry).
- **Old/exposed key remains FORBIDDEN** (D-V3-25); live tests use only a fresh Owner-injected key in OmniRoute.
- Agent Brain is **credentialless**: no provider-native credentials/base-URLs reach the child env (sanitized, gateway-required mode); OmniRoute owns all provider creds/rotation/selection/fallback.

## Startup prerequisites
- Daemon in **gateway-required mode**; sanitized child environment applied last.
- `MULTICA_PRODEX_REQUIRED=0` (Prodex default-OFF cold-recovery only; never hot; no dual router).
- Readiness gate + protocol admission before task launch.

## Route reachability (per D-V3-27 authoritative matrix)
- Antigravity (operational; revalidate), Claude, Codex, **Cline→Kimi-K2.7**, **Cline→GLM52**, **GLM52→NVIDIA (fallback, OmniRoute-owned/bounded)**, **Kiro→Opus48 (AWS)** — each reachable via OmniRoute with the required protocol/tools/reasoning/usage/cancel/error surface.

## Boundaries (unchanged)
- No exposed/revoked key use; no secret display. No production/cutover/Prodex. **No tier-20 capacity classification before G4-OBS** (ordinary live functional concurrency/failure tests allowed; capacity certification is NOT).
