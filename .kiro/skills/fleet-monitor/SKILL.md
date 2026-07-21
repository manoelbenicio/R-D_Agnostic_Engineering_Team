---
name: fleet-monitor
description: "Monitorar em tempo real o fleet de agentes do rollout Agent Brain/OmniRoute no ORQ2, lendo o Herdr local + board .deploy-control, e falar SOMENTE com o orquestrador (opus-4.8-orchestrator)."
license: MIT
metadata:
  author: Opus 4.8 (Principal Agentic Planning Orchestrator)
  version: "1.1"
---

# fleet-monitor — visão 360° do fleet migrado

## Topologia autoritativa (2026-07-21)

| Nó | Instance | Tailscale | Responsabilidade |
|---|---|---|---|
| ORQ1 | `i-0d9d441dd364039f9` | `100.118.244.61` | OmniRoute + stack DEV (Postgres/backend/frontend) |
| ORQ2 | `i-0af937456e125143d` | `100.110.178.47` | Herdr + agentes + board; este host |

O endpoint LAN legado foi aposentado e é bloqueado pelo dashboard. Não o consultar, nem como fallback.

## Fonte da verdade
- Estado vivo: `herdr agent list` **local no ORQ2**.
- Tarefas/evidências: `.deploy-control` local e `openspec/changes/build-omniroute-agent-brain/tasks.md`.
- Estado consolidado: pane local do `opus-4.8-orchestrator` (atualmente configurável por `FLEET_ORCHESTRATOR_PANE`, default `w5:p1`).
- Script: `scripts/dashboard/fleet_dashboard.py` (Python stdlib).

## Monitor
```bash
python3 scripts/dashboard/fleet_dashboard.py --once
python3 scripts/dashboard/fleet_dashboard.py --json
python3 scripts/dashboard/fleet_dashboard.py --report
python3 scripts/dashboard/fleet_dashboard.py --interval 3
```
O default é `--ssh local`. O valor ORQ2 (`100.110.178.47`) também é reconhecido como local, evitando SSH para o próprio nó. Use ORQ1 apenas para verificações read-only da stack DEV quando a tarefa exigir; ORQ1 não é o host do Herdr.

## Canal Tech-Lead → SOMENTE o orquestrador
```bash
python3 scripts/dashboard/fleet_dashboard.py --msg "status geral do fleet?"
python3 scripts/dashboard/fleet_dashboard.py --status
python3 scripts/dashboard/fleet_dashboard.py --read
```
Por baixo, o script usa `herdr pane run`/`herdr pane read` localmente no pane configurado do orquestrador. Nunca envie mensagens diretamente às lanes.

## Configuração opcional
- `FLEET_SSH_HOST` (default `local`; valores legados são neutralizados).
- `FLEET_BOARD` (default `.deploy-control` deste checkout).
- `FLEET_ORCHESTRATOR` (default `opus-4.8-orchestrator`).
- `FLEET_ORCHESTRATOR_PANE` (default atual `w5:p1`; reconfirmar após recriação do workspace).
- `RPP_FORCE_COLOR=1` ou `NO_COLOR=1`.

## Semântica e segurança
- `working`/`done`: progresso/conclusão; `idle`: entre tarefas; `blocked`: bloqueio; `unknown`: sem detecção.
- Monitor é read-only; a única escrita permitida é mensagem ao `opus-4.8-orchestrator`.
- Não registrar segredos, bodies, IDs brutos ou credenciais.
- Timeout/reachability de ORQ1 não autoriza fallback para infraestrutura aposentada.
