# Open Items

**BLOCKER:** None for local P6/P7 gate closeout.

**SCOPE NOTE:** Provider-backed PROD session evidence is not part of the local
`rpp.l2.v1` sidecar closeout and should be attached separately for a production
release.

**STANDING RULE:** Hotspot files `daemon.go`/`config.go`/`execenv.go` require explicit lock lines.

1. Copy artifacts into target repo.
2. Opus 4.8 records coordination.
3. Agents create check-in files.
4. Review/accept generated docs.
5. Produce implementation-specific commands.
6. Obtain owner approval before real PROD deploy.

## Open Items per Owner

### Codex#5.5#C (F3)
- [DONE] l2runtime client unwired into daemon lifecycle.
- [DONE] legacy Go rotation not gated for L2-owned sessions (one-router gate GREEN).
- [IN PROGRESS] StartSession wiring.
- [IN PROGRESS] Amending STALE files_locked (omitted 7 Go files).

### Product Owner
- [DECISION] F5 sign-off (8 not_validated vendor cells are GENUINE gaps).
- [DECISION] F0 canary go.
- [O1] (hygiene) .env.production is git-tracked despite .gitignore rule **/.env.production (only public values; owner to decide untrack).

### GLM#52#B (F4)
- [IN PROGRESS] Conformance/QA items (C1-C6, replay, capability, redaction-LIVE, kill-switch-LIVE, profile-fail-closed-LIVE) - pending real-exec.
- Postgres/Redis state backend.
- Redaction policy, audit event taxonomy, secrets boundary.

### GLM#52#A (F6)
- Start QA/conformance (smoke, replay, PROD validation plan).

### Codex#5.5#B & GLM#52#A (F9)
- [DEFERRED/GATED] Reset-claim empirical validation (planning DONE; gated on real account state).

### Codex#5.5#D (F7)
- [IN PROGRESS] Runbook wording reconcile.

### Opus 4.8
- [DONE] Ownership audit (NO collisions; C amending stale locks; minor outliers self-attributed).
- Vendor capabilities `not_validated` resolution or owner acceptance.
- Owner approval for F7 runbook (NO-GO for deploy until approved).
