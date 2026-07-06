# T3 Smart Context Reality

- Agent: Codex#5.5#A
- Timestamp: 2026-07-05T14:02:55Z
- Scope reviewed:
  - `multica-auth-work/prodex-sidecar/src/`
  - `multica-auth-work/server`
- Result: PASS for the expected finding. Current implementation is label/config only; no real token saving or smart compaction logic was found.

## Keyword Search

Broad command:

```bash
rg -n -i 'smart_context|shadow|exact|compaction|token_saving|summarize' multica-auth-work/prodex-sidecar multica-auth-work/server
```

The broad search returned many generic `exact`/`shadow` matches unrelated to smart context. Counts across code using substring matching:

```text
smart_context=18
shadow=52
exact=569
compaction=1
token_saving=0
summarize=25
```

Focused smart-context path:

```bash
rg -n -i 'smart_context|SmartContext|PRODEX_SMART_CONTEXT|smart context|token_saving|compaction|summarize|\bshadow\b|\bexact\b' \
  multica-auth-work/prodex-sidecar/src/main.rs \
  multica-auth-work/server/internal/daemon/prodex.go \
  multica-auth-work/server/internal/daemon/prodex_test.go \
  multica-auth-work/server/internal/l2runtime/client.go \
  multica-auth-work/server/internal/l2runtime/client_test.go
```

Output:

```text
multica-auth-work/prodex-sidecar/src/main.rs:273:            .unwrap_or("smart_context")
multica-auth-work/prodex-sidecar/src/main.rs:416:    let smart_context_mode =
multica-auth-work/prodex-sidecar/src/main.rs:417:        if switch_applies(tenant_id, provider, profile_id, &session_id, "smart_context") {
multica-auth-work/prodex-sidecar/src/main.rs:418:            "exact"
multica-auth-work/prodex-sidecar/src/main.rs:420:            "shadow"
multica-auth-work/prodex-sidecar/src/main.rs:445:        "smart_context_mode": smart_context_mode
multica-auth-work/server/internal/l2runtime/client_test.go:105:		if req.ContractVersion != ContractVersion || req.Feature != "smart_context" || req.State != "disabled" {
multica-auth-work/server/internal/l2runtime/client_test.go:336:		Feature:     "smart_context",
multica-auth-work/server/internal/daemon/prodex_test.go:61:	if !cfg.SmartContextShadow || cfg.SmartContextCanary != "0" || !cfg.KillSwitchDefaultOn {
multica-auth-work/server/internal/daemon/prodex_test.go:71:		SmartContextShadow:  true,
multica-auth-work/server/internal/daemon/prodex_test.go:72:		SmartContextCanary:  "1",
multica-auth-work/server/internal/daemon/prodex_test.go:89:		env["PRODEX_SMART_CONTEXT_SHADOW"] != "1" ||
multica-auth-work/server/internal/daemon/prodex_test.go:90:		env["PRODEX_SMART_CONTEXT_CANARY_PERCENT"] != "1" ||
multica-auth-work/server/internal/daemon/prodex_test.go:97:	for _, key := range []string{"PRODEX_HOME", "PRODEX_SMART_CONTEXT_SHADOW", "MULTICA_PRODEX_COMMIT"} {
multica-auth-work/server/internal/daemon/prodex.go:40:		SmartContextShadow:  envBoolDefault("MULTICA_PRODEX_SMART_CONTEXT_SHADOW", true),
multica-auth-work/server/internal/daemon/prodex.go:41:		SmartContextCanary:  strings.TrimSpace(os.Getenv("MULTICA_PRODEX_SMART_CONTEXT_CANARY_PERCENT")),
multica-auth-work/server/internal/daemon/prodex.go:44:	if cfg.SmartContextCanary == "" {
multica-auth-work/server/internal/daemon/prodex.go:45:		cfg.SmartContextCanary = "0"
multica-auth-work/server/internal/daemon/prodex.go:96:	if d.cfg.Prodex.SmartContextShadow {
multica-auth-work/server/internal/daemon/prodex.go:97:		agentEnv["PRODEX_SMART_CONTEXT_SHADOW"] = "1"
multica-auth-work/server/internal/daemon/prodex.go:99:	if d.cfg.Prodex.SmartContextCanary != "" {
multica-auth-work/server/internal/daemon/prodex.go:100:		agentEnv["PRODEX_SMART_CONTEXT_CANARY_PERCENT"] = d.cfg.Prodex.SmartContextCanary
multica-auth-work/server/internal/l2runtime/client.go:40:	"smart_context":   {},
multica-auth-work/server/internal/l2runtime/client.go:90:	"smart_context": {},
multica-auth-work/server/internal/l2runtime/client.go:272:	SmartContext         map[string]any       `json:"smart_context,omitempty"`
multica-auth-work/server/internal/l2runtime/client.go:286:	SmartContextMode string `json:"smart_context_mode"`
```

Token-reduction keyword search in the active smart-context path:

```bash
rg -n -i 'token_count|tokens|before.*after|after.*before|truncate|truncat|compact|compaction|summar|token_saving|savings|saving' \
  multica-auth-work/prodex-sidecar/src/main.rs \
  multica-auth-work/server/internal/daemon/prodex.go \
  multica-auth-work/server/internal/l2runtime/client.go
```

Output:

```text
multica-auth-work/server/internal/l2runtime/client.go:69:	"spend_savings":       {},
multica-auth-work/server/internal/l2runtime/client.go:119:	"spend_savings":      {},
multica-auth-work/server/internal/l2runtime/client.go:136:	"spend_savings":    {"tenant_id", "session_id", "runtime_request_id", "spend_savings"},
```

## Code Path Analysis

Sidecar `session/start` chooses a string and returns it:

```text
multica-auth-work/prodex-sidecar/src/main.rs:416:    let smart_context_mode =
multica-auth-work/prodex-sidecar/src/main.rs:417:        if switch_applies(tenant_id, provider, profile_id, &session_id, "smart_context") {
multica-auth-work/prodex-sidecar/src/main.rs:418:            "exact"
multica-auth-work/prodex-sidecar/src/main.rs:420:            "shadow"
multica-auth-work/prodex-sidecar/src/main.rs:445:        "smart_context_mode": smart_context_mode
```

No input message/context field is read in `handle_session_start`; the response does not include token counts, transformed content, before/after metrics, truncation metadata, or savings.

Daemon config only forwards environment knobs:

```text
multica-auth-work/server/internal/daemon/prodex.go:40:		SmartContextShadow:  envBoolDefault("MULTICA_PRODEX_SMART_CONTEXT_SHADOW", true),
multica-auth-work/server/internal/daemon/prodex.go:41:		SmartContextCanary:  strings.TrimSpace(os.Getenv("MULTICA_PRODEX_SMART_CONTEXT_CANARY_PERCENT")),
multica-auth-work/server/internal/daemon/prodex.go:97:		agentEnv["PRODEX_SMART_CONTEXT_SHADOW"] = "1"
multica-auth-work/server/internal/daemon/prodex.go:100:		agentEnv["PRODEX_SMART_CONTEXT_CANARY_PERCENT"] = d.cfg.Prodex.SmartContextCanary
```

The daemon pushes a desired policy map with `"mode": "shadow"` and `"canary_percent": 0`, but this is policy data only:

```text
multica-auth-work/server/internal/daemon/l2_runtime.go:283:		SmartContext: map[string]any{
multica-auth-work/server/internal/daemon/l2_runtime.go:284:			"mode":           "shadow",
multica-auth-work/server/internal/daemon/l2_runtime.go:285:			"canary_percent": 0,
```

The Go `StartSessionResponse` does not include `SmartContextMode`, so the sidecar's `smart_context_mode` field is discarded by normal Go unmarshalling:

```text
multica-auth-work/server/internal/l2runtime/client.go:360:type StartSessionResponse struct {
multica-auth-work/server/internal/l2runtime/client.go:361:	ContractVersion  string `json:"contract_version"`
multica-auth-work/server/internal/l2runtime/client.go:362:	RequestID        string `json:"request_id"`
multica-auth-work/server/internal/l2runtime/client.go:363:	RuntimeSessionID string `json:"runtime_session_id"`
multica-auth-work/server/internal/l2runtime/client.go:364:	RouterOwner      string `json:"router_owner"`
multica-auth-work/server/internal/l2runtime/client.go:365:	EventStreamURL   string `json:"event_stream_url"`
multica-auth-work/server/internal/l2runtime/client.go:366:	RuntimeEndpoint  string `json:"runtime_endpoint"`
multica-auth-work/server/internal/l2runtime/client.go:367:	RuntimeLogRef    string `json:"runtime_log_ref"`
```

## Live Session With Large Context

Initial sandboxed bind failed:

```text
thread 'main' (2) panicked at src/main.rs:553:37:
failed to bind sidecar: Os { code: 1, kind: PermissionDenied, message: "Operation not permitted" }
```

The sidecar was then started with approved escalation on `127.0.0.1:43127`:

```text
prodex-sidecar listening on 127.0.0.1:43127
```

First large-context session, default `shadow` mode:

```text
REQUEST_BYTES=1392296
CONTEXT_CHARS_PER_FIELD=684000
RESPONSE_BYTES=323
RESPONSE_JSON={"contract_version":"rpp.l2.v1","event_stream_url":"http://127.0.0.1:43117/v1/events/stream?session_id=session-t3-large-context","request_id":"req-t3-large-context","router_owner":"rust_l2","runtime_endpoint":"loopback","runtime_log_ref":"memory","runtime_session_id":"rt-1783260678315328680","smart_context_mode":"shadow"}
```

Second large-context session, after applying a `smart_context` kill-switch that makes the sidecar return `exact`:

```text
KILLSWITCH_REQUEST_BYTES=248
KILLSWITCH_RESPONSE_BYTES=104
KILLSWITCH_RESPONSE_JSON={"applied":true,"contract_version":"rpp.l2.v1","effective_at":"next_request","request_id":"req-t3-kill"}
REQUEST_BYTES=1392294
RESPONSE_BYTES=320
RESPONSE_JSON={"contract_version":"rpp.l2.v1","event_stream_url":"http://127.0.0.1:43117/v1/events/stream?session_id=session-t3-exact","request_id":"req-t3-large-context-exact","router_owner":"rust_l2","runtime_endpoint":"loopback","runtime_log_ref":"memory","runtime_session_id":"rt-1783260838924092326","smart_context_mode":"exact"}
```

Sidecar was stopped after the run.

## Conclusion

No real token-reduction logic exists in the reviewed implementation. There is no before/after token comparison, no tokenizer, no prompt/context truncation, no summarization pipeline, no transformed context returned or persisted, and no emitted smart-context savings metric. The observed behavior is a label-only `smart_context_mode` value plus config/policy/env plumbing.
