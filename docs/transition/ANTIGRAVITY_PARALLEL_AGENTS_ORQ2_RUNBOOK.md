# Antigravity-Compatible Parallel Agents on ORQ2

Status date: 2026-07-20 (America/Sao_Paulo)

## 1. Verified addresses

| Purpose | Address | Audience |
| --- | --- | --- |
| Browser UI through the established local SSH tunnel | `http://127.0.0.1:20129/` | Windows/WSL operator workstation |
| OmniRoute private UI/API root | `http://orq1:20128/` | ORQ1/ORQ2 over Tailscale |
| OpenAI-compatible API root | `http://orq1:20128/v1` | controlled compatible clients |
| Anthropic Messages API | `http://orq1:20128/v1/messages` | Claude Code gateway frontend |
| Model registry | `http://orq1:20128/v1/models` | authenticated metadata/readiness checks |

The raw Tailscale-IP form is `http://100.118.244.61:20128/`. Prefer the MagicDNS name `orq1` from ORQ2 so normal IP changes do not require client reconfiguration. The local browser tunnel remains on port `20129` because local port `20128` was already occupied.

## 2. Current verified deployment

- OmniRoute runs only on ORQ1 and binds `100.118.244.61:20128`.
- ORQ2 resolves `orq1` to `100.118.244.61` and reaches TCP port `20128`.
- Unauthenticated `GET /v1/models` returns `401`, as required.
- The protected OmniRoute virtual key exists on ORQ2 at `/etc/agent-brain/secrets/omniroute-inference-key`, owner `ec2-user:ec2-user`, mode `600`.
- Claude Code `2.1.215` is installed on ORQ2.
- `claude-omniroute --version` succeeds without inference.
- An authenticated metadata-only registry request exposes 15 `agy/...` routes.
- No paid/model inference was executed during provisioning.

## 3. Critical Antigravity decision

The installed native Antigravity CLI is `agy 1.1.4`. Its verified command surface has no base-URL, endpoint, proxy, or gateway override. Therefore:

1. Native `agy` cannot currently be claimed to use OmniRoute.
2. Native `agy` remains fail-closed for gateway-required project execution.
3. Do not use DNS interception, transparent proxying, binary patching, or another workaround to pretend native compatibility.
4. Antigravity-owned models are used through the approved compatible frontend: Claude Code using the Anthropic Messages contract and an exact `agy/...` route.
5. OpenSpec task `5.8` remains the source of truth for native Agy acceptance. It is not complete until an actual supported endpoint contract is proven or the fallback path is formally accepted with evidence.

This preserves the target boundary: ORQ1/OmniRoute owns provider credentials and routing; ORQ2 workers receive only the scoped OmniRoute virtual key.

## 4. ORQ2 launcher

The reproducible launcher is `scripts/ops/claude-omniroute`. Its installed copy is `~/.local/bin/claude-omniroute` on ORQ2.

The launcher:

- fails before startup if ORQ1 is unreachable or the virtual key is missing;
- reads the key from the protected file and never places it in an argument;
- removes `ANTHROPIC_API_KEY` to prevent direct-provider fallback;
- sets `ANTHROPIC_BASE_URL=http://orq1:20128`;
- supplies the key as `ANTHROPIC_AUTH_TOKEN`;
- defaults to `agy/claude-opus-4-6-thinking`;
- permits another exact OmniRoute model only through an explicit `--model` argument.

List the currently registered Antigravity routes:

```bash
ssh orq2
omniroute-models
```

Start an interactive worker from its assigned worktree:

```bash
ssh orq2
cd ~/worktrees/<task-id>
agent_cred_isolation_status
claude-omniroute
```

Select another registered route explicitly:

```bash
claude-omniroute --model agy/claude-sonnet-4-6
```

Do not export or paste the virtual key manually. Do not run native `agy` for a task that is declared gateway-required.

## 5. Parallel operating model

Use one main architect/orchestrator plus three initial write workers on ORQ2. ORQ2 has approximately 8 GiB RAM; increase beyond three simultaneous write/build workers only after observing stable memory and swap. Model execution is remote, but repository builds and multiple CLI processes still consume local resources.

Each worker requires all of the following before it starts:

1. One unique OpenSpec task or bounded deliverable.
2. One explicit file/directory ownership set with no overlap.
3. One dedicated Git branch.
4. One dedicated Git worktree.
5. One isolated Herdr pane/credential home.
6. One written acceptance result returned to the main architect.

The architect is the single writer for task assignment, `AGENT_LEDGER.md` integration records, the integration branch, and final merge decisions. Workers do not self-assign adjacent tasks and do not edit the same files concurrently.

## 6. Worktree creation

The canonical clone is read-only for orchestration and integration. Never run multiple write agents in the same checkout.

```bash
BASE=integration/dev-transition-candidate-20260719
TASK=5.8-antigravity-fallback
BRANCH=topic/orq2-${TASK}
WORKTREE="$HOME/worktrees/${TASK}"

git -C "$HOME/R-D_Agnostic_Engineering_Team" fetch origin
git -C "$HOME/R-D_Agnostic_Engineering_Team" worktree add \
  -b "$BRANCH" "$WORKTREE" "origin/$BASE"
```

Before launching the worker:

```bash
git -C "$WORKTREE" status --short --branch
git -C "$WORKTREE" rev-parse HEAD
```

The expected starting commit must match the architect's recorded base. At handoff, the worker reports task ID, branch, base commit, final commit, files changed, focused validation, known gaps, and whether it pushed the topic branch.

## 7. No-duplicate-work control

The main architect must perform this sequence for every worker:

1. Read the current OpenSpec task state and the latest handoff/ledger.
2. Check whether another branch or worktree already owns the task or files.
3. Record the assignment before launching the worker.
4. Give the worker only its bounded task, files, acceptance criteria, and prohibited areas.
5. Require the worker to stop and report if it discovers overlapping ownership.
6. Review and integrate one topic branch at a time.
7. Update OpenSpec and the integration ledger only after evidence is accepted.
8. Delete the worktree only after its commit is pushed and integrated or explicitly abandoned.

## 8. Immediate recommended team

| Role | Scope | Write policy |
| --- | --- | --- |
| Main architect | decisions, dispatch, integration, OpenSpec/ledger truth | sole integration writer |
| Worker A | native Agy/fallback task `5.8` evidence and enforcement | only assigned runtime/docs paths |
| Worker B | accepted Antigravity route conformance portion of `8.2` | only assigned conformance/evidence paths |
| Worker C | remaining independent P0 task selected from current OpenSpec state | non-overlapping paths only |

Do not start ten uncontrolled writers. Additional agents improve speed only when tasks and files are independent; otherwise they create merge conflicts, duplicated investigation, and inconsistent decisions.

## 9. Stop and recovery

Stop all Claude-compatible workers on ORQ2 without touching OmniRoute on ORQ1:

```bash
pkill -TERM -u ec2-user -f 'claude|claude-omniroute'
sleep 5
pkill -KILL -u ec2-user -f 'claude|claude-omniroute' 2>/dev/null || true
```

Confirm no worker remains:

```bash
pgrep -a -u ec2-user -f 'claude|claude-omniroute' || true
```

Do not delete worktrees, branches, or uncommitted changes as part of process cleanup. Inspect and preserve each worktree before removal.

## 10. Remaining acceptance limitations

- Native Agy endpoint support is not proven.
- OpenSpec `5.8` remains open.
- Full accepted-path conformance under `8.2` remains open.
- The 20/50/100 capacity gates are not proven by this operational setup.
- The currently provisioned path enables controlled parallel continuation; it does not constitute final OmniRoute production cutover sign-off.
