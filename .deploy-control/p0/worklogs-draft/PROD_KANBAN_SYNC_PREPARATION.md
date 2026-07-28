# INVENTÁRIO MECÂNICO DO BACKLOG E PREPARAÇÃO DE SINCRONIZAÇÃO (P0/P1 PROD)

- **Estado Global:** DRAFT_NOT_POSTED (Aguardando Sinalização Explícita de Autenticação por Kiro)
- **Gate 0 Preflight:** PASS (Concluído em < 2s em 2026-07-27T17:46:54Z; Go 1.26.1, gofmt, git 2.50.1, jq 1.8.1, sha256, cfn-lint 1.46.0, 15 GiB livres)
- **Agente Registrar:** Agy-P0-A8 (wB:p2)
- **Liderança Técnica AWS:** Kiro-Opus5
- **Autoridade de Integração:** Codex56-TL
- **Data UTC:** 2026-07-27T17:46:48Z

---

## 1. Inventário Mecânico de Issues do Backlog (Sem Leader / Stalled / Sem Runtime)

| Card | UUID do Card | Líder / Assignee | Estado dos Portões (Gates) | Estado de Execução / Runtime | Causa de Parada / Blocker | Ação Necessária & ETA |
|---|---|---|---|---|---|---|
| **ORQ-13** | `2aefcb3d-97b3-4e70-a4b0-ae3729b7981d` | **UNASSIGNED / STALLED** | BLOCK (Linter ST1003) | Desativado (Off) | SQLC gera `var max_seq`, `var token_hash`, etc em `snake_case` | Ajustar SQLC ou isenção ST1003 (Pós-SQLC) |
| **ORQ-17** | `7d873133-16d5-42c6-8595-629d6fb16251` | Agy-P0-A8 (Worktree auth tests PASS) | Stage3B BLOCK | Desativado (Serve Off) | Provisionamento de credenciais do owner no Stage 3B. RCA `user=0` corrigido para banco vivo `1/0/1` | Runbook de credenciais pelo Owner |
| **ORQ-26** | `e966922d-c6a5-4812-87bb-8b9576ccbc60` | Agy-P0-A7 | **PASS** (Consumer Tests) | Ativo | Validação E2E em browser autenticado | Sincronizar no sinal de Kiro (Imediato) |
| **ORQ-32** | `b01925fe-e914-422a-812e-f63cada274dc` | **UNASSIGNED / STALLED** | BLOCK (Dual-token ausente) | Desativado (Off) | Backend Go lê apenas `HANDSHAKE_TOKEN` único (sem suporte a token secundário) | PR de suporte a token secundário |
| **ORQ-38** | `fd5c4d55-8ce5-412f-9f15-db666831bfdb` | Codex56#A | **PASS** (Contrato GET /api/issues) | Ativo | Auditoria do contrato concluída (HTTP 400 fail-closed se sem workspace_id) | Sincronizar no sinal de Kiro (Imediato) |
| **ORQ-39** | `c03941bc-3bde-4de1-ab19-1ba93de0ad51` | Agy-P0-A8 (Plano V3) | DESIGN PASS / EXEC BLOCK | Desativado (Off) | Bloqueio de congelamento (Kanban Freeze) em workflows de automação E2E | Autorização de desblocagem pelo GTL |
| **ORQ-41** | `666f1ead-7fe9-4051-bab9-5d0a936c4701` | Agy-P0-A8 (Resgatado de Opus48#D) | **CODE_PASS / EVIDENCE_PASS** (`c047c0b`) | Flag **OFF** | `MULTICA_EXECUTION_TRIGGER_DECOUPLED` desativado aguardando `LANE-DB`, `W2`, `W3` | Sincronizar no sinal de Kiro (Imediato) |
| **ORQ-42** | `64bfcae0-b867-4812-ad33-ae03ef7f25ae` | **UNASSIGNED / STALLED** | V5 BLOCK (Design OK) | Desativado (Off) | Janela de manutenção e autorização do owner no Secrets Manager | Execução na janela do Owner |
| **ORQ-43** | `d1149dd3-9da8-4678-a4b7-d98f3eddca14` | **UNASSIGNED / STALLED** | DESIGN PASS | Desativado (Off) | Design de ciclo de vida de token `mdt_` aguardando agendamento | Agendamento de implementação |
| **ORQ-44** | `abd12d6a-16a5-439b-bc52-74ec4bb6b231` | **UNASSIGNED / STALLED** | DESIGN PASS | Desativado (Off) | Design de rotação de chave de inferência OmniRoute aguardando agendamento | Agendamento de rotação |

---

## 2. Protocolo de Sincronização Sem Duplicação de Mutações

1. Mantidos os 10 rascunhos de notas `/note ` (`is_note=true`) com marcadores de idempotência (`WORKLOG-V1:<id>:<sha>`).
2. Cards puramente de status mantidos sem atribuição (`UNASSIGNED`).
3. Zero requisições enviadas antes do login autenticado confirmado por Kiro.
