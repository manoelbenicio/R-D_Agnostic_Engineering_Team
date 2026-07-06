# FIX — Isolamento de Credencial por Conta · Documento Central

> Único documento de referência para EXECUTAR o fix. Só o que interessa (full paths). Sem histórico/discussões.
> Host: fleet `manoelneto-laptop` (Windows: trocar `/mnt/c/` por `C:\`).

## Escopo (inegociável)
- Entrega ÚNICA cobrindo TODOS os vendors: **Codex, Kiro, Antigravity, GLM, Cline, OpenCode**. Sem fases, sem adiar.
- Persistência **PostgreSQL** (tabelas `rotation_*`). SQLite proibido (exceto store nativo do vendor).
- Isolamento por conta: dir próprio `0700` (config dentro do home) + **COPIAR** credencial (nunca symlink) +
  injetar env nativa por vendor. **FAIL-CLOSED**: sem conta atribuída, NÃO cair no home global.
- Rotação (detector regex + HTTP 429) cobrindo TODOS os vendors acima.

## 1. Spec aprovada (o QUE fazer)
- /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/openspec/changes/agent-credential-isolation/proposal.md
- /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/openspec/changes/agent-credential-isolation/design.md
- /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/openspec/changes/agent-credential-isolation/tasks.md
- /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/openspec/changes/agent-credential-isolation/specs/agent-credential-isolation/spec.md
- /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/openspec/changes/agent-credential-isolation/auth-inventory.md

## 2. Implementação de referência a PORTAR (o COMO — já pronto no AOP)
- /mnt/c/VMs/Projects/AOP/control-plane/sessions_api/service.py   (`_prepare_isolated_paths` + `_vendor_env`)
- /mnt/c/VMs/Projects/AOP/control-plane/seats/pool.py             (`Seat.get_env` + `SeatPool` lease)
- /mnt/c/VMs/Projects/AOP/control-plane/rotation/models.py
- /mnt/c/VMs/Projects/AOP/control-plane/rotation/detector.py
- /mnt/c/VMs/Projects/AOP/control-plane/rotation/service.py
- /mnt/c/VMs/Projects/AOP/control-plane/rotation/trigger.py
- /mnt/c/VMs/Projects/AOP/control-plane/rotation/auth.py
- /mnt/c/VMs/Projects/AOP/control-plane/rotation/pool.py
- /mnt/c/VMs/Projects/AOP/docs/30-COMPONENTES/36-ROTACAO-CONTAS-TOKEN.md  (spec autoritativa da rotação)

## 3. Ponto cirúrgico no código (ONDE alterar)
- /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/execenv/codex_home.go
- /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/daemon.go
- /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work/server/migrations/123_rotation.up.sql
- /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/runtime_isolation_test.go  (gate de teste)

## 4. Matriz por vendor (tudo nesta entrega)
| Vendor | Env de isolamento | A fazer |
|---|---|---|
| Codex | CODEX_HOME | manter + fail-closed |
| Kiro | XDG_DATA_HOME | corrigir (hoje usa KIRO_HOME) + detector |
| Antigravity | HOME (`~/.gemini/antigravity-cli`) | entrada própria |
| GLM | env próprio | criar entrada + detector já existe |
| Cline | data-dir próprio | criar entrada + detector |
| OpenCode | env próprio | criar entrada + detector |

## 5. Aceite (fechar só com os 6 verdes)
1. Duas contas do mesmo vendor coexistem sem sobreposição.
2. Rotação automática ao esgotar a conta ativa.
3. Nenhum segredo em log.
4. `runtime_isolation_test.go` verde.
