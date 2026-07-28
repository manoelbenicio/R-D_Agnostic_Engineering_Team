# Matriz de Inventário Git READ-ONLY para Liberação (ORQ-13, 17, 26, 38, 39, 41)

- **Autor:** Antigravity (wB:p1 / w8:p2)
- **Data/Hora UTC:** `2026-07-27T16:16:50Z`
- **Destinatários:** General-Tech-Lead (Codex56-TL w5:pC), KIRO-PRINCIPAL-TL (wB:p1), Agente `A8`
- **Governança:** Release Control Git Inventory Audit (`ORQ-13`, `ORQ-17`, `ORQ-26`, `ORQ-38`, `ORQ-39`, `ORQ-41`)
- **Modo:** INVENTÁRIO MECÂNICO READ-ONLY — Zero `checkout`, zero `stash`, zero `add`, zero `commit`, zero `fetch`, zero `push`. Zero alteração de arquivos ou quadro.

---

## 1. Tabela Executiva do Inventário Git por Card

| Card | Caminho do Worktree | Branch | HEAD SHA | Merge-Base | Status | Staged / Untracked | Commit Local? | Cruzamento com Evidências | Status do Cruzamento |
|---|---|---|---|---|---|---|---|---|---|
| **ORQ-13** | `/home/ec2-user/workspace/worktrees/gtl-i03-orq13-phase1` | `agent/opus48-b/orq-13-thinking-level` | `0cb8aebb5aff79cb430b3740d22fadc53c0116fd` | `0cb8aebb5aff79cb430b3740d22fadc53c0116fd` | **DIRTY** | 1 / 4 (7 mod) | **NÃO** | Runbook V2 Aprovado (`orq13-wave0-sqlc-baseline-runbook-v2.md`) | `UNCOMMITTED_DIRTY_WORKTREE` |
| **ORQ-17** | `/home/ec2-user/workspace/worktrees/gtl-orq17-auth-tests` | `agent/agy-a8/orq17-auth-regression` | `9cf3296fcf2c8d5ff72f439df16492e6cd76c1fa` | `0cb8aebb5aff79cb430b3740d22fadc53c0116fd` | **CLEAN** | 0 / 0 | **SIM** | Commit `9cf3296fcf2c8d5ff72f439df16492e6cd76c1fa` (`orq17-auth-tests-minimal-fix.md`) | **`MATCHED_EXACT`** |
| **ORQ-26** | `/home/ec2-user/workspace/worktrees/ci-orq26-db-gate` | `ci/orq26-db-gate` | `d2447183c5fb7d93c07ca4afdd86d4ab9ac374ee` | `0cb8aebb5aff79cb430b3740d22fadc53c0116fd` | **CLEAN** | 0 / 0 | **SIM** | Commit `d2447183c5fb7d93c07ca4afdd86d4ab9ac374ee` (`orq26-github-auth-unblock-runbook.md`) | **`MATCHED_EXACT`** |
| **ORQ-26** | `/home/ec2-user/workspace/worktrees/gtl-orq26` | `agent/kiro-opus5/orq-26-contract-fix` | `0cb8aebb5aff79cb430b3740d22fadc53c0116fd` | `0cb8aebb5aff79cb430b3740d22fadc53c0116fd` | **DIRTY** | 1 / 1 (2 mod) | **NÃO** | Rascunho de alteração de contrato não commitado | `UNCOMMITTED_DIRTY_WORKTREE` |
| **ORQ-26** | `/home/ec2-user/workspace/worktrees/gtl-orq26-consumer-tests` | `agent/agy-a7/orq26-consumer-tests` | `0cb8aebb5aff79cb430b3740d22fadc53c0116fd` | `0cb8aebb5aff79cb430b3740d22fadc53c0116fd` | **DIRTY** | 1 / 0 (2 mod) | **NÃO** | Testes de consumidor não commitados | `UNCOMMITTED_DIRTY_WORKTREE` |
| **ORQ-38** | `/home/ec2-user/workspace/worktrees/gtl-orq38-get-contract` | `agent/codex-b/orq38-get-contract` | `0cb8aebb5aff79cb430b3740d22fadc53c0116fd` | `0cb8aebb5aff79cb430b3740d22fadc53c0116fd` | **DIRTY** | 1 / 3 (3 mod) | **NÃO** | Plano de Testes Aprovado (`orq38-get-issue-fail-closed-test-plan.md`) | `UNCOMMITTED_DIRTY_WORKTREE (FLAGGED)` |
| **ORQ-39** | `/home/ec2-user/workspace/worktrees/ci-orq39-browser-qa` | `ci/orq39-browser-qa-gate` | `0cb8aebb5aff79cb430b3740d22fadc53c0116fd` | `0cb8aebb5aff79cb430b3740d22fadc53c0116fd` | **CLEAN** | 0 / 0 | **NÃO** | Desenho V4 Aprovado (`orq39-browser-qa-v4-amendment-design.md`) | `UNCOMMITTED_NO_CARD_COMMIT` |
| **ORQ-41** | `/home/ec2-user/workspace/worktrees/gtl-orq41-w4-autopilot` | `agent/opus48-d/orq41-w4-autopilot` | `e0b0155d0f0d46a9da3724ecbfc51a5167bd2ebe` | `0cb8aebb5aff79cb430b3740d22fadc53c0116fd` | **CLEAN** | 0 / 0 | **SIM** | Claim `c047c0b7ed71...` (`orq41-w4-c047c0b-code-review.md`) | **`SUPERSEDED_BY_COMMIT_e0b0155`** |

---

## 2. Apontamento de Incoerências e Relação de Commits (Cross-Reference Findings)

1. **ORQ-17 (`9cf3296fcf2c8d5ff72f439df16492e6cd76c1fa`)**:
   - **Status**: **`MATCHED EXACT`**.
   - O commit `9cf3296fcf2c8d5ff72f439df16492e6cd76c1fa` existe no worktree `/home/ec2-user/workspace/worktrees/gtl-orq17-auth-tests` na branch `agent/agy-a8/orq17-auth-regression`. Worktree 100% LIMPO.
2. **ORQ-26 (`d2447183c5fb7d93c07ca4afdd86d4ab9ac374ee`)**:
   - **Status**: **`MATCHED EXACT`**.
   - O commit `d2447183c5fb7d93c07ca4afdd86d4ab9ac374ee` existe no worktree `/home/ec2-user/workspace/worktrees/ci-orq26-db-gate` na branch `ci/orq26-db-gate`. Worktree 100% LIMPO.
3. **ORQ-41 (`c047c0b7ed710f260e76a6ee50257b2a5f96dc70`)**:
   - **Status**: **`SUPERSEDED BY COMMIT e0b0155d0f0d46a9da3724ecbfc51a5167bd2ebe`**.
   - O commit `c047c0b` foi revisado com BLOCK em `orq41-w4-c047c0b-code-review.md` e corrigido no novo commit `e0b0155d0f0d46a9da3724ecbfc51a5167bd2ebe`. Worktree 100% LIMPO.
4. **ORQ-13, ORQ-38, ORQ-39**:
   - Possuem planos/desenhos aprovados em `.deploy-control/p0/evidence/`, porém **NÃO possuem commit local de card** criado na branch (ORQ-13 e ORQ-38 contêm alterações não commitadas em worktree dirty; ORQ-39 está em base limpa aguardando autorização).

---

## 3. Arquivo Formato Máquina (TSV)

- **Caminho do TSV**: `.deploy-control/p0/evidence/orq-git-inventory-matrix.tsv`

---

## 4. Declaração de FILES_LOCKED

- `multica-auth-work/server/internal/middleware/auth.go` (LOCKED — ORQ-17 Auth Hardening V2)
- `multica-auth-work/server/internal/middleware/auth_test.go` (LOCKED — ORQ-17 Tests)

---

## 5. Veredito Final
- **STATUS: PASS (INVENTÁRIO GIT READ-ONLY CONCLUÍDO)**
- **Documento Gravado**: `.deploy-control/p0/evidence/orq-git-inventory-matrix.md`
- *Operação 100% Read-Only. Nenhuma mutação git ou de arquivos executada.*
