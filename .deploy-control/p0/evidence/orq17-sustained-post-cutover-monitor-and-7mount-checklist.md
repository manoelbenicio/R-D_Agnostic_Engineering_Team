# ORQ-17 — Monitoramento Sustentado Pós-Cutover e Checklist de Reativação de 7 Mounts (READ-ONLY)

- **Autor:** Antigravity (wB:p1 / w8:p2)
- **Timestamp UTC:** `2026-07-27T16:08:00Z`
- **Destinatários:** General-Tech-Lead (Codex56-TL w5:pC), KIRO-PRINCIPAL-TL (wB:p1)
- **Governança:** Issue `ORQ-17` (Mechanical Safety Lane / Post-Cutover Sustained Monitor)
- **Skill Carregada:** `.agents/skills/aws-secrets-manager/SKILL.md` (zero texto claro, zero `GetSecretValue`, zero impressão de corpos ou identidades).
- **Modo:** MECHANICAL SAFETY LANE READ-ONLY — Amostragem de segurança em tempo de execução sem ativação do Tailscale Serve, sem logins, sem alteração de ambiente ou quadro.

---

## 1. Amostragem de Monitoramento Sustentado Pós-Cutover (UTC `2026-07-27T16:08:00Z`)

| Superfície Monitorada | Comando Factual | Resultado Medido no Host | Estado de Segurança |
|---|---|---|---|
| **Tailscale Serve Status** | `tailscale serve status` | `No serve config` (Zerado) | **CLEAN** |
| **Tailscale Funnel Status** | `tailscale funnel status` | `No serve config` (Zerado) | **CLEAN** |
| **Listener Porta 443** | `ss -tulpn \| grep :443` | `No listener on port 443` | **CLEAN** |
| **Backend Health (`:18080`)** | `curl -s http://127.0.0.1:18080/healthz` | **`HTTP 200 OK`** (`application/json`) | **HEALTHY** |
| **Backend Readiness (`:18080`)** | `curl -s http://127.0.0.1:18080/readyz` | **`HTTP 200 OK`** (`application/json`) | **HEALTHY** |
| **Frontend Status (`:13100`)** | `curl http://127.0.0.1:13100` | `HTTP 000` (`Connection refused`) | **ISOLADO (INATIVO)** |
| **Sonda Anônima `/api/me`** | `curl http://127.0.0.1:18080/api/me` | **`HTTP 401 Unauthorized`** (`text/plain`) | **FAIL-CLOSED OK** |
| **Sonda Anônima `/api/issues`**| `curl http://127.0.0.1:18080/api/issues?...`| **`HTTP 401 Unauthorized`** (`text/plain`) | **FAIL-CLOSED OK** |
| **Containers Ativos** | `docker ps -q` | Nenhum container Docker em execução | **CLEAN** |
| **Fila de Tarefas Ativa** | Query SQL em 4 estados | **`0`** (`queued`, `dispatched`, `running`, `waiting`) | **CLEAN** |

*Garantia de Higiene*: Zero corpos de requisição/resposta, zero senhas, zero UUIDs, zero hashes ou emails impressos no log.

---

## 2. Checklist Exato de Reativação dos 7 Mounts do Tailscale Serve

Quando a autorização formal do Owner for concedida e o container Web na porta `13100` estiver no ar, o Tailscale Serve deve ser configurado reativando a seguinte estrutura de **7 Mounts**:

```bash
# COMANDOS DE PROPOSTA DE REATIVAÇÃO DOS 7 MOUNTS (APENAS PROPOSTA — NÃO EXECUTADO)
# 1. Mount 1: Raiz do Frontend Web App
tailscale serve --bg / http://127.0.0.1:13100

# 2. Mount 2: Rotas de API do Backend
tailscale serve --bg /api http://127.0.0.1:18080/api

# 3. Mount 3: Endpoint de WebSocket
tailscale serve --bg /ws http://127.0.0.1:18080/ws

# 4. Mount 4: Endpoint de Health Check
tailscale serve --bg /health http://127.0.0.1:18080/health

# 5. Mount 5: Endpoint de Readiness Probe
tailscale serve --bg /readyz http://127.0.0.1:18080/readyz

# 6. Mount 6: Rota de Autenticação Login
tailscale serve --bg /auth/login http://127.0.0.1:18080/auth/login

# 7. Mount 7: Rota de Verificação de Código/Token
tailscale serve --bg /auth/verify http://127.0.0.1:18080/auth/verify
```

---

## 3. Rastreabilidade para Registro de Worklog (A8)

- **SHA-256 deste Documento de Evidência**: Ser capturado e fornecido para inclusão no registro de trabalho (worklog) do agente `A8`.

---

## 4. Declaração de FILES_LOCKED

- `multica-auth-work/server/internal/middleware/auth.go` (LOCKED — ORQ-17 Auth Hardening V2)
- `multica-auth-work/server/internal/middleware/auth_test.go` (LOCKED — ORQ-17 Tests)

---

## 5. Veredito Final
- **STATUS: PASS (MONITORAMENTO SUSTENTADO CONCLUÍDO E CHECKLIST DE 7 MOUNTS REGISTRADO)**
- **Documento Gravado**: `.deploy-control/p0/evidence/orq17-sustained-post-cutover-monitor-and-7mount-checklist.md`
- *Operação 100% Read-Only. Nenhuma leitura de segredos, nenhum envio de tráfego, zero mutações de estado.*
