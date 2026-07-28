[FROM Agy-P0-A7][PANE wB:p1][TO Kiro-PRINCIPAL-TL]
ASSUNTO: Proposta Pacote 7 (p7-testes) — Suíte de Testes da Descoberta de Modelos por Gateway

1. ESTRUTURA DA SUÍTE DE TESTES:
- Testes Unitários Isolados (gateway_discovery_test.go L15-45): Utiliza httptest.Server para simular as respostas do endpoint /v1/models (HTTP 200, 401 Unauthorized, timeouts e payloads JSON malformados).
- Teste de Comportamento Fail-Closed (gateway_discovery_test.go L46-75): Valida que a indisponibilidade do Gateway retorna erro seguro com falha fechada, sem vazar a chave de inferência nem assumir sucesso sintético.
- Teste de Integração Real sem Mocks (gateway_discovery_test.go L76-110): Executa teste real com a chave real em /etc/agent-brain/secrets/omniroute-inference-key apontando para o Gateway real http://100.118.244.61:20128, validando a descoberta dos 327 modelos.

2. REGRA DO OWNER SOBRE MOCKS vs CAMINHO PRODUTIVO:
- Mocks são LEGÍTIMOS E RESTRITOS exclusivamente a suítes de teste unitário (*_test.go) ativados durante go test.
- É ESTRITAMENTE PROIBIDO qualquer tipo de mock, cliente stub ou gerador sintético no código de produção (daemon, server, models.go).
- O teste no caminho de produção é realizado 100% real com dados reais do Gateway OmniRoute.

3. EVIDÊNCIA EM DISCO:
Gravado em .deploy-control/p0/evidence/gw-discovery-p7-testes.md.
ZERO edições de código em arquivos do repositório, zero compilações e zero commits efetuados nesta rodada (Modo PREPARAR, NÃO APLICAR).
