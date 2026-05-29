## Problema

FinOps Tier 1 em v1 usa estimativa grosseira baseada em wall-clock multiplicado por taxa por hora do provider. A UI marca esses valores com aviso de estimativa, mas o calculo nao reflete token usage real, diferencas por modelo, tool calls, retries ou cobranca efetiva.

## Solucao proposta

Implementar FinOps Tier 2 com parsing de token usage real a partir de APIs, respostas e logs dos providers. O sistema deve capturar input tokens, output tokens, modelo usado, provider, terminal/session/canvas associados e calcular custo mais preciso por provider e por canvas.

## Escopo

- Integrar usage parsing para OpenAI, Anthropic, Google e AWS.
- Normalizar modelos, unidades de billing e moeda.
- Persistir usage events por session, terminal, canvas e provider.
- Recalcular custos por janela temporal e comparar com Tier 1.
- Exibir confianca/accuracia do custo e fallback quando usage real nao existir.
- Criar testes de parsing para payloads reais e simulados.

## Dependencias de v1

- FinOps Tier 1 e `PROVIDER_COST_PER_HOUR`.
- Terminal/session/canvas identifiers.
- CAO client e provider validation.
- Dashboard e FinOps pages para exibir custos.
- Avisos v1 de estimativa permanecem como fallback quando nao houver usage real.
