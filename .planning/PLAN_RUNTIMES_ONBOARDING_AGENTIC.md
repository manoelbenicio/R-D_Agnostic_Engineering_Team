# PLANO AGÊNTICO — Runtimes nativos (NIM/Cline) + Descoberta de modelos + Rework de Onboarding

> **Status:** PLANO (nenhuma execução iniciada). Autor: Kiro (default). Data: 2026-07-12.
> **Regra:** o orquestrador **Kiro NÃO produz código** — só coordena, sequencia, revisa check-ins, resolve conflito de arquivo compartilhado e roda a integração/gates.
> **Objetivo:** máximo paralelismo, ownership de arquivos sem colisão, evidência em disco.

## Orquestrador
- **Kiro (orchestrator)** — não escreve código. Responsável por: distribuir waves, sequenciar edições de arquivos compartilhados (`config.go`, `agent.go`), revisar check-ins, rodar build/test de integração em container, resolver conflitos, dar o DONE final.

## Protocolo de check-in (obrigatório para TODOS os agentes)
Cada agente escreve em disco, em `.deploy-control/`:
- **ANTES de começar:** `CHECKIN_<AgentName>_<UTC-ISO8601>_START.md` — escopo, arquivos que vai tocar, dependências, riscos.
- **DEPOIS de terminar:** `CHECKIN_<AgentName>_<UTC-ISO8601>_DONE.md` — o que fez, arquivos alterados, evidência (saída de build/test), bloqueios.
- Formato do nome: `<agentname>+<timestamp>`. Verde-em-container antes de DONE. Sem segredo em log. Commits atômicos.

## Itens de trabalho
1. **NVIDIA NIM — runtime NATIVO do zero** (não via opencode). Backend OpenAI-compatible próprio contra `https://integrate.api.nvidia.com/v1`: SSE streaming, loop agêntico (tool-calling, edição de arquivo), `usageMetadata`→TokenUsage, isolamento de credencial + detecção de rotação, catálogo de modelos + níveis de effort.
2. **CLINE — backend NATIVO do zero.** `pkg/agent/cline.go` dirigindo `cline --acp --json` (reusar máquina ACP de kiro/kimi); probe + factory + SupportedTypes. (`cline_home.go` e `rotation_detector_cline.go` já existem.)
3. **Descoberta de modelos ("nada do CLI").** Fluxo assíncrono de model-list trava/lento (`agy models` ~20s, sem timeout/cache adequado, UI vazia). Corrigir timeout, cache, surface de erro e garantir que a UI popula.
4. **Rework de Onboarding/Frontend.** Remover a landing de marketing + patrocinadores e o fluxo de **código por email**. Login limpo, **mesmas cores/design do kanban e do menu Agentes** (design-system atual), sem "baianada". *(Decisão do dono pendente: modelo de auth final — login/senha simples vs. sem-fricção. NÃO implementar até o dono confirmar.)*

## Ownership de arquivos (evitar colisão)
- Backends em arquivos próprios (`nim.go`, `cline.go`) → sem conflito entre si.
- **Arquivos compartilhados** (`internal/daemon/config.go` probe, `pkg/agent/agent.go` factory+SupportedTypes, `requiresCredentialIsolation`): **somente o Integrador (Kiro) edita**, juntando as necessidades de NIM+Cline num único passo sequenciado (Wave 2). Agentes entregam um "patch de wiring" descrito no check-in; Kiro aplica.

## Waves (paralelismo)

### Wave 1 — paralela (arquivos disjuntos)
| Agente | Item | Arquivos (próprios) |
|---|---|---|
| **Agent-1 (NIM-Core)** | 1 | `server/pkg/agent/nim.go` (+`nim_test.go`): HTTP/SSE, loop agêntico, usageMetadata |
| **Agent-2 (NIM-Isolation)** | 1 | `server/internal/daemon/execenv/nim_home.go`, `server/internal/daemon/rotation_detector_nim.go`, `server/internal/rotation/detector_nim.go` (+tests) |
| **Agent-3 (Cline-Core)** | 2 | `server/pkg/agent/cline.go` (+`cline_test.go`) via `--acp` |
| **Agent-4 (Discovery-Fix)** | 3 | `server/internal/daemon/*model*`, `server/pkg/agent/models.go` (discovery/cache/timeout) — coordenar com Kiro se tocar models.go |
| **Agent-5 (Frontend-Auth)** | 4 | `apps/web/app/(auth)/**`, `packages/views/auth/**`, remover `app/(landing)`, `features/landing`, `content/use-cases`, sponsors |
| **Agent-6 (Frontend-Design/QA)** | 4 | design-system/tokens de cor (paridade kanban/agentes), i18n cleanup, `web` build/test |

### Wave 2 — integração (sequencial, Kiro)
- Kiro aplica os patches de wiring: `config.go` (probes `MULTICA_NIM_PATH`/`nim`, `MULTICA_CLINE_PATH`/`cline`), `agent.go` (`New()` cases + `SupportedTypes` para `nim` e `cline`), `requiresCredentialIsolation` (+`nim`).
- Rebuild `server/bin/multica` + imagem backend; restart daemon; verificar runtimes **nim** e **cline** aparecem e listam modelos.
- Integrar frontend (Agent-5+6): build web, subir imagem local, validar login novo + ausência de sponsors/email-code.

### Wave 3 — verificação/gates (Kiro)
- Testes verdes em container (Go + web). Smoke: criar agente em cada runtime novo (nim, cline), rodar 1 task, ver execução + tokens. UAT do onboarding.
- Check-ins DONE de todos + relatório de integração em `.deploy-control/`.

## Dependências e paralelismo
- Wave 1: 6 agentes **100% paralelos** (arquivos disjuntos; shared files reservados p/ Wave 2).
- Wave 2 depende de Wave 1 (NIM-Core+Isolation, Cline-Core, Frontend).
- Frontend (Agent-5/6) é independente do Go → pode concluir e integrar em paralelo ao backend.

## Riscos
- API interna do NIM (auth consumer) — validar credencial/fluxo antes de codar o loop; documentar fonte.
- `config.go`/`agent.go` compartilhados → risco de merge; mitigado por Kiro ser o único editor (Wave 2).
- `agy models` lento → item 3 deve dar timeout/cache p/ não travar a UI.
- Auth do onboarding: **decisão do dono pendente** — não codar antes de confirmar.

## Entregáveis
- Runtimes **nim** e **cline** nativos, aparecendo na UI com modelos + effort.
- Descoberta de modelos confiável (UI popula, sem travar).
- Onboarding/login novo no design do app, sem sponsors/email-code.
- Check-ins START/DONE de cada agente + relatório de integração, todos em `.deploy-control/`.
