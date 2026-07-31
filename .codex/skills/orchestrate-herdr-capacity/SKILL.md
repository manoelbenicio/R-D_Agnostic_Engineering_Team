---
name: orchestrate-herdr-capacity
description: Inspect and coordinate this project's coding-agent capacity through Herdr while preserving Kanban authority, credential isolation, deduplication, and one-task ownership. Use when asked to keep agents busy, run agents continuously, inspect idle or blocked agents, allocate parallel work, monitor agent panes, or explain why configured capacity is not runnable.
---

# Orchestrate Herdr Capacity

Read `../herdr/SKILL.md` completely before issuing any Herdr command. Use its CLI and
socket references for syntax. Verify:

```bash
test "${HERDR_ENV:-}" = 1
```

Stop Herdr control if this check fails.

## Measure real capacity

Inspect:

```bash
herdr agent list
herdr workspace list
herdr pane current --current
```

Use `herdr agent get`, `herdr agent read`, and `herdr agent explain` for ambiguous
states. Treat `idle`, `done`, `blocked`, and `unknown` as UI lifecycle observations,
not proof of runnable product capacity.

Reconcile each apparent lane with Kanban and metadata-only operational evidence:

- active or queued task;
- provider quota or access failure;
- approved credential assignment;
- physical credential slot presence;
- refresh-revoked or degraded account state;
- required model/reviewer capability;
- owned branch, worktree, and file set.

Never inspect secret contents or change credential assignments, slots, auth files,
provider sessions, `HOME`, or `CODEX_HOME` to make a lane look healthy.

## Allocate without duplicating work

Before prompting or dispatching, use `$reconcile-kanban-openspec` to classify the
issue and prove a missing delta.

Product implementation, repair, documentation, tests, investigations, and retries
must be represented and dispatched through Kanban. Do not create a hidden Herdr
implementation path.

Use Herdr for:

- inspecting live agent state and output;
- coordinating an already authorized task;
- independent review or consultation where the operating model permits it;
- ordinary non-product commands and monitoring;
- notifying the human about attention or blockers.

Keep every genuinely healthy lane assigned when bounded, non-overlapping work exists.
Do not create artificial work, duplicate paid tasks, or predictable pre-execution
failures merely to occupy an idle record.

## Preserve topology and ownership

Use one bounded task, branch, worktree, and file set per agent. Stop on overlapping
ownership. Keep background work unfocused with `--no-focus`; use explicit IDs or unique
agent names and parse returned JSON.

When creating authorized review capacity:

1. inspect layout;
2. split right or down without stealing focus;
3. start the requested agent in an available shell pane;
4. send one bounded prompt;
5. wait on semantic state;
6. read and record the result;
7. do not close panes or workspaces you did not create.

Prefer:

```bash
herdr agent prompt <agent> "<bounded task>" --wait --timeout 120000
herdr agent read <agent> --source recent-unwrapped --lines 120
```

If a task is already working, do not send a second task whose completion could satisfy
the same wait. If a provider fails after producing a commit or report, recover the
artifact before retrying.

## Report utilization truthfully

Maintain:

```text
agent | Herdr state | Kanban task | provider/account gate | physical slot |
owned worktree/files | next eligible delta | disposition
```

State separately:

- configured agents;
- runtime records online;
- credentials marked available;
- physically runnable lanes;
- lanes actively doing useful non-duplicate work.

Escalate owner-only repairs with exact evidence. Never equate a configured or idle
agent count with usable 24x7 capacity.
