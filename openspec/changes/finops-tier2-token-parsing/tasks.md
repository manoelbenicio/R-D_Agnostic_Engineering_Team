# finops-tier2-token-parsing — Implementation Tasks

> Owner: **DB** (`src/finops/`) + **SUP** (`src/shared/storage/` for the new
> IDB store). Builds on Tier 1 (`cost-estimate.ts`) which stays as the
> fallback when no real usage exists.

## 1. Usage parsing (DB) — DONE

- [x] 1.1 `src/finops/token-usage.ts`: normalise provider payloads into a
      single `TokenUsage` shape (provider, model, input/output/total tokens).
- [x] 1.2 Parsers for OpenAI (`usage.prompt_tokens/completion_tokens`),
      Anthropic (`usage.input_tokens/output_tokens`), Google
      (`usageMetadata.*TokenCount`), AWS Bedrock (Converse `usage.*` +
      InvokeModel `amazon-bedrock-invocationMetrics`).
- [x] 1.3 `parseUsage(provider, payload, modelId?)` dispatcher; returns `null`
      when no usage is present (so callers fall back to Tier 1).

## 2. Costing + confidence (DB) — DONE

- [x] 2.1 `src/finops/token-cost.ts`: per-model price table
      (`TOKEN_PRICES`, USD / 1M tokens, input+output split) with
      longest-match `resolveTokenPrice` so versioned model ids resolve to
      their family.
- [x] 2.2 `computeTokenCost(usage)` → measured cost + `confidence`.
- [x] 2.3 `aggregateTokenCost(events)` → total + by provider/canvas/model,
      `totalTokens`, `unpricedEvents`, and an overall confidence
      (`measured` / `mixed` / `estimated`).

## 3. Persistence (SUP) — DONE

- [x] 3.1 IDB schema bumped `1 → 2`; new `usage_events` store (keyPath `id`,
      indexes `by-canvas`, `by-timestamp`) in `migrations.ts` + `idb.ts`.
- [x] 3.2 `src/finops/usage-repository.ts`: `recordUsage`, `listUsageEvents`,
      `listUsageEventsByCanvas`, `listUsageEventsInWindow`.

## 4. UI (DB) — DONE

- [x] 4.1 `src/finops/use-token-cost.ts`: react-query hook reading the window
      and aggregating; falls back to `estimated` confidence when empty.
- [x] 4.2 FinopsPage: new "Token Cost (Tier 2)" KPI card with a confidence
      badge (Measured / Partially measured / Estimated). Tier 1 estimate +
      `COST_ESTIMATE_DISCLAIMER` remain as the fallback surface.

## 5. Tests (DB) — DONE

- [x] 5.1 `src/finops/__tests__/token-cost.test.ts`: parser coverage for all
      four providers (real + simulated payload shapes), price resolution,
      measured/estimated/mixed confidence, aggregation by provider/canvas.

## 6. Capture wiring — PARTIAL (SPA helper done; CAO call site BLOCKED)

Blocked on the runtime exposing per-response usage. The parsers + store are
ready; the SPA-side capture helper is now in place. The only remaining work is
on the CAO side: surfacing the usage payload so the helper can be called.

- [x] 6.0 `src/finops/usage-capture.ts`: `captureUsageFromPayload(provider,
      payload, ctx, modelId?)` — conditional parse+record. Records a Tier 2
      `UsageEvent` only when the payload carries a real usage block; otherwise
      no-op + `null` so Tier 1 stays the fallback. No usage data is synthesised.
      Covered by `__tests__/usage-capture.test.ts`. A dependency-free
      cross-capability seam already exists in `@/shared/usage-capture-bus`
      (concrete adapter in `src/shell/usage-capture-adapter.ts`) to let the
      terminal pipeline reach finops without a sideways import once a payload
      exists.
- [ ] 6.1 **[BLOCKED — CAO]** Have CAO surface provider usage payloads per
      terminal turn (response body or a usage event endpoint). Today terminal
      output reaches the SPA only as raw binary xterm frames
      (`src/api/connect-terminal-socket.ts` → `src/terminal/use-terminal-stream.ts`);
      there is no structured per-response `usage` block anywhere in the SPA, so
      no real call site can be added without inventing data.
- [ ] 6.2 **[BLOCKED on 6.1]** From the terminal-stream pipeline, register the
      shell adapter (`setUsageCaptureBus`) and delegate completed turns through
      `getUsageCaptureBus().captureTurnUsage(...)` →
      `captureUsageFromPayload(...)` with `{ sessionName, terminalId, canvasId }`.
- [ ] 6.3 Normalise currency (USD only today) and refresh `TOKEN_PRICES`
      against a maintained source. (TODO retained; prices + currency unchanged.)

## Out of scope

- Tier 3 (provider billing-API reconciliation).
- Retries / tool-call token accounting beyond what the provider usage block
  reports.
