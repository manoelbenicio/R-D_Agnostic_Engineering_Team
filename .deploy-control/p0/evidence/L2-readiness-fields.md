# L2 — Strict-readiness fields unmet by deployed OmniRoute /v1/models

- agent: `Codex56#A` · lane: `L2` · task: `L2-READINESS-FIELDS` · pane: `w7:p3`
- lock: `.deploy-control/p0/evidence/L2-readiness-fields.md`
- MODE: **READ-ONLY**; no OmniRoute-internal changes; no secret; no inference; no deploy.
- endpoint: ORQ1 Tailscale `http://100.118.244.61:20128`.

> DETERMINATION: from the deployed **non-secret** vantage, `/v1/models` is auth-gated (401) and its
> unauthenticated response **does not carry `X-OmniRoute-Registry-Version`**, so the three registry-
> dependent strict-readiness fields — **ModelRegistryReady, SelectedModelReady, SelectedProtocolReady**
> (for codex and for auto/coding) — are **unmet/undeterminable** without an authenticated metadata read.
> Confirming their actual met/unmet values requires an authorized non-inference `/v1/models` read.

## 1. Strict-readiness contract (source of truth, read-only)

`server/internal/daemon/gateway/health_models.go` `ReadinessChecker.CheckGatewayReadiness` sets, in
order, then applies the strict fail-closed policy (`policy.Evaluate`; constructor rejects any
non-`ReadinessStrict`/non-`FailClosed` policy):

| # | Field | Pass condition | Source dependency |
|---|---|---|---|
| 1 | `Live` | `client.CheckLiveness` OK | `EndpointSet.Liveness` HTTP 2xx |
| 2 | `Authenticated` | `client.CheckReadiness` OK | `EndpointSet.Readiness` HTTP 2xx (authenticated) |
| 3 | `ModelRegistryReady` | `registrySnapshot.Version != ""` | `registry.Snapshot` ← `/v1/models` enriched `registry_version` body + `X-OmniRoute-Registry-Version` header (`HeaderRegistryVersion`) |
| 4 | `SelectedModelReady` | `snapshot.Models[RouteModel]` exists **and** `model.Available` | authenticated enriched `/v1/models` rows |
| 5 | `SelectedProtocolReady` | `model.Capability.Protocol == request.Protocol` | authenticated enriched `/v1/models` rows |

Registry integrity (`registry.go`/`projection.go`): version must be present; header-vs-body mismatch → `ErrorProtocol` (no upstream echo). `/v1/models` alone is a catalog gate, not inference proof.

## 2. Deployed non-secret observations (probes)

| Probe | Result |
|---|---|
| `GET /v1/models` (unauthenticated) | **401** (auth-gated; REQUIRE_API_KEY) — confirmed F3 |
| `/v1/models` response headers | `content-type: application/json`, `x-omniroute-route-class: CLIENT_API`; **`X-OmniRoute-Registry-Version` ABSENT**; no `WWW-Authenticate` |
| `GET /health/live` | 404 (source-default liveness path not served on this deployment) |
| `GET /health/ready` | 404 |
| `GET /status` | 200 (HTML dashboard) |

## 3. Field-by-field determination (codex + auto/coding)

| Field | Deployed (non-secret) determination |
|---|---|
| `Live` | **Undeterminable / likely config-mismatch:** the source-default `EndpointSet.Liveness=/health/live` returns **404** here. If the deployed daemon is configured with that path, liveness would FAIL; the deployment's actual liveness path is not exposed non-secret (only `/status` 200 HTML). Flag for config confirmation. |
| `Authenticated` | **Undeterminable non-secret:** `/v1/models` (used as readiness under devModelsCompat) returns **401** without the stable secret; with the daemon's key it would authenticate. Cannot verify without the secret. |
| `ModelRegistryReady` | **UNMET/undeterminable:** requires `registrySnapshot.Version != ""`, sourced from the enriched `/v1/models` body + `X-OmniRoute-Registry-Version` header. The unauthenticated response carries **no such header** and no enriched body → Version is unobtainable non-secret. |
| `SelectedModelReady` — **codex** | **UNMET/undeterminable:** needs the codex `RouteModel` row + `Available` from the authenticated registry. Also the exact codex `RouteModel` was **UNDECLARED** in `g1-model-route-matrix.md` (Codex/OpenAI row) — doubly blocked. |
| `SelectedModelReady` — **auto / coding** | **UNMET/undeterminable:** `auto`/`coding` route names cannot be resolved to exact registry rows or `Available` without the authenticated enriched `/v1/models`. |
| `SelectedProtocolReady` — **codex** | **UNMET/undeterminable:** expected `ProtocolOpenAIResponses` (codex frontend), but per-model `Capability.Protocol` is only in the authenticated registry rows. |
| `SelectedProtocolReady` — **auto / coding** | **UNMET/undeterminable:** per-model protocol unreadable non-secret. |

## 4. Exact answer

- **`X-OmniRoute-Registry-Version` header present on unauthenticated `/v1/models`?** → **NO.**
- **Strict-readiness fields that cannot be satisfied/verified from the deployed non-secret surface:**
  `ModelRegistryReady`, `SelectedModelReady`, `SelectedProtocolReady` (for codex and for auto/coding);
  `Live` is at risk (source `/health/live` → 404) and `Authenticated` is unverifiable — both pending
  the deployment's real endpoint config + the stable secret.

## 5. Blocker / next action (for authenticated confirmation)

- The met/unmet **values** for fields 2–5 require an authorized **non-inference** authenticated
  `GET /v1/models` metadata read (registry version + per-model `protocol`/`available` for the codex
  and auto/coding routes; headers/bounded fields only, no body). Owner: OmniRoute operator + Principal
  Orchestrator (`w5:p9`); D-V3-25(B) gates the authenticated step. Also: confirm the deployment's
  actual `EndpointSet.Liveness/Readiness` paths (source defaults `/health/live|ready` 404 here).

## 6. Non-claims
- No secret used, no inference, no OmniRoute-internal changes, no deploy; no raw body printed.
- Fields 2–5 are reported as unmet/undeterminable **from the non-secret vantage**; their true values
  are not asserted (would require the authenticated read above).
