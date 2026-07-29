# T20 OTLP Receiver Source Code Security Audit (v2 — Corrected Official-Shape Receiver)

- **Date:** 2026-07-24T00:10:00Z
- **Scope:** `multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go`, `otlpreceiver_test.go`
- **Reviewer:** Antigravity (Independent Security & Pair Programming AI Agent)
- **Status:** **RE-AUDIT COMPLETED — 7 PASS / 1 CONCURRENCY RACE FINDING DETECTED**

---

## Executive Summary & PASS/FAIL Itemized Findings

This security re-audit evaluates the corrected official-shape OTLP HTTP/JSON receiver (`otlpreceiver.go`) designed for Claude Code telemetry ingestion (`scope.name == "claude_code"` and `logRecord.attributes["event.name"] == "api_request"`).

| # | Security / Architecture Invariant | Finding | Primary Source Code Reference (`file:line`) | Description & Risk Assessment |
|---|---|---|---|---|
| **1** | **Resource Trust Boundary** | **PASS** | [`otlpreceiver.go:L237-L242`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go#L237-L242), [`L262-L263`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go#L262-L263) | `agent_brain.task_id` and `agent_brain.request_id` are extracted strictly from `resourceLogs.resource.attributes` (injected by trusted daemon). Log-record attributes cannot overwrite resource correlation. |
| **2** | **Spoofing Assumptions (Loopback Child)** | **PASS** | [`otlpreceiver.go:L148-L159`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go#L148-L159), [`L201-L204`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go#L201-L204), [`L446-L460`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go#L446-L460) | Binds strictly to `127.0.0.1:port`. `isLoopbackRemote` validates `req.RemoteAddr` via `net.ParseIP` and `ip.IsLoopback()`, rejecting non-loopback clients with HTTP 403. |
| **3** | **Header / Content Omission** | **PASS** | [`otlpreceiver.go:L181-L185`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go#L181-L185), [`L205-L208`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go#L205-L208), [`L356-L368`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go#L356-L368) | Log `body` field is intentionally omitted from Go JSON struct definitions (never parsed). HTTP headers except `Content-Type` are ignored. Zero logging libraries or `Authorization` header reads. |
| **4** | **Bounds & Hard Resource Caps** | **PASS** | [`otlpreceiver.go:L214`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go#L214) (`MaxBytesReader`), [`L249`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go#L249) (`MaxRecords`), [`L315`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go#L315) (`MaxDedupKeys`), [`L421`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go#L421) (`MaxValueLen`) | Body (256 KiB cap), records (1000 cap), values (512 bytes cap), FIFO dedup key table (65536 cap), server read/write timeouts (10s). |
| **5** | **Listener Lifecycle & Shutdown** | **PASS** | [`otlpreceiver.go:L148-L159`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go#L148-L159) | Standard `*http.Server` construct with explicit `ReadTimeout`, `WriteTimeout`, `IdleTimeout`, `ReadHeaderTimeout`, `MaxHeaderBytes`. Supports `Shutdown(ctx)`. |
| **6** | **Persisted `Span` Conversion Suitability** | **PASS** | [`otlpreceiver.go:L74-L84`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go#L74-L84), [`contract.go:L375-L379`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/e2e/contract.go#L375-L379) | `SanitizedRecord` fields map 1-to-1 to `e2e.HopRoute` span requirements (`RequestID`, `TrustedTaskID`, `TrustedRequestID`, `Model`, `Status`, `DurationMs`). |
| **7** | **Incomplete Correlation Fail-Closed Drop** | **PASS** | [`otlpreceiver.go:L266-L269`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go#L266-L269) | A record missing `RequestID`, `TrustedTaskID`, or `TrustedRequestID` is dropped, not persisted, and counted in `Stats.DroppedIncomplete`. |
| **8** | **Idempotency & Deduplication Concurrency** | **FAIL (RACE)** | [`otlpreceiver.go:L283-L290`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go#L283-L290), [`L309-L323`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go#L309-L323) | **Concurrency State Bug:** `reserve(key)` marks `seen[key]` BEFORE `sink.Record()` succeeds. A concurrent request with the same key arriving during sink execution gets `reserve() == false`, assumes dedup success, and returns `HTTP 200 OK`. If the leader sink write fails, leader releases the key and returns `503`, leaving follower with a false `200 OK` response while **zero** records were persisted. |

---

## Detailed Analyses & Audit Evidence

### 1. Resource Trust Boundary
- **Code Reference:** [`otlpreceiver.go:L237-L242`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go#L237-L242)
  ```go
  trustedTask, taskOK := r.trustedResourceValue(rl.Resource, resAttrTaskID)
  trustedReq, reqOK := r.trustedResourceValue(rl.Resource, resAttrRequestID)
  ```
- **Analysis:** `agent_brain.task_id` and `agent_brain.request_id` are extracted strictly from `resourceLogs.resource.attributes` (injected into the child environment by the daemon). Log record attributes are evaluated separately in `sanitizeLog()` and are never merged into `TrustedTaskID` or `TrustedRequestID`.
- **Verdict:** **PASS**

### 2. Spoofing Assumptions (Loopback Child)
- **Code Reference:** [`otlpreceiver.go:L201-L204`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go#L201-L204), [`L446-L460`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go#L446-L460)
  ```go
  if !isLoopbackRemote(req.RemoteAddr) {
      http.Error(w, "forbidden", http.StatusForbidden)
      return
  }
  ```
- **Analysis:** Server binds to `127.0.0.1:port`. `isLoopbackRemote` parses host IP with `net.ParseIP` and checks `ip.IsLoopback()`. Requests from non-loopback IP addresses are immediately rejected with `403 Forbidden`.
- **Verdict:** **PASS**

### 3. Header & Content Omission
- **Code Reference:** [`otlpreceiver.go:L181-L185`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go#L181-L185)
  ```go
  type logRecord struct {
      TimeUnixNano string     `json:"timeUnixNano"`
      Attributes   []keyValue `json:"attributes"`
      // Body intentionally omitted: content is never read.
  }
  ```
- **Analysis:** The `body` element of OTLP log records is omitted from the Go struct model, so `json.Unmarshal` ignores it completely. Request headers (except `Content-Type`) are never read or logged. No `log` or `slog` packages are imported.
- **Verdict:** **PASS**

### 4. Bounds & Hard Limits
- **Code Reference:** [`otlpreceiver.go:L214`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go#L214) (`MaxBytesReader`), [`L249`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go#L249) (`MaxRecords`), [`L421`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go#L421) (`MaxValueLen`), [`L315`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go#L315) (`MaxDedupKeys`).
- **Analysis:** Payload sizes >256 KiB return 400 Bad Request. Log record count >1000 returns 413 Payload Too Large. Attribute scalar length >512 or containing control characters returns 400 Bad Request. Dedup ring buffer is hard-capped at 65536 entries.
- **Verdict:** **PASS**

### 5. Listener Lifecycle
- **Code Reference:** [`otlpreceiver.go:L148-L159`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go#L148-L159)
- **Analysis:** `LoopbackServer` configures bounded `ReadTimeout`, `ReadHeaderTimeout`, `WriteTimeout`, `IdleTimeout`, and `MaxHeaderBytes` on a standard Go `*http.Server`.
- **Verdict:** **PASS**

### 6. Persisted `Span` Conversion Suitability
- **Code Reference:** [`otlpreceiver.go:L74-L84`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go#L74-L84), [`contract.go:L375-L379`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/e2e/contract.go#L375-L379)
- **Analysis:** `SanitizedRecord` contains `RequestID`, `TrustedTaskID`, `TrustedRequestID`, `Model`, `Status`, `DurationMs`, and `StartUnixNano`. These fields directly populate an `e2e.HopRoute` span (hop 5 in the correlation contract).
- **Verdict:** **PASS**

### 7. Incomplete Correlation Fail-Closed Drop
- **Code Reference:** [`otlpreceiver.go:L266-L269`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go#L266-L269)
  ```go
  if rec.RequestID == "" || rec.TrustedTaskID == "" || rec.TrustedRequestID == "" {
      incomplete++
      continue
  }
  ```
- **Analysis:** Target records missing any of the 3 required correlation keys are dropped, counted in `Stats.DroppedIncomplete`, and omitted from persistence.
- **Verdict:** **PASS**

### 8. Concurrency Bug & Race Condition in Deduplication (FAIL)
- **Code Reference:** [`otlpreceiver.go:L283-L290`](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/otlpreceiver/otlpreceiver.go#L283-L290)
  ```go
  if !r.reserve(p.key) {
      dups++ // idempotency: already persisted (OTLP retry) — skip
      continue
  }
  if err := r.sink.Record(p.rec); err != nil {
      r.release(p.key) // not persisted -> allow a later retry to re-persist
      ...
  }
  ```
- **Vulnerability Mechanism:**
  1. Request A (Leader) calls `r.reserve(key)`. `seen[key]` is set to `struct{}{}`.
  2. Request A enters `r.sink.Record(p.rec)` (which may block or take time due to I/O).
  3. Request B (Follower) with the *exact same idempotency key* arrives concurrently.
  4. Request B calls `r.reserve(key)`. Since `seen[key]` is already set by Request A, `r.reserve(key)` returns `false`.
  5. Request B treats this as "already persisted", increments `dups++`, skips sink writing, and finishes with `HTTP 200 OK`.
  6. Request A's `r.sink.Record(p.rec)` encounters an error (e.g. disk/db error, transient failure) and returns non-nil.
  7. Request A calls `r.release(key)` and returns `HTTP 503 Service Unavailable`.
  8. **Result:** Request B returned `HTTP 200 OK` claiming deduplication success, but **ZERO** records were actually persisted in the sink!
- **Recommended Remediation:**
  Use a 3-state key tracking model (`in-flight`, `persisted`, or mutex singleflight) so concurrent requests wait for in-flight sink writes to complete before deciding whether to dedup or retry, or only mark keys as `seen` *after* `r.sink.Record()` returns success.

---

## Audit Conclusion

The corrected OTLP receiver source code (`otlpreceiver.go`) successfully satisfies 7 of 8 critical security invariants (including strict loopback binding, resource trust boundary, content/identity omission, and fail-closed dropping of incomplete records).

However, **Item 8 (Deduplication Concurrency)** receives a **FAIL** finding due to a state race condition where concurrent identical requests can return a false `200 OK` while zero records are written to disk upon leader sink failure.
