# Herdr — skill do Orquestrador (Opus 4.8)

> Herdr (herdr.dev, `ogulcancelik/herdr`) é a **plataforma de multiplexação** onde o fleet de agentes roda.
> Ferramenta EXTERNA (não versionada aqui) — instalada por `curl`/`npx`. Este doc é a skill operacional do
> Tech-Lead. Fonte canônica: https://herdr.dev/docs/ · CLI: /docs/cli-reference/ · Socket: /docs/socket-api/
> **Regra: nunca inventar flag/comando Herdr — verificar contra a doc.**

## Papel
**Opus 4.8 = Sr. SME AI Solutions Architect & Principal Agentic Planning Orchestrator (Tech-Lead / POC principal do fleet).**
Roda num pane rotulado/identificado **`opus-4.8-orchestrator`** (`herdr agent rename "$HERDR_PANE_ID" opus-4.8-orchestrator`).

## Instalação
```bash
curl -fsSL https://herdr.dev/install.sh | sh      # linux/macos
npx skills add ogulcancelik/herdr --skill herdr -g # skill de controle nos agentes (global)
```
Guardrail: um agente só opera Herdr se `HERDR_ENV=1` (senão não está num pane gerenciado → parar).

## Modelo de coordenação (bidirecional)
```
 Agente  ── herdr agent send opus-4.8-orchestrator "[<agente>] <msg>"  ──▶  Tech-Lead (Opus 4.8)
         ── herdr notification show "[<agente>] BLOCKED" --sound request ─▶
 Tech-Lead ── events.subscribe (pane.agent_status_changed: blocked|done) ─▶ monitora fleet sem polling
           ── herdr agent list / agent read <name> / agent send / agent start ─▶ dirige o fleet
```
- **ids de pane NÃO são duráveis** (compactam) → usar **nomes de agente** (`agent rename` + `agent send <name>`); reler ids via `pane list`/`agent list` quando necessário.
- Coordenar espera: `herdr agent wait <name> --status done|idle --timeout <ms>`.
- Ler estado/saída: `herdr agent read <name> --source recent --lines N`.

## Integrations por vendor/agente do fleet
| Vendor/Agente | Integration Herdr | Efeito |
|---|---|---|
| Codex (agentes #A–D) | `herdr integration install codex` | session identity + restore; usa `CODEX_HOME` |
| OpenCode (vendor) | `herdr integration install opencode` | lifecycle + session (`opencode --session <id>`) |
| Kiro / Antigravity / Cline | — (não há) | só screen-manifest detection |
| GLM#52 / Gemini (agentes) | — (não listados) | só screen detection (ok; sem restore nativo) |

## ⚠️ Ponto de validação (F6 conformance)
Três coisas passam a tocar `CODEX_HOME`/hooks do Codex: (1) nosso isolamento por conta (`~/.codex-acctN`),
(2) o **prodex** (perfis/hooks Codex), (3) a **integration Herdr codex** (`[features] hooks=true`).
Validar que coexistem sem se pisar antes de habilitar em PROD.

## Skills Herdr de referência (fonte primária)
- Guia (ensinar humano): https://herdr.dev/agent-guide.md
- SKILL.md (controlar Herdr de dentro do pane): https://github.com/ogulcancelik/herdr/blob/master/SKILL.md
- Integrations: https://herdr.dev/docs/integrations/ · CLI: /docs/cli-reference/ · Socket API: /docs/socket-api/
