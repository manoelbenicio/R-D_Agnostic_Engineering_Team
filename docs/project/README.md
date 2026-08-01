# Documentação do Projeto — Isolamento de Credencial OAuth por Conta

> **Authority boundary — reconciled 2026-08-01**
>
> - **CURRENT_OBSERVED:** ORQ2 has eight intentional physical credential homes mapped one-to-one to eight legacy-registry terminals and live panes; no duplicate or unreferenced home was observed.
> - **SOURCE_VALIDATED:** allocator-v2 and Runtime Manager/Postgres contracts exist in the reconciled candidate source, but allocator-v2 is not installed and Runtime Manager/Postgres account control is not deployed state. The installed registry remains legacy v1.
> - **PLANNED:** automatic account rotation, Runtime Manager activation, Postgres-backed account control, and higher capacity profiles are TO-BE only.
> - **CANONICAL AUTHORITY:** `credential-account-home-restoration` REQ-01..23 plus compatible qualified REQ-24..35 control. Discovery is dynamic and opaque; static slot-number grammars and allowlists are forbidden.
> - **APPROVAL_GATED:** commit/push identity, production DB access, deployment, restart, ORQ1 cutover/rollout, allocator-v2 installation, and capacity expansion require separate evidence and fresh owner gates; none is authorized or performed by this candidate.
> - **CHRONOLOGY:** any single-global-home or single-slot wording below describes historical behavior or a conditional migration starting point, not the current eight-home observed state.


Mudança **cirúrgica** no mecanismo de autenticação do Multica: separar credenciais
OAuth por conta para eliminar sobreposição entre múltiplas contas do mesmo vendor,
e (Fase 2) automatizar a troca ao esgotar a janela de ~5h. O produto permanece
íntegro; desenvolvimento na cópia local `multica-auth-work/`.

## Índice

| Doc | Conteúdo |
|-----|----------|
| [00-overview-why.md](00-overview-why.md) | Visão geral, problema, objetivo, princípios, fases (o **porquê**) |
| [01-as-is.md](01-as-is.md) | Estado atual: como a auth funciona hoje e as lacunas |
| [02-to-be.md](02-to-be.md) | Estado alvo: mecanismo por vendor, fluxos Fase 1 e 2, aceite |
| [03-requirements.md](03-requirements.md) | Requisitos funcionais e não-funcionais + rastreabilidade |
| [04-architecture.md](04-architecture.md) | Componentes, diagramas, contrato de env, modelo de dados |
| [05-observability.md](05-observability.md) | Grafana/Prometheus easy-deploy; cobertura de todo componente |

## Arquitetura visual (HTML interativo, estilo command-center)

| Arquivo | Conteúdo |
|---------|----------|
| [architecture_as_is.html](architecture_as_is.html) | AS-IS visual: credencial global compartilhada e o ponto de sobreposição |
| [architecture_to_be.html](architecture_to_be.html) | TO-BE visual: contas isoladas + injeção por vendor + rotação (Fase 2) |

> Diagramas SVG animados, nós clicáveis com notas, tema escuro — mesmo estilo dos
> `architecture_macro/deep/micro.html`. Abrir no navegador.

## Artefatos relacionados (OpenSpec)

Autoridade canônica atual: `openspec/changes/credential-account-home-restoration/`:
- `proposal.md`, `design.md`, `tasks.md`, `evidence.md` e `specs/credential-account-home/spec.md` — REQ-01..23 aceitos mais REQ-24..35 qualificados.

Referência histórica somente: `openspec/changes/archive/2026-07-22-agent-credential-isolation/`. Seus inventários e planos não substituem a autoridade canônica nem comprovam estado instalado/deployed.

## Resumo executivo

- **Vendors:** Codex (`CODEX_HOME`), Kiro (`XDG_DATA_HOME`/`KIRO_API_KEY`),
  Antigravity (`HOME`). Sem Claude direto.
- **Persistência:** Postgres-only.
- **Segurança:** store/restore AS-IS, nenhum segredo em log/label.
- **Verificação:** build em container `golang:1.26-alpine`; suítes tocadas verdes.
- **Observabilidade:** todo componente com `/metrics` + dashboards + alertas.
