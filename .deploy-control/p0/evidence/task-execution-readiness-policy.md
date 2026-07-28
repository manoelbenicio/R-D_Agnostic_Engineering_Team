# Task execution readiness policy

Effective: 2026-07-28 UTC  
Authority: General Tech Manager

No implementation or runtime task may move to `in_progress` until its task-specific readiness manifest is complete. Read-only discovery needed to complete the manifest is allowed. The manifest is a concise delta, not a repeated fleet inventory.

## Required fields

1. **Outcome and scope** — concrete deliverable, explicit exclusions, acceptance criteria.
2. **Ownership** — executor, independent reviewer, branch/worktree and non-overlapping `FILES_LOCKED`.
3. **Tools** — exact commands/versions required; present/absent; preparation owner and command for anything absent.
4. **Access** — repository, API, database, AWS, runtime and board permissions required; secret classification and safe resolution path.
5. **Infrastructure** — host, service, ports, ephemeral dependencies, space/cache budget and teardown.
6. **Inputs and dependencies** — required commits, migrations, reservations, IDs and external decisions already available.
7. **Verification** — exact build/test/gate commands, nominal test names/counts, anti-skip/anti-false-green assertions.
8. **Rollback** — recovery path for every mutation; backup requirement for material data changes.
9. **Plan and ETA** — milestones with owner and ETA; blocker escalation must state why/who/where/when/action.

## Start rule

The executor emits `READY` only when all required tools, access and inputs are actually available. Missing preparation is resolved before implementation begins. A task may not report a known missing tool, permission or service for the first time midway through execution.

An unforeseen dependency discovered only after code or runtime inspection is not misconduct, but it must include evidence that it was not discoverable in the readiness scan, its exact impact, and the fastest safe resolution. Cost alone is never a blocker.

## Dispatch rule

The Leader verifies the readiness manifest before assigning an execution-triggering comment. Status/acceptance cards remain human-only and must not generate paid tasks. Provider quota/auth health is checked immediately before dispatch; a known-red provider is not assigned blindly.

## Completion rule

Completion requires the declared acceptance evidence, clean ownership handoff, teardown confirmation and Kanban update. `ok` without nominal test execution is not evidence when a harness can exit before tests run.

## Kiro/Opus rate and concurrency throttling

A positive subscription balance is not proof of request capacity, but it prevents classifying response-stream throttling as `provider_quota_limit` without an explicit quota-exhaustion signal. The 2026-07-28 Opus48-A incident occurred with KIRO PRO MAX at 60.4% remaining and is classified as probable rate/concurrency throttle (`provider_server_error`), not exhausted quota.

For every Kiro/Opus lane:

1. Permit at most **one heavy tool/request in flight per agent**. A heavy operation includes a long DB/build/test gate, provider request, large repository scan or equivalent sustained call.
2. Spawn **zero subagents during gates**. The assigned executor owns the serial gate directly.
3. Issue **no simultaneous retries**, whether within one agent or across sibling Kiro/Opus lanes reacting to the same throttle event.
4. Retry only with **serial backoff** after proving the prior request is terminal and workspace/agent active counts permit it. Record source task, attempt count and next allowed action.
5. Do not translate a throttle into quota exhaustion unless the provider returns an explicit quota/balance signal. Repeated throttling is an external-capacity blocker and should trigger handoff/reassignment rather than a retry storm.
6. Read-only discovery may be READY independently of mutation readiness when its manifest has exact scope, no-write gates, tools/access and teardown. Install, cutover, secret, DB or host mutation remains NOT_READY until its separate authority and reviewer gates pass.
