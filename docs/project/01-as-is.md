# 01 — Estado Atual (AS-IS)

**Data:** 2026-07-01 · Verificado no servidor de produção (192.168.15.6) + código.

---

## 1. Como a autenticação de agente funciona hoje

O Multica roda agentes CLI (Codex, Kiro, Antigravity) em panes. Cada tarefa recebe
um ambiente preparado pelo `execenv`. Para o Codex, é criado um `CODEX_HOME` por
tarefa, mas o **`auth.json` é symlinkado para um único `~/.codex` global**.

### Trecho crítico (fonte de verdade)
- `server/internal/daemon/execenv/codex_home.go`:
  - `codexSymlinkedFiles = ["auth.json"]` → **credencial global compartilhada**.
  - `codexCopiedFiles` = config.toml/config.json/instructions.md (isolados por cópia).
  - Comentário no código: *"Symlinks share state (e.g. auth tokens) so changes
    propagate automatically."* → é **design deliberado** de compartilhar UMA conta.
- `server/internal/daemon/daemon.go` (~l.3380): monta `agentEnv` por tarefa e injeta
  `CODEX_HOME`, `CURSOR_DATA_DIR`, `OPENCLAW_CONFIG_PATH`. **É o ponto de injeção**.
- `codex_user_skills.go`: *"Codex is the only runtime whose HOME is redirected to a
  per-task directory"* → Kiro/Antigravity **não** recebem home isolado hoje.

## 2. Anatomia real de credencial por vendor (no servidor)

| Vendor | Store real | Observação |
|--------|-----------|------------|
| **Codex** | `~/.codex/auth.json` (0600) | arquivo plano; chaves: access_token, id_token, refresh_token, account_id, last_refresh. Portável entre máquinas. |
| **Kiro** (fork Amazon Q) | `~/.local/share/kiro-cli/data.sqlite3` (tabela `auth_kv`) | NÃO usa `~/.kiro`. Também aceita `KIRO_API_KEY` (headless). |
| **Antigravity (agy)** | `~/.gemini/antigravity-cli/antigravity-oauth-token` | sem subcomando de login/flag de config-dir; isola via `HOME`. |

> Na máquina de produção existe **apenas 1 diretório por vendor** — nenhuma
> estrutura multi-conta ainda. Confirma a causa-raiz da sobreposição.

## 3. Sintoma observável (o que o operador vive)

1. Loga conta A do Codex → funciona.
2. Loga conta B do Codex → `~/.codex/auth.json` é sobrescrito; conta A cai.
3. Agente esgota a janela de ~5h → para na tela pedindo re-login manual.
4. Não há troca automática; capacidade ociosa de outras contas é desperdiçada.

## 4. Persistência hoje

- Multica: Postgres (server Go) para o produto; SQLite legado permanece em
  HerdMaster (`herdmaster.db`) e no store nativo do Kiro (`data.sqlite3`).
- Histórico relatado: SQLite deu `database is locked` sob concorrência → migração
  para Postgres com pool de conexões (decisão firme: **Postgres-only** daqui pra frente).

## 5. Sistema de credenciais existente (referência)

Já existe um app de auth de cliente (padrão `auth_routes.py`) que **armazena o estado
de sessão OAuth / config-dir por conta e serve de volta AS-IS**. É o mesmo conceito
que vamos reaproveitar — mesmos agentes, mesmos vendors, mesmo protocolo OAuth.

## 6. Lacunas do AS-IS (o que falta)

- G1: sem isolamento de credencial por conta (uma conta por vendor de cada vez).
- G2: sem seleção/atribuição de conta → agente.
- G3: sem detecção de esgotamento nem troca automática (Fase 2).
- G4: sem observabilidade dedicada de credencial/cota/rotação.
