# T20 — real-run manifest collector (scripts/ops; self-test green, live NOT run)

- agent: **Opus48#C** · lane **tier20-workload** · task **T20-MANIFEST-COLLECTOR** · pane `w8:p1`
- as-of (UTC): `2026-07-24T00:17Z` · HEAD `a6d50986aee7f1981a323b29c9d90f175b4b6f8b`
- lock: `scripts/ops/t20_manifest_collector.py` (NEW, ops tooling — **not product source**) + this evidence.
- check-in: `.deploy-control/p0/checkins/Opus48-C__T20-MANIFEST-COLLECTOR__20260724T001611Z.json`

## 0. Result

**DONE — collector built and unit self-test green (temp fixtures only).** It produces the `t20-run-manifest.v1` that the hardened Tier-20 verifier (`t20_hardened_verify_test.go` gates) binds to. It **only reads real run inputs** and **fails closed** on anything missing/malformed/count-mismatched/stale/unverifiable; it **never generates spans/IDs, never pads to a count, never infers a missing value**. Live invocation is documented; **no live run performed and no live PASS claimed.**

## 1. What it collects (all from real inputs; no generation/inference)

| Field | Source | Fail-closed rule |
|---|---|---|
| `run_id`, `window_start`, `window_end` | launcher (CLI args) | run_id required (never generated); window must be tz-aware and end>start |
| `expected_tasks` | launched-batch record `tasks.json` | **exactly** `--expected-count` (default 20) unique non-empty task IDs; 19/21/dup → fail |
| `exports.backend` / `exports.daemon` | span export files | `{inode, mtime_utc, size, sha256}`; missing/unreadable → fail; mtime outside `[window±grace]` → **stale → fail** |
| `launched_procs` | `pids.json` (sampled live during the run) + `/proc` re-verify | each `{pid,starttime}` must be int >0 (no sentinels); if the pid is still alive at collection, live `/proc/<pid>/stat` field-22 starttime MUST match the sample (else **PID reuse/spoof → fail**); sample missing starttime → cannot infer → fail |
| `persisted` | backend terminal rows `rows.json` | **exactly** 20 `{row_id,status}`, unique non-empty; <20 or dup → fail |

Robust `/proc/<pid>/stat` parse: splits after the final `)` and reads field-22 (`after[19]`), so a comm containing spaces/parens cannot skew the starttime.

## 2. Fail-closed guarantees (verified by self-test)

`--self-test` (temp fixtures, no live data, no real `/proc`) — `python3 scripts/ops/t20_manifest_collector.py --self-test` → **Ran 11 tests … OK** (exit 0):
- `test_happy_path` — valid fixture → `status=COLLECTED`, 20 tasks / 20 procs / 20 rows, 64-hex sha256, inode>0, all starttimes>0.
- `test_missing_export_fails_closed`, `test_stale_export_fails_closed` (mtime before window).
- `test_wrong_task_count_fails_closed` (19≠20), `test_wrong_persisted_count_fails_closed`, `test_duplicate_persisted_row_fails_closed`.
- `test_pid_starttime_mismatch_fails_closed` (live `/proc` starttime ≠ sample ⇒ reuse/spoof), `test_pid_sample_without_starttime_fails_closed` (cannot infer), `test_sentinel_pid_fails_closed` (pid 0).
- `test_dead_proc_uses_sampled_starttime` — process exited by collection time (no `/proc` dir): the during-run sampled starttime is authoritative and recorded (`alive_at_collection=false`), never fabricated.
- `test_missing_run_id_fails_closed`.

CLI smoke: `--run-id x` with no other inputs → `{"status":"FAILED_CLOSED","reason":"missing required inputs: …"}` exit **2**, no manifest emitted.

## 3. Evidence — exact commands + exit codes
```text
python3 -m py_compile scripts/ops/t20_manifest_collector.py                 -> COMPILE_OK ; exit 0
python3 scripts/ops/t20_manifest_collector.py --self-test                   -> Ran 11 tests OK ; exit 0
python3 scripts/ops/t20_manifest_collector.py --run-id x                    -> FAILED_CLOSED (missing inputs) ; exit 2
git diff --check -- scripts/ops/t20_manifest_collector.py                    -> PASS ; exit 0
```

## 4. Live invocation (documented; NOT run here; NO live PASS claimed)
During the authorized tier-20 run: a concurrent sampler (e.g. `scripts/ops/t20_resource_sampler.py`) records each observed child PID + its `/proc/<pid>/stat` field-22 starttime → `pids.json`; the launcher records the exact 20 enqueued task IDs → `tasks.json`; the backend records the 20 persisted terminal rows → `rows.json`; the instrumented harness exports spans → `backend.json`/`daemon.json`. Then:
```
python3 scripts/ops/t20_manifest_collector.py \
  --run-id "$RUN_ID" --window-start "$START" --window-end "$END" \
  --batch-tasks tasks.json --backend-export backend.json --daemon-export daemon.json \
  --pid-samples pids.json --persisted-rows rows.json --out t20-run-manifest.json
```
Exit 0 + manifest ⇒ provenance collected; non-zero + `FAILED_CLOSED` ⇒ unusable. The manifest feeds the hardened verifier's `t20RunManifest` gates (run_id/window/exact-task-set/real-proc/persisted). **This has not been executed; no live PASS is asserted.**

## 5. Scope / non-claims
- **No product source**: the collector is `scripts/ops` tooling (outside the server module); no product `.go`/shared source edited. Complements the existing `scripts/ops/t20_resource_sampler.py` (peer, not modified).
- Self-test uses **temp fixtures only** (never `/tmp` fabrication accepted as real; a real run must supply real inputs); no live `/proc`, no DB, no network, no inference/deploy.
- Never generates spans/IDs; never pads a count; never infers a missing value — every shortfall fails closed.
- Does not itself verify traces (that is the hardened verifier); it only produces the provenance manifest the verifier binds to.
- Does not authorize tier-20 activation (9.2), higher tiers, cutover, or production.

## 6. Status
- STATUS: DONE. DELIVERED: fail-closed manifest collector (§1) + 11-case temp-fixture self-test green (§2/§3) + documented (unrun) live invocation (§4). FILE: `scripts/ops/t20_manifest_collector.py` + this evidence; no product source; no live run.
