---
name: herdr-dispatch
description: "Assign a GSD task-ID to a Herdr pane, submit the prompt with herdr pane run, collect agent output, validate evidence against EVIDENCE_CONTRACT.md, and mark DONE/BLOCKED. Use only inside Herdr (HERDR_ENV=1) and only after §7.1 = AUTORIZADO. Never accept DONE without a traceable artifact."
---

# herdr-dispatch — task dispatch + evidence validation

Execution + validation loop for Agent Brain tasks. Operator is the TL (Claude/GLM-5.2),
delegation-only. Workers are the four Codex agents running in Herdr panes (see
`herdr-fleet`).

## Preconditions (fail closed)

1. Herdr gate:
   ```bash
   test "${HERDR_ENV:-}" = 1
   ```
   Fail → stop.

2. Authorization: §7.1 of `OMNIROUTE_ARCHITECT_RESPONSE.md` MUST be `AUTORIZADO` AND the
   G1 freeze complete. Dispatching *production* task work before that is forbidden.
   (Read-only inspection is allowed.) If unsure, stop and surface the question.

3. The task MUST be a real task-ID in a `phases/G*/PLAN.md` (see `tasks.md`). No task
   outside a PLAN.md is dispatched (Golden Rule 11). Confirm the task-ID, its AB-REQ(s),
   its acceptance/evidence ID, its owning stream (Codex 1-4), and its `files_locked`.

## Learn the CLI, do not guess

```bash
herdr --help
herdr pane
herdr wait
```
Most commands print JSON; parse IDs/state from JSON. Do not run bare `herdr`. Do not probe
mutating nested commands without args (some execute with defaults).

## Dispatch loop

Given: task-ID `T`, AB-REQ(s), owning stream agent pane `P`, files_locked, evidence
ID `EV-...`.

1. Write/confirm the check-in in `.planning/agent-brain-v3/AGENT_LEDGER.md` for stream→P
   (status IN_PROGRESS, files_locked disjoint, start UTC) before any pane input.

2. Make sure the pane's agent is `idle` (or `done`) before submitting:

   ```bash
   herdr pane get <P>
   herdr wait agent-status <P> --status idle --timeout 30000
   ```
   `blocked` = needs input; `unknown` = not a detected/integrated agent → stop and report.

3. Submit the task. `pane run` sends text + Enter. Give the worker a precise, scoped
   prompt that names the task-ID, AB-REQ, files_locked (disjoint), the acceptance/evidence
   ID, and the redaction/no-secret rule. Do not add non-interactive flags.

   ```bash
   herdr pane run <P> "<concise scoped task: task-ID T; AB-REQ-x; files_locked: ...; acceptance EV-...; no secrets in output; follow EVIDENCE_CONTRACT>"
   ```

4. Watch the agent start working, then (for background work) wait for completion:

   ```bash
   herdr wait agent-status <P> --status working --timeout 30000
   herdr wait agent-status <P> --status done --timeout 120000
   ```
   If the user is watching the tab, completion may report `idle` instead — treat `idle`
   or `done` as completed. Always inspect before waiting (`pane get`, `pane read`) — do
   not blind-wait.

5. Collect the worker's reported result WITHOUT secrets/content:

   ```bash
   herdr pane read <P> --source recent-unwrapped --lines 200
   ```
   Use `--format ansi` only if terminal styling is itself evidence. Otherwise use text.
   Use the right read source: `visible` (viewport), `recent` (renders soft wraps),
   `recent-unwrapped` (joined — prefer for logs/transcripts), `detection` (bottom buffer).

## Evidence validation (binding — from EVIDENCE_CONTRACT.md)

A task is NOT done because the worker says so. DONE requires a traceable, reproducible
artifact. Distinguish **reviewed · implemented · verified · accepted**.

Reject (mark INVALID, revert to BLOCKED, keep the audit trail) if any:
- No provenance: missing command, host, OmniRoute version/digest, UTC timestamp, who ran.
- Non-real topology: localhost/temp port/mock/stub when a real one is required.
- 200 on `/v1/models` used as proof of protocol fidelity (it proves connectivity only).
- Failure "evidence" that describes a procedure instead of showing before/after.
- Numbers identical across independent runs/sessions (fabrication tell).
- Kill-switch / rollback described, not executed live.
- Logs not scrubbed: any match for secret/token/cookie/key in the evidence → INVALID.
- Sign-off forged (authored "as" the owner or another agent).
- Evidence predates the check-in/execution, or has no task-ID.

Smart Context: SC01–SC10 require shadow→canary→exact whole-request fallback→self-check
evidence OR a signed product+security waiver (see DECISIONS.md PD-03). A claim of "OmniRoute
has a similar feature" is NOT parity.

## Decide: DONE or BLOCKED

- DONE → update AGENT_LEDGER (status DONE, progress 100, evidence ID + location, end UTC),
  link evidence to the AB-REQ acceptance ID in `EVIDENCE_INDEX.md`. Only the TL marks DONE.
- BLOCKED → update AGENT_LEDGER (status BLOCKED + exact reason + what is needed), send the
  worker a follow-up via `herdr pane run <P> "<precise correction or question>"` or escalate
  to the owner. "Better an honest BLOCKED than a fake DONE."
- Anything needing a scope change, waiver, destructive action, production change, secret
  exposure, Prodex removal, tier 50/100, or cutover-default → ESCALATE, do not self-decide.

## Rules / safety

- Use `--no-focus` for background work; keep focus on the TL's pane.
- Use `--current` or an explicit pane ID; never rely on another client's focused pane.
- Inspect before waiting. Read current output first, then wait for the next state/output.
- Do not close panes/tabs/workspaces/sessions you did not create unless explicitly asked.
- Never run `herdr server stop` or kill the main process unprompted.
- The TL does not execute product code or run `prodex`/provider CLI directly. The TL
  dispatches and validates; workers execute.
- No secrets in prompts you send either — never paste the OmniRoute key, provider creds,
  cookies, prompts, repo content, or tool payloads into a pane.
