---
name: run-guarded-release
description: Execute this project's daemon or backend release waves with exact source identity, clean reproducible builds, queue-zero admission locking, automatic rollback, runtime verification, bounded canaries, production smokes, and evidence reconciliation. Use for cutovers, deployments, migrations, canary retries, or rollback decisions.
---

# Run a Guarded Release

Fail closed. A release is authorized only for the exact reviewed commit and artifact.
Do not build from a dirty production checkout and do not deploy while tasks are active.

## Establish the release identity

Verify:

- exact commit, parent, branch, remote ref, and changed files;
- required adversarial and GTL review verdicts;
- clean committed source independent of the production checkout;
- focused tests, full required gates, and deterministic build identity;
- candidate artifact or image digest, size, mode, and embedded revision;
- preserved rollback artifact and a tested rollback command.

Do not substitute a nearby commit, rebuilt dirty tree, or previously rejected artifact.

## Revalidate immediately before mutation

Check current Git, process, service, container, database, and queue state. A historical
green check is not a live gate.

Require two queue-zero observations and acquire the established PostgreSQL admission
lock in one held transaction:

```sql
BEGIN;
SET lock_timeout='15s';
LOCK TABLE agent_task_queue IN SHARE MODE;
SELECT CASE WHEN EXISTS (
  SELECT 1 FROM agent_task_queue
  WHERE status IN ('queued','dispatched','running','waiting_local_directory')
) THEN 'ACTIVE_QUEUE_PRESENT' ELSE 'ADMISSION_FREEZE_HELD' END;
```

Hold the same database session and transaction through the mutation. Abort on active
queue, lock timeout, identity drift, missing rollback, or any failed prerequisite.
Commit or roll back through that same session.

## Protect secrets and isolation

Never print credentials, tokens, headers, cookies, secret values, or credential-file
contents. Use metadata-only checks. Do not copy auth files, change physical credential
slots, alter login isolation, or repair provider sessions during a release.

Use established secret-resolution procedures inside the target child process. Never
call plaintext secret retrieval APIs.

## Perform the cutover

Prepare automatic rollback before touching the live target. Promote the exact artifact
atomically, restart only the intended service or container, and retain previous
identities until verification completes.

For backend releases:

- use the transactional deployment wrapper;
- apply only the reviewed migration set;
- preserve the durable environment source;
- verify the image revision label and migration state.

For daemon releases:

- verify executable mode and checksum before promotion;
- verify the systemd unit, PID, binary identity, health, and restart count afterward;
- preserve the prior executable as the rollback target.

## Verify before admitting work

Confirm:

- expected process/container identity and exact revision;
- health locally and through the production path;
- zero unexpected restarts;
- daemon/runtime visibility and provider discovery;
- queue remains zero and no launch regression is present.

Release the admission lock only after these checks pass.

Run a bounded canary exactly as authorized. Do not repeat a failed product task more
than the approved retry count. Validate credential snapshot, token-only task home,
native execution evidence, and the absence of the targeted failure signature.

## Roll back on any failed gate

Automatically restore the preserved artifact or image when identity, health, migration,
runtime visibility, restart, or canary checks fail. Reconfirm the rollback identity and
health. Keep the card open and record the failure without fabricating acceptance.

## Reconcile the release

Record exact commits, artifacts, tests, migration state, runtime observations, canary
task IDs, rollback readiness, and limitations in Kanban and OpenSpec. Distinguish
tested candidate from production-live state. Close cards only after GTL acceptance and
all named smokes pass.
