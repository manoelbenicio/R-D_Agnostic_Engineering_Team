# Scope & Target Selection Review Evidence: OTLP Receiver

- **Date:** 2026-07-24T00:28:30Z
- **Scope:** `multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go`, `otlpreceiver_test.go`
- **Reviewer:** Antigravity (AI Pair Programmer)

---

## Executive Summary & Verification Matrix

| Target Selection / Scope Invariant | Code Mechanism / Struct Definition | Verdict | Finding Summary |
|---|---|---|---|
| **1. Target Selection Matching** | Scope `sl.Scope.Name == "claude_code"` AND log attribute `event.name == "api_request"`. | **PASS** | `otlpreceiver.go` matches `ExpectedScopeName` (`"claude_code"`) at `scopeLogs` level and `TargetEventName` (`"api_request"`) at `logRecord` level. Both are validated independently. |
| **2. No Dependence on Invented Scope** | Standard Scope & Event constants used without arbitrary/invented fallbacks. | **PASS** | `ExpectedScopeName` (`"claude_code"`) and `TargetEventName` (`"api_request"`) are closed constants. Records with mismatched scope or event name are safely skipped without side effects. |
| **3. `ObservedScope` Leak Prevention** | Closed `SanitizedRecord` struct shape; no untrusted scope persistence. | **PASS** | `SanitizedRecord` contains NO field for `ObservedScope` or raw scope strings. Unvalidated scope strings can never be persisted by design. |
| **4. Focused Unit Tests** | Package test suite verification. | **PASS** | `TestScopeValidatedSeparatelyFromEventName`, `TestOfficialShapeAllowlistTrustedCorrelationAndContentOff`, and `TestFailClosedDropAndCountIncomplete` pass cleanly (`14/14 PASS`, `0.041s`). |

---

## Detailed Inspection Findings

### 1. Independent Scope & Event Name Validation
- **File:** [otlpreceiver.go](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go#L255-L275)
- **Mechanism:**
  ```go
  for _, sl := range rl.ScopeLogs {
      if strings.TrimSpace(sl.Scope.Name) != ExpectedScopeName {
          continue // not Claude Code telemetry — validate scope separately from event.name
      }
      for _, lr := range sl.LogRecords {
          rec, isTarget, bad := r.sanitizeLog(lr)
          if !isTarget {
              continue // scope matched but event.name != api_request — drop, not counted
          }
  ```
- **Analysis:**
  - Scope name is matched strictly against `ExpectedScopeName` (`"claude_code"`). Mismatched scope logs are ignored (`continue`).
  - Log attribute `event.name` is matched strictly against `TargetEventName` (`"api_request"`). Mismatched event records are dropped (`continue`).

---

### 2. Closed `SanitizedRecord` Struct Invariant
- **File:** [otlpreceiver.go](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go#L75-L85)
- **Struct Definition:**
  ```go
  type SanitizedRecord struct {
      EventName        string // always TargetEventName ("api_request")
      RequestID        string // log attribute request_id
      ClientRequestID  string // log attribute client_request_id
      Model            string
      Status           string
      DurationMs       int64
      StartUnixNano    uint64
      TrustedTaskID    string // resource agent_brain.task_id (trusted correlation)
      TrustedRequestID string // resource agent_brain.request_id (trusted correlation)
  }
  ```
- **Analysis:**
  - Struct fields are strictly typed and limited to trusted metadata and allowlisted identifiers.
  - There is no field to store raw scope names or unvalidated metadata strings.

---

## Test Verification Output

Ran focused test suite:
```text
=== RUN   TestOfficialShapeAllowlistTrustedCorrelationAndContentOff
--- PASS: TestOfficialShapeAllowlistTrustedCorrelationAndContentOff (0.00s)
=== RUN   TestScopeValidatedSeparatelyFromEventName
--- PASS: TestScopeValidatedSeparatelyFromEventName (0.00s)
=== RUN   TestFailClosedDropAndCountIncomplete
--- PASS: TestFailClosedDropAndCountIncomplete (0.00s)
PASS
ok  	github.com/multica-ai/multica/server/internal/daemon/observability/otlpreceiver	0.041s
```
All **14 package tests** passed cleanly.
