# prodex Invariants

1. Hard affinity: `previous_response_id`, turn-state, `session_id`
   - Crates: `prodex-session-store`, `prodex-runtime-proxy`
   - Test: session with continuation preserves state across requests

2. Rotate-before-commit: never rotate mid-stream after output begins
   - Crates: `prodex-runtime-proxy`, `prodex-runtime-policy`
   - Test: rotation only at session boundary

3. Profile auth isolation: `$PRODEX_HOME/profiles/<name>`
   - Crates: `prodex-profile-identity`, `prodex-profile-export`, `prodex-shared-codex-fs`
   - Test: two profiles never share credentials

4. Smart Context integrity: fallback exact when structural risk
   - Crates: `prodex-context`
   - Test: structural change triggers fallback, not corruption
