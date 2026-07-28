# Owner ruling — Kanban-only agent dispatch

- **Effective:** 2026-07-28T16:49:00Z
- **Authority:** Owner / General Tech Manager
- **Status:** ACTIVE

## Rule

Every new executable agent action must begin through the product Kanban by changing the card assignee through the supported API. The assignment must create exactly one product task.

This applies to implementation, review, diagnosis, testing, integration preparation and operations. Status-only changes and human-only `/note` comments are not dispatch mechanisms.

## Preconditions

Before assignment, the mechanical operator must verify:

1. the card is the exact intended card and project;
2. the target agent is unique, active, non-archived and belongs to the workspace;
3. its runtime is online and compatible with the requested provider/model;
4. the agent and card have no active task or conflicting ownership;
5. required tools, access, services, disk budget and secret-safety instructions are present;
6. the assignee update creates exactly one task, followed by GET-back and active-task verification.

If a precondition fails, no assignment is performed and the exact failure is reported to the GTM.

## Herdr boundary

Herdr is limited to supervision, pane health, communication, milestone collection and steering of already authorized work. It must not launch a parallel execution for a card dispatched through Kanban.

Work already running through Herdr when this ruling became effective may finish in place. It must not be re-dispatched through Kanban mid-flight.

## Reviews and follow-ups

Independent reviews are also executable work and therefore use Kanban assignment. The original author is recorded in a human-only note before the reviewer assignment when necessary for auditability.

## Exception

Only explicit owner/GTM emergency-containment authorization may bypass this rule. The exception must name the target, reason, executor and rollback before execution.

## Reserved agent

Kiro-Opus5 remains reserved for the owner and is excluded from automatic dispatch.
