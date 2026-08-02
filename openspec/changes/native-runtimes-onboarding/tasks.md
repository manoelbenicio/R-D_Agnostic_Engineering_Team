# Tasks

> Execução: coders (codex & cia). Validação de cada entrega: Kiro. Check-in START/DONE por agente.

## Wave 1 — paralela
- [x] 1.1 Agent-1: `server/pkg/agent/nim.go` — backend NIM OpenAI-compatible (SSE, loop agêntico, usageMetadata→TokenUsage) + testes — VALIDADO Kiro (container `pkg/agent` verde)
- [x] 1.2 Agent-2: isolamento/rotação NIM — `execenv/nim_home.go`, `rotation_detector_nim.go`, `rotation/detector_nim.go` + testes
- [x] 1.3 Agent-3: `server/pkg/agent/cline.go` — backend nativo Cline 3.x via `cline --acp` (ACP JSON-RPC 2.0 por stdin/stdout); `--json` é modo separado incompatível e é filtrado, conforme teste direto do argv
- [ ] 1.4 Agent-4: descoberta de modelos — REOPEN. ORQ-87 proved the current `pkg/agent` timeout/cache slice, but the candidate-bound `internal/daemon` slice could not compile offline because required Go modules were absent from the isolated cache.
- [ ] 1.5 Agent-5 onboarding frontend — REOPEN. Candidate bytes are pinned by ORQ-87, but current frontend tests/typecheck did not execute because pnpm 10.28.2 and the dependency tree were unavailable offline. No `reconciliation/SOURCE_PARITY_EVIDENCE.md` exists.
- [ ] 1.6 Agent-6 design parity/i18n/web QA — REOPEN. Earlier evidence records scoped tests and typechecks, but also records the production web build as blocked/non-claim; ORQ-87 could not execute the current web toolchain offline.
- [ ] 1.7 Agent-1 backend password auth — REOPEN. Candidate bytes, including `auth_routes_test.go`, are pinned by ORQ-87, but the focused current auth suite could not compile offline because required Go modules were absent. Historical attribution and residual-scope qualifications remain.

## Wave 2 — integração (Kiro)
- [ ] 2.1 Wiring `config.go`: probes `nim` e `cline` — REOPEN pending candidate-bound daemon-package execution.
- [x] 2.2 Wiring `agent.go`: `New()` cases + `SupportedTypes` (nim, cline)
- [ ] 2.3 `requiresCredentialIsolation` += `nim` — REOPEN pending candidate-bound daemon-package execution.
- [ ] 2.4a Backend binary/image build scope — OPEN. ORQ-87's offline build and vet attempts stopped at missing-module setup; no binary/image acceptance is claimed.
- [ ] 2.4b Daemon restart and online `nim`/`cline` rollout — NOT PERFORMED and outside ORQ-87. Supersession is a scope disposition, not completed acceptance.
- [ ] 2.5a Production web build scope — OPEN. ORQ-87's forced-offline invocation stopped before build because the pinned pnpm toolchain was unavailable.
- [ ] 2.5b Local web service startup and live onboarding validation — NOT PERFORMED and outside ORQ-87.

## Wave 3 — verificação (Kiro valida)
- [ ] 3.1 Go + web source tests — OPEN. ORQ-87 has one current bounded Go package pass; remaining Go and web slices have preserved setup failures, not passing assertions.
- [ ] 3.2 `nim`/`cline` deployed smoke — NOT PERFORMED and explicitly outside ORQ-87.
- [ ] 3.3 Deployed onboarding UAT — NOT PERFORMED and explicitly outside ORQ-87.
- [ ] 3.4 Integration closure — OPEN. Neither `reconciliation/SECOND_CHANGE_REVIEW.md` nor `reconciliation/SOURCE_PARITY_EVIDENCE.md` exists, and ORQ-87 did not fabricate either artifact.

## Decisão (resolvida pelo dono — 2026-07-12)
- [x] 0.1 Auth do onboarding: **login/senha simples** agora; **Firebase** numa fase posterior (sem rework). Task 1.5 desbloqueada.

## ORQ-87 candidate-bound reconciliation — 2026-08-02

The recovered all-checked state was invalid. The reconciled classification is five bounded-direct criteria (`0.1`, `1.1`, `1.2`, `1.3`, `2.2`), five reopened criteria (`1.4`, `1.5`, `1.7`, `2.1`, `2.3`), and nine contradicted completion claims reopened above (`1.6`, `2.4a`, `2.4b`, `2.5a`, `2.5b`, `3.1`, `3.2`, `3.3`, `3.4`). Exact candidate hashes, commands, exit codes, contradictions, and residuals are recorded under `.deploy-control/evidence/ORQ-87/`.
