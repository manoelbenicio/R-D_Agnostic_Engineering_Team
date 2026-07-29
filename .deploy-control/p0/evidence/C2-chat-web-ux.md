# C2 — Chat Web UX (verify untargeted-omits-target / picker-sends-explicit-agent)

- agent `Opus48#B` · lane `C2` · pane `w6:p2` · task `C2-CHAT-WEB-UX`
- lock: `multica-auth-work/packages/views/chat` (exclusive) · check-in receipt `CHECKIN__Opus48-B__C2__C2-CHAT-WEB-UX__20260722T121947Z.json`
- preflight: HEAD `a6d50986…`; go1.26.1; node v22.23.1; root disk full / `/tmp` tmpfs.
- posture: read-only review within lock; **no product edit** (no verifiable/contract-correct change possible — see blockers).

## Review findings (static, within chat/**)

1. **Direct-agent picker sends an explicit agent — SATISFIED (reviewed).**
   `AgentDropdown` (`chat-window.tsx`) calls `onSelect(agent)` with the full selected `Agent`; the
   consumer sets `selectedAgentId`. Encoded by `chat-window.agent-dropdown.test.tsx`
   ("keeps the current agent marked and selects another agent" → `onSelect` called with `agents[2]`).

2. **"New untargeted chat OMITS target" — NOT resolvable inside chat/** (out-of-lock contract).**
   There is **no `target` domain concept anywhere in `packages/views/chat/**`** (every `target` match is a
   DOM `e.target` event, unrelated). The only agent routing is
   `chat-window.tsx:339` `createSession.mutateAsync({ agent_id: activeAgent.id, title })`, where
   `activeAgent = selectedAgentId ?? availableAgents[0]`. Whether an *untargeted* chat should omit the
   agent/target field is governed by the **`useCreateChatSession` / createChat request contract in
   `packages/core`** (imported at `chat-window.tsx:42`), which is **outside my exclusive lock**
   (`packages/views/chat` only). No chat/**-local field exists to omit; any change to omit a target is a
   cross-boundary (packages/core / chat-orchestration backend) contract decision.

## Blockers (concrete; owner; next action)

- **BLK-C2-TESTS — cannot execute focused web tests.** `multica-auth-work/node_modules` and
  `node_modules/.bin/vitest` are **ABSENT**; the views test script is `vitest run`. Dependency-install is
  **forbidden** by the dispatch. So I cannot run/verify the chat focused tests
  (`chat-window.agent-dropdown.test.tsx`, `chat-input.test.tsx`) or any fix. Owner: Manager (w5:p1) /
  environment — provision `node_modules`/vitest for the web workspace, or authorize a bounded install.
  Next: with the runner available, run `vitest run packages/views/chat` and record exit + executed tests.
- **BLK-C2-CONTRACT — untargeted target-omission is out of lock.** The create-chat request shape
  (`agent_id`/target optionality) lives in `packages/core` (`useCreateChatSession`) — not
  `packages/views/chat`. Owner: `packages/core` chat-orchestration owner / Principal. Next: confirm the
  createChat contract for untargeted sessions (omit agent_id vs default), then C2 wires chat/** to it and
  verifies via the focused tests once BLK-C2-TESTS clears.

## Non-claims
No product edit (no verifiable, contract-correct, in-lock change available). No focused web test executed
(runner absent, install forbidden). No OmniRoute probe/map/provider/account/credential/session/rotation.
No deploy/inference/secret/dependency-install/commit/push/destructive-git; no OpenSpec checkbox. live_runs=false.
