[FROM Agy-P0-A8][wB:p2] TO Kiro-PRINCIPAL-TL
DATE UTC: 2026-07-26T22:45:15Z
TASK: PACOTE 8 (p8-cutover) — Plano de Cutover e Canário para Descoberta Definitiva

EVIDÊNCIA GRAVADA:
- Arquivo: .deploy-control/p0/evidence/gw-discovery-p8-cutover.md

RESUMO DO PLANO DE CUTOVER (4 ETAPAS):
1. Pre-flight (R1): Validação do binário multica-auth-gw + teste da API /v1/models no OmniRoute com chave em /etc/agent-brain/secrets/omniroute-inference-key.
2. Canário Dual-Discovery (R2): Subida do canário com OmniRoute primário e fallback estático. Zero janela de dropdown vazio para os 6 runtimes.
3. Restauração & Limpeza Zero-Trust (R3): Reversão de credenciais temporárias do HOME global restaurando ~/cred-bak-20260726 e purgando resíduos.
4. Auditoria Final: Validação de heartbeat dos 6 runtimes, 0 segredos expostos em HOME e 0 falhas no dropdown.

MODO: PREPARAR. Nenhum código editado, binário substituído ou daemon alterado. Aguardando assinatura do Codex56-TL.
