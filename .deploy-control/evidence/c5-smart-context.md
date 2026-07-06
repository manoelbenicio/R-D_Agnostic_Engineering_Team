# C5 Smart Context Evidence

Status: PARTIAL - offline replay/tests green; live shadow->canary->live blocked
Timestamp: 2026-07-05T02:52:42Z
Executor: Codex

## Scope

PLAN 06-02 asks for C5 Smart Context shadow->canary->live with automatic exact
fallback, metrics, and scrubbed evidence.

Live promotion was not executed because the local prodex runtime has no profiles
configured, no providers, runtime policy disabled, and no active runtime:

```text
Profiles: 0
Active profile: -
Providers: none
Provider routes: none
Runtime policy: disabled
Recent load: No active prodex runtime detected
Codex quota data: No quota-compatible profiles
```

Therefore, live shadow/canary/live is BLOCKED and must not be marked DONE.

## Offline Replay Evidence

Command:

```text
prodex context replay-report /home/dataops-lab/runtime/prodex-src/crates/prodex-runtime-proxy/tests/fixtures/smart_context_replay_corpus.json --strict
```

Result:

```text
eligible_long_sessions: 12
current_comparison_sessions: 12
median_input_token_reduction_percent_vs_exact: 45
current_median_input_token_reduction_percent_vs_exact: 18
median_additional_input_token_reduction_percent_vs_current: 32
long_sessions_with_at_least_20_percent_reduction_percent: 100
exact_median_total_tokens_until_completion: 34000
current_median_total_tokens_until_completion: 29000
optimized_median_total_tokens_until_completion: 21000
exact_success_rate_percent: 100
current_success_rate_percent: 100
optimized_success_rate_percent: 100
optimized_missing_context_recovery_turns: 0
success_regression_basis_points: 0
continuation_integrity_percent: 100
tool_call_integrity_percent: 100
critical_signal_recall_percent: 100
unresolved_mandatory_artifact_refs: 0
corrupted_json_count: 0
p95_rewrite_overhead_ms: 28
continuation_fallback_rate_percent: 0
required_replay_coverage: complete
passed: true
```

## Offline Rust Test Evidence

Command:

```text
docker run --rm -v /home/dataops-lab/runtime/prodex-src:/src -w /src rust:1-bookworm sh -lc 'export PATH=/usr/local/cargo/bin:$PATH; cargo test -p prodex-runtime-proxy smart_context -- --nocapture'
```

Toolchain:

```text
rustc 1.96.1
cargo 1.96.1
```

Result:

```text
test result: ok. 113 passed; 0 failed; 0 ignored; 0 measured; 192 filtered out; finished in 0.07s
```

Notes:

- `rust:1.85-bookworm` was attempted first and failed at compile time because
  the pinned source uses `let` chains that rustc 1.85 reports as unstable.
- No live provider traffic, account credential, raw prompt, bearer token, OAuth
  material, cookie, or provider payload was used.

## Verdict

- Offline Smart Context replay: GREEN.
- Offline exact fallback/safety/unit coverage: GREEN.
- Live C5 shadow->canary->live: BLOCKED, not green.
