# Proposal — TL/Manager Chat Orchestration (native & standard)

## Why
Hoje agentes ficam soltos e qualquer um responde ao chat. O dono quer o comportamento
**padrão** do sistema: toda task/chat chega a um **TL/Manager** (squad leader) que **esclarece
dúvidas**, **documenta** (abrindo um OpenSpec explore do zero quando fizer sentido), **planeja**
e só então **delega** aos agentes envolvidos — sintetizando o resultado. O leader coordena e
(quando delegation-only) **não produz** código.

## What Changes
- **ADDED** modelo padrão: chat/task roteia por default para um **TL/Manager (squad leader)**.
- **ADDED** protocolo do leader: esclarecer → (opcional) OpenSpec explore para documentar →
  planejar → delegar aos agentes envolvidos → sintetizar/entregar.
- **ADDED** escape hatch: o usuário PODE endereçar um runtime/agente específico direto, para
  tarefas pontuais, sem passar pelo TL.
- **MODIFIED** setup default do workspace: existir um squad TL/Manager com leader configurado
  e o chat roteado a ele por padrão.
- **MODIFIED** toda ativacao executavel de agente parte do Kanban por assignee/API e gera uma
  unica task de produto; Herdr fica restrito a supervisao read-only.

## Impact
- Config/instruções (identity do leader) + roteamento default de chat/task. Reusa primitivos
  existentes (squad leader + delegação + Squad Operating Protocol).
- Execução: coders. **Kiro planeja e valida.**
- O General Tech Leader continua decidindo prioridade, escopo, aceite e integracao; o operador
  do Kanban executa somente transicoes e despachos explicitamente autorizados.

## Provenance & Evidence Boundary (ORQ-88 / REC-CHAT-CLAIMS-01)
- Authoritative Candidate OID: `a5aa53e8e89d2845cacfbc82ca851fbd18f9a505` (Tree OID: `b3d3f484fa7204308d085c17a99db8fe7b586f6e`).
- 12 OpenSpec claims reconciled: DIRECT 3 (`1.3`, `2.2`, `2.3`), CORROBORATED 5 (`0.1`, `0.2`, `0.3`, `1.1`, `1.4`), CLAIMED 3 (`1.2`, `1.5`, `2.1`), MISSING 1 (`2.4`), CONTRADICTED 0.
- Sealed isolated PostgreSQL execution verified via E1 (`CHAT-CLOSEOUT.md`, inner SHA-256 `5c247789...`).
- Open gaps preserved: Task 1.2 (isolated PG test required), Task 2.1 (live LLM synthesis unproven), Task 2.4 (native Kanban dispatch missing), and Root Owner attribution waiver requirement for `1.1`/`1.4` checkbox authority.

## Non-goals
- Impedir conversa direta com um agente (o escape hatch é requisito).
- Forçar OpenSpec explore em toda task (é a critério do leader quando há dúvida/complexidade).
