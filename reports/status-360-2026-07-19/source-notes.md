# OpenSpec Delivery 360 — Source and Reproducibility Notes

Snapshot date: 2026-07-19 (America/Sao_Paulo)

## Task inventory

Source files:

- `openspec/changes/agent-credential-isolation/tasks.md`
- `openspec/changes/build-omniroute-agent-brain/tasks.md`
- `openspec/changes/chat-orchestration-standard/tasks.md`
- `openspec/changes/native-runtimes-onboarding/tasks.md`
- `openspec/changes/persist-prodex-runtime-integration/tasks.md`
- `openspec/changes/rotation-parity-polyglot/tasks.md`

Reproduction command:

```bash
for f in openspec/changes/*/tasks.md; do
  total=$(rg -c '^\s*- \[[ xX]\]' "$f" || true)
  donec=$(rg -c '^\s*- \[[xX]\]' "$f" || true)
  open=$(rg -c '^\s*- \[ \]' "$f" || true)
  printf '%s\t%s\t%s\t%s\n' "$f" "$total" "$donec" "$open"
done
```

Reviewed result:

| Change | Complete | Open | Total |
|---|---:|---:|---:|
| agent-credential-isolation | 4 | 17 | 21 |
| build-omniroute-agent-brain | 51 | 34 | 85 |
| chat-orchestration-standard | 4 | 6 | 10 |
| native-runtimes-onboarding | 9 | 8 | 17 |
| persist-prodex-runtime-integration | 0 | 16 | 16 |
| rotation-parity-polyglot | 78 | 0 | 78 |
| **Total** | **146** | **81** | **227** |

Active incomplete changes exclude `rotation-parity-polyglot`: 68 complete of 149, or 45.6%.

## Architecture and acceptance state

Primary documents:

- `openspec/changes/build-omniroute-agent-brain/architecture.md`
- `.planning/agent-brain-v3/STATE.md`
- `.planning/agent-brain-v3/EVIDENCE_INDEX.md`
- `.planning/agent-brain-v3/G3_SECURITY_CORRECTION_PLAN.md`

The AS-IS/TO-BE table is a qualitative reconciliation of these files. “Current actual” never upgrades a documentary, synthetic, local-only, compiled-only, or fail-closed result to live or production acceptance.

## Git and recoverability snapshot

Read-only commands used:

```bash
git status --porcelain=v1
git diff --stat
git diff --cached --stat
git ls-files --others --exclude-standard
git rev-list --left-right --count origin/main...HEAD
git branch --contains 5106de35
```

Independent internal audit result:

- 469 dirty entries: 92 modified, 5 added, 1 deleted, 371 untracked.
- 11 staged frontend files; latest ownership review excludes them from push.
- `HEAD` at `b657129`; no commit ahead of locally cached `origin/main`.
- Local branch `backup/wip-snapshot-20260718T202300Z` contains commit `5106de35`.
- No remote branch was found containing `5106de35` in the local ref snapshot.
- This internal audit was performed by a read-only sub-agent named `kiro_audit`; it was not the visible Kiro/Herdr TL session.

## Fresh verification

Commands and results from the current checkout:

| Command | Result | Evidence |
|---|---|---|
| `pnpm typecheck` | PASS | 6 successful Turbo tasks, 7m14s |
| `pnpm test` | FAIL | 569 passing tests; 8 Vitest worker-start timeouts; exit 1 |
| `cargo test` | PASS | 4 passed, 0 failed |
| `go test ./...` | NOT RUN | `go: command not found` |

No database-backed integration suite, deployment, credential mutation, fetch, commit, or push was performed.

## Chart map

| Report section | Question | Chart | Fields | Claim |
|---|---|---|---|---|
| Legacy completion masks the active delivery gap | How are checked and open tasks distributed by change? | Horizontal stacked bar | `change`, `completed`, `open` | The completed 78-task legacy stream inflates the all-program headline; active work remains 45.6% complete. |

Palette policy: two-root categorical (completed versus open) with labels and an exact lookup table so the distinction does not rely on color alone.
