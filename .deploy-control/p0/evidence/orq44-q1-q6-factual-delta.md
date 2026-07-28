# ORQ-44 — Q1–Q6 factual delta (READ-ONLY)

**Skill:** `aws-secrets-manager` loaded in full. No secret value, SMA,
`GetSecretValue`, `asm-exec`, provider call, Docker, DB, task, restart, board
or rotation was executed.

## Facts established locally

- **Health derivation:** `healthPortForProfile` is deterministic. Empty profile
  maps to `19514`; a named profile maps to `19514 + 1 + sum(profile bytes)%1000`.
  The runbook must still prove the profile source and bind socket PID,
  `MainPID`, `/health.pid`, and `/health.daemon_id`; a config override or stale
  listener cannot be accepted from port/HTTP 200 alone.
- **Identity:** `/health` publishes daemon ID, PID and OS; daemon `/api/daemon/ws`
  uses the `mdt_` hash path and is unrelated to this gateway key. `agent_brain`
  readiness fields are snapshots from the last `admitTask`, not a post-swap
  credential probe.
- **Canary:** `/v1/models` is the configured readiness endpoint and the design
  caps it at one pre-swap request, no retry/loop. Its quota/cost semantics are
  not proven by repository facts.
- **Secret reference:** the only candidate identity in the artifacts is
  `prod/multica/omniroute-inference-key`; no authorized metadata response proved
  that it exists, its region, JSON key or account. This remains Q6, not a fact.

## Q1–Q6 answers

| Q | Factual result | Gate |
|---|---|---|
| Q1 overlap/revoke | No provider evidence. The local consumer holds one file reference and sends one credential per request; overlap semantics are external. | **BLOCK: operator answer required** |
| Q2 key scope | No provider metadata or account evidence. Cannot determine whether key is account/global or the affected account set. | **BLOCK: operator answer required** |
| Q3 `/v1/models` cost/introspection | Repository confirms endpoint purpose, not quota billing or a zero-cost introspection alternative. | **BLOCK: owner/operator must declare one-call cost and authorize it** |
| Q4 `AWSPREVIOUS` | `describe-secret`/`list-secret-version-ids` are metadata-only and permitted by the skill, but were not executed and no result is available. | **BLOCK until metadata boolean is captured by authorized operator** |
| Q5 post-swap positive proof | No cost-free post-swap positive probe is established. The safe available proof is pre-swap one-call canary plus post-swap regression/health; stronger proof needs explicit cost authorization. | **BLOCK: owner decision required** |
| Q6 secret identity | Candidate name only; full ARN, region, JSON key, stage and IAM identity are unproven. | **BLOCK: owner/operator must provide/confirm metadata** |

## Exact external action required

An authorized operator must provide redacted booleans/metadata only: full ARN
and region (no value), confirmed JSON key/stage, `AWSPREVIOUS` present or
absent, provider overlap/revoke scope and quota semantics for one `/v1/models`
call, and the owner decision on Q5. If any answer is unavailable or
`AccessDenied` occurs, the runbook remains stopped. No agent should infer these
facts or invoke the provider to discover them.

The V5/V6 proposal should be amended to mark Q1–Q6 with these exact blockers;
V4c must remain explicitly labeled as stale snapshot/non-proof, and no
`agent_brain` readiness field may be treated as a key-validity result.
