# Tasks

> Execução: coders (codex & cia). Validação de cada entrega: Kiro. Check-in START/DONE por agente.

## Wave 1 — paralela
- [ ] 1.1 Agent-1: `server/pkg/agent/nim.go` — backend NIM OpenAI-compatible (SSE, loop agêntico, usageMetadata→TokenUsage) + testes
- [ ] 1.2 Agent-2: isolamento/rotação NIM — `execenv/nim_home.go`, `rotation_detector_nim.go`, `rotation/detector_nim.go` + testes
- [ ] 1.3 Agent-3: `server/pkg/agent/cline.go` — backend nativo via `cline --acp --json` + testes
- [ ] 1.4 Agent-4: descoberta de modelos — timeout + cache + surface de erro no fluxo model-list; UI popula
- [ ] 1.5 Agent-5: onboarding — remover `(landing)`/sponsors/`content/use-cases` + fluxo de código por email; login no design-system
- [ ] 1.6 Agent-6: paridade de design (cores kanban/agentes), i18n, build/test do web

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
