# External Operator Ticket: OmniRoute Deployed Readiness & Catalog Access

**Ticket ID**: `TICKET-OR-20260723-READINESS-01`  
**Target System**: `OmniRoute Model Gateway (ORQ1)`  
**Target Endpoint**: `http://100.118.244.61:20128` (Tailscale)  
**Severity**: `Blocker` (Blocks OpenSpec Task 6.1 Deployed Readiness & Task 6.5 Single Live Acceptance Run)  
**Submitter**: Main Brain Orchestration Team / Kiro Supervisor  
**Timestamp**: 2026-07-23T00:18:00Z  
**Policy Compliance**: Based strictly on factual HTTP probe evidence (`F3-deployed-readiness.md`). Zero secret keys read, zero inference executed, zero internal diagnosis speculation.  

---

## 1. Summary of Issue

The Main Brain agent system requires a non-secret or metadata-only mechanism to verify OmniRoute gateway liveness, registry schema version (`X-OmniRoute-Registry-Version`), and selected model availability (`cp/cline-pass/glm-5.2`, `claude_code_kimi_2.7_Code`).

Currently, the ORQ1 Tailscale endpoint (`http://100.118.244.61:20128`) is **live and reachable**, but the model catalog (`/v1/models`) returns **HTTP `401 Unauthorized`** (`REQUIRE_API_KEY=true`), and standard unauthenticated health paths (`/health/live`, `/health/ready`) return **HTTP `404 Not Found`**. Because Main Brain probing lanes operate under a strict non-secret posture (`SecretsPresent == false`), the system cannot verify registry revision consistency or model availability, leaving Task 6.1 deployed readiness **BLOCKED-external**.

---

## 2. Expected Readiness Contract (Main Brain)

Main Brain's `GatewayAdmissionController` (`internal/daemon/brain/admission.go`) enforces a strict, fail-closed readiness contract (`ReadinessStrict`). Prior to process launch, it evaluates five boolean gates:

1. **`Live`**: Base TCP/HTTP reachability.
2. **`Authenticated`**: Gateway catalog access permitted without 401/403 errors.
3. **`ModelRegistryReady`**: Presence of `X-OmniRoute-Registry-Version` header.
4. **`SelectedModelReady`**: Requested model ID present in registry catalog with `Available == true`.
5. **`SelectedProtocolReady`**: Model protocol capability matches requirement (`openai-chat` or `anthropic-messages`).

To satisfy this contract without exposing inference credentials to probing lanes, Main Brain requires either a **non-secret readiness/revision endpoint** or a **scoped metadata-only API key**.

---

## 3. Observed Probing & Readiness Results

Direct empirical probe findings against `http://100.118.244.61:20128` (from evidence file `F3-deployed-readiness.md`):

| Endpoint Probe | HTTP Status | Curl Exit Code | Observed Behavior / Headers |
|---|---:|---:|---|
| `GET /status` | **200 OK** | `0` | `Content-Type: text/html` (Dashboard UI present, sub-40ms latency) |
| `GET /` | **307 Temporary Redirect** | `0` | Redirects to `/dashboard` |
| `GET /v1/models` | **401 Unauthorized** | `0` | `x-omniroute-route-class: CLIENT_API`; `REQUIRE_API_KEY` enforced. **No `X-OmniRoute-Registry-Version` header present on 401 response.** |
| `GET /v1/health` | **401 Unauthorized** | `0` | Auth-gated |
| `GET /health/live` | **404 Not Found** | `0` | Source-configured path not served on this deployment |
| `GET /health/ready` | **404 Not Found** | `0` | Source-configured path not served on this deployment |
| `GET /healthz`, `/version`, `/ping` | **404 Not Found** | `0` | No non-secret version/health path found |

### Endpoint Health Metadata
- **Host**: `100.118.244.61` (ORQ1 Tailscale)
- **Port**: `20128`
- **Network Status**: Reachable (`curl_rc = 0`), sub-40ms TCP latency.
- **Security Posture**: API-key enforcement confirmed active.

---

## 4. Impact on Main Brain Deployment

1. **Task 6.1 Status**: Main Brain's source-level fail-closed code is 100% verified (`C7-readiness-consumption.md`), but **deployed 6.1 readiness proof remains `BLOCKED-external`** because `/v1/models` is auth-gated and no non-secret version/readiness endpoint exists.
2. **Task 6.5 Status**: The single live acceptance run (`live_runs.*.authorized = false`) cannot be executed until readiness is verified.

---

## 5. Target Auto-Coding Models Pending Readiness Declaration

The following model IDs are declared in Main Brain contracts and await ready status in the OmniRoute catalog:

1. **`cp/cline-pass/glm-5.2`** (Primary GLM auto-coding, OpenAI Chat `/v1/chat/completions`)
2. **`nvidia/z-ai/glm-5.2`** (Fallback GLM auto-coding, OpenAI Chat `/v1/chat/completions`)
3. **`claude_code_kimi_2.7_Code`** (Claude Code auto-coding, Anthropic Messages `/v1/messages`)

---

## 6. Requested Operator Action

Please perform **either Option A (Recommended)** or **Option B**:

### Option A (Recommended — Non-Secret Readiness Endpoint)
Expose an unauthenticated liveness/readiness/version endpoint (e.g. `/healthz`, `/version`, or unauthenticated `/v1/models` headers) that returns:
- Header: `X-OmniRoute-Registry-Version: <version_string>`
- Body (or headers): Per-model `available` status for `cp/cline-pass/glm-5.2` and `claude_code_kimi_2.7_Code`.

### Option B (Scoped Metadata API Key)
Provide a scoped, non-inference API key permitted exclusively for `GET /v1/models` metadata reads, allowing Main Brain readiness verification to inspect `X-OmniRoute-Registry-Version` and model availability prior to process launch.

---

**Attachment / Reference**:  
- `F3-deployed-readiness.md` (Empirical probe log)  
- `READINESS-routes.md` (Declared route model IDs)  
- `C7-readiness-consumption.md` (Main Brain consume-only contract)  
