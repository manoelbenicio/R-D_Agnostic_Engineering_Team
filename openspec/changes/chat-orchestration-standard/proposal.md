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

## Impact
- Config/instruções (identity do leader) + roteamento default de chat/task. Reusa primitivos
  existentes (squad leader + delegação + Squad Operating Protocol).
- Execução: coders. **Kiro planeja e valida.**

## Non-goals
- Impedir conversa direta com um agente (o escape hatch é requisito).
- Forçar OpenSpec explore em toda task (é a critério do leader quando há dúvida/complexidade).
