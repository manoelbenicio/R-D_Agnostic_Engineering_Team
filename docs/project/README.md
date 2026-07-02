# Documentação do Projeto — Isolamento de Credencial OAuth por Conta

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

`openspec/changes/agent-credential-isolation/`:
- `proposal.md`, `tasks.md`, `specs/.../spec.md` — a mudança formal.
- `design.md` — escopo travado, gaps/riscos, decisões, achados de pesquisa.
- `auth-inventory.md` — as três camadas de auth mapeadas nos 4 projetos.

## Resumo executivo

- **Vendors:** Codex (`CODEX_HOME`), Kiro (`XDG_DATA_HOME`/`KIRO_API_KEY`),
  Antigravity (`HOME`). Sem Claude direto.
- **Persistência:** Postgres-only.
- **Segurança:** store/restore AS-IS, nenhum segredo em log/label.
- **Verificação:** build em container `golang:1.26-alpine`; suítes tocadas verdes.
- **Observabilidade:** todo componente com `/metrics` + dashboards + alertas.
