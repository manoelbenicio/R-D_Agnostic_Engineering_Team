# Tasks

> Execução: coders (codex & cia). Validação de cada entrega: Kiro. Check-in START/DONE por agente.

## Wave 1 — paralela
- [x] 1.1 Agent-1: `server/pkg/agent/nim.go` — backend NIM OpenAI-compatible (SSE, loop agêntico, usageMetadata→TokenUsage) + testes — VALIDADO Kiro (container `pkg/agent` verde)
- [x] 1.2 Agent-2: isolamento/rotação NIM — `execenv/nim_home.go`, `rotation_detector_nim.go`, `rotation/detector_nim.go` + testes
- [x] 1.3 Agent-3: `server/pkg/agent/cline.go` — backend nativo via `cline --acp --json` + testes
- [x] 1.4 Agent-4: descoberta de modelos — timeout + cache + surface de erro no fluxo model-list; UI popula — VALIDADO Kiro (container `pkg/agent` + `internal/daemon` verdes)
- [x] 1.5 Agent-5: onboarding (FRONTEND) — remover `(landing)`/`features/landing`/`content/use-cases`/sponsors + fluxo de código por email; `AuthService` interface + `SimpleAuthService`→`api.login()` (Firebase-ready) + UI login/senha no design-system; manter Google OAuth/CLI callback/desktop handoff. **A5 é o dono da remoção de marketing/landing/sponsors** (A6 NÃO remove marketing).
- [x] 1.6 Agent-6: paridade de design (tokens/cores kanban/agentes), limpeza de i18n, harness build/test do web e QA. **NÃO remove marketing (isso é A5)**; arquivos disjuntos de A5.
- [ ] 1.7 Agent-1 (BACKEND, novo — desbloqueia 1.5): `POST /auth/login` (username/senha) em `cmd/server/router.go` + credential store (Postgres, hash bcrypt/argon2) atrás de interface `AuthProvider` (Firebase-ready, sem rework); remover `/auth/send-code` + `/auth/verify-code`; manter `/auth/google` + `/auth/logout`. Contrato request/response coordenado pelo Kiro com Agent-5 (`packages/core/api/client.ts` + UI).

## Wave 2 — integração (Kiro)
- [ ] 2.1 Wiring `config.go`: probes `nim` e `cline`
- [ ] 2.2 Wiring `agent.go`: `New()` cases + `SupportedTypes` (nim, cline)
- [ ] 2.3 `requiresCredentialIsolation` += `nim`
- [ ] 2.4 Rebuild `server/bin/multica` + imagem backend; restart daemon; runtimes `nim`/`cline` online
- [ ] 2.5 Build + subir web local; validar onboarding novo

## Wave 3 — verificação (Kiro valida)
- [ ] 3.1 Testes verdes Go + web em container
- [ ] 3.2 Smoke: criar agente em `nim` e `cline`, rodar 1 task, ver execução + tokens
- [ ] 3.3 UAT onboarding (sem sponsors/email-code; cores idênticas)
- [ ] 3.4 Check-ins DONE + relatório de integração em `.deploy-control/`

## Decisão (resolvida pelo dono — 2026-07-12)
- [x] 0.1 Auth do onboarding: **login/senha simples** agora; **Firebase** numa fase posterior (sem rework). Task 1.5 desbloqueada.
