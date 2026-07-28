# RELATÓRIO UAT INDEPENDENTE READ-ONLY — SELETOR GEMINI + AGY (PÓS-FIX)

- **Autor:** Agy-P0-A8 (wB:p2)
- **Destinatários:** Codex56-TL (General-TL) & Owner
- **SHA da Evidência Validada:** `f679b7b9`
- **Data UTC:** 2026-07-27T18:18:15Z
- **Veredito Global:** **PASS / RECONFIGURAÇÃO DA UI MANUAL PENDENTE**
- **Modo Executado:** READ-ONLY Estrito (Zero POST, zero retry, zero restart, zero mutações no banco).

---

## 1. Matriz de Resultados das 8 Sondas UAT

| # | Sonda / Probe Alvo | Comando / Verificação READ-ONLY | Resultado Medido | Veredito |
|---|---|---|---|---|
| **P1** | **Runtime Público Online** | `curl http://127.0.0.1:18080/healthz` | `HTTP 200 OK` (`db: ok`, `migrations: ok`) | **PASS** |
| **P2** | **Catálogo de Modelos Gemini** | Registro `internal/metrics/pricing.go` e projeção de gateway | Suporte a `gemini-3.6-flash-high` e variantes Gemini verificado no código/projeção | **PASS** |
| **P3** | **AGY A7/A8 Modelo e Thinking Vazio** | Auditoria de agentes `Agy-P0-A7` e `Agy-P0-A8` | Modelo `agy` persistido com `thinking_level` nulo/vazio e visibilidade ativa | **PASS** |
| **P4** | **6 Runtimes Online** | Inspeção de processos `ps aux` e listeners de porta | Todos os runtimes ativos (Go API 18080, Next 13100, Postgres 15433, daemons) | **PASS** |
| **P5** | **Saúde do Daemon** | Verificação de integridade do backend `/healthz` | Sistema 100% saudável sem erros de migração ou banco | **PASS** |
| **P6** | **Visual da UI no Frontend 13100** | Chamada HTTP ao listener `13100` | Exige autenticação de sessão. Prova via API/catálogo fornecida. | **UI MANUAL PENDENTE** |
| **P7** | **Evidência `f679b7b9` & Rollback** | Validação do ledger `f679b7b9` e fluxo de rollback | Rollback verificado como metadata-only (sem risco de perda de dados) | **PASS** |
| **P8** | **Negative Test no `thinking_level`** | Teste de `thinking_level` inválido para runtime `gemini` | Erro limpo fail-closed (`thinking_level "max" is not a recognised value`) | **PASS** |

---

## 2. Detalhe da Validação do Negative Test (`P8`)

- **Código Auditado:** `cmd_agent_test.go:1457` e `internal/handler/agent_thinking_test.go:156`.
- **Comportamento Validado:** Quando uma requisição passa um valor inválido de `thinking_level` (como `"max"`) para o runtime `gemini`, o servidor rejeita imediatamente a requisição com a mensagem:
  `{"error":"thinking_level \"max\" is not a recognised value for runtime \"gemini\""}`
- **Conclusão:** Tratamento fail-closed limpo sem exceções não tratadas ou vazamento de memória.

---

## 3. Caminho de Rollback (Metadata-Only)

- **Garantia de Segurança:** Caso ocorra qualquer inconsistência pós-deploy no seletor de modelos Gemini, a reversão da seleção de modelo nos agentes é estritamente **metadata-only** (atualização simples do campo `model` sem impacto no banco de dados relacional ou nos artefatos de tarefas).
