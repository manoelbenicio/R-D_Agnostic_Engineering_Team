# FIX — Isolamento de Credencial por Conta · Documento Central

> Único documento para EXECUTAR o fix. Objetivo, full paths, sem histórico. Host fleet `manoelneto-laptop` (Windows: `/mnt/c/`→`C:\`).
> Verificado contra o código em 2026-07-06 (TL OPUS#46 + Codex#5.5#A).

## Objetivo (ALVO P0)
Isolamento de credencial por conta para **6 vendors** — **Codex, Kiro, Antigravity, GLM, Cline, OpenCode** — em UMA entrega, sem fases.
Persistência **PostgreSQL**. **Copiar** credencial (nunca symlink). **FAIL-CLOSED** (sem conta atribuída → NÃO usar credencial compartilhada).

## ESTADO ATUAL vs ALVO (não confundir)
| | Estado ATUAL (no código) | ALVO P0 (esta entrega) |
|---|---|---|
| Vendors isolados | **3**: Codex, Kiro, Antigravity | **6**: + GLM, Cline, OpenCode |
| GLM / Cline / OpenCode | `_vendor_env` retorna `{}` (SEM isolamento) | env própria por vendor |
| Sem atribuição de conta | **fallback p/ credencial compartilhada** (o bug) | **fail-closed** |
| Symlink `auth.json` | fallback symlink ainda ativo | sempre copiar |

## O BUG a eliminar (explícito, com linha)
1. **`daemon.go:3870-3889`** (`credentialAccountHomeForTask`): retorna `""` quando não há atribuição / conta indisponível / vendor não bate → daemon usa **credencial compartilhada**. É a violação do fail-closed. **Tornar fail-closed** (não deixar rodar sem conta isolada).
   `/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/daemon.go`
2. **`codex_home.go`**: fallback de **symlink** para `~/.codex/auth.json` compartilhado quando não há AccountHome → **sempre copiar**.
   `/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/execenv/codex_home.go`
3. **`_vendor_env` / `agent.go` / `execenv.go`**: adicionar env de isolamento para **GLM, Cline, OpenCode** (hoje `{}`).

## PostgreSQL — nomes REAIS das tabelas (migration 123)
`/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work/server/migrations/123_rotation.up.sql`
→ **`accounts`, `credentials`, `assignments`, `rotation_events`** (só `rotation_events` tem prefixo). Não existe `rotation_accounts/credentials/assignments`.

## 1. Spec aprovada (o QUE fazer)
- /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/openspec/changes/agent-credential-isolation/proposal.md
- /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/openspec/changes/agent-credential-isolation/design.md
- /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/openspec/changes/agent-credential-isolation/tasks.md
- /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/openspec/changes/agent-credential-isolation/specs/agent-credential-isolation/spec.md
- /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/openspec/changes/agent-credential-isolation/auth-inventory.md

## 2. Implementação de referência a PORTAR (o COMO — pronto no AOP)
- /mnt/c/VMs/Projects/AOP/control-plane/sessions_api/service.py   (`_prepare_isolated_paths` + `_vendor_env`)
- /mnt/c/VMs/Projects/AOP/control-plane/seats/pool.py             (`Seat.get_env` + `SeatPool` lease)
- /mnt/c/VMs/Projects/AOP/control-plane/rotation/models.py · detector.py · service.py · trigger.py · auth.py · pool.py
- /mnt/c/VMs/Projects/AOP/docs/30-COMPONENTES/36-ROTACAO-CONTAS-TOKEN.md   (spec autoritativa da rotação)

## 3. Ponto cirúrgico (ONDE alterar)
- /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/execenv/codex_home.go
- /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/daemon.go  (linhas 3870-3889)
- /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work/server/migrations/123_rotation.up.sql
- /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/runtime_isolation_test.go  (gate)

## 4. Matriz por vendor (ATUAL → A FAZER nesta entrega)
| Vendor | Env | Atual | A fazer |
|---|---|---|---|
| Codex | `CODEX_HOME` | ✅ isolado | fail-closed (remover fallback compartilhado) |
| Kiro | `XDG_DATA_HOME` | ⚠️ usa `KIRO_HOME` | corrigir p/ `XDG_DATA_HOME` |
| Antigravity | `HOME` (`~/.gemini/antigravity-cli`) | ⚠️ só HOME genérico | entrada própria |
| GLM | env próprio | ❌ `{}` | criar entrada + isolamento |
| Cline | data-dir próprio | ❌ `{}` | criar entrada + isolamento + detector |
| OpenCode | env próprio | ❌ `{}` | criar entrada + isolamento + detector |

## 5. Aceite (fechar só com os 6 verdes)
1. Duas contas do mesmo vendor coexistem sem sobreposição (para os 6).
2. Rotação automática ao esgotar a conta ativa.
3. **Fail-closed provado**: sem atribuição → NÃO usa credencial compartilhada.
4. Nenhum segredo em log.
5. `runtime_isolation_test.go` verde.
