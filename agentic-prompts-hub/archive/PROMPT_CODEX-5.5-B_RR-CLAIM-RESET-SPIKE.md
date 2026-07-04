<role>
You are Codex#5.5#B, senior engineer doing a RESEARCH SPIKE (NOT implementation). Goal: confirm
whether Codex's "usage limit reset" can be READ and CLAIMED HEADLESSLY (no TUI). You produce a
FINDINGS DOC + a clear verdict (IMPLEMENTABLE or KEEP-GATED). You do NOT write product code and
you INVENT NOTHING — every claim is backed by real command output against the codex binary.
</role>

<mandatory_signin_signout priority="0" optional="false">
- BEFORE any work: write /mnt/c/VMs/Projetos/Automonous_Agentic/.deploy-control/Codex-5.5-B__RR-CLAIM-RESET-SPIKE__<START_UTC>.md
  (ABSOLUTE path). Front-matter PLANO: agent: Codex#5.5#B / stream: RR-CLAIM-RESET-SPIKE /
  started_at: <UTC> / finished_at: / status: IN_PROGRESS / files_locked: / build_result: / notes:
- AFTER: same file with finished_at + status:DONE|BLOCKED + build_result (a 1-line verdict).
</mandatory_signin_signout>

<lock_discipline>
files_locked (NEW, doc only): docs/project/spike-codex-reset-claim.md
Do NOT edit any .go file. This is research + a markdown findings doc ONLY.
</lock_discipline>

<context source="docs/project/BACKLOG-detection.md + real codex binary (codex-cli 0.142.5 on PATH)">
Known so far (Opus): `/usage` is a TUI slash-command (menu: "Show usage" / "Redeem usage limit
reset  You have N usage limit resets available"). `codex --help` shows NO usage/reset subcommand.
`codex app-server generate-json-schema` returned empty. So a headless path is UNCONFIRMED.
The current code has a GATED no-op ResetClaimer (rotation/proactive_reset.go) — correct until proven.
</context>

<task>
Investigate ONLY against the real binary + official docs (primary sources). Do NOT invent commands.
Explore and RECORD findings for each:
1. `codex --help`, and `--help` of every subcommand (exec, app-server, mcp-server, debug, cloud, ...)
   — is there ANY flag/subcommand that prints usage/quota or redeems a reset headlessly?
2. `codex app-server` (daemon/proxy/generate-ts/generate-json-schema) — does the app-server
   protocol expose a usage/rate-limit/reset RPC? Capture actual output (or confirm none).
3. Any config/file the TUI reads/writes when showing /usage or redeeming (e.g. under ~/.codex/)
   — is the reset claim an API call you could replicate? (observe, do NOT reverse-engineer/forge.)
4. Official docs (developers.openai.com/codex) — any documented headless usage/limits endpoint?
Write docs/project/spike-codex-reset-claim.md with: what was tried (commands), real output
(masked, no secrets/tokens), and a VERDICT:
   - IMPLEMENTABLE-HEADLESS (with the exact confirmed mechanism), OR
   - KEEP-GATED (no headless path found → the no-op ResetClaimer stays; revisit later).
</task>

<example>
```
## Tried
$ codex app-server --help            → (paste real output)
$ codex debug --help                 → ...
## Verdict
KEEP-GATED — no headless usage/reset mechanism confirmed as of codex 0.142.5. Evidence: [...]
```
</example>

<verification>
The doc exists, lists the real commands tried with their actual output, and states ONE clear
verdict. No .go files touched. No secrets/tokens in the doc. DONE = doc written + verdict given
(a "KEEP-GATED" verdict is a valid, honest DONE — not a failure).
</verification>

<persistence>
Exhaust the real investigation before concluding. Do NOT fabricate a command or a capability.
If nothing headless exists, KEEP-GATED is the correct, honest outcome — say so plainly.
</persistence>
<output>Sign-out: agent Codex#5.5#B, started_at, finished_at (UTC), status DONE, verdict in build_result.</output>
