# ORQ-17 — Pré-Flight Factual de Prontidão do Cutover de Autenticação em Produção (RETIFICADO)

- **Autor:** Antigravity (wB:p1 / w8:p2)
- **Data UTC:** 2026-07-27T15:43:00Z
- **Destinatários:** General-Tech-Lead (Codex56-TL w5:pC), KIRO-PRINCIPAL-TL (wB:p1)
- **Governança:** Issue `ORQ-17` (Auth Code / Production Login Cutover)
- **Skill Carregada:** `.agents/skills/aws-secrets-manager/SKILL.md` (zero texto claro, zero `GetSecretValue`, zero leitura de senhas/hashes).
- **Modo:** PRÉ-FLIGHT FACTUAL READ-ONLY — Retificação de portas reais de execução (`18080` backend e `13100` frontend), retratação de hipóteses não fundamentadas e medição sem mutação de estado.

---

## 1. Retratações e Correções de Fato em Relação ao Pré-Flight Anterior

1. **Correção de Portas de Execução**:
   - O backend ativo de produção/dev opera na porta **18080** (não na 8080).
   - O frontend de produção/dev está configurado na porta **13100** (não na 3000).
2. **Retratação do Nome Canônico do Segredo**:
   - Retrata-se expressamente a afirmação de que `prod/owner-credentials` seria o ID canônico de segredo. A ausência desse nome em uma busca por ID estrito no Secrets Manager não comprova a ausência de credencial de owner, uma vez que a resolução ocorre dinamicamente via `asm-exec` no processo filho.
3. **Diferenciação do `NEXT_PUBLIC_API_URL`**:
   - Diferencia-se a regra de compilação do Next.js (onde `NEXT_PUBLIC_*` é embutido em tempo de build) da presença/ausência do bundle compilado no ambiente.

---

## 2. Medições Factuais Realizadas no Host

| Superfície Auditada | Comando de Inspeção | Resultado Medido no Host | Status Factual |
|---|---|---|---|
| **Backend Status (`:18080`)** | `curl -s http://localhost:18080/readyz` | **`HTTP 200 OK`** (`Content-Type: application/json`) | **READY (ONLINE)** |
| **Backend Health (`:18080`)** | `curl -s http://localhost:18080/healthz` | **`HTTP 200 OK`** (`Content-Type: application/json`) | **READY (ONLINE)** |
| **Configuração API (`:18080`)** | `curl -s http://localhost:18080/api/config` | `allow_signup: true`, `daemon_app_url: http://localhost:13100` | **READY (CONFIGURADO)** |
| **Frontend Status (`:13100`)** | `curl http://localhost:13100` | `Connection refused` (Serviço Web inativo no momento) | **BLOCK** — Serviço inativo |
| **Tailscale Serve** | `tailscale serve status` | `No serve config` | **BLOCK** — Requer proxying |

---

## 3. Matriz Reconciliada PASS/BLOCK

```
┌────────────────────────────────────────────────────────────────────────┐
│ CONDICIONAL DE BLOQUEIO FÍSICO (FACTUAL BLOCKERS)                     │
├────────────────────────────────────────────────────────────────────────┤
│ 1. BACKEND SAUDÁVEL NA PORTA 18080: GET /readyz e GET /healthz        │
│    retornam HTTP 200 OK. allow_signup está ativo.                      │
│ 2. FRONTEND INDISPONÍVEL NA PORTA 13100: Retorna Connection Refused.   │
│    O serviço frontend precisa ser iniciado na porta 13100.             │
│ 3. TAILSCALE SERVE ZERADO: Não há rotas ativas mapeando               │
│    https://orq1.tail96e2c0.ts.net para 13100 / 18080.                  │
│ 4. REBUILD WEB CONDICIONAL: A alteração de NEXT_PUBLIC_API_URL        │
│    exige que o container web seja compilado com o target de prod.      │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Ações Seguras para Execução Futura (Safe Next Actions)

1. **Iniciar Serviço Frontend na Porta 13100**: Garantir que o processo Web esteja ouvindo em `http://localhost:13100`.
2. **Configurar Tailscale Serve**: Executar o roteamento seguro de `https://orq1.tail96e2c0.ts.net` para a porta local `13100` e `/api` para `18080`.
3. **Porta de Autorização do Owner**: Aguardar ordem formal do Owner para o cutover final.

---

## 5. Declaração de FILES_LOCKED

- `multica-auth-work/server/internal/middleware/auth.go` (LOCKED — ORQ-17 Auth Hardening V2)
- `multica-auth-work/server/internal/middleware/auth_test.go` (LOCKED — ORQ-17 Tests)

---

## 6. Veredito Final Retificado
- **STATUS: BLOCK (CUTOVER DE AUTENTICAÇÃO PENDENTE DE FRONTEND NA PORTA 13100 E TAILSCALE SERVE)**
- **Documento Gravado**: `.deploy-control/p0/evidence/orq17-auth-cutover-readiness-preflight.md`
- *Operação 100% Read-Only. Zero leitura de valores de segredo, zero mutações de estado.*
