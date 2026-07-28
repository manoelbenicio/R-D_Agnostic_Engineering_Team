# Design — TL/Manager Orchestration

## Fluxo padrão (native)
```
Chat/Task ─▶ TL/Manager (squad leader)
                 │
                 ├─ 1. Esclarecer: faz perguntas até não haver dúvida
                 ├─ 2. Documentar: se complexo/ambíguo, abre OpenSpec explore do zero
                 ├─ 3. Planejar: quebra em subtasks, define agentes envolvidos
                 ├─ 4. Delegar: distribui aos agentes (membros do squad)
                 └─ 5. Sintetizar: coleta resultados e entrega
```
- Reusa primitivos existentes: roteamento **squad→leader**, **delegação** a membros, **Squad
  Operating Protocol**, papel **delegation-only** (leader não produz).

## Escape hatch (pontual)
```
Chat direto ─▶ @agente/runtime específico ─▶ executa a tarefa pontual (sem TL)
```
- Preservar o roteamento direto atual (chat com agente privado / @-mention a um agente).

## Peças a definir (config + instruções, não runtime nativo novo)
1. **Squad TL/Manager default** por workspace: 1 leader + membros (os coders).
2. **Identity/instructions do leader** com o protocolo (esclarecer → openspec explore →
   planejar → delegar → sintetizar) + marcador `## Squad Operating Protocol` (ativa `IsSquadLeader`).
3. **Roteamento default do chat**: mensagens sem destino explícito → squad TL; com `@agente` → direto.
4. **Regras**: leader delegation-only não posta produção; usa `multica squad activity`/delegação.

## Decisões (resolvidas pelo dono — 2026-07-12)
- Roteamento default do chat: **sempre por squad** (→ TL/Manager).
- **OpenSpec mandatório**; projeto do zero **sem doc**: o TL **inicia a doc (explore/proposal) ou o projeto não acontece** (gate duro).
- Squad TL padrão + membros e limiar de explore: TL abre OpenSpec sempre que iniciar trabalho do zero sem doc; demais casos a critério do leader.

## Validação (Kiro)
- Smoke: chat sem destino → cai no TL; TL faz pergunta, delega, sintetiza. Chat `@codex` → direto.

## Despacho mecanico oficial — 2026-07-28

- O TL define card, prioridade, escopo, acceptance, executor e ETA.
- O operador atribui o card pelo Kanban/API; a mudanca de assignee cria exatamente uma task.
- Follow-up ocorre por comentario normal no card somente depois de a task anterior estar
  terminal; `/note` e content-only e nunca dispara agente.
- Herdr pode ser usado para leitura de estado, nunca para iniciar ou repetir trabalho.
- Status visual sem assignee/task nao representa execucao; `in_progress` exige ownership
  visivel e uma execucao real ou milestone ativo verificavel.
- O leader ajuda, direciona, revisa e integra, mas nao escreve em paralelo nos arquivos do
  executor designado.
