> **Nao certifica release.** Os checkboxes deste change referem-se ao escopo do proprio
> build-omniroute-agent-brain e **nao** certificam a integracao credential-account-home /
> C1+C2+C3+C4 como pronta para release. Naquela integracao o estado corrente e HOLD / NOT READY:
> `go build ./...` passa e as suites de pacote revisadas estao verdes. A composicao concreta do
> Runtime Manager em PostgreSQL esta implementada e validada na fonte compartilhada `spe6`, com
> idempotencia duravel, locking de parent, UUID esperado fail-closed, `apply_class` persistido e
> mount sob router/middleware no startup; ela ainda **nao** foi importada e revisada na arvore de
> release aceita nem verificada no alvo. Todos os gates locais estao aprovados, incluindo banco real
> em PostgreSQL descartavel, suite completa com e sem `-race` e `go vet ./...`. O HOLD remanescente e
> a transferencia seletiva serial do owner, a revisao aditiva de `router.go`/`main.go`, a verificacao
> de bytes e spill de sqlc no alvo, e os gates consolidados de Go, C4 e SPE-7 no alvo. O
> trabalho sistemico de companions na arvore aceita esta pendente, e a correcao de identidade de
> credencial e apenas de codigo-fonte. A autoridade de fixture SPE-7 ja e livre de Git, com suite
> 28/28 aprovada com `git` ausente do `PATH`. Nao houve push, deploy ou restart.
> Ver `credential-account-home-restoration/tasks.md` Fase 5.
>
> **Nomes externos canonicos do Runtime Manager:** `version` como discriminador de documento,
> `configuration_digest` e `capability_digest` como digests, sempre hex minusculo de 64 caracteres
> sem prefixo. Nao existem `active_digest` nem `effective_configuration_digest`. O `schema_version`
> interno do preimage de digest **permanece inalterado**, pois renomea-lo mudaria todo digest ja
> produzido e invalidaria fixtures pinados.

## 1. Single-router runtime removal

- [x] 1.1 Delete dedicated alternate-router binaries, sidecars, runtime packages and legacy account-rotation package.
- [x] 1.2 Delete dedicated daemon startup/facade/profile/session consumers and tests.
- [x] 1.3 Delete dedicated smoke scripts, prompts, operational documents and obsolete OpenSpec changes.
- [x] 1.4 Remove alternate startup, shutdown, credential-home selection, retry/rotation and task-time routing branches from `daemon.go`.
- [x] 1.5 Remove alternate config structs, environment aliases and CLI migration flag.

## 2. OmniRoute-only Main Brain contract

- [x] 2.1 Restrict router identity to `omniroute` and reject non-gateway task contracts.
- [x] 2.2 Make a missing Agent Brain/OmniRoute admission plan fail closed before CLI launch.
- [x] 2.3 Preserve workspace/repository/worktree, context/skills, process lifecycle, watchdog, cancellation, stream, commit-ledger and terminal-result paths.
- [x] 2.4 Keep controlled `runtimeenv`/`execenv` construction and provider-secret/direct-endpoint exclusion.
- [x] 2.5 Remove direct provider credential discovery and provider-auth environment injection.

## 3. Recovery and health

- [x] 3.1 Implement `NORMAL(omniroute)` and `DEGRADED(none)` recovery states only.
- [x] 3.2 Require a session boundary for outage/restore and strict readiness for restore.
- [x] 3.3 Add tests proving an outage never promotes an alternate router.
- [x] 3.4 Remove alternate runtime and migration-mode health fields; report OmniRoute authority and Main Brain diagnostics.

## 4. Product and operations preservation

- [x] 4.1 Preserve backend/web, projects, squads, Kanban issues/tasks, Postgres and terminal/session persistence.
- [x] 4.2 Update README and self-host Compose comments for the Main Brain/OmniRoute topology.
- [x] 4.3 Rewrite rollout and rollback runbooks to keep admission closed instead of activating a direct/alternate route.
- [x] 4.4 Update typed deployment catalog wording and secret-reference failure policy.
- [x] 4.5 Document and exercise Kanban-only dispatch with exactly one product task per
  activation; prohibit parallel Herdr execution.

## 5. Verification

- [x] 5.1 Run `gofmt` on every modified Go file using an available pinned Go toolchain.
- [x] 5.2 Run targeted tests for `internal/daemon/brain`, `internal/daemon/gateway`, `internal/daemon/runtimeenv`, `internal/daemon/execenv`, `internal/daemon/deploy` and `internal/daemon`.
- [x] 5.3 Run a server-wide build/test check and resolve compile fallout from deleted symbols.
- [x] 5.4 Run `openspec validate build-omniroute-agent-brain --strict`.
- [x] 5.5 Run residual scans proving no active startup/config/health/recovery/fallback reference remains outside preserved historical evidence.

## 6. Operational work still requiring external readiness/evidence

- [x] 6.1 Consume the owner/operator OmniRoute immutable-revision and selected-route/protocol readiness declaration, and verify only Main Brain's strict fail-closed reaction to ready/not-ready signals; do not inspect, test or validate OmniRoute provider/model mappings, accounts, credentials, sessions or rotation internals.
- [x] 6.2 Complete metadata-only end-to-end correlation for ingress, queue, daemon, CLI, gateway, terminal persistence and UI delivery.
  > **Limitacao registrada (2026-08-01):** `agent.Result` nao expoe um `ProcessID` real, portanto o
  > `proc_id` do span de CLI permanece **fail-closed e nao fabricado**, com o wiring do recorder
  > preservado. A correlacao e completa em todos os outros elos; o `proc_id` de CLI e ausente por
  > decisao, nunca inventado. Fecha-lo exige expor o identificador de processo real em
  > `agent.Result`, o que esta fora do escopo deste change.
- [ ] 6.3 Validate and approve the 20-task bounded capacity profile; keep the current lower limit until evidence passes.
- [ ] 6.4 Validate 50- and 100-task profiles only after the lower tier and observability gates pass.
- [x] 6.5 Perform an owner-approved Kanban → Main Brain → OmniRoute → terminal acceptance run in the controlled environment; do not perform it as part of basic source validation.
