# ORQ-17 — Índice Canônico de Supersessão e Rastreabilidade de Evidências (V2 COM ACLS DO OWNER)

- **Autor:** Antigravity (wB:p1 / w8:p2)
- **Data/Hora UTC:** `2026-07-27T16:27:40Z`
- **Destinatários:** General-Tech-Lead (Codex56-TL w5:pC), KIRO-PRINCIPAL-TL (wB:p1), Agente `A8`
- **Governança:** Issue `ORQ-17` (Canonical Evidence Supersession Index V2)
- **Modo:** DOCUMENTAÇÃO PURA (MECÂNICA LOCAL) — Zero chamadas de rede, API, DB, Docker ou SSH. Preservação integral de todos os hashes e histórico anteriores.

---

## 1. Tabela Canônica de Supersessão de SHAs de Evidência

| Versão da Evidência | SHA-256 da Evidência | Arquivo de Evidência Relacionado | Status de Governança | Razão / Justificativa Factual |
|---|---|---|---|---|
| **V1 (Monitor Inicial)** | `4b22742148bbfe0052bbe26df7f66c1de32415c9c029dbf102477063a0ba2c14` | `orq17-sustained-post-cutover-monitor-and-7mount-checklist.md` | **`RETRACTED`** | Mounts incorretos propostos (`/health`, `/readyz`, `/auth/verify`) e sonda de frontend realizada localmente no ORQ2. |
| **V2 (Reconciliação 1)** | `f1b30d7229b0e2b4ce0b2d2a56130669b3b8207fbd1c4cc66d13bf143849ad39` | `orq17-sustained-post-cutover-monitor-and-7mount-checklist-v2.md` | **`RETRACTED`** | Falsa hipótese de outage/rebuild do frontend (causada por sondagem no ORQ2) e inclusão de 9 comandos com duplicatas. |
| **V3 (Baseline Prontidão)** | `9447a2df1098ab44522ca765d7c174918712d6acaf9d097110216419988e0b1d` | `orq17-auth-cutover-v3-final-readiness.md` | **`APPROVED READINESS BASELINE`** | Baseline factual provado no nó ORQ1 (`100.118.244.61`, frontend `1b3b6c9ee32a` Up 13h, backend `75e4416f06e9` Up 13m). |
| **V4 (Estabilidade Monitor)** | `14517fff8c4286c4fe0f75edcf1828add4463271145e45f14b8b4ce91a43152b` | `orq17-mechanical-monitor-v4.md` | **`APPROVED STABILITY MONITOR`** | 3 amostras UTC consecutivas no host ORQ1 confirmando invariância de containers, status 401 fail-closed e fila zerada. |
| **ACL Scope (Owner)** | `1b8265de1f80de41a937964fbc99d2d1908971f24b7c8de0c9cc353b0ebd778a` | `orq17-tailscale-acls-owner-evidence.md` | **`APPROVED OWNER ACL EVIDENCE`** | Evidência OWNER-PROVIDED de `ACLs.png` (SHA256 `ef8fa8a10ae1...`). Confirma regra `src=* dst=* ip=*` e fecha a lacuna U-1. |

---

## 2. Rastreabilidade dos Estágios da ORQ-17

- **Estágio 1 (Backup & Auditoria Janela)**:
  - Artefato: `.deploy-control/p0/evidence/gtl-orq17-window-audit-and-auth-cutover-plan.md`
- **Estágio 2 (Preflight & Cutover Gate A)**:
  - Artefatos: `.deploy-control/p0/evidence/orq17-auth-cutover-readiness-preflight.md` e `.deploy-control/p0/evidence/gtl-orq17-gate-a-execution.md`
- **Correção da Suíte de Testes de Autenticação**:
  - Artefato: `.deploy-control/p0/evidence/orq17-auth-tests-minimal-fix.md`
  - Commits: Parent `40909d85ee1962745e436cc540c2323f95e59820` -> Local Fix `9cf3296fcf2c8d5ff72f439df16492e6cd76c1fa` (Alinhado `AUTH_TOKEN_TTL` com o contrato `sync.Once`).
- **Estágio 3B (Auditoria & Retificação de Prontidão de Login)**:
  - Artefato: `.deploy-control/p0/evidence/orq17-stage3-login-readiness.md` (Contagens agregadas `0` para `user_password_credential` e `"user"`).
- **Evidência de Escopo de Rede (Owner ACLs)**:
  - Artefato: `.deploy-control/p0/evidence/orq17-tailscale-acls-owner-evidence.md` (SHA-256 `1b8265de1f80de41a937964fbc99d2d1908971f24b7c8de0c9cc353b0ebd778a`).

---

## 3. Avaliação de Escopo de Rede e Raio de Impacto (Blast Radius)

- **Regra de ACL Registrada**: `src=* dst=* ip=*` (`action: accept`).
- **Raio de Impacto (Blast Radius)**:
  Configura acesso **Full-Mesh interno** entre todos os usuários, dispositivos, portas e protocolos autenticados na Tailnet privada.
- **Sem Implicação de Funnel ou Internet Pública**:
  A regra de ACL se restringe estritamente aos nós membros da Tailnet privada. **NÃO IMPLICA NEM AUTORIZA exposição pública via Tailscale Funnel**.
- **Portão Obrigatório Pré-Serve**:
  A aplicação do Tailscale Serve permanece **RIGOROSAMENTE BLOQUEADA** até que o provisionamento prévio de credenciais do Owner no Secrets Manager seja concluído e a validação do primeiro login real seja realizada com sucesso.

---

## 4. Estrutura Canônica Final dos 7 Mounts do Tailscale Serve

Os **7 Mounts Canônicos** consolidados e sem duplicatas para aplicação futura sob autorização formal no nó ORQ1 são:

1. `tailscale serve --bg --https=443 --set-path=/auth/login http://127.0.0.1:18080/auth/login`
2. `tailscale serve --bg --https=443 --set-path=/auth/google http://127.0.0.1:18080/auth/google`
3. `tailscale serve --bg --https=443 --set-path=/auth/logout http://127.0.0.1:18080/auth/logout`
4. `tailscale serve --bg --https=443 --set-path=/ws http://127.0.0.1:18080/ws`
5. `tailscale serve --bg --https=443 --set-path=/api http://127.0.0.1:18080/api`
6. `tailscale serve --bg --https=443 --set-path=/uploads http://127.0.0.1:18080/uploads`
7. `tailscale serve --bg --https=443 --set-path=/ http://127.0.0.1:13100/`

---

## 5. Estado Atual do Bloqueio (Current Blocker)

- **STATUS EXATO DO BLOQUEIO**: **`BLOCKED_ON_CREDENTIALS`**
- **Justificativa**: Tanto o frontend quanto o backend no ORQ1 estão 100% saudáveis e operacionais. O sistema permanece bloqueado **exclusivamente aguardando o provisionamento prévio das credenciais do Owner no AWS Secrets Manager** e a validação do primeiro login real.

---

## 6. Declaração de FILES_LOCKED

- `multica-auth-work/server/internal/middleware/auth.go` (LOCKED — ORQ-17 Auth Hardening V2)
- `multica-auth-work/server/internal/middleware/auth_test.go` (LOCKED — ORQ-17 Tests)

---

## 7. Veredito Final
- **STATUS: CANONICAL INDEX UPDATED WITH OWNER ACL EVIDENCE — BLOCKED_ON_CREDENTIALS**
- **Documento Gravado**: `.deploy-control/p0/evidence/orq17-canonical-evidence-supersession-index.md`
- *Operação 100% Mecânica e Local. Zero chamadas de rede, API ou mutações de quadro.*
