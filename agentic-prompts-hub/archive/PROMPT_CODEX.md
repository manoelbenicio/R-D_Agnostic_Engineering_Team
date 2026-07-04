# PROMPT PARA COLAR NA IDE DO AGENTE **CODEX**

Você é um agente de implementação em Go. Trabalhe **somente** na cópia de trabalho:
`/mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work/` (NUNCA o source original).

## Objetivo
Implementar isolamento de credencial por conta para **Kiro** e **Antigravity**,
seguindo o contrato já publicado (Codex já foi feito como piloto). Criar
**arquivos NOVOS por vendor** — NÃO edite `execenv.go` nem `daemon.go` (esses são
de dono único de outro agente; a fiação no core é feita por ele).

## Protocolo de check-in/out (OBRIGATÓRIO — antes de editar)
1. Leia `Automonous_Agentic/.deploy-control/*.md` com `status: IN_PROGRESS`.
2. Confirme que nenhum arquivo que você vai tocar está em `files_locked` de outro.
3. Crie seu check-in ANTES de editar:
   `.deploy-control/CODEX__W-VENDORS__<START_UTC>.md` com:
   ```
   agent: CODEX
   stream: W-VENDORS
   started_at: <UTC ISO>
   finished_at:
   status: IN_PROGRESS
   files_locked:
     - server/internal/daemon/execenv/kiro_home.go
     - server/internal/daemon/execenv/kiro_home_test.go
     - server/internal/daemon/execenv/antigravity_home.go
     - server/internal/daemon/execenv/antigravity_home_test.go
   depends_on: [W-INT-contract]
   build_result:
   notes:
   ```
4. Ao terminar: preencha `finished_at`, `status: DONE`, `build_result` (verde) e
   a lista de arquivos. Se travar: `status: BLOCKED` + motivo em `notes`.

## Contrato publicado (siga à risca)
- Campo de entrada já existe no core: `PrepareParams.CredentialAccountHome` e
  `ReuseParams.CredentialAccountHome` (string; vazio = fallback global histórico).
- Padrão de referência: `execenv/codex_home.go` → `prepareCodexHomeWithOpts` com
  `CodexHomeOptions.AccountHome` e o helper `seedAccountAuth(accountHome, home, logger)`
  que copia `<accountHome>/auth.json` → `<home>/auth.json` via `syncCopiedFile`.
- **Você entrega apenas funções novas** em arquivos novos; o dono do core chama
  elas. Exponha assinaturas claras:
  - `prepareKiroHome(home string, opts KiroHomeOptions, logger *slog.Logger) error`
  - `prepareAntigravityHome(home string, opts AntigravityHomeOptions, logger *slog.Logger) error`
  - cada `*HomeOptions` deve ter um campo `AccountHome string`.

## O que implementar

### Kiro (`kiro_home.go`)
- Alavanca de isolamento: **`XDG_DATA_HOME`** por conta (o binário é fork do
  Amazon Q e IGNORA `KIRO_HOME`). O store nativo do Kiro é
  `~/.local/share/kiro-cli/data.sqlite3`.
- Prepare deve, quando `AccountHome != ""`: garantir um dir isolado por conta e
  (se existir) restaurar AS-IS o `data.sqlite3` da conta OU aceitar
  `KIRO_API_KEY` como alternativa headless (documente qual caminho no código).
- `AccountHome == ""` → **no-op** (fallback: comportamento atual do produto).
- `mkdir` do dir alvo antes do uso; permissões 0700.
- NUNCA logar conteúdo de credencial (apenas caminho/tipo/mtime).

### Antigravity (`antigravity_home.go`)
- Alavanca: **`HOME`** por conta → o CLI lê `~/.gemini/antigravity-cli/…token`.
- Prepare deve, quando `AccountHome != ""`: montar um HOME isolado da conta e
  restaurar AS-IS o token dir da conta. Vazio → no-op (fallback).

## Testes (obrigatórios, no padrão do codex_home_account_test.go)
- Prove isolamento: contas A e B recebem credenciais próprias; alteração em A não
  afeta B; arquivo é cópia (não symlink compartilhado global).
- Prove fallback: `AccountHome == ""` não quebra e não força cópia.

## Verificação (antes do check-out DONE)
```
cd /mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work
docker run --rm -v "$PWD":/src -w /src/server golang:1.26-alpine \
  sh -c "go build ./internal/daemon/... && go test ./internal/daemon/execenv/"
```
Só marque DONE com build + testes VERDES.

## Regras
- Postgres-only se precisar persistir algo (NUNCA SQLite próprio; o sqlite do
  Kiro é store NATIVO do vendor, tolerado só dentro do home isolado).
- De-branding: em arquivo que você criar, use nomes neutros; não introduza novas
  strings "Multica".
- Não toque em orquestração/canvas/dispatch/UI. Só o caminho de auth.
- Se algo do contrato faltar, marque BLOCKED e descreva — não improvise no core.
