# OWNER RULING — Política Oficial de Execução, Custos/Tokens e Atribuição de Cards (READ-ONLY)

- **Autor:** Antigravity (wB:p1 / w8:p2)
- **Data/Hora UTC:** `2026-07-27T17:26:05Z`
- **Destinatários:** General-Tech-Lead (Codex56-TL w5:pC), KIRO-PRINCIPAL-TL (wB:p1), Todos os Agentes
- **Governança:** Direct Owner Ruling — Execution & Cost Authorization Policy
- **Modo:** REGISTRO DE GOVERNANÇA READ-ONLY — Incorporação imediata das regras do Owner no fluxo operacional de todos os agentes.

---

## 1. Diretriz Primária: Custos e Tokens NÃO são Bloqueador

- **Regra de Execução**: Custo financeiro ou consumo de tokens **NÃO constituem bloqueador** para execução de tarefas reais autorizadas.
- **Modelo e Agentes**: Toda tarefa técnica real autorizada pode ser imediatamente atribuída e executada utilizando o agente ou modelo de IA mais adequado para a tarefa (ex: Pro, Flash, subagentes dedicados).
- **Proibição de Parada por Custo**: Agentes **NÃO DEVEM solicitar autorização prévia ao Owner apenas por motivos de consumo de tokens ou custo**.

---

## 2. Matriz Exclusiva de Bloqueadores Reais (O Que Permanece Bloqueado)

Permanecem **RIGOROSAMENTE BLOQUEADAS** apenas as operações que incorram nas seguintes violações de segurança e governança:

1. **Duplicidade ou Acidente (`duplicidade/acidente`)**: Esforço duplicado entre agentes ou execuções não intencionais.
2. **Conflito de Ownership (`conflito de ownership`)**: Mutação de arquivos ou escopos pertencentes a outro agente ou raia travada.
3. **Exposição de Segredos (`segredo exposto`)**: Leitura direta ou log de segredos (estrita aplicação da skill `aws-secrets-manager`).
4. **Mutação Fora de Escopo (`mutação fora de escopo`)**: Alterações em arquivos protegidos ou fora da tarefa atribuída.
5. **Risco de Dados sem Backup/Rollback (`risco de dados sem backup/rollback`)**: Mutações destrutivas sem plano de reversão e backup factual.

---

## 3. Política de Atribuição de Cards no Quadro (Assignee Policy)

- **Cards Puramente de Status / Monitoramento**:
  - Permanecem **sem assignee** (`unassigned`), pois não demandam execução de código ou alteração de repositório.
- **Cards Reais de Implementação e Correção**:
  - **PODEM E DEVEM disparar execução técnica imediata** pelo agente responsável assim que a raia/portão estiver liberado.

---

## 4. Veredito Final
- **STATUS: OWNER RULING RECORDED & ACTIVE ACROSS ALL LANES**
- **Documento Gravado**: `.deploy-control/p0/evidence/owner-ruling-cost-and-execution-policy.md`
- *Operação de Governança 100% Read-Only.*
