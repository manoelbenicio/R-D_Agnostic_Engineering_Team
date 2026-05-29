## Problema

AgentVerse v1 mitiga o risco R3 no nivel de prompt: o supervisor LLM recebe um bloco `canvas-topology` e deve respeitar a topologia declarada no canvas. Isso reduz risco, mas nao impede que o modelo ignore o grafo e tente executar `handoff`, `assign` ou `send_message` para um agente fora das arestas permitidas.

## Solucao proposta

Projetar e implementar um Validation Proxy que intercepte chamadas CAO relacionadas a comunicacao entre agentes e valide cada `handoff`, `assign` e `send_message` contra o grafo do Canvas Document implantado. Chamadas que violarem a topologia devem ser bloqueadas com erro explicito e auditavel.

## Escopo

- Definir o contrato do proxy e o modelo de grafo em runtime.
- Interceptar chamadas CAO de handoff, assign e send_message.
- Validar origem, destino e tipo de aresta contra a topologia do canvas.
- Decidir instalacao no CAO, SPA-side, ou camada hibrida.
- Expor erros de violacao de topologia para UI, logs e testes.
- Criar testes de contrato e integracao cobrindo arestas validas, invalidas e canvas degradado.

## Dependencias de v1

- Canvas Document schema e edge types de v1.
- Canvas Reconciler e estado deployado do canvas.
- CAO HTTP/WebSocket integration.
- Prompt-level mitigation v1 com `canvas-topology` permanece como defesa em profundidade.
