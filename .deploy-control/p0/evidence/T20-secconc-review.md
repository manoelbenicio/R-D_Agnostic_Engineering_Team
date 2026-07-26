# T20 Security & Concurrency Review Evidence

- **Date:** 2026-07-23T22:38:00Z
- **Scope:** `multica-auth-work/server/internal/daemon/gateway/client.go`, `brain_integration.go`, `daemon.go`
- **Reviewer:** Antigravity (AI Pair Programmer)

---

## Executive Summary

| Item | Focus / Invariants Assessed | Verdict | Key Findings |
|---|---|---|---|
| **1. `cancelOnCloseBody` & context lifecycle** | Double-close idempotency, context leak prevention, error-path context cancellation. | **PASS** | `do()` defers context cancellation if `handedOff` is false. On 2xx success, `handedOff = true` and `cancel` is stored in `cancelOnCloseBody`. `Close()` calls `b.cancel()`, which is safe and idempotent in Go `context`. Non-2xx responses close the body and cancel context immediately. |
| **2. `classifyBodyReadError` classification** | `errResponseTooLarge` (protocol, non-retryable) vs connection read error (transport, retryable). | **PASS** | `classifyBodyReadError` checks `errors.Is(err, errResponseTooLarge)` returning `ErrorProtocol` (`Detail: "response_too_large"`), non-retryable. Any other read error returns `ErrorTransport` with `Retryable: true`. |
| **3. `safeErrorToken` diagnostics** | Sanitization against URL, host, port, credentials, or body payload leakage. | **PASS** | Unwraps `*url.Error` to strip the target URL. Pattern matches error string to standard tokens (`"unexpected_eof"`, `"conn_reset"`, `"dial"`, `"tls"`, etc.) and truncates unknown errors to 80 chars with `other:` prefix after URL removal. |
| **4. Transport `DisableKeepAlives`** | Stale connection reuse prevention for control-plane probes under proxy close. | **PASS** | `defaultHTTPClient` sets `transport.DisableKeepAlives = true` to force fresh connections for control-plane probes, avoiding stale pooled connections prematurely closed by OmniRoute fronting proxies. |
| **5. `admitGroup` singleflight** | Singleflight coalescing for `/v1/models` without serializing admitted task execution. | **PASS** | `r.admitGroup.Do("gateway-readiness", ...)` coalesces concurrent readiness probes into a single evaluation using a detached context `context.WithoutCancel(ctx)`. Admitted verdict fans out to callers while capacity leases are acquired independently per task. |
| **6. `holdForCapacityBarrier` gating** | Dev + gateway-required gating and context cancellation. | **PASS** | Gated strictly by `d.cfg.AgentBrain.DevelopmentEnabled && d.cfg.AgentBrain.Neutral.Gateway.Required` and `AGENT_BRAIN_TEST_TASK_HOLD_MS > 0`. Honors `ctx.Done()` via `select`. |
| **7. `StrictReadinessPolicy` invariant** | Contract unchanged and fail-closed behavior preserved. | **PASS** | `StrictReadinessPolicy()` remains unchanged (`ReadinessStrict`, `FailClosed: true`, all 5 checks required). `admitRetryLoop` only returns fresh ready decisions and fails closed on deadline exhaustion or deterministic rejection. |

---

## Detailed Item Analyses

### Item 1: `cancelOnCloseBody` & Handed-Off Cancel Guard
- **File:** [client.go](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/gateway/client.go#L221-L293)
- **Mechanism:**
  ```go
  requestCtx, cancel := context.WithTimeout(ctx, c.requestTimeout)
  handedOff := false
  defer func() {
      if !handedOff {
          cancel()
      }
  }()
  ```
  On non-2xx status or transport/credential error, `handedOff` remains `false`, executing `cancel()` immediately on function exit and closing response body if present.
  On status `2xx`, `response.Body = &cancelOnCloseBody{ReadCloser: response.Body, cancel: cancel}` wraps the body and sets `handedOff = true`.
- **Idempotency & Leaks:** Calling `b.Close()` invokes `b.cancel()`. In Go, `context.CancelFunc` is completely safe and idempotent across multiple invocations. If `Close()` is called multiple times, subsequent calls safely invoke `cancel()` without panic or side effects.
- **Verdict:** **PASS**

### Item 2: `classifyBodyReadError` Error Classification
- **File:** [client.go](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/gateway/client.go#L400-L430)
- **Mechanism:**
  ```go
  func classifyBodyReadError(operation string, err error) *GatewayError {
      if errors.Is(err, errResponseTooLarge) {
          return &GatewayError{Operation: operation, Class: ErrorProtocol, Detail: "response_too_large"}
      }
      return &GatewayError{Operation: operation, Class: ErrorTransport, Retryable: true, Detail: safeErrorToken(err)}
  }
  ```
- **Analysis:** `errResponseTooLarge` signifies a size limit breach (protocol/policy violation), correctly classified as `ErrorProtocol` with `Retryable: false` (default false for GatewayError struct). Other read errors (such as unexpected EOF, connection reset during body read) are classified as `ErrorTransport` with `Retryable: true`.
- **Verdict:** **PASS**

### Item 3: `safeErrorToken` Sanitization
- **File:** [client.go](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/gateway/client.go#L313-L352)
- **Mechanism:**
  - Extracts underlying cause if `err` is a `*url.Error` (`err = ue.Err`), discarding `ue.URL`.
  - Matches against generic strings (`"unexpected EOF"`, `"connection reset"`, `"dial "`, etc.).
  - Default case caps sanitized string to 80 characters after URL stripping.
- **Security Check:** Prevents leaking credentials in query parameters, authorization headers, target IP/ports, or sensitive path tokens into logs and error details.
- **Verdict:** **PASS**

### Item 4: `DisableKeepAlives` in Default HTTP Client
- **File:** [client.go](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/gateway/client.go#L122-L137)
- **Mechanism:** `transport.DisableKeepAlives = true` is configured on `defaultHTTPClient`.
- **Analysis:** Fronting proxies (e.g. OmniRoute) close idle keep-alive connections earlier than standard client timeouts, leading to flaky `EOF` or `connection reset` errors on low-frequency control-plane probes. Disabling keep-alives forces a clean TCP connection per request.
- **Verdict:** **PASS**

### Item 5: `admitGroup` Singleflight Coalescing
- **File:** [brain_integration.go](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/brain_integration.go#L200-L224)
- **Mechanism:**
  ```go
  v, _, _ := r.admitGroup.Do("gateway-readiness", func() (interface{}, error) {
      rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), readinessAdmissionWait()+5*time.Second)
      defer cancel()
      d, e := admitRetryLoop(rctx, readinessAdmissionWait(), func(c context.Context) (brain.AdmissionDecision, error) {
          return admission.Admit(c, task)
      })
      return admitResult{decision: d, err: e}, nil
  })
  ```
- **Analysis:**
  - Coalesces concurrent readiness checks into one execution of `admitRetryLoop`.
  - Detaches context via `context.WithoutCancel(ctx)` so one caller's cancellation does not cancel the shared in-flight probe for all other waiters.
  - Returns the shared `admitResult` to all concurrent waiters without serializing subsequent task execution or capacity lease acquisition.
- **Verdict:** **PASS**

### Item 6: `holdForCapacityBarrier` Concurrency Barrier
- **File:** [daemon.go](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/daemon.go#L3245-L3269)
- **Mechanism:**
  ```go
  if !(d.cfg.AgentBrain.DevelopmentEnabled && d.cfg.AgentBrain.Neutral.Gateway.Required) {
      return nil
  }
  ...
  select {
  case <-ctx.Done():
      return ctx.Err()
  case <-time.After(time.Duration(ms) * time.Millisecond):
      return nil
  }
  ```
- **Analysis:** Strictly gated on both `DevelopmentEnabled` and `Neutral.Gateway.Required`. Read from `AGENT_BRAIN_TEST_TASK_HOLD_MS`. Immediately returns if disabled or unset. Listens to `ctx.Done()` for clean cancellation.
- **Verdict:** **PASS**

### Item 7: `StrictReadinessPolicy` Unchanged
- **Files:** [config.go](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/brain/config.go#L110-L121), [brain_integration.go](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/brain_integration.go#L178-L266)
- **Mechanism:** `StrictReadinessPolicy()` defines `ReadinessStrict` with all checks set to `true` and `FailClosed: true`. `admitRetryLoop` retries transient errors within a bounded window, but fails closed if the window is exhausted or if a deterministic rejection occurs.
- **Verdict:** **PASS**

---

## Test Verification Summary

Ran unit test suite across gateway and daemon packages:
- `TestClientDoDoesNotCancelBeforeLargeBodyRead`: **PASS**
- `TestClientDoNonSuccessCancelsAndClosesImmediately`: **PASS**
- `TestClientDoBodyCloseIsIdempotent`: **PASS**
- `TestRegistryTransientThenSuccessAdmitsFreshReady`: **PASS**
- `TestAdmitRetryLoop_TransientThenFreshReady`: **PASS**
- `TestAdmitRetryLoop_PersistentFailsClosed`: **PASS**
- `TestAdmitRetryLoop_DeterministicRejectionImmediate`: **PASS**
