# L1 — Readiness-declaration adapter trace (how the daemon consumes an approved enriched declaration)

- agent `Opus48#B` · lane `L1-TRACE` · pane `w6:p2` · task `L1-DECLARATION-TRACE`
- check-in receipt: `.deploy-control/p0/checkins/Opus48-B__L1-DECLARATION-TRACE__20260722T231400Z.json`
- posture: **READ-ONLY** (no edits). Source at HEAD `a6d50986…`. All paths under `server/internal/daemon/`.

## Chain: OmniRoute /v1/models → Registry → ReadinessChecker → admission

1. **Fetch** — `brain_integration.go:210` `gateway.NewRegistry(ModelsFetchFunc(client.FetchModels), ttl)`.
   `gateway.Registry.Snapshot` (`registry.go:89`) is TTL-cached; on refresh it calls
   `fetcher.FetchModels` (`:118`) → `buildSnapshot` (`:121`).
2. **ReadinessChecker** — `brain_integration.go:232` `gateway.NewReadinessChecker(client, registry,
   r.config.Neutral.Gateway.Readiness, correlation)` (strict, fail-closed policy).

## A. Registry version header — source & consumption
- **Wire header:** `X-OmniRoute-Registry-Version` = `HeaderRegistryVersion` (`health_models.go:10`).
- **Client reconciliation** (`client.go FetchModels`):
  - enriched (non-dev) path: reads `response.Header.Get(HeaderRegistryVersion)` (`:195`); if the JSON body
    `registry_version` (`ModelsDocument.RegistryVersion`, `health_models.go:19`) is non-empty **and differs**
    from the header → `ErrorProtocol` (fail closed, `:197`); otherwise the **header value wins**
    (`document.RegistryVersion = headerVersion`, `:200`).
  - dev path: header value is stamped into the projection (`ProjectOmniRouteModels(native, header)`, `:189`).
- **Validation** (`registry.go:221` `buildSnapshot`): version must be non-empty, ≤128 chars, no
  `\r\n\x00`, and `len(Models) ∈ [1, MaxRegistryModels]`, else `ErrorProtocol`.
- **Readiness gate:** `ReadinessChecker` sets `ModelRegistryReady = registrySnapshot.Version != ""`
  (`health_models.go`). So the operator/OmniRoute declaration is "approved" for readiness only when it
  carries a valid registry version.

## B. capability.protocol — source & consumption
- **Per-model source:** each enriched row's `protocol` JSON field (`ModelDocument.Protocol`,
  `health_models.go:22`) → `protocolFromWire(row.Protocol)` in `buildSnapshot` (`registry.go`), which
  accepts only `anthropic-messages | openai-responses | openai-chat | antigravity`, else `ErrorProtocol`
  (fail closed). Stored as `ModelSpec.Capability.Protocol` (`registry.go:261-263`).
- **Enriched-row completeness (fail-closed):** the row must also have non-nil
  `streaming/tools/reasoning/structured_output/available` pointers, `context_limit > 0`, and a non-empty
  bounded `account_pool`, else `registry.refresh ErrorProtocol` (`registry.go` ~`:234-241`). A bare
  OpenAI-basic `/v1/models` row is therefore rejected unless projected (see C).
- **Consumption:**
  - `ReadinessChecker`: `SelectedModelReady = model.Available`; `SelectedProtocolReady =
    model.Capability.Protocol == request.Protocol` (`health_models.go`).
  - `Registry.ValidateCapability` (`registry.go`): `requirement.Protocol != "" && capability.Protocol !=
    requirement.Protocol → ErrorCapability`; also rejects requested streaming/tools/reasoning/
    structured_output/min-context not met by the declared capability.

## C. Operator declaration config/file — the ONLY operator surface
- **There is NO operator-supplied static capability/protocol declaration file.** The authoritative
  declaration is the OmniRoute `/v1/models` response itself (header `X-OmniRoute-Registry-Version` +
  per-row enriched `protocol`/capability/`available`).
- **Single operator knob:** env `OMNIROUTE_DEV_MODELS_COMPAT=1` (`model_projection.go:13`;
  `brain_integration.go:220`; `client.go:153,180`). When set, `client.FetchModels` decodes OmniRoute's
  native OpenAI-basic `/v1/models` and **projects** it into the enriched schema via
  `ProjectOmniRouteModels` (`model_projection.go:93`):
  - **only** `approvedProjectionRouteModel = "claude_code_kimi_2.7_Code"` (`:46`) is marked
    `available=true` with `Protocol = approvedProjectionProtocol` (`approvedProjectionRow`, `:123`);
    **every other id → `unavailableProjectionRow`** (`available=false`, inert `context_limit=1`).
  - registry version defaults to `omniroute-dev-compat-v1` (`:56`) only when the header is absent.
  This is a DEV/operator compatibility projection for a registry that hasn't published enriched rows yet —
  it is **not** a credential/account/model-mapping declaration and grants selectability to exactly one
  projected route. When the flag is unset, the enriched path (A/B) is used verbatim.
- Deploy runbook (`deploy/topology.go:102`) references an *"effective non-secret configuration revision"*
  at brain launch (frozen gateway-required/base-URL/secret-file-ref/strict-readiness/tier-20) — operator
  config for gateway wiring, **not** a capability/protocol declaration.

## Summary
An "approved enriched readiness declaration" enters solely through OmniRoute `/v1/models`:
header `X-OmniRoute-Registry-Version` (reconciled header-wins, mismatch fail-closed) supplies the registry
version; each row's `protocol` (+ full capability booleans, context, account_pool, `available`) supplies
`ModelSpec.Capability` after `protocolFromWire` validation. The daemon consumes it via the TTL-cached
`Registry` and the strict `ReadinessChecker` (liveness → authenticated → registry-version → selected-model
`available` → selected-protocol match). The only operator override is the `OMNIROUTE_DEV_MODELS_COMPAT=1`
projection (single selectable route); there is no static operator capability-declaration file.

## Non-claims
Static read-only trace only; no runtime/live probe; no edit; no secret read; no inference. OmniRoute
internals (account/credential/rotation/model-mapping) not probed — only the daemon-side consumption path.
