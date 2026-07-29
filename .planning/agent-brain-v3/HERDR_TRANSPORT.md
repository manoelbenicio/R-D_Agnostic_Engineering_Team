# Herdr transport — Kiro/Opus-4.8 + Codex#56#A co-leads ↔ Codex workers

Kiro/Opus-4.8 owns planning/adjudication. Codex#56#A is operational co-lead: independently
verifies state/evidence, records decisions, operates Herdr, monitors transitions and controls
dispatch. Kiro does not send competing worker instructions; Codex#56#A does not unilaterally
change architecture. The owner closed the former Claude pane `w3:p5` on 2026-07-18.

## Preconditions

```bash
test "${HERDR_ENV:-}" = 1
herdr --help
herdr pane
herdr wait
```

Never run bare `herdr`, never change focus for background work, and never close a pane or
session unless the owner explicitly requests it.

## Current workspace and stable pane map

| Role | Pane | Label |
|---|---|---|
| Planning/adjudication TL | `w3:p3` | `Kiro#Opus48-TL` |
| Operational co-lead / transport / verification | `w3:p1` | `Codex#56#A` |
| Codex 1 / Brain-integrator | `w3:pD` | `codex1-brain` |
| Codex 2 / Gateway | `w3:p8` | `codex2-gateway` |
| Codex 3 / Runtime-security | `w3:p9` | `codex3-runtime` |
| Codex 4 / Ops-parity | `w3:pA` | `codex4-ops` |
| Spare | `w3:pB` | `Codex#56#F` |

Treat every pane ID as opaque. Refresh the map after any move, split, close, or restart:

```bash
herdr workspace list
herdr pane list --workspace w3
```

## Inspect without stealing focus

```bash
herdr pane get w3:pD
herdr pane read w3:pD --source recent-unwrapped --lines 120
```

Replace the pane ID with `w3:p8`, `w3:p9`, or `w3:pA`. Inspect current output before
waiting for future state.

## Dispatch and follow up

The latest phase entry in `DISPATCH_QUEUE.md` is authoritative. Historical G1/G2 prompts are
audit records and must not be replayed. Submit one current prompt and Enter:

```bash
herdr pane run <pane-id> "<approved co-lead prompt>"
```

Use the same command for a follow-up. Do not use argv prompts, non-interactive agent flags,
or separate `send-text`/`send-keys` for normal prompts.

## Wait and collect

```bash
herdr wait agent-status <pane-id> --status working --timeout 30000
herdr wait agent-status <pane-id> --status done --timeout 120000
herdr pane get <pane-id>
herdr pane read <pane-id> --source recent-unwrapped --lines 160
```

If the pane is visible to the user, completion may report `idle` rather than `done`. Treat
`idle` or `done` as completed only after inspecting the pane and required evidence files.
On timeout, inspect `pane get` and `pane read`; do not declare failure from timeout alone.

## Evidence and safety

- Never accept DONE without the artifact and evidence ID required by `DISPATCH_QUEUE.md`.
- Never print, relay, or inspect raw credentials, tokens, cookies, auth payloads, or secret files.
- Never reset, stash, revert, or discard the preserved Prodex baseline.
- Codex 1 exclusively owns the Prodex/central-daemon hotspots.
- Record real timestamps, pane IDs, status, and evidence in `AGENT_LEDGER.md`.
- Do not dispatch through the current Multica daemon until G3 credentialless wiring and the
  isolation smoke pass. Use the already isolated Herdr panes meanwhile.

## Credential-isolation monitoring

The four reauthentication events on 2026-07-18 created distinct physical private login
directories. During monitoring: global `~/.codex/auth.json` remained absent, saved slots did
not change, all auth files remained mode `0600`, and no shared inode or login-to-login content
duplication appeared. Any future global write, slot mutation, symlink, shared inode, or new
cross-login duplicate is a critical stop condition.
