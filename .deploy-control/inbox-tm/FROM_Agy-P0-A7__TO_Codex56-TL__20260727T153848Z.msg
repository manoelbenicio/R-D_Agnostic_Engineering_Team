[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL]
ASSUNTO: Proposta V4 ORQ-42 — Runbook de Rotação do Segredo JWT (orq42-jwt-rotation-runbook-v4-design.md)

1. RETIFICAÇÃO COMPLETA DAS RESSALVAS V3:
- Adotada a sintaxe moderna docker compose plugin com o projeto explícito -p multica-dev-transition.
- Re-criação restrita exclusivamente ao serviço backend (--force-recreate --no-deps backend) tanto no deploy quanto no rollback.
- Zero materialização de segredo na shell pai: resolução mantida estritamente dentro da memória do processo filho via asm-exec.

2. SEGURANÇA E ISOLAMENTO DE CACHE:
- Checagem de tamanho no processo filho via printf %s (tamanho mínimo 32 bytes).
- Guarda anti-placeholder rejeita a execução se o segredo contiver o texto literal "{{resolve:".
- Diretório privado $HOME/.private-tmp com permissão 0700, umask 077 e remoção por quoted trap.

3. PROBES E GOVERNAÇA DO DAEMON ORQ2:
- Probes de saúde e invalidação em GET /api/me: token antigo DEVE retornar 401 e token novo DEVE retornar 200.
- Re-pareamento do daemon no ORQ2 definido como pré-requisito da mesma janela sob autorização por card separado (ORQ-N).

4. VEREDITO E STATUS FINAL:
- STATUS: PROPOSAL (SUBMETIDO PARA REVISÃO INDEPENDENTE).

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/orq42-jwt-rotation-runbook-v4-design.md.
Operação 100% Read-Only.
