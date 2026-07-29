# G4 accelerated development-validation packet

Prepared read-only by Kiro/Opus-4.8 on 2026-07-18. Persisted by Codex#56#A so the plan does
not depend on pane transcript context. **Do not dispatch until G3 evidence and isolation smoke
are independently accepted.** Use isolated Herdr panes, never the current Multica daemon.

## Global constraints for every worker

- §7.1 Waves 0–3/tier-20 development validation only; no production claim.
- PD-01: preserve dirty baseline; no reset/stash/revert/discard.
- PD-08: no credential/auth/secret read, copy, print, rewrite, rotation, quarantine or mutation.
- Synthetic or reference-only values only. No live provider accounts or traffic.
- Central daemon/config/health/cmd/go.mod/execenv/models/brain/prodex files are Codex1-only.
- No cutover, Prodex removal, tier 50/100 or native 5.6–5.8 acceptance.
- Stop if work requires a real secret, Multica daemon, live provider, G3-unaccepted behavior,
  central hotspot edit or tier activation.

## Stream A — Codex2 `w3:p8`: gateway protocol/failure tests

Tasks 8.1, 8.4–8.7 gateway portion · EV-G4-01/04/05/06/07 · scope `gateway/**` only.

Build synthetic offline protocol-conformance and failure-injection tests: Messages, Responses,
Chat and Antigravity mock transports; concurrent strict RR and continuation affinity; expired,
revoked, quota, scoped/global 429, 5xx, timeout and malformed upstream; safe pre-output retry,
no replay after output/tool commit, dedup and cancel release; synthetic account lifecycle and
restart/rollback. Run focused test/race/vet. Stop rather than use live OmniRoute/accounts/secrets.

## Stream B — Codex3 `w3:p9`: credentialless isolation/adapters

Tasks 8.2/8.3 · EV-G4-02/03/COD/ADP/NIM/AGY · scope `runtimeenv/**` and owned
`pkg/agent/{claude,codex,kimi,nim,antigravity}.go`; execenv/models are read-only.

Prove child env/homes/process/logs contain no provider credential, auth file, cookie or direct
provider endpoint with synthetic/reference-only values. Exercise Claude/Codex trusted gateway
paths against the gateway mock. For Kimi/GLM/NVIDIA/NIM/Agy, validate fail-closed contracts only;
do not claim native acceptance. Run focused test/vet. Any provider auth read/copy is STOP.

## Stream C — Codex4 `w3:pA`: evidence/observability/capacity

Tasks 8.1/8.4–8.8 records and 9.1 · EV-G4-01..08/CAP · scope `observability/**`, `deploy/**`,
`evidence/g4-*.md` and `EVIDENCE_INDEX.md`.

After A/B artifacts exist, generate redacted checklist/parity records with honest
Supported/Partial/Not-supported disposition. Execute only the synthetic 20-task development
profile and record acceptance/completion/failure, p50/p95/p99 selection/TTFT/E2E, retry/fallback,
queue/resources/sockets and mock-account fairness. Do not enable tier 20; task 9.2 remains Codex1.

## Ordering and modernization

1. Accept G3 evidence and credentialless isolation smoke.
2. Run A+B in parallel; C consolidates after their artifacts exist.
3. Event-driven read-only status monitoring over schema-v1 events replaces repeated transcript reads.
4. Codex4 makes evidence generation deterministic with provenance and synthetic-only manifests.
5. Vendor/model visibility UI repair starts only after active-path safety.
6. Kanban MUL-2..25 stays parked; reconcile/supersede MUL-11/12/15 and automate dispatch only
   after credentialless execution is proven end-to-end.
