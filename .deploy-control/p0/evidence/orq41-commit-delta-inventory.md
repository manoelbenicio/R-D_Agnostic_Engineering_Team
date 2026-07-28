# Inventário Mecânico do Delta de Commits da ORQ-41 (`c047c0b..e0b0155`) (READ-ONLY)

- **Autor:** Antigravity (wB:p1 / w8:p2)
- **Data/Hora UTC:** `2026-07-27T16:18:15Z`
- **Destinatários:** General-Tech-Lead (Codex56-TL w5:pC), KIRO-PRINCIPAL-TL (wB:p1), Revisor Independente
- **Governança:** Issue `ORQ-41` (W4 Autopilot Rescue Commit Delta Inventory)
- **Worktree Inspecionado:** `/home/ec2-user/workspace/worktrees/gtl-orq41-w4-autopilot`
- **Modo:** INVENTÁRIO MECÂNICO READ-ONLY — Sem interpretação de corretude, sem execução de testes, sem edição de arquivos, sem mutações em Git ou quadro.

---

## 1. Rastreabilidade da Cadeia de Commits e Mensagem de Commit

- **Intervalo de Comparação**: `c047c0b7ed710f260e76a6ee50257b2a5f96dc70..e0b0155d0f0d46a9da3724ecbfc51a5167bd2ebe`
- **SHA Pai Direto de `e0b0155`**: `c047c0b7ed710f260e76a6ee50257b2a5f96dc70`
- **Autor do Commit**: `kiro-codex56a <codex56a@orq2.local>`
- **Mensagem do Commit (`e0b0155`)**:
  ```text
  fix(autopilot): keep replay gate off-path and correlate by prior task

  W4 correction on top of c047c0b. No amend, no migrations, no generated code,
  no router.go.

  - The replay gate no longer runs unless it is explicitly armed. NewAutopilotService
    leaves it disarmed, so admission behaviour is identical to the pre-gate Autopilot:
    the nil-hook path can no longer refuse 100% of dispatches in production.
  - Correlation is the prior *task*, resolved through AutopilotReplayCorrelation.
    The autopilot ID is never used as a ledger key; the first-run case (no prior
    task) is allowed without consulting the gate.
  - Gate and resolver are injected once via NewAutopilotServiceWithReplayGate and
    are immutable afterwards. SetReplayGateHook is gone, removing the late-setter
    race on a service that is already serving.
  - PutDaemonLedgerSummary validates the reserved transport contract and fails
    closed with 503; it no longer answers 200 "recorded" while writing nothing.
    Registration requirements that only W3 can satisfy are documented in place.
  - No HMAC secret work is claimed: CommitLedgerHMACSecret is still never assigned
    and is outside the W4 file set in the approved V3 manifest.

  Wired-on behaviour stays deferred behind the two interfaces because the durable
  ledger summary store and the prior-task lookup are owned by LANE-DB and W2.
  ```

---

## 2. Estatística Exata do Delta e Lista de Arquivos Alterados

### Relatório de `git diff --stat c047c0b..e0b0155`:
```text
 .../internal/handler/daemon_ledger_summary.go      |  45 ++---
 .../internal/handler/daemon_ledger_summary_test.go |  22 +--
 .../server/internal/service/autopilot.go           | 130 +++++++++++--
 .../internal/service/autopilot_replay_test.go      | 209 +++++++++++++++------
 4 files changed, 297 insertions(+), 109 deletions(-)
```

### Lista Factual de Arquivos Mutados:
1. `multica-auth-work/server/internal/handler/daemon_ledger_summary.go` (+18, -27)
2. `multica-auth-work/server/internal/handler/daemon_ledger_summary_test.go` (+7, -15)
3. `multica-auth-work/server/internal/service/autopilot.go` (+115, -15)
4. `multica-auth-work/server/internal/service/autopilot_replay_test.go` (+157, -52)

---

## 3. Estado de Limpeza do Worktree e Verificação `git show --check`

- **Estado do Worktree (`git status --short`)**: **`100% CLEAN`** (Zero arquivos modificados ou não rastreados).
- **Resultado de `git show --check e0b0155`**: **`PASS`** (Zero erros de sintaxe de whitespace ou formatação).

---

## 4. Verificação de Correspondência de Caminhos Proibidos (Forbidden Paths Check)

| Palavra-Chave Proibida | Arquivos Inspecionados no Diff | Status da Verificação |
|---|---|---|
| `router` | Nenhum arquivo alterado em `router` ou `router.go` | **CLEAN** |
| `migrations` | Nenhum arquivo alterado em `migrations/` | **CLEAN** |
| `generated` | Nenhum arquivo alterado em `generated/` | **CLEAN** |
| `queries` | Nenhum arquivo alterado em `queries/` | **CLEAN** |
| `commitledger` | Nenhum arquivo alterado em `commitledger` | **CLEAN** |
| `task` | Nenhum arquivo alterado em `task` | **CLEAN** |
| `daemon` | Encontrado em `daemon_ledger_summary.go` e `daemon_ledger_summary_test.go` | **PERMITIDO (Parte do manifesto W4)** |

---

## 5. Arquivo Formato Máquina (TSV)

- **Caminho do TSV**: `.deploy-control/p0/evidence/orq41-commit-delta-inventory.tsv`

---

## 6. Declaração de FILES_LOCKED

- `multica-auth-work/server/internal/middleware/auth.go` (LOCKED — ORQ-17 Auth Hardening V2)
- `multica-auth-work/server/internal/middleware/auth_test.go` (LOCKED — ORQ-17 Tests)

---

## 7. Veredito Final
- **STATUS: PASS (INVENTÁRIO MECÂNICO DO DELTA CONCLUÍDO E REGISTRADO)**
- **Documento Gravado**: `.deploy-control/p0/evidence/orq41-commit-delta-inventory.md`
- *Operação 100% Mecânica e Read-Only. Nenhuma interpretação de corretude ou alteração realizada.*
