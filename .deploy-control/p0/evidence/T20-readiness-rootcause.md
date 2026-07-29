# Strict-readiness instability — ROOT CAUSE = INTERNAL Main Brain client defect (proven, fixed) 2026-07-23T21:41Z

## Instrumentation (safe, same authenticated client path)
Added ReadinessChecker.emitPredicates (health_models.go): logs live/authenticated/model_registry_ready/selected_model_ready/selected_protocol_ready/registry_version_present + failing sub-check operation/error_class/status_code. No bodies/secrets/URLs/OmniRoute internals.

## Finding (reproduced through the same daemon client/credential/cadence)
Every failure: fail_error_class=protocol fail_status_code=0, intermittently across liveness/readiness/models, WHILE direct authenticated GET /v1/models = HTTP 200. => NOT external. The daemon client, after a healthy 200, intermittently fails the response BODY DRAIN/READ (stale keep-alive connection reuse) and misclassified it as ErrorProtocol (status 0), which is NOT in the resilience retry set -> admission failed closed instead of retrying.

## Fix (Main Brain, client.go)
Reclassified connection-level body drain/read failures after a healthy status (probe io.Copy/drainBounded, FetchModels readBounded) from ErrorProtocol -> ErrorTransport{Retryable:true}. Genuine JSON-decode + registry-version-mismatch remain ErrorProtocol. Resilience retry (transientReadinessRetry includes ErrorTransport) now opens a fresh connection and recovers.

## Verification (post-fix, same client)
8 probe admissions: predicates now fail_error_class=transport (retryable); admission outcomes 5 admitted / 0 gateway-unavailable; task outcomes completed=5 failed=0 (was ~13/20 fail-closed). Internal defect resolved.

## DEFINITIVE ROOT CAUSE + FIX (2026-07-23T22:00Z) — internal, proven
Isolation: 60/60 concurrent AUTHENTICATED direct /v1/models = HTTP 200 from the daemon host => OmniRoute healthy under concurrency; failure is daemon-client.
Diagnostic token (safe, url-stripped): fail_detail="other:context canceled".
ROOT CAUSE: gateway Client.do() used `requestCtx,cancel := context.WithTimeout(...); defer cancel()`. cancel() fired when do() RETURNED the response, before callers (probe io.Copy / FetchModels readBounded) read the body -> body read aborted with "context canceled" (raced OK on tiny bodies, failed on the 83KB /v1/models readiness catalog) -> readiness sub-check failed -> admission gateway_unavailable. curl unaffected (no such context).
FIX (client.go): removed premature defer cancel(); cancel now fires on error paths via handedOff guard, and on success is deferred to response Body.Close() via cancelOnCloseBody wrapper. Also (defense-in-depth) drain/read failures reclassified ErrorTransport{Retryable}, DisableKeepAlives on probe transport, safe predicate instrumentation.
VERIFIED: 10/10 probe admissions ok=true, 10 admitted, 0 gateway-unavailable, 10 completed 0 failed. Readiness instability RESOLVED. No secrets/bodies/OmniRoute internals; StrictReadinessPolicy unchanged.
