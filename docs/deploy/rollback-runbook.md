# Rollback Runbook

Status: PRE-DEPLOY REQUIRED

## 1. Rollback Goal

Return runtime launch from `prodex` path to known-good raw `codex` path or to
previous approved launch configuration.

Rollback must preserve:

- profile isolation;
- audit trail;
- logs;
- evidence.

## 2. Rollback Triggers

Rollback if:

- secret leak;
- profile isolation failure;
- continuation affinity failure;
- sidecar health failure;
- Go daemon failure;
- kill switch failure;
- Smart Context corruption without exact fallback;
- owner requests rollback.

## 3. Rollback Steps

1. Apply global kill switch:

```text
smart_context=disabled
gateway=disabled
auto_redeem=disabled
provider_bridge=disabled
```

2. Stop new sessions through prodex path.
3. Preserve in-flight sessions only if safe; otherwise stop with reason.
4. Restore raw codex launch path.
5. Restart or reload Go daemon as required.
6. Verify raw codex smoke session.
7. Verify no prodex route is selected for new sessions.
8. Record `rollback_succeeded`.

## 4. Rollback Success Criteria

- new session launches through raw codex or previous approved path;
- Go daemon healthy;
- no new prodex runtime events for new sessions;
- existing audit preserved;
- operator evidence scrubbed.

## 5. Rollback Evidence

Required:

- rollback trigger;
- timestamp;
- command summary with secrets redacted;
- health result;
- smoke result;
- owner notification.

