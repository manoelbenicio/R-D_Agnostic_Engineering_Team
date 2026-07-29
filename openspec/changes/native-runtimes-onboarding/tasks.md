# Tasks — Native Runtimes Onboarding (v6.2 Reconciliation)

## Wave 1 — Implementation
- [x] 1.1 Agent-1: `server/pkg/agent/nim.go` — **SUPERSEDED** per owner ruling (NIM deliberately removed from canonical source; zero `nim.go`/`nim_home.go` at production SHA `15626386da2725af8e8d4ac611754cffe359fe31`)
- [x] 1.2 Agent-2: `execenv/nim_home.go`, `rotation_detector_nim.go` — **SUPERSEDED** per owner ruling (NIM removed from canonical source)
- [x] 1.3 Agent-3: `server/pkg/agent/cline.go` — Backend nativo Cline 3.x via `cline --acp` (ACP JSON-RPC 2.0 por stdin/stdout); `--json` filtrado — COMPLETO no fonte canônico
- [x] 1.4 Agent-4: Descoberta de modelos — timeout + cache + surface de erro no fluxo model-list; UI popula — COMPLETO
- [x] 1.5 Agent-5: Onboarding frontend — remover landing/sponsors + fluxo de código por email; UI login/senha no design-system — **ACEITO ORQ-51**
- [x] 1.6 Agent-6: Paridade de design (tokens/cores kanban/agentes), limpeza de i18n, harness build/test web — **ACEITO ORQ-52**
- [x] 1.7 Agent-1 (Backend Auth): `POST /auth/login` (username/senha) em `cmd/server/router.go` + credential store atrás de `AuthProvider` — COMPLETO

## Wave 2 — Integração
- [x] 2.1 Wiring `config.go`: probe `cline` — COMPLETO (`MULTICA_CLINE_PATH`/`cline`)
- [x] 2.2 Wiring `agent.go`: `New()` cases + `SupportedTypes` (cline) — COMPLETO
- [x] 2.3 `requiresCredentialIsolation` (+`nim`) — **SUPERSEDED** (NIM removido do fonte canônico)
- [ ] 2.4 Rebuild & restart daemon com runtime `cline` ativo — EM ANDAMENTO (depende do deploy do daemon durável ORQ-66; ORQ2 daemon segue binário `88ca`)
- [x] 2.5 Build + subir web local; validar onboarding novo — **ACEITO ORQ-51 / ORQ-52**

## Wave 3 — Verificação & Live Canary
- [x] 3.1 Testes verdes Go + web em container — COMPLETO (`openspec validate --all --strict` 5/5 pass)
- [ ] 3.2 Smoke / Live Canary Cline: criar agente em `cline`, rodar task, ver execução + tokens — PENDENTE deploy do daemon durável ORQ-66 (live CLI é `cline 3.0.46`)
- [x] 3.3 UAT onboarding (sem sponsors/email-code; cores idênticas) — **ACEITO ORQ-51 / ORQ-52**
- [x] 3.4 Check-ins DONE + relatório de integração em `.deploy-control/` — COMPLETO

## Decisão do Dono (2026-07-12 / 2026-07-29)
- [x] 0.1 Auth do onboarding: **login/senha simples** agora; **Firebase** em fase posterior (sem rework). Task 1.5 aceita via ORQ-51.
- [x] 0.2 Remoção do NIM: **NVIDIA NIM deliberadamente removido** do fonte canônico (zero `nim.go`/`nim_home.go` em `15626386da2725af8e8d4ac611754cffe359fe31`). Obsoleto / SUPERSEDED.
