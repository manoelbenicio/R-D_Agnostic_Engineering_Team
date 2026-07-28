# OWNER RULING — Separação Estrita entre Coordenação e Execução Técnica (READ-ONLY)

- **Autor:** Antigravity (wB:p1 / w8:p2)
- **Data/Hora UTC:** `2026-07-27T17:28:45Z`
- **Destinatários:** General-Tech-Lead (Codex56-TL w5:pC), KIRO-PRINCIPAL-TL (wB:p1), Todos os Agentes
- **Governança:** Direct Owner Ruling — Coordination vs Execution Separation Policy
- **Modo:** REGISTRO DE GOVERNANÇA READ-ONLY — Incorporação imediata da matriz de responsabilidades e isolamento de escrita entre o General-TL e os Agentes Executores.

---

## 1. Atribuições do General-Tech-Lead (Codex56-TL)

- **Escopo Exclusivo**:
  1. Orientação estratégica e direcionamento das raias de trabalho (*Steering*).
  2. Definição de prioridades da equipe e aceite formal do **Gate 0 (Preflight Inventory)**.
  3. Decisão de *Ownership*, resolução de conflitos e integração final entre raias.
  4. Monitoramento passivo e governança do progresso das tarefas.
- **Proibição de Escrita**: O General-TL **NÃO executa comandos técnicos em paralelo no mesmo alvo** nem toma para si os artefatos de um agente executor sem handoff formal.

---

## 2. Atribuições do Agente Designado (Executor Exclusivo)

- **Escopo Exclusivo**:
  1. O agente designado para a tarefa é o **ÚNICO EXECUTOR E ESCRITOR (*Sole Writer*)** dos arquivos, processos, repositórios e artefatos de evidência atribuídos.
  2. **Padrão de Escrita Segura**: Obrigatoriedade de utilização de saída temporária única (ex: `$HOME/.private-tmp/` com permissão `0700`) seguida de **promoção atômica** (`mv` / *atomic rename*).
  3. **Reporte de Milestones**: Notificação ativa dos marcos (*milestones*) atingidos para o General-TL via `herdr`.

---

## 3. Regras de Intervenção Direta

- **Exceção Única para Intervenção**: Qualquer intervenção técnica direta do General-TL no arquivo ou repositório de um agente exige:
  - **Handoff Explícito Formalizado** no diretório `.deploy-control/`, OU
  - **Contenção Emergencial Pré-Autorizada** (com documentação imediata do desvio à posteriori).
- **Sem Handoff**: Fora destas condições, o General-TL atua estritamente em modo de direcionamento e coordenação (*Steering Only*).

---

## 4. Veredito Final
- **STATUS: OWNER RULING ON COORDINATION/EXECUTION SEPARATION RECORDED**
- **Documento Gravado**: `.deploy-control/p0/evidence/owner-ruling-coordination-execution-separation.md`
- *Operação de Governança 100% Read-Only.*
