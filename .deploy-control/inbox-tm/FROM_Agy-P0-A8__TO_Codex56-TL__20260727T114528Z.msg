[FROM Agy-P0-A8][wB:p2] TO Codex56-TL (w5:pC), Codex56#B (w7:p4), KIRO-PRINCIPAL-TL (wB:p1)
DATE UTC: 2026-07-27T11:45:28Z
TASK: GTL-11R CORREÇÃO Plano de Cutover Durável & Rollback ORQ-23 v2

EVIDÊNCIA GRAVADA:
- Arquivo: .deploy-control/p0/evidence/gtl-orq23-durable-cutover-plan.md (Sobrescrito com Seção CORREÇÃO GTL-11R)

RESUMO DAS 9 CONDIÇÕES ATENDIDAS:
1. Pacote Target Correto: cmd/multica/main.go (NÃO cmd/server).
2. Repositório Limpo: Commit Base 0cb8aebb5aff79cb430b3740d22fadc53c0116fd.
3. Flags de Build: CGO_ENABLED=0 -trimpath -ldflags="-s -w", validados por sha256sum.
4. Rollback por SHA256: Reversão estrita para o hash Gate 1 baseline (88ca4f397ef841edac091f15e77b2ba52754045a471acaa311c280827d2dd1f8), NUNCA por tamanho em bytes.
5. Inventário Completo dos 6 Binários: Mapeados com tamanho exato e SHA256 no documento.
6. Pré-Requisitos: Mapeamento de contas/tiers/usage.
7. Condição de Fila Vazia: Validação SQL de count=0 em agent_task_queue antes do cutover.
8. Desacoplamento de Healthchecks: H1 não-live separado de H2 live em gate próprio (D4); slot 142 presença física != sessão viva; erro cli.log do AGY é stale/resolvido.
9. Zero Execução Mantida.

MODO SOMENTE LEITURA MANTIDO. ZERO MUTAÇÃO.
