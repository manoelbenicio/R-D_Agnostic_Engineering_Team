# Rollback Runbook

Status: DRAFT FOR OWNER REVIEW - PROD DEPLOY NO-GO

This runbook defines the path back from pinned prodex AS-IS to the known-good raw Codex launch path or the previous approved launch configuration. It must be reviewed before any PROD deploy.

## 1. Rollback Goal

Rollback restores new sessions to raw Codex or the previous approved runtime path while preserving:

- profile isolation;
- durable audit rows;
- scrubbed runtime logs;
- deployment evidence;
- owner notification trail;
- Postgres state integrity.

Rollback must not delete audit evidence or credential material. It must not move credentials from ext4 to 9p.

## 2. Rollback Triggers

Immediate rollback triggers:

- raw secret detected in logs, traces, events, evidence, screenshots, or command output;
- `PRODEX_HOME`, `CODEX_HOME`, OAuth/profile store, token cache, or `auth.json` detected on 9p/shared mount;
- credential file mode is not 0600 or credential directory mode is not 0700;
- profile switch is not fail-closed;
- session continuation affinity failure;
- Smart Context protocol/tool-call/JSON/continuation corruption without exact fallback;
- kill switch unavailable or unconfirmed;
- sidecar health/readiness failure;
- event ingest cannot write durable audit rows;
- Postgres unavailable for required state;
- Go daemon unhealthy;
- owner or Opus 4.8 requests rollback.

## 3. Rollback Controls

Use the most specific kill switch scope that stops the failure, then broaden if confirmation fails:

```text
feature=smart_context state=disabled effective_at=next_request
feature=gateway state=disabled effective_at=immediate
feature=auto_redeem state=disabled effective_at=immediate
feature=provider_bridge state=disabled effective_at=immediate
```

If a scoped kill switch cannot be confirmed, treat that as critical and restore the raw Codex launch path for new sessions.

## 4. Rollback Steps

These steps are executable only by the approved operator during an open incident or deploy window.

1. Declare rollback and record trigger, timestamp, owner/operator, and deploy id.
2. Freeze new prodex sessions at the Multica admission layer.
3. Apply kill switches for Smart Context, gateway, auto-redeem, and provider bridge.
4. Confirm kill switch state from durable store and runtime event acknowledgement if available.
5. Stop or drain prodex-backed sessions:
   - preserve in-flight sessions only if continuation and profile isolation are known safe;
   - otherwise stop with reason `kill_switch`, `runtime_error`, or `operator_requested`.
6. Restore previous raw Codex launch configuration.
7. Restart or reload the minimum required Go daemon component.
8. Verify Go daemon health.
9. Start a controlled raw Codex smoke session.
10. Confirm no new session selects prodex runtime routing.
11. Confirm event ingest and audit ledger remain available.
12. Preserve prodex logs and evidence after redaction; do not delete them.
13. Notify owner and Opus 4.8 with scrubbed summary.

## 5. Data Handling During Rollback

Do not:

- copy credentials or profile stores to evidence;
- paste raw database or Redis URLs;
- paste raw bearer tokens;
- delete audit rows;
- run destructive migrations;
- downgrade schema without an explicit owner-approved migration rollback.

Do:

- retain durable audit in Postgres;
- mark sessions with stop reason and rollback id;
- record scrubbed command summaries;
- capture hashes of relevant config, not secret values.

## 6. Rollback Success Criteria

All must be true:

- new sessions launch through raw Codex or previous approved path;
- no new prodex runtime events appear for new sessions after rollback boundary;
- Go daemon healthy;
- Postgres reachable and audit rows preserved;
- kill switch state recorded;
- redaction smoke still passes;
- credential/profile ext4 and permission invariants still pass;
- owner receives scrubbed rollback completion notice.

## 7. Rollback Evidence

Store scrubbed evidence under `.deploy-control/evidence/`:

```text
rollback_id
deploy_id
trigger
timestamp
operator
owner_notification_ref
kill_switch_result
raw_codex_smoke_result
Go health result
Postgres/audit result
redaction result
remaining risks
```

Evidence may include profile aliases or hashed account ids. It must not include raw account emails unless explicitly required, and must never include secrets.

## 8. Post-Rollback Follow-Up

After rollback:

- leave prodex deploy disabled until owner re-approval;
- keep Smart Context canary/live disabled;
- open an incident record for the trigger;
- attach scrubbed metrics and logs;
- require ADR or owner decision for any architecture change before retry.
