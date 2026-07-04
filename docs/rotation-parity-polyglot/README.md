# Rotation-Parity Polyglot — Fase ativa (final, antes do deploy)

Marco atual do Multica PaaS: rotação multi-vendor + otimização de contexto.
Decisão: **prodex AS-IS em PROD agora** → alvo **polyglot** (Go L4 control plane + Rust L2).

## Documentos desta pasta (ordem de leitura)
| # | Arquivo | O que é |
|---|---------|---------|
| 01 | `01_PRD.md` | PRD com o gap, a solução (prodex) e o feedback do R&D |
| 02 | `02_ADR-001-arquitetura.md` | Decisão de arquitetura (prodex as-is→PROD; polyglot alvo; roteador único) |
| 03 | `03_PLATFORM_PLAN_360.md` | Consolidação 360°: itens herdados, ownership, invariantes, waves |

## Onde estão as outras peças (frameworks — ficam no home próprio)
- **OpenSpec change:** `openspec/changes/rotation-parity-polyglot/{proposal,design,tasks}.md`
  - Supersede: `openspec/changes/rotation-router/proposal.md`
- **Plano agêntico (board ativo):** `.deploy-control/MASTER_ROTATION_PARITY_POLYGLOT.md`
- **Prompts dos 8 agentes:** `agentic-prompts-hub/new_prompts/` (ativos) · `agentic-prompts-hub/archive/` (consumidos)

## Histórico do projeto
Planejamento anterior (fase Go, consumido) e execução passada: `docs/99_arquivados/`.

## Estado
Somente planejamento/documentação. **Nenhum agente despachado, nenhum deploy executado.**
Deploy PROD real fica **gated** ao OK do dono após o runbook (F7).
