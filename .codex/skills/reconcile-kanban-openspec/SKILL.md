---
name: reconcile-kanban-openspec
description: Reconcile this project's Kanban issues, agent tasks, Git branches, source code, tests, deployments, and OpenSpec without redoing valid work. Use when triaging open or in-review cards, deciding whether to dispatch work, accepting or rejecting an agent delivery, aligning documentation with operational reality, or closing and blocking issues with evidence.
---

# Reconcile Kanban and OpenSpec

Treat Kanban/API as the authority for executable product work and OpenSpec as the
canonical requirements and evidence documentation where established. Treat the live
filesystem, Git remotes, runtime, and database as current truth.

## Reconcile before acting

Inspect only the evidence needed for the target issue:

1. Read the issue, acceptance criteria, metadata, and latest discussion.
2. List existing tasks, including failed, cancelled, and completed attempts.
3. Inspect assigned agents, task status, branches, remote refs, worktrees, commits,
   ancestry, diffs, tests, artifacts, deployment state, and prior GTL/Kiro verdicts.
4. Inspect the corresponding OpenSpec change and validate whether its claims match
   source and production.
5. Consult preserved history only to resolve a specific uncertainty.

Never infer missing implementation from an open card or a terminal task. Classify the
issue before dispatch:

- **A — already done and accepted:** reconcile the card and documentation only.
- **B — implemented, not accepted or deployed:** perform only review, gate, integration,
  deployment, or evidence steps.
- **C — partially complete:** dispatch only the remaining bounded delta.
- **D — failed or rejected, valid replacement present:** use the replacement.
- **E — genuinely not done:** dispatch the smallest non-overlapping Kanban task.
- **F — external, owner, or security blocked:** record exact evidence and owner action.

## Dispatch safely

Dispatch implementation, repair, investigation, documentation, testing, and retries
through Kanban. Do not use a hidden Herdr task as a product-work path.

Before creating a task, prove:

- no active or completed task already covers the delta;
- no branch, commit, artifact, or accepted replacement already supplies it;
- the selected agent has a healthy provider, approved credential assignment, and
  physical slot where required;
- the issue and agent have no active task;
- the file set does not overlap another active owner.

Use one bounded task, branch, worktree, and non-overlapping file set per agent. Do not
create work merely to make an idle counter decrease.

## Accept artifacts, not status labels

Independently verify lineage, exact changed files, behavior, tests, and remote SHA.
A provider reporting failure after commit, push, or `goal_complete` does not invalidate
a proven artifact. Recover and accept that artifact, then perform only its missing
integration step.

Require the established Principal Kiro/Opus gate where the architecture calls for it.
Gemini or documentation agents cannot approve technical quality. GTL owns acceptance,
rejection, integration, deployment, and closure.

## Keep all evidence layers distinct

Use these terms precisely:

- **source:** code exists at an exact commit;
- **tested:** named checks passed against that commit;
- **candidate:** a preserved build or image is eligible for a release decision;
- **accepted:** the required reviewer and GTL accepted it;
- **deployed/live:** production identity and behavior were observed;
- **blocked:** a named condition prevents the next gate.

Never promote evidence from a lower layer to a higher one.

## Reconcile OpenSpec continuously

Update OpenSpec in the same acceptance or deployment wave. Record the requirement,
implementation commit and branch, tests, deployment state, and remaining limitation.
Remove superseded claims instead of preserving stale `.planning` or `.deploy-control`
material. Never claim acceptance or deployment before it occurs.

Run:

```bash
openspec validate --all --strict --no-interactive
git diff --check
```

Integrate accepted documentation into the canonical docs branch with an ancestry check
and normal non-force fast-forward. Verify the remote tip exactly.

## Maintain the GTL ledger

For every open issue retain:

```text
issue | classification | reused evidence | missing delta | task/agent |
commit | review | tests | deployment | OpenSpec | disposition/blocker
```

Close only when canonical evidence proves the acceptance criteria. Otherwise leave the
card blocked with the exact condition, responsible owner, and required next action.
