agent: Codex#5.5#B
stream: DEPLOY-ROLLBACK-TEST-PROOF
phase: P12
priority: P0
status: DONE
progress: 100
started_at: 2026-07-06T03:15:00Z
finished_at: 2026-07-06T03:21:50Z
files_locked:
  - .deploy-control/Codex-5.5-B__DEPLOY-ROLLBACK-TEST-PROOF__20260706T031500Z.md
  - .deploy-control/evidence/deploy-rollback-test-proof.md
  - .deploy-control/evidence/deploy-rollback-test-proof.sh
depends_on: specs/deploy-rollback
plan_ref: specs/deploy-rollback/spec.md
build_result: |
  PASS — .deploy-control/evidence/deploy-rollback-test-proof.sh
  Evidence: .deploy-control/evidence/deploy-rollback-test-proof.md
  Pinned prodex binary: prodex 0.246.0
  Raw codex after rollback: codex-cli 0.142.5
  Kill-switch tests:
    - tenant smart_context: proxy_rewrite -> exact -> proxy_rewrite
    - provider gateway: HTTP 200 -> 423 blocked -> 200 restored; other provider unaffected
    - profile auto_redeem: active true -> false; other profile unaffected
  Rollback test:
    - one command rewrote isolated runtime env from prodex/L2 to raw codex
    - MULTICA_PRODEX_ENABLED=0, MULTICA_L2_ENABLED=0
    - prodex/L2 routing keys removed
  Scrub check: 0 matches
notes: Evidence-Gated Done. Used isolated PRODEX_HOME/CODEX_HOME and temporary sidecar/gateway ports. auto_redeem has no session_block hook in current sidecar, so the test verifies disable/restore through killswitch status and profile-scope isolation. No commits.
