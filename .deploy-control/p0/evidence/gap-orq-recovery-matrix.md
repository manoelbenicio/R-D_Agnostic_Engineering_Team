# ORQ Recovery Matrix v3 — ORQ-12/13/15/16/17/18/21/22/23

**auditor**: Antigravity w8:p2  
**timestamp-v3**: 2026-07-27T10:21Z  
**fonte de dados**: API direta `GET /api/issues?workspace_id=...` (sem CLI), `issue run-messages` (storage real)  
**modo**: read-only — nenhuma ação executada  

---

## Correções desta versão

1. **assignee=none estava errado**: a API com `workspace_id` no query param retorna campo incorreto; a API com `?workspace_id=...` diretamente no endpoint correto confirma `assignee_type=agent` + `assignee_id` preenchido em todas as 9 issues. IDs de agente não expostos neste documento.
2. **`runs.tool_use_count=0` era stale**: campo de cache. `issue run-messages` (storage real) confirma os counts 57/83/31/45/57/87 — são da **task atual única** de cada issue, não de runs anteriores.
3. **postgresql17-contrib**: `rpm -q postgresql17-contrib` no ORQ2 retorna `not installed`. O tool_use em ORQ-21 foi tentativa — **não há prova de conclusão** sem tool_result correspondente. Reclassificado como tentativa ambígua.

---

## Tabela consolidada

| Issue | Status | Assignee | task_id (atual) | created_at task | tool_use | tool_result | Veredito | Motivo |
|---|---|---|---|---|---|---|---|---|
| ORQ-12 | todo | agent (preenchido) | d4f7f744 | 2026-07-27T02:42:13Z | 57 | 1 | ⚠️ OWNER/AMBÍGUO | 1 tool_result confirmado; workspace pode ter estado parcial; precisa de diff antes de rerun |
| ORQ-13 | todo | agent (preenchido) | 00c87814 | 2026-07-27T02:42:13Z | 83 | 5 | ⚠️ OWNER/AMBÍGUO | 5 tool_result + múltiplos strReplace em Go/TS; edições parcialmente aplicadas; requer inspeção de diff |
| ORQ-15 | todo | agent (preenchido) | e79257cd | não iniciou | 0 | 0 | ⚠️ OWNER | agent_error.unknown, started_at=None; causa não visível; nenhum efeito confirmado |
| ORQ-16 | todo | agent (preenchido) | ed0301e0 | não iniciou | 0 | 0 | ⚠️ OWNER | agent_error.unknown, started_at=None; causa não visível; nenhum efeito confirmado |
| ORQ-17 | todo | agent (preenchido) | e0197c7e | 2026-07-27T02:42:13Z | 31 | 0 | ⚠️ OWNER/AMBÍGUO | 0 tool_result = nenhum efeito durável confirmado; tool_use prova intenção, não conclusão |
| ORQ-18 | todo | agent (preenchido) | 91e70c79 | 2026-07-27T02:42:13Z | 45 | 1 | ⚠️ OWNER/AMBÍGUO | 1 tool_result; efeito parcial possível; requer inspeção |
| ORQ-21 | todo | agent (preenchido) | e39199ed | 2026-07-27T02:42:13Z | 57 | 4 | ⚠️ OWNER/AMBÍGUO | 4 tool_result; tentativa de `sudo dnf install postgresql17-contrib` detectada em tool_use mas `rpm -q` no ORQ2 confirma **não instalado** — tentativa ambígua, não conclusão |
| ORQ-22 | in_progress | agent (preenchido) | (task ativa) | — | 21 | 0 | ⚠️ OWNER | Status kanban in_progress mas 0 tool_result; task em execução ou presa; não interromper sem autorização |
| ORQ-23 | todo | agent (preenchido) | ff121b28 | 2026-07-27T02:42:13Z | 87 | 3 | ⚠️ OWNER/AMBÍGUO | 3 tool_result; edições em arquivos possíveis; requer diff antes de rerun |

---

## Regra aplicada

- **tool_use sem tool_result** = tentativa/intenção registrada pelo agente, **sem prova de conclusão**. Não afirma efeito colateral.
- **tool_result presente** = agente recebeu resposta do ambiente — efeito possível mas não caracteriza o tipo de mutação sem inspeção do conteúdo.
- **ORQ-22** permanece `in_progress` com task ativa (21 tool_use, 0 tool_result): não rerun, não interromper sem owner.

---

## Nota sobre unicidade de task

Cada issue tem exatamente 1 task no histórico (retornada por `issue runs`). Os counts 57/83/31/45/57/87 são da mesma task atual — não existem runs anteriores separados. A API retorna 1 run por issue porque só existe 1.

*Nenhuma ação executada. Nenhum rerun disparado. Apenas leitura e análise.*
