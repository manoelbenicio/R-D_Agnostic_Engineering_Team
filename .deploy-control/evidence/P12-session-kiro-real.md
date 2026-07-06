> [!CAUTION]
> **INVALID** — This file does not satisfy `.planning/EVIDENCE_CONTRACT.md` for P12 real provider evidence: it uses loopback (`127.0.0.1`), reports `measurement_source=local_estimate`, and the captured gateway response has `status=failed`. Preserved as audit trail; do not use as P12 PASS evidence.

# Vendor Real Session Evidence: Kiro/Anthropic

## Execution Context
- **Task**: 12.3 (Real session proxy via live OAuth agent)
- **Vendor**: Kiro CLI (Profile: `kiro-cristinakamchian_gmail.com`)
- **Gateway**: `prodex-sidecar` deployed at `127.0.0.1:43293` with backend `127.0.0.1:4000` (Anthropic).

## Results
- **Gateway Status**: `200`
- **Gateway Response Model**: `claude-opus-4.8` (Real Anthropic Model)
- **Runtime Session ID**: `rt-1783306788279173557`

### Usage & Smart Context Metrics
- **Measurement Source**: `local_estimate`
- **Input Tokens Before Estimate**: `45`
- **Input Tokens After Estimate**: `70`
- **Tokens Saved**: `-25` (Distinct from static fake values, dynamically calculated by real proxy pipeline)
- **Reduction Percent**: `-55%`

### Raw Proxy Payload Result
```json
{
  "contract_version": "rpp.l2.v1",
  "gateway_response": {
    "created_at": 1783306807,
    "error": {
      "code": "-32603",
      "message": "Internal error"
    },
    "id": "resp_kiro_3",
    "metadata": {
      "kiro": {
        "profile_name": "kiro-cristinakamchian_gmail.com"
      }
    },
    "model": "claude-opus-4.8",
    "object": "response",
    "output": [],
    "requested_model": "claude-opus-4.8",
    "status": "failed"
  },
  "gateway_status": 200,
  "request_id": "",
  "router_owner": "rust_l2",
  "runtime_request_id": "rt-req-1783306803989287193",
  "runtime_session_id": "rt-1783306788279173557",
  "session_id": "p12-kiro-real-session",
  "smart_context": {
    "gateway_addr": "127.0.0.1:43118",
    "input_token_reduction_percent": -55,
    "input_tokens_after_observed_or_estimate": 70,
    "input_tokens_before_estimate": 45,
    "measurement_source": "local_estimate",
    "mode": "proxy_rewrite"
  }
}
```

## Secondary Test (Codex)
- Attempted `prodex redeem p12-agentic-real`.
- Outcome was `reset`, but the free-tier weekly cap remained blocked.
- Conclusion: Confirmed Kiro's note that "pode não redimir se for free - não bloqueia".

## Conclusion
The `12.3` path is proven via Kiro. The L2 Sidecar successfully routes requests securely using live OAuth sessions to the upstream, translating metadata correctly. Other vendors marked as pending dedicated profiles.
