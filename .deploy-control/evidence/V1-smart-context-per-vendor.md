# V1.1 Smart Context Per-Vendor Payload Shapes

agent: Codex#5.5#B
stream: V1-SMART-CONTEXT-PER-VENDOR
timestamp_utc: 2026-07-06T00:51:08Z
status: PASS_WITH_CAVEAT

## Scope

Measured four vendor payload shapes through the existing sidecar/gateway stack:

- sidecar: `127.0.0.1:43292`
- gateway: `127.0.0.1:43291`
- fake upstream: `127.0.0.1:43290`

Each test created a session, then posted a runtime envelope to:

```text
POST /v1/runtime/proxy?session_id=<vendor-session>
```

The envelope omitted `gateway_path`, matching the C5 default path behavior. Therefore the sidecar/gateway defaulted to:

```text
/v1/responses
```

No `prodex-sidecar/` files were edited.

## Fake Upstream Caveat

The currently running fake upstream is:

```text
python3 /mnt/c/VMs/Projects/Automonous_Agentic/scripts/smoke/fake-upstream-logging.py --host 127.0.0.1 --port 43290
```

It returns `404` for `/v1/responses`:

```text
HTTP/1.0 404 Not Found
Server: FakeOpenAIUpstream/1.0 Python/3.12.3
Content-Type: application/json

{"error":{"message":"not found"}}
```

It does support `/v1/chat/completions`, but this V1.1 run intentionally preserved the C5 sidecar default `/v1/responses`. Because `/v1/responses` returns `404`, the sidecar reported Smart Context token savings using `measurement_source="local_estimate"` for all vendors.

## Results

| Vendor | Provider | Payload shape | Runtime status | Gateway status | Measurement source | Tokens before | Tokens after | Tokens saved | Reduction |
|---|---|---|---:|---:|---|---:|---:|---:|---:|
| Codex/OpenAI | `codex` | Responses API `{model,input,instructions}` | 200 | 404 | `local_estimate` | 4,169 | 8 | 4,161 | 99% |
| Kiro/Anthropic | `kiro` | Messages API `{model,messages,max_tokens}` | 200 | 404 | `local_estimate` | 4,158 | 8 | 4,150 | 99% |
| Antigravity/Gemini | `antigravity` | Gemini API `{model,contents}` | 200 | 404 | `local_estimate` | 4,154 | 8 | 4,146 | 99% |
| Cline/OpenRouter | `cline-openrouter` | OpenAI-compatible `{model,messages,max_tokens}` | 200 | 404 | `local_estimate` | 4,155 | 8 | 4,147 | 99% |

All four vendor payload shapes satisfied the requested `tokens_saved>0` condition through the sidecar Smart Context counters.

## Raw Runtime Responses

### Codex/OpenAI

Payload shape:

```json
{"model":"gpt-4.1","input":[{"role":"user","content":"BLOCO_16KIB"}],"instructions":"preserve identifiers exactly and answer ok","max_output_tokens":1}
```

Runtime response:

```json
{"contract_version":"rpp.l2.v1","gateway_response":{"error":{"message":"not found"}},"gateway_status":404,"request_id":"req-v1-runtime-1","router_owner":"rust_l2","runtime_request_id":"rt-v1-runtime-1","runtime_session_id":"rt-1783299045739305987","session_id":"v1-vendor-1-codex","smart_context":{"gateway_addr":"127.0.0.1:43291","input_token_reduction_percent":99,"input_tokens_after_observed_or_estimate":8,"input_tokens_before_estimate":4169,"measurement_source":"local_estimate","mode":"proxy_rewrite"}}
```

### Kiro/Anthropic

Payload shape:

```json
{"model":"claude-sonnet-4-20250514","messages":[{"role":"user","content":"BLOCO_16KIB"}],"max_tokens":1}
```

Runtime response:

```json
{"contract_version":"rpp.l2.v1","gateway_response":{"error":{"message":"not found"}},"gateway_status":404,"request_id":"req-v1-runtime-2","router_owner":"rust_l2","runtime_request_id":"rt-v1-runtime-2","runtime_session_id":"rt-1783299045744806273","session_id":"v1-vendor-2-kiro","smart_context":{"gateway_addr":"127.0.0.1:43291","input_token_reduction_percent":99,"input_tokens_after_observed_or_estimate":8,"input_tokens_before_estimate":4158,"measurement_source":"local_estimate","mode":"proxy_rewrite"}}
```

### Antigravity/Gemini

Payload shape:

```json
{"model":"gemini-2.5-pro","contents":[{"role":"user","parts":[{"text":"BLOCO_16KIB"}]}]}
```

Runtime response:

```json
{"contract_version":"rpp.l2.v1","gateway_response":{"error":{"message":"not found"}},"gateway_status":404,"request_id":"req-v1-runtime-3","router_owner":"rust_l2","runtime_request_id":"rt-v1-runtime-3","runtime_session_id":"rt-1783299045747759785","session_id":"v1-vendor-3-antigravity","smart_context":{"gateway_addr":"127.0.0.1:43291","input_token_reduction_percent":99,"input_tokens_after_observed_or_estimate":8,"input_tokens_before_estimate":4154,"measurement_source":"local_estimate","mode":"proxy_rewrite"}}
```

### Cline/OpenRouter

Payload shape:

```json
{"model":"openrouter/auto","messages":[{"role":"user","content":"BLOCO_16KIB"}],"max_tokens":1}
```

Runtime response:

```json
{"contract_version":"rpp.l2.v1","gateway_response":{"error":{"message":"not found"}},"gateway_status":404,"request_id":"req-v1-runtime-4","router_owner":"rust_l2","runtime_request_id":"rt-v1-runtime-4","runtime_session_id":"rt-1783299045750772820","session_id":"v1-vendor-4-cline-openrouter","smart_context":{"gateway_addr":"127.0.0.1:43291","input_token_reduction_percent":99,"input_tokens_after_observed_or_estimate":8,"input_tokens_before_estimate":4155,"measurement_source":"local_estimate","mode":"proxy_rewrite"}}
```

## Verdict

V1.1 per-vendor payload-shape test confirms `tokens_saved>0` for:

- Codex/OpenAI
- Kiro/Anthropic
- Antigravity/Gemini
- Cline/OpenRouter

Operational caveat: because the existing fake upstream returns `404` for `/v1/responses`, this run proves savings through sidecar Smart Context local estimates rather than upstream usage counters. A follow-up upstream fixture that returns `200` plus usage for `/v1/responses` would be required to convert this from `PASS_WITH_CAVEAT` to full upstream-backed PASS.
