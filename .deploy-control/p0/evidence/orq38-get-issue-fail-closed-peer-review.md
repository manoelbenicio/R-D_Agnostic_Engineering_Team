# ORQ-38 — Peer Review de Governança: Plano de Testes Fail-Closed para `GET issue`

- **Autor do Peer Review:** Antigravity (wB:p1 / w8:p2)
- **Documento Auditado:** `.deploy-control/p0/evidence/orq38-get-issue-fail-closed-test-plan.md`
- **Data UTC:** 2026-07-27T15:17:00Z
- **Destinatários:** General-Tech-Lead (Codex56-TL w5:pC), KIRO-PRINCIPAL-TL (wB:p1)
- **Governança:** Issue `ORQ-38`
- **Modo:** PEER REVIEW ADVERSARIAL READ-ONLY — Zero edição de código, zero implementação, zero compilação/testes rodados, zero mutação no banco/quadro.

---

## 1. Veredito Final
- **VEREDITO: PASS (PLANO DE TESTES FAIL-CLOSED APROVADO PARA IMPLEMENTAÇÃO)**
- O documento `.deploy-control/p0/evidence/orq38-get-issue-fail-closed-test-plan.md` formaliza com precisão matemática os testes de regressão para a rota `GET /api/issues/{id}`, garante o comportamento de anti-enumeração cross-tenant (`HTTP 404`), preserva o escopo obrigatorio de workspace, proíbe testes "falso-positivos" sustainers de skip e isola limpidamente os 6 arquivos bloqueados (`FILES_LOCKED`).

---

## 2. Auditoria Adversarial dos Critérios de Governança e Segurança

### 2.1 Tratamanto Fail-Closed de Status HTTP e Anti-Enumeração Cross-Tenant
- **Anti-Enumeração Confirmada (S03 vs S05)**: A tentativa de consultar uma issue pertencente ao Workspace B por um usuário autenticado no Workspace A retorna estritamente **`HTTP 404 Not Found`** com payload idêntico a uma consulta por UUID inexistente (`{"error":"issue not found"}`). Não há vazamento de metadados nem indicação de existência.
- **Controle Positivo Pareado (S04)**: A mesma issue consultada sob o Workspace B correto retorna `HTTP 200 OK`, provando que o 404 em S03 não é um falso-positivo de ausência de dados.
- **Tratamento de Workspace Ausente/Inválido (S01/S02)**: Retorna **`HTTP 400 Bad Request`** via middleware `RequireWorkspaceMember` antes de atingir o handler.

### 2.2 Contrato Obrigatório de Identidade Completa (`id` + `identifier` + `number`)
- O cliente HTTP e a ferramenta CLI validam conjuntamente via `requireIssueIdentity` (C01-C13):
  - `id`: string UUID não-vazia.
  - `identifier`: string de projeto/issue não-vazia (ex: `ORQ-38`).
  - `number`: inteiro JSON positivo (ex: `38`).
- Rejeita valores nulos, strings vazias, números negativos, zero ou fracionários.
- **Prevenção de TOCTOU (C11)**: Confirma que o `id` da resposta de leitura final corresponde exatamente ao UUID resolvido na primeira consulta.

### 2.3 Impacto no Cliente CLI e Preservação de Código Produtivo
- O cliente HTTP adiciona a operação opt-in `GetJSONExpectedStatus`, exigindo estritamente `HTTP 200 OK` para consultas de issue, sem alterar o comportamento global do método legado `GetJSON`.
- **Zero Alteração em Produção do Servidor**: `router.go`, `workspace.go`, `issue.go` e `issue.sql` permanecem 100% intactos (byte a byte).

### 2.4 Proteção Contra Falsos-Positivos (False-Green Prevention)
- **Gate G4 (Log JSON Nominal)**: Como a suíte Go pode retornar código de saída zero quando testes são pulados por ausência de banco Postgres, o Gate G4 exige a captura de `go test -json` com o evento explícito `"Action":"pass"` para a função de teste `TestGetIssueWorkspaceFailClosed`. A ausência do evento é classificada como **FAIL/SKIP**, impedindo falso-verde.

### 2.5 Reserva Estrita de `FILES_LOCKED` (6 Arquivos)
- **Modificados (3)**: `internal/cli/client.go`, `cmd/multica/cmd_id_resolver.go`, `cmd/multica/cmd_issue.go`.
- **Novos Testes (3)**: `client_get_expected_status_test.go`, `cmd_issue_get_contract_test.go`, `issue_get_workspace_scope_test.go`.
- **Não-Sobreposição**: Sem colisão com `file.go`/`file_test.go` (`ORQ-26`) ou `auth.go` (`ORQ-17`).

---

## 3. Veredito Final
- **STATUS: PASS (PLANO DE TESTES FAIL-CLOSED APROVADO)**
- **Documento Gravado**: `.deploy-control/p0/evidence/orq38-get-issue-fail-closed-peer-review.md`
- *Operação 100% Read-Only. Nenhum código editado, nenhuma compilação ou teste executado.*
