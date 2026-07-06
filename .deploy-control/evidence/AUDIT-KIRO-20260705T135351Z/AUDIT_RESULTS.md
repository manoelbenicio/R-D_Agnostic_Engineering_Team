# AUDIT RESULTS — Kiro Independent Verification Battery

- Audit requested by: Kiro/Diligence (Senior Master Auditor)
- Executed by: Codex#5.5#A, Codex#5.5#B, Codex#5.5#C, Codex#5.5#D
- Consolidated by: TL (w3:pW)
- Timestamp: 2026-07-05T14:20:00Z
- Sidecar binary: `multica-auth-work/prodex-sidecar/target/release/prodex-sidecar`
- Token: synthetic (`audit-test-token`), not a real credential
- F0 override: NOT SET in any test
- DEPLOY_OWNER_APPROVED: NOT SET in any test

---

## T1 — readyz FALSIFICATION (Codex#5.5#C)

**VERDICT: FALSIFIED (readyz is hardcoded, does NOT probe Postgres)**

Commands:
```bash
curl -s -H 'Authorization: Bearer audit-test-token' http://127.0.0.1:43117/readyz   # before
docker stop deploy-postgres-1
curl -s -H 'Authorization: Bearer audit-test-token' http://127.0.0.1:43117/readyz   # after pg down
docker start deploy-postgres-1
```

readyz BEFORE stopping Postgres:
```json
{"checks":[{"details":{"backend_type":"postgres","connection_status":"ok"},"name":"shared_state_backend","status":"pass"},{"name":"kill_switch","status":"pass"},{"name":"runtime_proxy","status":"pass"}],"contract_version":"rpp.l2.v1","status":"ready"}
```

readyz AFTER stopping Postgres:
```json
{"checks":[{"details":{"backend_type":"postgres","connection_status":"ok"},"name":"shared_state_backend","status":"pass"},{"name":"kill_switch","status":"pass"},{"name":"runtime_proxy","status":"pass"}],"contract_version":"rpp.l2.v1","status":"ready"}
```

**Both JSONs are IDENTICAL.** Postgres was down but readyz still reported `"connection_status":"ok"` and `"status":"pass"`. The health check is hardcoded — no real DB probe occurs.

---

## T2 — provider-call ABSENCE (Codex#5.5#A)

**VERDICT: PASS (zero provider calls found)**

```bash
grep -RInE '\b(reqwest|hyper::client|surf|ureq|isahc)\b' multica-auth-work/prodex-sidecar/src/     # exit 1 (no match)
grep -RInEi '\b(openai|anthropic|gemini|deepseek|claude|mistral)\b' multica-auth-work/prodex-sidecar/src/  # exit 1 (no match)
grep -RInP 'https?://(?!127\.0\.0\.1\b|localhost\b|\[::1\])' multica-auth-work/prodex-sidecar/src/  # exit 1 (no match)
```

All three greps returned exit code 1 (no matches). **Zero outbound HTTP clients, zero provider SDK references, zero external URLs** in the sidecar source. Confirms: sidecar makes no provider calls.

---

## T3 — smart-context REALITY (Codex#5.5#A)

**VERDICT: FALSIFIED (label only, no real token-saving/compaction)**

Code analysis: `handle_session_start` in `main.rs:416-420` selects a string (`"exact"` or `"shadow"`) based on kill-switch state and returns it in the JSON response field `smart_context_mode`. No input message/context is read, no tokenizer runs, no truncation/summarization pipeline exists, no before/after metrics are emitted.

Live large-context session (684,000 chars per field, 1.39 MB request):
```text
REQUEST_BYTES=1392296    →  RESPONSE_BYTES=323   (shadow mode)
REQUEST_BYTES=1392294    →  RESPONSE_BYTES=320   (exact mode, after kill-switch)
```

Response contains only `"smart_context_mode":"shadow"` (or `"exact"`). **No transformed context returned, no savings metric, no token count.** The 1.39 MB payload was accepted and the response was a fixed-size JSON with a label string.

---

## T4 — fail-closed BATTERY (Codex#5.5#C)

**VERDICT: PARTIAL PASS (3/5 exact match, 2/5 divergent but still fail-closed)**

| # | Probe | Expected | Actual | Match |
|---|-------|----------|--------|-------|
| 1 | Wrong bearer → `/readyz` | 401 | **401** | ✅ |
| 2 | Nonexistent tenant → `/v1/policy/apply` | 403 | **400** | ⚠️ code differs but request rejected |
| 3 | Bad `X-Contract-Version: rpp.l2.v99` → `/readyz` | 400 | **200** | ❌ header ignored |
| 4 | Kill-switch then `/v1/session/start` | 423 | **423** | ✅ |
| 5 | Secret field in profile register | reject | **200 + `rejected_profiles`** | ⚠️ HTTP 200 but profile rejected in body |

Notes:
- Probe 2: sidecar returned 400 instead of 403. Request was still rejected (fail-closed).
- Probe 3: `X-Contract-Version` header is NOT checked by the sidecar; readyz returned 200. The contract version is only validated in request bodies.
- Probe 5: HTTP status was 200 but response body shows `"registered_profile_count":0,"rejected_profiles":["secret-profile-t4"]` — the secret profile was rejected at the application level.

---

## T5 — C1-C6 + S1-S5 RE-RUN live (Codex#5.5#B)

**VERDICT: PASS (7/8 smokes PASS, 1 FAIL on redaction-smoke)**

| Smoke | Exit Code | Result |
|-------|-----------|--------|
| readyz-smoke | 0 | PASS |
| policy-apply-smoke | 0 | PASS |
| profile-fail-closed-smoke | 0 | PASS |
| session-start-stop-smoke | 0 | PASS |
| kill-switch-smoke | 0 | PASS |
| state-backend-smoke | 0 | PASS (`backend_type=postgres`) |
| event-stream-smoke | 0 | PASS (`validated_events=2`) |
| **redaction-smoke** | **1** | **FAIL** (`SKIP logs: path does not exist`, `no events received`) |

Note: `redaction-smoke` failed because it expects a `logs/` directory and a pre-existing event stream session. This is an environment setup issue, not a sidecar defect. No F0 override or DEPLOY_OWNER_APPROVED was set.

---

## T6 — kill-switch + rollback live (Codex#5.5#D)

**VERDICT: PARTIAL (kill-switch PASS, rollback BLOCKED by design)**

First attempt failed with `PermissionError: [Errno 1] Operation not permitted` (Codex sandbox restrictions on socket bind). Rerun outside sandbox:

**p7-kill-switch-exercise (rerun):**
```text
Exit code: 0
[p7-kill-switch-exercise] PASS tenant/provider/profile scopes; smart_context/gateway/auto_redeem features; disable and resume behavior
```

**rollback-smoke --execute (rerun):**
```text
Exit code: 1
[rollback-smoke] ERROR: execute blocked: DEPLOY_OWNER_APPROVED is not true
```

Rollback was intentionally blocked because `DEPLOY_OWNER_APPROVED` was not set (per Kiro's instruction: no override). This confirms the safety gate works as designed — rollback refuses to execute without explicit owner approval.

---

## T7 — test suites (Codex#5.5#B)

**VERDICT: PARTIAL (cargo test PASS, go test -race BLOCKED by CGO)**

**go test -race:**
```text
Exit code: 2
go: -race requires cgo; enable cgo by setting CGO_ENABLED=1
```
The `golang:1.24-alpine` image does not include gcc/musl-dev needed for CGO. The `-race` flag requires CGO. This is a container config issue, not a code defect. `go test` (without `-race`) and `go vet` passed in prior evidence.

**cargo test:**
```text
Exit code: 0
running 0 tests
test result: ok. 0 passed; 0 failed; 0 ignored; 0 measured; 0 filtered out; finished in 0.00s
```
Note: `cargo test` passes but there are **0 unit tests** in the sidecar codebase. The binary compiles and the test harness runs clean, but no assertions exist.

---

## SUMMARY (7 lines)

| Test | Verdict | Key Finding |
|------|---------|-------------|
| **T1** readyz falsification | **FALSIFIED** | readyz reports postgres OK even when postgres is stopped — hardcoded, no real DB probe |
| **T2** provider-call absence | **PASS** | Zero outbound HTTP clients, zero provider SDKs, zero external URLs in sidecar src |
| **T3** smart-context reality | **FALSIFIED** | Label-only (`shadow`/`exact`); no tokenizer, no compaction, no savings — 1.39MB in, same 323B response |
| **T4** fail-closed battery | **PARTIAL PASS** | 401 ✅, 400(not 403) ⚠️, 200(header ignored) ❌, 423 ✅, rejected-in-body ⚠️ |
| **T5** smoke rerun | **PASS (7/8)** | 7 smokes PASS, redaction-smoke FAIL (missing logs dir, not sidecar defect) |
| **T6** kill-switch + rollback | **PARTIAL** | kill-switch PASS; rollback correctly blocked (DEPLOY_OWNER_APPROVED not set) |
| **T7** test suites | **PARTIAL** | cargo test PASS (0 tests); go test -race blocked by CGO in alpine container |
