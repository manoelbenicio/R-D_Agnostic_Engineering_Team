agent: Codex#5.5#B
stream: C5-FINAL-REMEASURE-3SIZES
phase: W2-C5
priority: P0
status: DONE
progress: 100
started_at: 2026-07-05T23:49:43Z
finished_at: 2026-07-05T23:53:43Z
files_locked:
  - .deploy-control/Codex-5.5-B__C5-FINAL-REMEASURE-3SIZES__20260705T234943Z.md
  - .deploy-control/evidence/C5-final-remeasure-3sizes.md
depends_on: DIAG-SMART-CONTEXT-COMPACTION
build_result: |
  PASS — sidecar /v1/runtime/proxy default /v1/responses with valid Responses API body:
    16KiB: client_body_bytes=16772 upstream_body_bytes=16605 tokens_before=4151 tokens_after=12 tokens_saved=4139 reduction_percent=99
    64KiB: client_body_bytes=66122 upstream_body_bytes=65930 tokens_before=16488 tokens_after=12 tokens_saved=16476 reduction_percent=99
    256KiB: client_body_bytes=263526 upstream_body_bytes=263331 tokens_before=65839 tokens_after=12 tokens_saved=65827 reduction_percent=99
notes: Final C5 remeasurement after DIAG proved tokens_saved>0. Measure sidecar /v1/runtime/proxy default /v1/responses with valid Responses API body at 16KiB, 64KiB, and 256KiB repeated context; capture client_body_bytes, upstream_body_bytes, tokens_before, tokens_after, tokens_saved, reduction_percent.

2026-07-05T23:53:43Z done: C5 final remeasure completed. Evidence in .deploy-control/evidence/C5-final-remeasure-3sizes.md. All three sizes report tokens_saved>0.
