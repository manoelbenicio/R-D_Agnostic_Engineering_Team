<role>
You are Codex#5.5#B, senior Go engineer. Wire the policy-driven selection (policy + fallback +
load-balance) into the rotation SELECTION path. This is a SERIAL hotspot stream — EXCLUSIVE lock
on service.go/pool.go. "Done" = additive integration (AS-IS preserved), green package + no regression.
</role>

<mandatory_signin_signout priority="0" optional="false">
- BEFORE editing: write /mnt/c/VMs/Projetos/Automonous_Agentic/.deploy-control/Codex-5.5-B__RR-INTEGRATE__<START_UTC>.md
  (ABSOLUTE path). Front-matter PLANO (sem bullets): agent: Codex#5.5#B / stream: RR-INTEGRATE /
  started_at: <UTC> / finished_at: / status: IN_PROGRESS / files_locked: / build_result: / notes:
- AFTER: same file with finished_at + status:DONE|BLOCKED + build_result. UM arquivo por stream.
</mandatory_signin_signout>

<lock_discipline enforcement="SERIAL — you are the ONLY agent on these files">
files_locked (HOTSPOT): server/internal/rotation/service.go, server/internal/rotation/pool.go
Before editing, read ALL IN_PROGRESS check-ins on the board. If ANY other stream locks service.go
or pool.go, STOP and wait. Do NOT touch contract.go/policy.go/fallback.go/loadbalance.go (READ-only).
Depends on: RR-POLICY + RR-FALLBACK (done). Use RR-LOADBALANCE funcs if present; else FALLBACK only.
</lock_discipline>

<context source="/mnt/c/VMs/Projetos/Automonous_Agentic/openspec/changes/rotation-router/design.md §5, §10 — ABSOLUTE path, read first">
Existing SelectNext does naive priority-drain. Make it policy-driven, ADDITIVELY.
Reusable pieces already merged: policy.go (ResolvePolicy, RotationPolicy, Ordered),
fallback.go (NextBackoff, Jitter, ClassifyError, RetryPlan), loadbalance.go (PickConsistent/Weighted).
</context>

<task>
In service.go/pool.go, make selection policy-driven:
- Resolve a policy by workType (default GENERAL when unset) via ResolvePolicy.
- FALLBACK type: iterate p.Ordered(); per item apply RetryPlan/ClassifyError/NextBackoff+Jitter;
  move to next item on FAILOVER/exhausted; chain empty → ErrNoAccountAvailable.
- LOAD_BALANCING type: pick via PickConsistent(traceID/agentID) (if loadbalance.go present).
- **AS-IS PRESERVED**: if no policy configured / resolver returns default absent → keep the
  current priority-drain behavior EXACTLY. Feature is additive, never regressive.
- Do NOT change contract.go signatures. Keep changes minimal and surgical.
</task>

<example>
```
// workType=REVIEW → resolves review policy → selects strongest-first via fallback
// item returns retryable(429) → RetryPlan retries w/ backoff before next item
// item returns FAILOVER_NOW(401) → immediately next item
// no policy → identical to today's priority-drain (assert unchanged)
```
</example>

<verification note="canonical gate + non-regression">
cd /mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work
docker run --rm -v "$PWD":/src -w /src/server golang:1.26-alpine sh -c "go build ./... && go vet ./internal/rotation/... && go test ./internal/rotation/ -v" 2>&1 | tail -20
If a DATABASE_URL is available, also re-run the rotation E2E (-run E2E) to prove no regression.
Paste tail. DONE only on green package + no regression.
</verification>

<persistence>Finish fully; fix-and-rerun on red; never DONE on red. BLOCKED if service.go/pool.go already locked by another agent.</persistence>
<output>Sign-out: agent Codex#5.5#B, started_at, finished_at (UTC), status DONE, green tail in build_result.</output>
