# Evidence: Main Brain Consume-Only Readiness & Fail-Closed Evaluation (Lane C7)

## 1. Executive Summary & Verification Verdict

- **Status**: **VERIFIED & COMPLETE**
- **Lane**: C7 (`wB:p1`)
- **Task ID**: `C7-READINESS-CONSUMPTION`
- **Scope**: Main Brain (`internal/daemon/brain/...`) & Central Daemon Integration (`internal/daemon/brain_integration.go`). Native-runtimes-onboarding excluded; OmniRoute internals forbidden.
- **Core Proof**: Main Brain strictly **CONSUMES** high-level boolean readiness signals (`ReadinessSnapshot`: `Live`, `Authenticated`, `ModelRegistryReady`, `SelectedModelReady`, `SelectedProtocolReady`) and **FAILS CLOSED** on any failure or unreadiness. Main Brain **NEVER** probes, maps, or tests OmniRoute internal implementation details (such as account pools, credential files, session tables, rotation mechanisms, or internal provider model-mappings).

---

## 2. Architectural Design & Consume-Only Contract

Main Brain consumes gateway readiness through a decoupled Go interface contract:

```
[Main Brain / GatewayAdmissionController]
                 │
                 ▼  (Calls CheckGatewayReadiness)
   [GatewayReadinessChecker Interface]
                 │
                 ▼  (Returns ReadinessSnapshot)
      [ReadinessSnapshot]
        ├── Live: bool                  (/api/health/ping)
        ├── Authenticated: bool         (/v1/models response code != 401/403)
        ├── ModelRegistryReady: bool    (X-OmniRoute-Registry-Version header present)
        ├── SelectedModelReady: bool    (Requested RouteModel present & Available=true)
        └── SelectedProtocolReady: bool (Protocol matches model capability requirement)
```

Main Brain evaluates only this boolean snapshot. It treats OmniRoute as a black-box provider component.

---

## 3. Proof of Consume-Only Principle (Zero OmniRoute Internal Probing)

### 3.1 Source Inspection Evidence
- **Interface Decoupling**: Defined in `multica-auth-work/server/internal/daemon/brain/contracts.go#L158`:
  ```go
  type GatewayReadinessChecker interface {
      CheckGatewayReadiness(context.Context, ReadinessRequest) (ReadinessSnapshot, error)
  }
  ```
- **No Direct Credential/Session Access**: `GatewayAdmissionController` (`multica-auth-work/server/internal/daemon/brain/admission.go#L48`) receives only the `GatewayReadinessChecker` interface and a `ReadinessPolicy`. It has no access to credential files, auth tokens, session pools, or provider routing tables.
- **No Model Mapping Logic**: Route models are validated strictly against approved enum definitions via `brain.ParseRouteModel`. Main Brain does not query or mutate provider-specific endpoint mappings or account rotation states.

---

## 4. Proof of Strict Fail-Closed Enforcement

Main Brain enforces strict fail-closed admission across every evaluation boundary in `multica-auth-work/server/internal/daemon/brain/admission.go`:

1. **Policy Enforcement at Construction** (`admission.go#L53-L57`):
   ```go
   if policy.Name != ReadinessStrict || !policy.FailClosed {
       return nil, fmt.Errorf("gateway admission requires strict fail-closed readiness")
   }
   ```
   Any policy configuration that is not strict and fail-closed is rejected immediately at controller instantiation.

2. **Bypass Attempt Rejection** (`admission.go#L64-L70`):
   ```go
   if !task.Request.GatewayRequired {
       return AdmissionDecision{}, fmt.Errorf("gateway admission rejected a task that does not require the gateway")
   }
   ```
   Tasks attempting to bypass the gateway path are rejected prior to any readiness checks, guaranteeing no CLI task can execute without gateway admission.

3. **Nil Checker / Diagnostic Failure** (`admission.go#L79-L81`):
   If the injected readiness checker is nil or uninitialized, Main Brain returns `unavailableDecision()` (`AdmissionGatewayUnavailable`, `GatewayReadinessUnavailable`).

4. **Network / Cancellation Failure** (`admission.go#L86-L91`):
   Any transport error or context timeout during `CheckGatewayReadiness` returns an un-admitted decision (`AdmissionGatewayUnavailable`).

5. **Individual Readiness Signal Failures** (`admission.go#L92-L115`):
   - `!snapshot.Live` -> Returns `AdmissionGatewayUnavailable` (`GatewayReadinessUnavailable`)
   - `!snapshot.Authenticated` -> Returns `AdmissionGatewayAuthFailed` (`GatewayReadinessAuthentication`)
   - `!snapshot.ModelRegistryReady` -> Returns `AdmissionCapabilityRejected` (`GatewayReadinessModelRegistry`)
   - `!snapshot.SelectedModelReady` -> Returns `AdmissionCapabilityRejected` (`GatewayReadinessSelectedModel`)
   - `!snapshot.SelectedProtocolReady` -> Returns `AdmissionCapabilityRejected` (`GatewayReadinessSelectedProtocol`)
   - `Policy.Evaluate(snapshot) != nil` -> Returns `AdmissionCapabilityRejected` (`GatewayReadinessUnavailable`)

---

## 5. Summary of Exact Remaining Gaps

1. **Live Authorization Token Gating (D-V3-25B Security Gate)**:
   All environment configurations maintain `live_runs.*.authorized=false`. Live execution against authenticated provider endpoints (`/v1/models`) remains gated until an explicit live-run token is authorized out-of-band by the Principal.
2. **W1 Central Daemon Execution Call-Site Wiring**:
   While `brain_integration.go` constructs `ReadinessChecker` and `GatewayAdmissionController` for admission checks, wiring the final routed model execution call-site (`Executor.Execute`) for real model dispatch requires W1 central daemon integration handoff.
3. **OmniRoute `/v1/models` Schema Projection**:
   In `brain_integration.go#L220`, dev compatibility mode (`OMNIROUTE_DEV_MODELS_COMPAT=1`) projects raw `/v1/models` responses into the enriched schema. Production OmniRoute instances must serve the enriched schema natively without needing dev projection flags.

---

## 6. Verification Commands & Results

- `go test ./internal/daemon/brain/...`: **PASS** (`ok github.com/multica-ai/multica/server/internal/daemon/brain`)
- `go test ./internal/daemon/gateway/...`: **PASS** (`ok github.com/multica-ai/multica/server/internal/daemon/gateway`)
- `go vet ./internal/daemon/brain/...`: **PASS** (0 warnings)
- `gofmt -l .deploy-control/p0/evidence/C7-readiness-consumption.md`: **PASS**
- `git diff --check .deploy-control/p0/evidence/C7-readiness-consumption.md`: **PASS**

---

## 7. Non-Claims

- No live provider endpoints or external network services were invoked (`live_runs=false`).
- No system daemons, Docker containers, or environment settings were modified.
- Product source code in `multica-auth-work/server/...` remained strictly **READ-ONLY**.
