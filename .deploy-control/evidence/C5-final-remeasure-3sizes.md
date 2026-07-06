# C5 Final Remeasure: Smart Context 3 Sizes

agent: Codex#5.5#B
stream: C5-FINAL-REMEASURE-3SIZES
timestamp_utc: 2026-07-05T23:53:43Z
status: PASS

## Scope

Final remeasurement after `DIAG-smart-context-compaction.md` proved `tokens_saved>0`.

All three requests used the sidecar runtime proxy:

```text
POST /v1/runtime/proxy?session_id=<session>
```

The request envelope omitted `gateway_path`, so the sidecar used its default gateway path:

```text
/v1/responses
```

Responses API body shape:

```json
{
  "model": "gpt-4.1",
  "instructions": "system prompt: preserve identifiers exactly and answer ok",
  "input": [
    {"role": "user", "content": "BLOCO"}
  ],
  "max_output_tokens": 1
}
```

`BLOCO` was repeated text sized to 16KiB, 64KiB, and 256KiB respectively.

## Environment

- fake upstream: `127.0.0.1:43310`
- sidecar: `127.0.0.1:43312`
- sidecar gateway: `127.0.0.1:43311`
- sidecar binary: `multica-auth-work/prodex-sidecar/target/release/prodex-sidecar`
- prodex binary: `bin/prodex`
- token used: scrubbed local test bearer

## Results

| Context | client_body_bytes | upstream_body_bytes | tokens_before | tokens_after | tokens_saved | reduction_percent | measurement_source |
|---:|---:|---:|---:|---:|---:|---:|---|
| 16KiB | 16,772 | 16,605 | 4,151 | 12 | 4,139 | 99% | gateway_usage |
| 64KiB | 66,122 | 65,930 | 16,488 | 12 | 16,476 | 99% | gateway_usage |
| 256KiB | 263,526 | 263,331 | 65,839 | 12 | 65,827 | 99% | gateway_usage |

## Fake Upstream Log

```text
UPSTREAM_POST path=/v1/responses len=16605
UPSTREAM_POST path=/v1/responses len=65930
UPSTREAM_POST path=/v1/responses len=263331
```

## Interpretation

`tokens_saved>0` is confirmed for all three sizes through the sidecar response counters:

- 16KiB: `tokens_saved=4139`
- 64KiB: `tokens_saved=16476`
- 256KiB: `tokens_saved=65827`

Important nuance: this final test used exactly one `input` item as requested. With one large content item, the upstream body size remains close to the forwarded Responses API body size. The earlier DIAG byte-size compaction was observed with duplicated large content across separate input items, which triggered Smart Context duplicate-text rewriting before upstream delivery.

Conclusion: C5 final remeasure passes the requested `tokens_saved>0` criterion at 16KiB, 64KiB, and 256KiB through sidecar `/v1/runtime/proxy` using default `/v1/responses`.
