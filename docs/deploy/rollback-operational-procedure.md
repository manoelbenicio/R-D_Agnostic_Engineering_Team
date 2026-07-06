# Rollback Operational Procedure

Status: DRY-RUN READY - LIVE EXECUTION F0-GATED

This procedure turns the rollback runbook into an operator checklist. It does not authorize production changes. LIVE rollback is allowed only during an approved incident or deploy window after the F0/F7 owner gate is open.

References:

- `docs/deploy/rollback-runbook.md`
- `docs/deploy/prod-rollout-runbook.md`
- `docs/contracts/f0-readiness-matrix.md`
- `.deploy-control/evidence/status-board.md`

## 1. Hard Gate

Before any LIVE rollback action, confirm all fields are true in the owner approval record:

```text
deploy_owner_approved: true
owner:
timestamp:
rollback_command_ref:
kill_switch_command_ref:
accepted_risk:
```

Current known state is F0-gated. If `deploy_owner_approved` is absent or false, perform dry-run validation only.

## 2. Dry-Run Procedure

Run from the repo root:

```bash
rg -n "deploy_owner_approved: false|NO-GO|Rollback" .deploy-control/evidence/status-board.md docs/deploy
bash scripts/smoke/kill-switch-smoke.sh --dry-run --feature smart_context
bash scripts/smoke/kill-switch-smoke.sh --dry-run --feature gateway
bash scripts/smoke/readyz-smoke.sh --dry-run
bash scripts/smoke/state-backend-smoke.sh --dry-run
bash scripts/smoke/redaction-smoke.sh --dry-run
```

Dry-run pass criteria:

- status board still shows no LIVE deploy authorization unless owner approval is explicitly recorded;
- kill-switch dry-runs print planned loopback requests only;
- readiness and state dry-runs reference `127.0.0.1` or loopback only;
- redaction dry-run uses fake markers only;
- no command starts, stops, restarts, deploys, migrates, or changes runtime traffic.

Record dry-run evidence under `.deploy-control/evidence/` with the timestamp, operator, command names, exit codes, and scrubbed summaries. Do not paste secrets, bearer tokens, database URLs, Redis URLs, raw prompts, raw tool outputs, or full runtime responses.

## 3. LIVE Rollback Procedure

Only execute this section when the owner approval gate is open and an incident/deploy window is active.

1. Declare rollback with a unique `rollback_id`, trigger, operator, owner, and UTC timestamp.
2. Freeze new prodex/L2-backed session admission in the Multica control plane.
3. Apply the smallest kill switch that stops the failure, then broaden if confirmation fails.
4. Confirm kill-switch state from the durable store and, when available, from the runtime event acknowledgement.
5. Drain or stop affected prodex/L2 sessions with a scrubbed stop reason.
6. Restore the previous raw Codex or previous approved runtime launch configuration using the owner-approved `rollback_command_ref`.
7. Restart or reload only the minimum required Multica runtime component.
8. Verify Go daemon health, Postgres audit state, redaction, and raw Codex smoke.
9. Confirm no new sessions select prodex/L2 runtime routing after the rollback boundary.
10. Notify owner and Opus 4.8 via the approved channel with scrubbed status only.

LIVE validation commands must include the script gates:

```bash
SMOKE_ALLOW_EXECUTE=1 \
SMOKE_TARGET_ENV=prod \
DEPLOY_OWNER_APPROVED=true \
L2_BASE_URL=http://127.0.0.1:43117 \
L2_BEARER_TOKEN=<from-approved-secret-boundary> \
bash scripts/smoke/readyz-smoke.sh --execute

SMOKE_ALLOW_EXECUTE=1 \
SMOKE_TARGET_ENV=prod \
DEPLOY_OWNER_APPROVED=true \
L2_BASE_URL=http://127.0.0.1:43117 \
L2_BEARER_TOKEN=<from-approved-secret-boundary> \
bash scripts/smoke/redaction-smoke.sh --execute
```

Do not run these commands with real credentials until F0/F7 owner approval is recorded. Do not paste command output if it contains secrets or raw payloads.

## 4. Success Criteria

Rollback is complete only when:

- new sessions use raw Codex or the previous approved runtime path;
- no new prodex/L2 runtime events appear for post-rollback sessions;
- Go daemon and Postgres audit state are healthy;
- kill-switch state and rollback boundary are recorded;
- redaction smoke passes;
- credential/profile filesystem invariants still pass;
- owner and Opus 4.8 receive a scrubbed completion notice.

## 5. Failure Handling

If rollback cannot be confirmed:

- keep admission frozen;
- keep broad kill switches disabled for the affected scope;
- preserve evidence and logs after redaction;
- escalate to owner and Opus 4.8 immediately;
- do not retry deploy or re-enable Smart Context/canary without a new owner decision.
