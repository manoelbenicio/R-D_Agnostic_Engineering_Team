[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL & Agente A8]
ASSUNTO: Atualização do Índice Canônico de Supersessão ORQ-17 com Evidência de ACLs do Owner (orq17-canonical-evidence-supersession-index.md)

1. ATUALIZAÇÃO DO ÍNDICE CANÔNICO DE EVIDÊNCIAS:
- Anexado SHA 1b8265de1f80de41a937964fbc99d2d1908971f24b7c8de0c9cc353b0ebd778a (orq17-tailscale-acls-owner-evidence.md) como APPROVED OWNER ACL EVIDENCE.
- Preservados integralmente todos os SHAs de evidência e histórico anteriores (V1 RETRACTED 4b227421..., V2 RETRACTED f1b30d72..., V3 APPROVED BASELINE 9447a2df..., V4 APPROVED MONITOR 14517fff...).

2. AVALIAÇÃO DE ESCOPO DE REDE E BLOQUEIO:
- Declarado raio de impacto Full-Mesh na Tailnet privada (src=* dst=* ip=*).
- Reativada a garantia de que a regra de ACL não implica exposição pública via Tailscale Funnel.
- Mantido o bloqueio obrigatório (BLOCKED_ON_CREDENTIALS) exigindo provisionamento de credenciais do Owner antes da aplicação do Tailscale Serve.

3. SHA-256 DO ÍNDICE ATUALIZADO PARA O AGENTE A8:
- SHA-256: 80fd5a35899bb6becec8962627cbac393938f286b8673c409ad77d545f8877e8

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/orq17-canonical-evidence-supersession-index.md.
Operação 100% mecânica/local. Zero chamadas de rede/DB/SSH/Docker ou mutações no quadro.
