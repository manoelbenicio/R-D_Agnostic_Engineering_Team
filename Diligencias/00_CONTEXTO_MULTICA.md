# CONTEXTO DO PRODUTO — Multica (leia PRIMEIRO)

> Onboarding self-contained: o que é o Multica, onde está o código, e como o projeto
> Rotation-Parity Polyglot se encaixa. Fiel ao README/AGENTS.md/CLAUDE.md do repo.

## 1. O que é o Multica
**Plataforma open-source de "managed agents"** — *"Your next 10 hires won't be human."*
Transforma coding agents em **colegas de time**: você atribui uma issue a um agente como faria a um
humano; ele pega o trabalho, escreve código, reporta bloqueios e atualiza status **autonomamente**.
Board + conversas + skills que compõem ao longo do tempo. **Vendor-neutral, self-hosted.**

- Funciona com: **Codex, Claude Code, GitHub Copilot CLI, OpenClaw, OpenCode, Hermes, Gemini, Pi, Cursor Agent, Kimi, Kiro CLI, Qoder CLI.**
- **Squads:** camada de roteamento — atribui a um grupo liderado por um agente, que delega ao membro certo.
- Site: multica.ai · Repo: `github.com/multica-ai/multica`.

## 2. Arquitetura do repo (monorepo)
Go backend + frontend monorepo (pnpm workspaces + Turborepo).
```
server/            Go backend (Chi router, sqlc, gorilla/websocket)  — módulo github.com/multica-ai/multica/server, Go 1.26.1
apps/web/          Next.js (App Router)
apps/desktop/      Electron
packages/core/     lógica de negócio headless (Zustand, React Query, API client)
packages/ui/       componentes atômicos (shadcn/Base UI)
packages/views/    páginas/componentes compartilhados
```
- **Estado:** React Query = server state; Postgres (`pgvector/pgvector:pg17`) + Redis.
- **Fonte de verdade de arquitetura/regras:** `CLAUDE.md` (raiz) — ler primeiro; `AGENTS.md` é ponteiro; `Makefile`/`package.json`/`pnpm-workspace.yaml` = comandos.

## 3. Pacotes Go relevantes (server/internal — do inventário)
| Pacote | Papel |
|--------|------|
| `handler` (143 arq.) | API HTTP (Chi) |
| `daemon` (49) | **lança e gerencia os agentes** (lifecycle, dispatch) — HOTSPOT |
| `daemon/execenv` (34) | **isolamento por conta/vendor**: CODEX_HOME por tarefa, HOME isolado (Antigravity/Kiro), copia auth.json por conta |
| `rotation` (32) | rotação de contas (caminho frio, Go) |
| `l2runtime` + `daemon/prodex.go` | **cliente/launcher do prodex** (o L2 Rust) |
| `auth` (13) | credenciais/tokens |
| `metrics`,`middleware`,`realtime`,`scheduler`,`storage` | infra |

## 4. Onde ESTE projeto (Rotation-Parity Polyglot) se encaixa
O Multica hoje lança CLIs de vendor (ex.: `codex` cru). **Este projeto faz o Multica lançar o `prodex`
(runtime Rust L2) no lugar** — para ter caminho quente: rotação pré-commit, afinidade de sessão,
**Smart Context/token-saver** (mandatório) e reset-claim. Divisão **polyglot**:
- **L4 Multica (Go)** = control plane frio: cadastro, policy, approved-accounts, kill-switch, observability. **NÃO roteia request em voo.**
- **L2 prodex (Rust)** = runtime quente: decide o request em voo (afinidade, fallback, Smart Context).
- Contrato local `rpp.l2.v1` entre os dois. Ver `ADR-001` + OpenSpec `rotation-parity-polyglot`.

## 5. Como buildar/rodar (via container, IPv6 OFF)
- Go: `golang:1.26-alpine` + `multica-gomod` cache (ver 00b_DEPENDENCY_SOURCES.md).
- prodex: `rust:1.85-bookworm` (edition 2024) + `prodex-cargo` cache.
- Datastores: docker-compose (`docker-compose.selfhost.yml`) — Postgres+Redis já up.

## 6. Invariantes (rotação) — inegociáveis
Roteador único por sessão · hard affinity (`previous_response_id`/turn/session) · rotate-before-commit ·
troca de perfil fail-closed · sem SQLite compartilhado · **sem segredo em log** · verde-em-container antes de DONE.

## 7. Glossário rápido
- **prodex** = runtime Rust (L2) open-source (`github.com/christiandoxa/prodex`) que faz o hot-path; usado AS-IS agora, fork depois.
- **Smart Context** = token-saver via prodex (reescreve contexto preservando campos de controle; fallback exato).
- **reset-claim** = recuperar crédito via `prodex redeem` (baixa prioridade, por último).
- **MCP (prodex)** = o prodex fala MCP: crate `prodex-mcp-stdio` (framing stdio p/ MCP servers) + tradução/passthrough de tool-calls MCP no runtime (anthropic/gemini/deepseek). Faz parte do hot-path → coberto no contrato/fork-map/QA.
- **Herdr** = multiplexer que roda os agentes-worker em panes (harness de execução do time).
