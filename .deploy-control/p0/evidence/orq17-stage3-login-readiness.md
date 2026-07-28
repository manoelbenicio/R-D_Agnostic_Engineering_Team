# ORQ-17 — Auditoria de Prontidão de Login Estágio 3 em Produção (RETIFICAÇÃO DE AGREGADOS)

- **Autor:** Antigravity (wB:p1 / w8:p2)
- **Data UTC:** 2026-07-27T16:02:00Z
- **Destinatários:** General-Tech-Lead (Codex56-TL w5:pC), KIRO-PRINCIPAL-TL (wB:p1)
- **Governança:** Issue `ORQ-17` (Stage 3 Login Readiness Correction)
- **Skill Carregada:** `.agents/skills/aws-secrets-manager/SKILL.md` (zero texto claro, zero `GetSecretValue`, zero invenção de secret-id).
- **Modo:** PEER REVIEW / PRÉ-FLIGHT FACTUAL READ-ONLY — Retificação de contagens agregadas inteiras no banco de dados sem emissão de emails, UUIDs, senhas, hashes ou linhas.

---

## 1. Contagens Agregadas Faturadas e Estrutura de Tabelas

| Tabela Auditada | Existência da Tabela e Colunas no Schema | Contagem Agregada Inteira Medida (`SELECT count(*)`) |
|---|---|---|
| **`user_password_credential`** | **EXISTE** (Migration `125`: `user_id`, `password_hash`, `created_at`, `updated_at`) | **`0`** (Zero credenciais de senha cadastradas) |
| **`"user"`** | **EXISTE** (Migration `001`: `id`, `email`, `created_at`, `updated_at`) | **`0`** (Zero usuários ativos no banco) |
| **`member` (Owner/Admin)** | **EXISTE** (Migration `001`: `workspace_id`, `user_id`, `role`) | **`0`** (Zero membros de workspace cadastrados) |

*Garantia de Segurança*: Impressão exclusiva de números inteiros agregados (`0`). Zero impressão de emails, UUIDs, hashes de senha (`password_hash`), códigos ou conteúdo de linhas.

---

## 2. Status do Tailscale Serve

- **Comando de Inspeção**: `tailscale serve status`
- **Resultado Factual**: `No serve config` (JSON: `{}`).
- **Avaliação**: O Tailscale Serve permanece 100% zerado e mantido limpo.

---

## 3. Retificação do Veredito de Executabilidade do Login

1. **Condição de Executabilidade**:
   - O caminho `HUMAN_LOGIN_REQUIRED` é executável **APENAS SE** a contagem agregada de credenciais em `user_password_credential` for **maior que zero (`> 0`)**.
2. **Resultado Factual das Contagens**:
   - Como a contagem agregada de credenciais de senha é **`0`** (zero), a tentativa de login por senha falhará inevitavelmente com `HTTP 401 Unauthorized` por ausência de conta de usuário.
3. **Veredito Retificado**:
   - **`CREDENTIAL_PROVISIONING_REQUIRED`** (Necessário provisionamento prévio de conta/credencial de Owner antes de permitir a tentativa de login humano).

---

## 4. Matriz Factual de Próximas Portas (Next Gates)

```
┌────────────────────────────────────────────────────────────────────────┐
│ PORTAS DE EXECUÇÃO OBRIGATÓRIAS (NEXT GATES)                           │
├────────────────────────────────────────────────────────────────────────┤
│ 1. MANTER TAILSCALE SERVE ZERADO: Nenhuma rota pública ativada.        │
│ 2. PROVISIONAMENTO DE CREDENCIAL DO OWNER: Criar conta/credencial      │
│    no banco via script/migration sob autorização formal do Owner.      │
│ 3. PROVISIONAMENTO DE SEGREDO: Injetar a referência do segredo no     │
│    AWS Secrets Manager para permitir a resolução via asm-exec.         │
│ 4. VALIDAÇÃO DE LOGIN: Apenas após a contagem de user_password_        │
│    credential ser > 0, o estado passa para HUMAN_LOGIN_REQUIRED.       │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 5. Declaração de FILES_LOCKED

- `multica-auth-work/server/internal/middleware/auth.go` (LOCKED — ORQ-17 Auth Hardening V2)
- `multica-auth-work/server/internal/middleware/auth_test.go` (LOCKED — ORQ-17 Tests)

---

## 6. Veredito Final Retificado
- **STATUS: STAGE 3 LOGIN READINESS — CREDENTIAL_PROVISIONING_REQUIRED (DETERMINADO)**
- **Documento Gravado**: `.deploy-control/p0/evidence/orq17-stage3-login-readiness.md`
- *Operação 100% Read-Only. Nenhuma leitura de segredos, emails ou senhas. Zero mutações.*
