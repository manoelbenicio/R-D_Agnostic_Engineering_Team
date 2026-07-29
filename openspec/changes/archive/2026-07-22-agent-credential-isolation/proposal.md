# Proposal: Isolamento de credencial por conta de agente

## Why

O operador mantém **múltiplas contas** por provedor (ex.: várias Codex, várias
Anthropic, várias Google). Hoje, na versão "pura" do Multica que roda nos
containers, a credencial de cada CLI vive num **home único global** por provedor
(ex.: `~/.codex/auth.json`). Logar numa segunda conta **sobrescreve** a primeira,
e ao esgotar créditos o operador precisa deslogar/relogar manualmente.

Verificação no servidor original (SSH, 2026-07-01): o HerdMaster/Herd **não**
tem isolamento de credencial multi-conta, e a máquina que roda os agentes tem
apenas **um** diretório por provedor (`~/.codex`, `~/.claude`, `~/.gemini`,
`~/.kiro`). Ou seja, a peça de isolamento multi-conta **ainda não existe**.

O que existe e é reaproveitável é o **contrato de providers** já definido em
`infra/cao/auth_routes.py` (dict `PROVIDERS`): para cada CLI, o diretório de
config e a **env var nativa** que o aponta:
- Codex → `CODEX_HOME` / `CODEX_CONFIG_DIR`
- Claude Code → `CLAUDE_CONFIG_DIR`
- Gemini CLI → `CLOUDSDK_CONFIG` / `GEMINI_CONFIG_DIR`
- Kiro CLI → `KIRO_HOME` / `KIRO_CONFIG_DIR`

A implementação em si — apontar cada agente para o diretório da conta atribuída
no momento do spawn — é **código novo**, guiado por esse contrato. A solução é a
que o operador definiu: **cada credencial numa pasta diferente**.

## What Changes

Replicar o modelo existente (fonte de verdade: `infra/cao/auth_routes.py`,
`src/api/session-discovery.ts`, `src/api/session-store.ts`,
`docs/session-management.md`):

- **Config dir por conta** com as env vars nativas de cada provedor, exatamente
  como o `PROVIDERS` de `auth_routes.py` já define:
  - Codex → `CODEX_HOME` / `CODEX_CONFIG_DIR`
  - Claude Code → `CLAUDE_CONFIG_DIR`
  - Gemini CLI → `CLOUDSDK_CONFIG` / `GEMINI_CONFIG_DIR`
  - Kiro CLI → `KIRO_HOME` / `KIRO_CONFIG_DIR`
- **Discovery / login / revoke** de sessões por conta, no mesmo contrato de API
  já existente (`GET /auth/sessions`, `POST /auth/login`,
  `DELETE /auth/sessions/:id`).
- **Atribuição sessão → agente/nó** (o modelo atual: o usuário escolhe a sessão
  por nó; `session_id` viaja até a env injetada no terminal via
  `resolveSessionEnv`).
- **Isolamento de env por terminal**: cada processo recebe só as env vars da sua
  conta — sem estado de credencial compartilhado.
- Aplicar aos mesmos agentes em uso: **codex, claude, gemini/agy, kiro** (e glm
  quando aplicável).

### Fase 2 incluída agora (por decisão do dono)

- **Rotação automática ao esgotar crédito/token**: detectar sessão
  `expired`/esgotada e reatribuir o agente para a próxima conta disponível do
  mesmo provedor, sem intervenção manual. Aproveita o `status`/`expires_at` que o
  discovery já expõe e o monitor de expiração já existente
  (`useSessionMonitor`/`isExpiringSoon`).

## Impact

- Affected specs: `agent-credential-isolation` (nova capability).
- Affected code (paridade com o modelo existente):
  - Backend de sessão: `infra/cao/auth_routes.py` (contrato `PROVIDERS`).
  - Frontend de sessão: `src/api/session-discovery.ts`, `src/api/session-store.ts`,
    `src/canvas-reconciler/reconciler.ts` (`resolveSessionEnv`).
  - Runtime dos agentes na versão pura (execenv), que deve montar o config dir
    por conta em vez do home global.
- Segurança: menor blast radius (uma conta comprometida não vaza para outras);
  segredos nunca no frontend; logs redigidos (`sanitizeForLog`).
