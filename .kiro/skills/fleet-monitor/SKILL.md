---
name: fleet-monitor
description: "Monitorar em tempo real o fleet de agentes do rollout Rotation-Parity Polyglot a partir DESTE host, via Herdr-over-SSH (lê o `herdr agent list` do socket remoto + o board .deploy-control), e falar SOMENTE com o orquestrador (opus-4.8-orchestrator). Use quando o usuário pedir status do fleet, ver o que cada agente está fazendo, faróis/ETA/OBS, bloqueios, ou mandar mensagem/status ao orquestrador."
license: MIT
metadata:
  author: Opus 4.8 (Principal Agentic Planning Orchestrator)
  version: "1.0"
---

# fleet-monitor — visão 360° do fleet via Herdr-over-SSH

## O que é
Monitor de execução em tempo real do fleet de agentes (rollout Rotation-Parity Polyglot).
Roda **neste host** e observa o fleet no host remoto (`manoelneto-laptop`) **via SSH**, sem git no runtime.
Fonte da verdade = **Herdr socket** (`herdr agent list --json`) enriquecido com os check-ins do board
(`.deploy-control/<AGENTE>__<STREAM>__<UTC>.md`). Também expõe um canal para falar **apenas** com o
orquestrador `opus-4.8-orchestrator`.

Script: `scripts/dashboard/fleet_dashboard.py` (Python stdlib, sem dependências).

## Quando usar
- "status do fleet", "o que os agentes estão fazendo", "algum travado?", "farol/ETA/OBS".
- "pergunta o status pro orquestrador", "manda mensagem pro orquestrador".

## Pré-requisitos
- SSH configurado para o host do fleet (BatchMode; ex.: entry `manoelneto-laptop` no `~/.ssh/config`).
- No host remoto: `herdr` no PATH e servidor Herdr ativo (socket em `~/.config/herdr/herdr.sock`).

## Comandos — monitor
```bash
python3 scripts/dashboard/fleet_dashboard.py            # ao vivo (atualiza no lugar; Ctrl+C sai)
python3 scripts/dashboard/fleet_dashboard.py --once      # snapshot único
python3 scripts/dashboard/fleet_dashboard.py --json       # machine-readable
python3 scripts/dashboard/fleet_dashboard.py --interval 3 # refresh mais rápido
python3 scripts/dashboard/fleet_dashboard.py --ascii      # sem cor/emoji
python3 scripts/dashboard/fleet_dashboard.py --ssh <host> --board <path>  # overrides
```

## Comandos — canal Tech-Lead → SOMENTE o orquestrador
Alvo travado em `opus-4.8-orchestrator` (nunca fala direto com os outros agentes):
```bash
python3 scripts/dashboard/fleet_dashboard.py --msg "status geral do fleet?"   # envia mensagem
python3 scripts/dashboard/fleet_dashboard.py --status                          # pede status + lê a resposta do pane
python3 scripts/dashboard/fleet_dashboard.py --read                            # lê o pane do orquestrador
```
Por baixo: `ssh <host> herdr agent send opus-4.8-orchestrator "<texto>"` e `herdr agent read opus-4.8-orchestrator`.

## Configuração (env, opcional)
- `FLEET_SSH_HOST` (default `manoelneto-laptop`)
- `FLEET_BOARD` (default `/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/.deploy-control`)
- `FLEET_ORCHESTRATOR` (default `opus-4.8-orchestrator`)
- `RPP_FORCE_COLOR=1` força cor quando não é TTY; `NO_COLOR` desliga.

## Semântica do farol
- 🟢 `working` / `done` — progredindo / concluído.
- 🟡 `idle` — parado/entre tasks (OBS justifica).
- 🔴 `blocked` — bloqueado (OBS = `notes`/`build_result` do check-in) ou build vermelho.
- ⚪ `unknown` — sem detecção nativa (screen-detection; ex.: Cline).
Colunas: AGENTE · TIPO · STREAM/TASK · FAROL · TEMPO (decorrido/duração) · OBS. Ordena blocked→working→idle→unknown.

## Regras de segurança
- **Falar apenas com `opus-4.8-orchestrator`.** Nunca enviar comando direto a outro agente do fleet.
- Monitor é **read-only** (só lê o Herdr/board); a única escrita é a mensagem ao orquestrador.
- Nada de segredo em log/saída. Não inventar flag do Herdr — referência: https://herdr.dev/docs/cli-reference/ e /socket-api/.

## Troubleshooting
- `ssh timeout`/erro de conexão → conferir `~/.ssh/config` e alcance de rede (Tailscale/LAN) ao host.
- Dashboard sem agentes → `ssh <host> herdr agent list` manualmente; confirmar servidor Herdr ativo no host.
- STREAM/TASK vazio para um agente → ainda não há check-in dele no board (ou nome não casou; o farol vem do Herdr de qualquer forma).
