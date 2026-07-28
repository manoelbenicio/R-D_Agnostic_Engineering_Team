[FROM Agy-P0-A8][wB:p2] TO Kiro-PRINCIPAL-TL
DATE UTC: 2026-07-26T21:41:15Z
TASK: Investigação de Custo por Tier de Reasoning no OmniRoute

INVESTIGAÇÃO EMPÍRICA:
1. `/api/pricing` -> Retorna HTTP 403 Forbidden com chave de inferência:
   `{"error":{"code":"AUTH_001","message":"Invalid management token","correlation_id":"..."}}`
   Header: `x-omniroute-route-class: MANAGEMENT`.

2. `/v1/models` -> Retorna HTTP 200 (327 modelos). Exemplo de objeto de modelo:
   `id: "codex/gpt-5.6-sol-high"`, `capabilities.effort_tiers: ["none","low","medium","high","xhigh"]`.
   Modelos expõem variações de esforço como IDs separados (`-low`, `-medium`, `-high`, `-xhigh`).
   Contudo, o endpoint `/v1/models` não expõe campos de valor numérico $/1k tokens.

3. IMPACTO DE CUSTO:
   Mudar de tier (ex: `low` -> `xhigh`) aumenta a geração de tokens de raciocínio/pensamento (output tokens) em até 3x-10x, o que eleva proporcionalmente o custo total registrado por requisição.
