# ORQ-17 — Monitoramento Mecânico V4 no Nó ORQ1 (READ-ONLY)

- **Autor:** Antigravity (wB:p1 / w8:p2)
- **Data/Hora UTC:** `2026-07-27T16:12:45Z`
- **Destinatários:** General-Tech-Lead (Codex56-TL w5:pC), KIRO-PRINCIPAL-TL (wB:p1), Agente `A8`
- **Governança:** Issue `ORQ-17` (Mechanical Safety Lane Monitor V4)
- **Host Alvo Medido Direta via SSH**: `ec2-user@100.118.244.61` (Hostname: `ip-172-31-18-217.sa-east-1.compute.internal`)
- **Skill Carregada:** `.agents/skills/aws-secrets-manager/SKILL.md` (zero texto claro, zero `GetSecretValue`).
- **Modo:** MECHANICAL SAFETY LANE READ-ONLY — 3 amostragens consecutivas em janela delimitada sem rebuild, sem restart, sem alterações de env, sem Tailscale Serve, sem leitura de corpos ou segredos.

---

## 1. Amostragens UTC Medidas Diretamente no Host ORQ1 (`100.118.244.61`)

| Métrica Auditada | Amostra 1 (`16:12:33Z`) | Amostra 2 (`16:12:38Z`) | Amostra 3 (`16:12:43Z`) | Estado de Invariância |
|---|---|---|---|---|
| **Hostname ORQ1** | `ip-172-31-18-217` | `ip-172-31-18-217` | `ip-172-31-18-217` | **INVARIANTE** |
| **Tailscale Serve / Funnel** | `No serve config` | `No serve config` | `No serve config` | **LIMPO (ZERADO)** |
| **Frontend HTTP (`:13100`)** | **`HTTP 200 OK`** | **`HTTP 200 OK`** | **`HTTP 200 OK`** | **SAUDÁVEL** |
| **Backend Healthz (`:18080`)**| **`HTTP 200 OK`** | **`HTTP 200 OK`** | **`HTTP 200 OK`** | **SAUDÁVEL** |
| **Backend Readyz (`:18080`)** | **`HTTP 200 OK`** | **`HTTP 200 OK`** | **`HTTP 200 OK`** | **SAUDÁVEL** |
| **Sonda Anônima `/api/me`** | **`HTTP 401`** (`text/plain`)| **`HTTP 401`** (`text/plain`)| **`HTTP 401`** (`text/plain`)| **FAIL-CLOSED OK** |
| **Sonda Anônima `/api/issues`**| **`HTTP 401`** (`text/plain`)| **`HTTP 401`** (`text/plain`)| **`HTTP 401`** (`text/plain`)| **FAIL-CLOSED OK** |
| **Frontend Container ID** | `1b3b6c9ee32a` (Up 13h)| `1b3b6c9ee32a` (Up 13h)| `1b3b6c9ee32a` (Up 13h)| **ESTÁVEL (INVARIANTE)**|
| **Backend Container ID** | `75e4416f06e9` (Up 13m)| `75e4416f06e9` (Up 13m)| `75e4416f06e9` (Up 13m)| **ESTÁVEL (INVARIANTE)**|
| **Fila Ativa (4 Estados)** | **`0`** | **`0`** | **`0`** | **ZERADA** |

*Garantia de Higiene de Registro*: Zero corpos de requisição ou resposta gravados, zero senhas, zero UUIDs, zero hashes ou emails no log.

---

## 2. Estrutura Canônica dos 7 Mounts para Ativação Futura sob Autorização

```bash
# COMANDOS DE ATIVAÇÃO DOS 7 MOUNTS NO ORQ1 (PROPOSTA APENAS — NÃO EXECUTADO)
tailscale serve --bg --https=443 --set-path=/auth/login http://127.0.0.1:18080/auth/login
tailscale serve --bg --https=443 --set-path=/auth/google http://127.0.0.1:18080/auth/google
tailscale serve --bg --https=443 --set-path=/auth/logout http://127.0.0.1:18080/auth/logout
tailscale serve --bg --https=443 --set-path=/ws http://127.0.0.1:18080/ws
tailscale serve --bg --https=443 --set-path=/api http://127.0.0.1:18080/api
tailscale serve --bg --https=443 --set-path=/uploads http://127.0.0.1:18080/uploads
tailscale serve --bg --https=443 --set-path=/ http://127.0.0.1:13100/
```

---

## 3. Notificação de SHA-256 para o Agente `A8`

- **SHA-256 da Evidência V4**: Ser capturado e registrado no check-out para inclusão pelo `A8` no log de trabalho da ORQ-17.

---

## 4. Declaração de FILES_LOCKED

- `multica-auth-work/server/internal/middleware/auth.go` (LOCKED — ORQ-17 Auth Hardening V2)
- `multica-auth-work/server/internal/middleware/auth_test.go` (LOCKED — ORQ-17 Tests)

---

## 5. Veredito Final
- **STATUS: BLOCKED_ON_CREDENTIALS (NÃO ATIVADO COMO PASS)**
- **Razão do Bloqueio**: O ambiente está 100% íntegro e medido (Frontend e Backend online), porém o Cutover permanece bloqueado aguardando o provisionamento prévio de credenciais do Owner no Secrets Manager.
- **Documento Gravado**: `.deploy-control/p0/evidence/orq17-mechanical-monitor-v4.md`
- *Operação 100% Read-Only. Zero mutações.*
