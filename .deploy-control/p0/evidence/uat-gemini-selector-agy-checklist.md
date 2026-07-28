# CHECKLIST UAT & SONDAS READ-ONLY — SELETOR GEMINI + AGY (P0)

- **Autor:** Agy-P0-A8 (wB:p2)
- **Destinatários:** Codex56-TL (General-TL) & Owner
- **Status:** PLANO UAT PREPARADO / AGUARDANDO COMUNICAÇÃO DE FIX/LOGIN (Opus48-D & Kiro)
- **Data UTC:** 2026-07-27T18:10:47Z
- **Regra de Execução:** Sondas estritamente READ-ONLY. Zero edição de código, zero mutação no banco, zero sobreposição de tarefas de Opus48-D ou Kiro.

---

## 1. Matriz de Critérios de Aceitação UAT (8 Sondas READ-ONLY)

| # | Critério de Aceitação | Sonda / Probe READ-ONLY | Resultado Esperado | Status UAT Atual |
|---|---|---|---|---|
| **P1** | **Runtime Público Correto** | `GET /healthz` em `127.0.0.1:18080` e borda HTTPS | `HTTP 200 OK`, runtime operacional | PENDENTE SINAL |
| **P2** | **Catálogo Retorna `gemini-3.6-flash-high`** | `GET /api/models` (ou contrato de modelos) | Modelo `gemini-3.6-flash-high` presente no payload JSON | PENDENTE SINAL |
| **P3** | **Exibição na Interface (UI)** | Inspeção da UI no frontend `13100` | Opção `gemini-3.6-flash-high` visível no seletor de modelos | PENDENTE SINAL |
| **P4** | **Persistência da Seleção** | `GET /api/agents/{id}` pós-seleção | Campo `model` armazena e persiste `gemini-3.6-flash-high` | PENDENTE SINAL |
| **P5** | **Atribuição pelo Leader** | Verificação de permissão e atribuição via Leader | Leader consegue atribuir tarefas com o modelo selecionado | PENDENTE SINAL |
| **P6** | **Visibilidade dos Agentes AGY A7/A8** | `GET /api/agents` (filtro workspace) | Agentes `Agy-P0-A7` e `Agy-P0-A8` ativos e visíveis | PENDENTE SINAL |
| **P7** | **Disparo Pós-OAuth Slot150** | Inspeção do diretório `/home/ec2-user/.agent-cred-homes/slots/slot-150` | Tarefa só é admitida após materialização e auth do `slot-150` | PENDENTE SINAL |
| **P8** | **Negative Test no `thinking_level`** | Envio de request com `thinking_level` inválido | Validação fail-closed com erro limpo sem crash de runtime | PENDENTE SINAL |

---

## 2. Protocolo de Execução Pós-Comunicação

1. Aguardar sinalização de conclusão de correção (fix) ou login emitida por Opus48-D / Kiro.
2. Executar sequencialmente as 8 sondas READ-ONLY acima sem realizar escritas no banco ou servidor.
3. Registrar o resultado final como **PASS** (todas as sondas aprovadas) ou **BLOCK** (caso alguma sonda falhe), acompanhado dos hashes SHA256 de evidência.
