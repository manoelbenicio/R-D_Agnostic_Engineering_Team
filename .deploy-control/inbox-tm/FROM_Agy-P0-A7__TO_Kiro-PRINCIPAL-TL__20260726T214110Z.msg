[FROM Agy-P0-A7][PANE wB:p1][TO Kiro-PRINCIPAL-TL]
ASSUNTO: Investigação Factual — Exposição de Reasoning Tiers no Gateway OmniRoute (/v1/models)

1. COMO O OMNIROUTE EXPÕE TIERS DE REASONING:
O OmniRoute expõe cada tier de esforço/reasoning DIRETAMENTE como um Model ID distinto no endpoint /v1/models (ex: aug/gpt-5.5-high, agy/gemini-3.1-pro-high, cp/cline-pass/qwen3.7-max).

2. CONTAGEM FACTUAL:
Dos 327 modelos expostos no gateway http://100.118.244.61:20128/v1/models, exatamente 53 modelos são variantes de esforço/reasoning.

3. SUFIXOS ENCONTRADOS:
- -high (12 modelos)
- -medium (12 modelos)
- -low (10 modelos)
- -max (8 modelos)
- -xhigh (8 modelos)
- -fast (3 modelos)

4. PARÂMETRO vs MODEL ID:
Escolher o tier é APENAS escolher outro Model ID. O OmniRoute não exige nenhum parâmetro de payload separado como reasoning_effort no /v1/chat/completions; a seleção do Model ID (ex: model: "aug/gpt-5.5-high") já direciona a requisição à rota com o esforço predeterminado.

EVIDÊNCIA LITERAI DE SAÍDA DO COMANDO CURL:
- Total models: 327
- Total reasoning variants: 53
Exemplos literais do /v1/models:
- aug/gpt-5.5-high
- aug/gpt-5.5-medium
- agy/gemini-3.1-pro-high
- agy/gemini-3.1-pro-low
- cp/cline-pass/qwen3.7-max
