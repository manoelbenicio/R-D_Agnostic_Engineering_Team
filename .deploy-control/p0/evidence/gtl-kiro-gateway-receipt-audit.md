# Auditoria READ-ONLY GTL-20 — Hipótese de Gateway Receipt para Kiro

## 1. Avaliação Factual e Veredito da Hipótese
- **VEREDITO: HIPÓTESE PASSIVA REJEITADA**.
- **Causa Factual**: O daemon inicia o `kiro-cli` via ACP JSON-RPC sobre stdin/stdout (\`kiro.go\` L30-38 e L66). As chamadas HTTP de inferência ocorrem diretamente entre o processo filho \`kiro-cli\` e o OmniRoute. O daemon **NÃO recebe a resposta HTTP nem os cabeçalhos do OmniRoute** no fluxo atual.
- **Consequência**: Tentar ler cabeçalhos \`x-omniroute-tokens-*\` passivamente no daemon sem interceptar o tráfego HTTP é inviável, pois tais cabeçalhos morrem no cliente \`kiro-cli\`.

## 2. Rastreamento de Payload, Headers e Ponto de Recepção
- **Rota no OmniRoute**: \`POST /v1/chat/completions\`.
- **Headers do OmniRoute**: \`x-omniroute-tokens-input\` e \`x-omniroute-tokens-output\`.
- **Onde o Dado É Recebido**: Atualmente recebido apenas no buffer interno de socket do \`kiro-cli\`. O protocolo ACP (\`session/prompt\`) entre o \`kiro-cli\` e o daemon omite a struct de usage quando o runtime não a expõe em JSON-RPC.

## 3. Correlação Segura de Identidade (\`task_id\` / \`account_id\` / \`tier\`)
- **Preservação de Segredo**: \`task_id\`, \`account_id\` pseudônimo e \`thinking_level\` podem ser correlacionados sem expor segredos se o daemon injetar cabeçalhos de contexto (\`X-Multica-Task-ID\`, \`X-Multica-Account-ID\`) no ambiente do processo filho.
- **Requisito Necessário**: Para que a captura ocorra, o daemon precisa subir um **Daemon Loopback Proxy** (\`HTTP_PROXY=http://127.0.0.1:<port>\`) que intercepta o tráfego de inferência do Kiro, lê os cabeçalhos de resposta do OmniRoute, e registra a contagem na \`task_usage\` ligada ao \`task_id\`.

## 4. Riscos de Double-Counting e Streaming (SSE)
- **Streaming (SSE)**: Respostas em \`stream: true\` devolvem múltiplos chunks \`data: {...}\`. O payload final contém \`usage\` apenas se \`stream_options: {"include_usage": true}\` estiver ativo.
- **Risco de Contagem Dupla**: Interceptar requisições HTTP sem desduplicação por \`request_id\` / \`turn_id\` pode somar o mesmo consumo no stream e no encerramento da task.
- **Garantia de Desduplicação**: O interceptor deve registrar o receipt indexado pela chave única \`(task_id, omniroute_request_id)\`.

## 5. Mapa de Especificação e Suíte de Testes (Sem Mudar Produção)
- **Patch Map Especificado**:
  1. \`server/internal/daemon/execenv/proxy.go\`: Injetar \`HTTP_PROXY\` apontando para o listener do daemon.
  2. \`server/internal/daemon/gateway/receipt_interceptor.go\`: Interceptar \`x-omniroute-tokens-*\` e gravar na \`task_usage\` via \`ReportTaskUsage\`.
- **Testes com \`httptest.Server\`**:
  * Simular \`kiro-cli\` através do proxy gravando tokens com sucesso.
  * Teste de fail-closed: se o proxy falhar, a tarefa não vaza credencial nem corrompe a contabilidade.
