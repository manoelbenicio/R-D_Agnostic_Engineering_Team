# Canvas Topology Prompt Augmentation

Used by `canvas-reconciler` (task 9.2). Per master spec section 4.4
step 5, Supervisor agent profiles installed by the reconciler get a
"canvas topology" block appended to their `system_prompt`. The block lists
which target profiles the agent may reach through `handoff`, `assign`, or
`send_message` edges derived from the current `CanvasDocument`.

## When It Runs

- Trigger: `reconcileCanvas()` installs or reinstalls an agent profile.
- Inputs: `CanvasDocument`, the current node id, and `canvas.edges`.
- Output: a string passed as `systemPrompt` to `generateProfileMarkdown()`.
- Current predicate: the block is appended when
  `node.data.is_entry_point || node.data.role === 'supervisor'`.

The implementation lives in `augmentSupervisorPrompt()` at
`src/canvas-reconciler/reconciler.ts:47-75`. `reconcileCanvas()` calls it
during first deploy/retry, node profile updates, and added-node deploys at
`src/canvas-reconciler/reconciler.ts:255-267`,
`src/canvas-reconciler/reconciler.ts:448-459`, and
`src/canvas-reconciler/reconciler.ts:524-535`.

## Edge To Directive Mapping

| Edge type | Directive emitted | Example line in prompt |
| --- | --- | --- |
| `handoff` | `Allowed Handoff Targets` | `- Developer (Profile: developer_<id>)` |
| `assign` | `Allowed Assign Targets` | `- Developer (Profile: developer_<id>)` |
| `send_message` | `Allowed Send Message Targets` | `- Reviewer (Profile: reviewer_<id>)` |
| `data-flow` | skipped; not a v1 control edge | n/a |

`CanvasEdge.type` is currently limited to `handoff`, `assign`, and
`send_message` in `src/shared/canvas-types.ts:22-28` and
`src/canvas-document/schema.ts:39-45`. The Canvas Builder defaults new edges
to `handoff`, lets users change them, and stores the selected type back on the
canvas edge (`src/canvas-builder/CanvasBuilderPage.tsx:349-365` and
`src/canvas-builder/CanvasBuilderPage.tsx:723-735`).

## Block Schema

`augmentSupervisorPrompt()` emits this exact shape
(`src/canvas-reconciler/reconciler.ts:70-75`):

```text

### Canvas Topology
Allowed Handoff Targets:
<handoff target lines or None>

Allowed Assign Targets:
<assign target lines or None>

Allowed Send Message Targets:
<send_message target lines or None>
```

Each target line is built from the target node display name and generated
profile name (`src/canvas-reconciler/reconciler.ts:55-67`):

```text
- <display_name> (Profile: <profile_name>_<node_id_with_underscores>)
```

The generated profile name is the same node profile naming convention used
when installing profiles and creating sessions.

## Worked Example

Canvas:

```text
Supervisor Node --handoff------> Developer Node
Supervisor Node --send_message-> Reviewer Node
Developer Node  --handoff------> Reviewer Node
```

For this Supervisor node:

```text
id: 00000000-0000-4000-8000-000000000001
profile_name: supervisor
display_name: Supervisor Node
system_prompt: Coordinates team.
```

With these outbound edges:

```text
handoff:
  target profile: developer_00000000_0000_4000_8000_000000000002
send_message:
  target profile: reviewer_00000000_0000_4000_8000_000000000003
```

The appended topology block is:

```text

### Canvas Topology
Allowed Handoff Targets:
- Developer Node (Profile: developer_00000000_0000_4000_8000_000000000002)

Allowed Assign Targets:
None

Allowed Send Message Targets:
- Reviewer Node (Profile: reviewer_00000000_0000_4000_8000_000000000003)
```

The final profile body is the original `system_prompt` followed by that block.

## Edge Cases

- No outbound edges: the block is still emitted. Each section contains `None`.
- Cycles: cycles are not traversed. Only direct outgoing edges where
  `edge.source === node.id` are listed.
- Missing target node: that edge is skipped.
- Duplicate edges of different types between the same pair: the same target
  appears once under each matching action section.
- Duplicate edges of the same type between the same pair: the line is repeated;
  there is no deduplication.
- Non-supervisor nodes: ordinary non-supervisor nodes do not receive the block,
  but any node marked `is_entry_point` does receive it under the current
  predicate.
- Profile updates after edit-after-deploy: when a node's profile content
  changes, the reconciler reinstalls that profile and regenerates this block
  from the edited canvas. Edge-only edits do not reinstall a supervisor profile
  in v1; they persist the canvas and set `edge_change_advisory`.

## Where It Is Consumed

- `src/canvas-reconciler/reconciler.ts:47-75`:
  `augmentSupervisorPrompt()` builds the block.
- `src/canvas-reconciler/reconciler.ts:255-267`:
  first deploy/retry profile install path passes the augmented prompt to
  `generateProfileMarkdown()`.
- `src/canvas-reconciler/reconciler.ts:448-459`:
  edit-after-deploy profile update path regenerates the augmented prompt.
- `src/canvas-reconciler/reconciler.ts:524-535`:
  edit-after-deploy added-node path regenerates the augmented prompt.
- `src/canvas-reconciler/reconciler.ts:164-175`:
  edge-only edits set `edge_change_advisory` and avoid CAO calls.
- `src/canvas-reconciler/index.ts:1-4`:
  the reconciler module re-exports the implementation.

Tests that exercise the deploy paths:

- `src/canvas-reconciler/__tests__/reconciler.test.ts:120-143`:
  `handles full happy-path 3-node deploy`.
- `src/canvas-reconciler/__tests__/reconciler.test.ts:399-461`:
  `handles diff-change-profile-content in edit-after-deploy`.
- `src/canvas-reconciler/__tests__/reconciler.test.ts:463-511`:
  `handles diff-display-only edits without triggering CAO calls`.

## Known Gaps

- There is no unit test that asserts the exact `Canvas Topology` prompt text.
  Existing reconciler tests cover deploy/profile-install behavior indirectly.
- The implementation emits action-section headers, not the peer-centric
  "You are connected to the following agents" format described in some
  planning notes.
- The implementation appends the block to every `is_entry_point` node, not only
  nodes whose role is `supervisor`.
- Edge-only edits do not regenerate the already-installed supervisor profile in
  v1. The design records this as a v1 limitation and relies on Tear Down plus
  redeploy when topology changes must affect the running supervisor.
