# CHECKLIST P1 DO OWNER — SELETOR GEMINI + AGY (READ-ONLY FOLLOW-UP)

- **Autor:** Agy-P0-A8 (wB:p2)
- **Destinatários:** Codex56-TL (General-TL) & Owner
- **Data UTC:** 2026-07-28T13:12:05Z
- **Status Backend/API:** **PASS (100% OPERACIONAL)**
- **Único Passo Pendente:** **Validação Visual da UI no Frontend 13100 pós-login**

---

## 1. Evidências de Prontidão Medidas (Backend / API / AGY)

- **Sonda 1 (Runtime Público):** `127.0.0.1:18080/healthz` -> `HTTP 200 OK` (`db: ok`, `migrations: ok`).
- **Sonda 2 (Catálogo & Precificação):** Modelos Google Gemini (`gemini-3.6-flash-high`, `gemini-3-flash`, `gemini-3.1-pro`, `gemini-2.5-pro`, `gemini-2.5-flash`) verificados e registrados em `internal/metrics/pricing.go`.
- **Sonda 3 (Isolamento AGY A7/A8 & Slot150):** Isolamento de credenciais Antigravity (`.gemini/antigravity-cli/antigravity-oauth-token`) via `HOME` e permissões `0700` ativas para `Agy-P0-A7` e `Agy-P0-A8`.
- **Sonda 4 (Rollback Path):** Reversão de modelo confirmada como **metadata-only** (sem impacto estrutural ou risco de perda de dados).

---

## 2. Checklist Conciso para o Owner (Único Passo Pendente em UI)

- [ ] **Passo 1 (Login na UI):** Efetuar login autenticado na interface web (`http://127.0.0.1:13100`).
- [ ] **Passo 2 (Seleção de Modelo):** Abrir as configurações de agente e verificar a presença de `gemini-3.6-flash-high` no menu suspenso (dropdown).
- [ ] **Passo 3 (Persistência):** Salvar a alteração e confirmar que a opção permanece selecionada após recarregar a página.
- [ ] **Passo 4 (Confirmação Kanban):** Sinalizar a conclusão do passo visual para o registrador Kanban efetuar o fecho final.

---

## 3. Garantias de Preservação
- Zero alteração de código ou runtime.
- Zero duplicação da conclusão de A7.
- Zero requisições POST, retry ou manipulação de segredos.
