# ORQ-17 — Reconciliação de Contagens de Banco de Dados e Análise de Causa Raiz (RCA)

- **Autor:** Antigravity (wB:p1 / w8:p2)
- **Data/Hora UTC:** `2026-07-27T17:22:40Z`
- **Destinatários:** General-Tech-Lead (Codex56-TL w5:pC), KIRO-PRINCIPAL-TL (wB:p1), Agente `A8`
- **Governança:** Issue `ORQ-17` (DB Counts Reconciliation RCA & Preventive Gate)
- **Skill Carregada:** `.agents/skills/aws-secrets-manager/SKILL.md` (zero texto claro, zero `GetSecretValue`).
- **Modo:** AUDITORIA E RECONCILIAÇÃO READ-ONLY — Zero mutação no banco de dados, API, segredos ou quadro.

---

## 1. Identificação da Evidência Anterior e Discrepância

- **Artefato Anterior Auditado**: `.deploy-control/p0/evidence/orq17-stage3-login-readiness.md`
- **Timestamp Anterior Registrado**: `2026-07-27T16:02:00Z`
- **Reclamação Anterior**:
  - `user`: `0`
  - `user_password_credential`: `0`
  - `member`: `0`

---

## 2. Medição Factual Exata no Ambiente Vivo do ORQ1

- **Host Alvo**: `ORQ1` (`100.118.244.61` / `ip-172-31-18-217.sa-east-1.compute.internal`)
- **Container do Banco**: `2a4a84897363` (`multica-dev-transition-postgres-1` running `pgvector/pgvector:pg17`)
- **Tupla DB / Usuário**: `multica_transition` / `multica_transition`
- **Timestamp UTC da Consulta**: `2026-07-27T17:22:35Z`
- **Comando de Consulta Exato Executado**:
  ```sql
  SELECT 
    (SELECT count(*) FROM "user") as user_count, 
    (SELECT count(*) FROM user_password_credential) as cred_count, 
    (SELECT count(*) FROM member) as member_count;
  ```
- **Resultado Factual Medido**:
  - **`user_count`**: **`1`** (Um usuário ativo registrado no banco)
  - **`cred_count` (`user_password_credential`)**: **`0`** (Zero credenciais de senha cadastradas)
  - **`member_count` (`member`)**: **`1`** (Um membro de workspace registrado)

---

## 3. Retratação e Correção Factual (RETRACTED WITH CORRECTION)

1. **Retratação da Alegação `user=0` e `member=0`**:
   - A alegação anterior de `user=0` e `member=0` no relatório de prontidão é formalmente marcada como **RETRACTED** (RETRATADA) por erro de amostragem no nome do banco de dados (consulta anterior efetuada em DB default `multica` em vez de `multica_transition`).
2. **Reafirmação Factual de `credential=0`**:
   - A contagem de `user_password_credential = 0` é **REAFIAMADA COMO FACTUALMENTE CORRETA**. Nenhuma senha ou hash de credencial existe no banco de dados.
3. **Impacto no Veredito de Prontidão de Login**:
   - Como `user_password_credential = 0`, a tentativa de login via senha permanece **IMPOSSÍVEL** (`HTTP 401 Unauthorized`), mantendo o veredito estrito de **`CREDENTIAL_PROVISIONING_REQUIRED`**. O login humano não é executável sem o provisionamento prévio da credencial de senha.

---

## 4. Portão Preventivo Obrigatório (Preventive Gate Rule)

Para qualquer auditoria futura de banco de dados ou prontidão de serviço, a evidência deve obrigatoriamente incluir e satisfazer a seguinte tripla:

```
┌────────────────────────────────────────────────────────────────────────┐
│ REQUISITOS DO PORTÃO PREVENTIVO DE AUDITORIA DE DB                      │
├────────────────────────────────────────────────────────────────────────┤
│ 1. TUPLA EXATA: Host / Container ID / Nome DB / Usuário DB            │
│    Exemplo: ORQ1 / 2a4a84897363 / multica_transition / multica_trans. │
│ 2. TRÊS CONTAGENS PADRONIZADAS:                                       │
│    - count(*) de "user"                                               │
│    - count(*) de user_password_credential                             │
│    - count(*) de member                                               │
│ 3. TIMESTAMP UTC EXATO DA CONSULTA: Ex: 2026-07-27T17:22:35Z          │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 5. Declaração de FILES_LOCKED

- `multica-auth-work/server/internal/middleware/auth.go` (LOCKED — ORQ-17 Auth Hardening V2)
- `multica-auth-work/server/internal/middleware/auth_test.go` (LOCKED — ORQ-17 Tests)

---

## 6. Veredito Final
- **STATUS: RECONCILIATION & RCA COMPLETED — CREDENTIAL_PROVISIONING_REQUIRED REAFFIRMED**
- **Documento Gravado**: `.deploy-control/p0/evidence/orq17-db-counts-reconciliation-rca.md`
- *Operação 100% Read-Only. Nenhuma mutação efetuada.*
