[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL]
ASSUNTO: Diagnóstico Factual de Indisponibilidade do Frontend ORQ-17 (orq17-frontend-outage-diagnosis.md)

1. CAUSA RAIZ IDENTIFICADA:
- O servidor backend Go está ativo e saudável em 127.0.0.1:18080 (tunnel SSH ativo).
- A falha Connection Refused na porta 13100 ocorre porque não há processo escutando em 127.0.0.1:13100.
- docker ps -a confirma que a aplicação web não está em container Docker ativo.

2. AVALIAÇÃO DE REBOOT VS REBUILD:
- O Next.js embute a variável NEXT_PUBLIC_API_URL de forma estática no build do pacote web (runtime-urls.ts).
- Reiniciar o processo sem recompilar manteria a URL legada. É OBRIGATÓRIO um REBUILD da aplicação web passando NEXT_PUBLIC_API_URL=https://orq1.tail96e2c0.ts.net para o cutover final.

3. COMANDOS DE RECUPERAÇÃO E ROLLBACK:
- Proposta de recuperação: NEXT_PUBLIC_API_URL="https://orq1.tail96e2c0.ts.net" PORT=13100 pnpm --filter @multica/web build && start & (PROPOSTA APENAS, NÃO EXECUTADO).
- Plano de rollback documentado.

4. VEREDITO FINAL:
- STATUS: DIAGNÓSTICO CONCLUÍDO.

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/orq17-frontend-outage-diagnosis.md.
Operação 100% Read-Only.
