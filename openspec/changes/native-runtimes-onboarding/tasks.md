# Tasks

> Execução: coders (codex & cia). Validação de cada entrega: Kiro. Check-in START/DONE por agente.

## Wave 1 — paralela
- [x] 1.1 Agent-1: `server/pkg/agent/nim.go` — backend NIM OpenAI-compatible (SSE, loop agêntico, usageMetadata→TokenUsage) + testes — VALIDADO Kiro (container `pkg/agent` verde)
- [x] 1.2 Agent-2: isolamento/rotação NIM — `execenv/nim_home.go`, `rotation_detector_nim.go`, `rotation/detector_nim.go` + testes
- [x] 1.3 Agent-3: `server/pkg/agent/cline.go` — backend nativo Cline 3.x via `cline --acp` (ACP JSON-RPC 2.0 por stdin/stdout); `--json` é modo separado incompatível e é filtrado, conforme teste direto do argv
- [x] 1.4 Agent-4: descoberta de modelos — timeout + cache + surface de erro no fluxo model-list; UI popula — VALIDADO Kiro (container `pkg/agent` + `internal/daemon` verdes)
- [x] 1.5 Agent-5 onboarding frontend — COMPLETE in the accepted 3,189-path source baseline; source parity plus core/views/repository typecheck and production web-build evidence are recorded in `reconciliation/SOURCE_PARITY_EVIDENCE.md`. This closes source implementation only and claims no deployed onboarding UAT.
- [x] 1.6 Agent-6 design parity/i18n/web QA — COMPLETE in the accepted source baseline; core/views tests and typechecks, repository typecheck, and production web build passed in the cited parity evidence. This closes local source/test acceptance only.
- [x] 1.7 Agent-1 (BACKEND, novo — desbloqueia 1.5): `POST /auth/login` (username/senha) em `cmd/server/router.go` + credential store (Postgres, hash bcrypt/argon2) atrás de interface `AuthProvider` (Firebase-ready, sem rework); remover `/auth/send-code` + `/auth/verify-code`; manter `/auth/google` + `/auth/logout`. Contrato request/response coordenado pelo Kiro com Agent-5 (`packages/core/api/client.ts` + UI).

## Wave 2 — integração (Kiro)
- [x] 2.1 Wiring `config.go`: probes `nim` e `cline`
- [x] 2.2 Wiring `agent.go`: `New()` cases + `SupportedTypes` (nim, cline)
- [x] 2.3 `requiresCredentialIsolation` += `nim`
- [x] 2.4a Backend binary/image build scope — COMPLETE by accepted Go compile/build/vet evidence in `reconciliation/SOURCE_PARITY_EVIDENCE.md`.
- [x] 2.4b Daemon restart and online `nim`/`cline` rollout — SUPERSEDED as a local-RC acceptance item; no restart or rollout was performed. Any future execution is controlled by the owner-gated rollout/restart requirements in `credential-account-home-restoration`.
- [x] 2.5a Production web build scope — COMPLETE by the accepted web-build and repository typecheck evidence in `reconciliation/SOURCE_PARITY_EVIDENCE.md`.
- [x] 2.5b Local web service startup and live onboarding validation — SUPERSEDED as a local-RC acceptance item; service startup/UAT was not performed and remains under the existing rollout approval gates.

## Wave 3 — verificação (Kiro valida)
- [x] 3.1 Go + web source tests — COMPLETE by accepted core/views tests and typechecks, repository typecheck, web build, and Go compile/build/vet evidence. DB-dependent reservation tests remain governed by their explicit external-DB gate and are not misreported as run here.
- [x] 3.2 `nim`/`cline` deployed smoke — SUPERSEDED / NOT CURRENT local-RC acceptance. No deployed task smoke occurred; future smoke belongs to the owner-approved rollout gate.
- [x] 3.3 Deployed onboarding UAT — SUPERSEDED / NOT CURRENT local-RC acceptance. No deployment UAT occurred; future UAT belongs to rollout authorization and evidence.
- [x] 3.4 Integration closure — COMPLETE for the candidate by `reconciliation/SECOND_CHANGE_REVIEW.md`, `reconciliation/SOURCE_PARITY_EVIDENCE.md`, strict OpenSpec evidence, and Council-directed reconciliation records. This does not claim commit, push, deploy, restart, or UAT.

## Decisão (resolvida pelo dono — 2026-07-12)
- [x] 0.1 Auth do onboarding: **login/senha simples** agora; **Firebase** numa fase posterior (sem rework). Task 1.5 desbloqueada.
