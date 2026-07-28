# RELATÓRIO RETRATADO — AUDITORIA DE DUPLICIDADE DE AGENTES

- **Status:** RETRACTED (PREMISSA FALSA)
- **Autor:** Agy-P0-A8 (wB:p2)
- **Data UTC da Retratação:** 2026-07-27T18:08:42Z

---

## 1. Motivo da Retratação (Premissa Falsa Refutada)

- **Alegação Prévia:** Existência de agentes duplicados `Codex-A` e `Codex-B` no mesmo workspace.
- **Fato Medido no Código e Esquema:**
  - A restrição de integridade `agent_workspace_name_unique UNIQUE (workspace_id, name)` **JÁ ESTÁ APLICADA** na migração `046_agent_unique_name.up.sql:22`.
  - As instâncias registradas como `Codex-A` e `Codex-B` pertencem a **WORKSPACES DIFERENTES**.
  - Não há violação de unicidade nem ambiguidade no mesmo workspace.

---

## 2. Substituição pelo Bug Real (`UpdateAgent` HTTP 500 -> 409)

- **Bug Real Identificado:** No manipulador `UpdateAgent` (`internal/handler/agent.go`), quando ocorre uma violação de unicidade de nome (`pgErr.Code == "23505"` na constraint `agent_workspace_name_unique`), o servidor retornava `HTTP 500` exposto com detalhes do driver de banco de dados em vez de um erro `HTTP 409` limpo e content-free.
- **Resolução em Andamento:** O agente **Codex56-A** já está implementando o tratamento do erro `23505` para retornar `HTTP 409 Conflict`.
