# Design — Agentic Execution Plan

## Roles
- **Kiro (orchestrator)** — NÃO produz código. Coordena waves, sequencia edições de arquivos
  compartilhados, revisa check-ins, roda build/test de integração, valida cada entrega, dá DONE.
- **6 coder agents (codex & cia)** — produzem o código nas suas trilhas.

## Check-in protocol (obrigatório)
Cada agente escreve em `.deploy-control/`:
- ANTES: `CHECKIN_<agentname>_<UTC-ISO8601>_START.md` (escopo, arquivos, deps, riscos).
- DEPOIS: `CHECKIN_<agentname>_<UTC-ISO8601>_DONE.md` (o que fez, arquivos, evidência de build/test).
- Verde-em-container antes de DONE. Sem segredo em log. Commits atômicos.

## Ownership de arquivos (anti-colisão)
- Backends em arquivos próprios (`nim.go`, `cline.go`) → sem conflito.
- Arquivos COMPARTILHADOS (`internal/daemon/config.go` probe, `pkg/agent/agent.go` factory+SupportedTypes,
  `requiresCredentialIsolation`) → **somente Kiro edita** na Wave 2, juntando NIM+Cline num passo.

## Waves

### Wave 1 (6 agentes paralelos, arquivos disjuntos)
```
Agent-1 NIM-Core      -> server/pkg/agent/nim.go (+test): OpenAI-compat, SSE, loop agêntico, usageMetadata
Agent-2 NIM-Isolation -> execenv/nim_home.go, rotation_detector_nim.go, rotation/detector_nim.go (+tests)
Agent-3 Cline-Core    -> server/pkg/agent/cline.go (+test) via `cline --acp` (ACP JSON-RPC por stdin/stdout)
Agent-4 Discovery-Fix -> internal/daemon (model-list flow) + models.go discovery: timeout/cache/erro
Agent-5 Frontend-Auth -> apps/web/app/(auth), packages/views/auth; remover (landing)/sponsors/use-cases/email-code
Agent-6 Frontend-QA   -> design-system (paridade de cores kanban/agentes), i18n, build/test web
```

### Wave 2 (integração — Kiro, sequencial)
- Aplicar wiring: `config.go` probes (`MULTICA_NIM_PATH`/`nim`, `MULTICA_CLINE_PATH`/`cline`);
  `agent.go` `New()` cases + `SupportedTypes`; `requiresCredentialIsolation` (+`nim`).
- Rebuild `server/bin/multica` + imagem backend; restart daemon; runtimes `nim` e `cline` aparecem.
- Integrar frontend: build web local, validar login novo sem sponsors/email-code.

### Wave 3 (verificação/gates — Kiro valida)
- Testes verdes (Go + web) em container. Smoke: criar agente em `nim` e `cline`, rodar 1 task, ver execução + tokens.
- UAT do onboarding. Check-ins DONE + relatório de integração em `.deploy-control/`.

## Paralelismo
- Wave 1 = 6 agentes 100% paralelos. Frontend (5/6) independe do Go → integra em paralelo.
- Wave 2 depende de Wave 1. Wave 3 valida tudo.

## Decisões / riscos
- **Auth do onboarding**: decisão do dono PENDENTE (login/senha vs sem-fricção) — bloqueia Agent-5.
- **NIM auth**: validar credencial/fluxo do gateway antes de codar o loop; documentar fonte.
- **Compatibilidade Cline 3.x**: usar somente `cline --acp` para o transporte ACP. Apesar de
  as mensagens ACP serem JSON-RPC 2.0, o flag CLI `--json` seleciona outro modo headless e
  encerra antes do handshake quando combinado com `--acp`. O teste direto do backend fornece
  `--json` como argumento customizado hostil e confirma que o argv final o remove, preservando
  `--acp` e a sessão ACP por stdin/stdout.
- `agy models` lento → item 3 precisa timeout+cache.
- Shared files → só Kiro edita (Wave 2).
