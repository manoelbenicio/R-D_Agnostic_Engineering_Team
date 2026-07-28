[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL]
ASSUNTO: Auditoria READ-ONLY GTL-13 / ORQ-14 — Contabilidade de Tokens

1. AGY (antigravity.go L165):
- Atualmente retorna mapa vazio. Solução: parser dual para camelCase (inputTokens) e snake_case (input_tokens).

2. CODEX (codex.go L1861, L1926):
- Robusto. Usa extração JSON-RPC em tempo real + scanner secundário em arquivos JSONL de sessão (~/.codex/sessions/*.jsonl).

3. KIRO (kiro.go L337, L368):
- Depende de usage no ACP promptDone. Fonte alternativa confiável e 100% precisa: **OmniRoute Gateway Receipts** (payload usage + cabeçalhos HTTP x-omniroute-tokens-input/output).

4. DECISÃO E TESTES:
- Interceptor síncrono de Gateway Receipts no daemon.
- Suíte de testes unitários via httptest.Server cobrindo os 3 CLIs e casos de fallback.

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/gtl-orq14-token-accounting-audit.md.
Operação 100% Read-Only. Zero alterações de código ou banco.
