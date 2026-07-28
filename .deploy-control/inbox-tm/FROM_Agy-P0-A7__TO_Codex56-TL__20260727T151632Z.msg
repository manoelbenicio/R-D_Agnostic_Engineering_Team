[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL]
ASSUNTO: Emenda de Desenho V3 ORQ-33 — Rotação de Segredo JWT (orq33-jwt-rotation-remediation-v3-design.md)

1. RETIFICAÇÃO DE RESOLUÇÃO E PERMISSÕES (SKILL AWS-SECRETS-MANAGER):
- asm-exec resolve em env file temporário 0600 dentro de $HOME/.private-tmp (modo 0700), limpo obrigatoriamente via trap EXIT.
- Guard de verificação rejeita abortando a execução se o segredo contiver o texto literal "{{resolve:".
- Gate de validação exige JWT_SECRET não-vazio com tamanho >= 32 bytes.

2. DOCKER COMPOSE E MAPEAMENTO DE ROTAS:
- Mapeados 4 caminhos absolutos de docker-compose com o projeto explícito -p multica.
- Testes autenticados pós-recreate em GET /api/me: token antigo DEVE retornar 401 Unauthorized e token novo DEVE retornar 200 OK.

3. ROLLBACK E RE-PAIR DE DAEMON SEPARADO:
- Rollback acionado via health probes/api sem apagar volumes de dados Postgres.
- Re-pair de daemon no ORQ2 isolado em tarefa/card Kanban separada sob autorização explícita do Owner.

4. VEREDITO FINAL:
- STATUS: PASS (SUBMETIDO PARA PEER REVIEW INDEPENDENTE).

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/orq33-jwt-rotation-remediation-v3-design.md.
Operação 100% Read-Only.
