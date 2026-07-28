# ORQ-38 — Auditoria do contrato de `GET /api/issues/{uuid}` sem `workspace_id` (READ-ONLY)

Auditor: Codex56#A (`w7:p3`) · UTC 2026-07-27T13:41Z · ORQ-38 (`fd5c4d55…`)
Modo: somente leitura. Nenhum patch, nenhuma chamada de API de escrita, nenhum deploy/restart.

## 0. Veredito: **o servidor NÃO devolve objeto nulo — ele já falha fechado com 400.**

A premissa que eu mesmo levantei no K05 ("`GET /api/issues/{uuid}` sem `workspace_id` devolve corpo com
campos nulos em vez de 404") está **CORRIGIDA por medição**. Medido agora, no mesmo backend:

```text
GET /api/issues/666f1ead-…                              -> HTTP 400  {"error":"workspace_id or workspace_slug is required"}
GET /api/issues/666f1ead-…?workspace_id=20fce817-…      -> HTTP 200
GET /api/issues/00000000-0000-0000-0000-000000000000?workspace_id=…  -> HTTP 404 {"error":"issue not found"}
GET /api/issues/666f1ead-…?workspace_id=not-a-uuid      -> HTTP 400  {"error":"invalid workspace_id"}
```

Os quatro casos são fail-closed e distinguem corretamente **falta de escopo (400)**, **escopo inválido
(400)** e **inexistência dentro do escopo (404)**.

## 1. Contrato mapeado (handler → query)

```text
router.go:734                r.Get("/", h.GetIssue)                     // sob /api/issues/{id}
handler/issue.go:1583-1588   GetIssue: id := chi.URLParam(r,"id"); issue, ok := h.loadIssueForUser(w,r,id); if !ok { return }
handler/handler.go:568-606   loadIssueForUser:
   569  requireUserID(w,r)                              -> 401 se sem usuário
   573  workspaceID := h.resolveWorkspaceID(r)
   574-577  if workspaceID == "" { writeError(400, "workspace_id is required"); return false }   <-- FALHA FECHADA
   582  resolveIssueByIdentifier(ctx, id, workspaceID)  -> aceita "ORQ-41" além de UUID
   586-591  ParseUUID(id) falhou           -> 404 "issue not found"
   592-596  ParseUUID(workspaceID) falhou  -> 400 "invalid workspace_id"
   597-604  Queries.GetIssueInWorkspace(ID, WorkspaceID); err != nil -> 404 "issue not found"
handler/handler.go:445-447   resolveWorkspaceID -> middleware.ResolveWorkspaceIDFromRequest(r, h.Queries)
                             (aceita workspace_id OU workspace_slug; a mensagem real de erro cita ambos)
```

A query é **escopada por workspace** (`GetIssueInWorkspace(ID, WorkspaceID)`), não por UUID isolado —
isso é a propriedade de segurança correta: um UUID válido de outro workspace resulta em 404, não em
leitura cruzada.

## 2. Então o que causou o relato falso de identidade?

**O cliente/verificador, não o servidor.** A minha própria sonda no K05 foi:

```text
curl -s "http://127.0.0.1:18080/api/issues/$U" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('id'), d.get('identifier'), d.get('number'))"
```

Três defeitos de verificação, todos do lado do cliente:

1. **`curl -s` sem `-w '%{http_code}'` e sem `--fail`**: o 400 foi descartado silenciosamente.
2. **`d.get('campo')` em um corpo de erro**: `{"error": …}` não tem `id`/`identifier`/`number`, então
   `dict.get` devolveu `None` — foi isso que apareceu como "objeto de forma nula". O servidor nunca
   emitiu um objeto de issue com campos nulos.
3. **Ausência de `workspace_id` na URL**, quando o contrato o exige.

Contribuição para o incidente de identidade: um verificador nesse formato **não distingue** "card não
existe" de "minha requisição está malformada". Foi o que produziu, no meu K05, a conclusão errada de
que "o card não existe" e alimentou o relato de identificadores divergentes. A tabela de dados estava
sã — a auditoria de unicidade que rodei depois (Counter de `number` e `identifier`) mostrou zero
duplicata.

## 3. Onde ainda há risco real (fail-closed que falta)

Nada a corrigir no `GET` de issue. O que falta é **contrato de verificação** e **cobertura de teste**:

| # | Lacuna | Evidência | Proposta |
|---|---|---|---|
| 1 | Nenhum teste de handler cobre as mensagens/códigos deste caminho | `grep -rl "workspace_id or workspace_slug is required\|workspace_id is required" internal/handler/*_test.go` → **vazio** | 4 testes de regressão (§4) |
| 2 | Verificadores da frota usam `curl -s \| python d.get(...)` | K05/K07 desta mesma lane | padrão obrigatório de verificação (§5) |
| 3 | Mensagem do handler (`"workspace_id is required"`) difere da mensagem real observada (`"workspace_id or workspace_slug is required"`) | handler.go:575 vs corpo medido | alinhar texto ou documentar que a mensagem vem do middleware; sem isso, grep por mensagem em teste falha |

## 4. Testes de regressão propostos (não escritos, não executados)

Em `internal/handler/issue_get_workspace_scope_test.go`:

1. `TestGetIssueWithoutWorkspaceReturns400` — sem `workspace_id`/`workspace_slug`: espera **400** e
   corpo com chave `error`; **assert explícito de que o corpo NÃO contém `id`/`identifier`/`number`**.
2. `TestGetIssueWithInvalidWorkspaceReturns400` — `workspace_id=not-a-uuid`: **400** `invalid workspace_id`.
3. `TestGetIssueForeignWorkspaceReturns404` — UUID real de outro workspace: **404**, e nenhum campo do
   issue no corpo (prova de não-vazamento cross-tenant).
4. `TestGetIssueUnknownUUIDReturns404` — UUID sintático válido inexistente: **404**.
5. `TestGetIssueByIdentifierRequiresWorkspace` — `ORQ-41` sem workspace: **400**, não 404, porque o
   identificador só é resolvível dentro de um escopo (`handler.go:582`).

Todos herméticos, sem banco compartilhado, `-count=1`, sem skip global.

## 5. Padrão de verificação obrigatório para a frota (o fail-closed que realmente falta)

```bash
# leitura de card: falha fechada por status, e só então parse
resp=$(curl -sS -w '\n%{http_code}' --max-time 10 \
  "http://127.0.0.1:18080/api/issues/$UUID?workspace_id=$WS")
code=$(printf '%s' "$resp" | tail -1); body=$(printf '%s' "$resp" | sed '$d')
[ "$code" = "200" ] || { echo "VERIFY-FAIL http=$code body=$body"; exit 1; }
printf '%s' "$body" | python3 -c '
import json,sys
d=json.load(sys.stdin)
for k in ("id","identifier","number"):
    if d.get(k) in (None,""): raise SystemExit("VERIFY-FAIL campo ausente: "+k)
print(d["identifier"], d["number"], d["id"])'
```

Regras: (a) sempre enviar `workspace_id`; (b) tratar qualquer código ≠ 200 como **falha de verificação**,
nunca como "não existe"; (c) exigir presença dos três campos antes de afirmar identidade; (d) provar
unicidade por agregação (`Counter` de `number` e `identifier`) e não por leitura individual.

## 6. Não-alegações

- Não patchei, não chamei API de escrita, não rodei teste, build, deploy ou restart nesta auditoria.
- **Retrato** a minha afirmação anterior de "objeto de forma nula": era artefato do meu cliente. O
  servidor responde 400. Mantenho o restante do achado do K05 (a leitura sem `workspace_id` engana um
  verificador ingênuo) — apenas a atribuição de causa muda de servidor para cliente.
- Não testei `workspace_slug` como alternativa nem o caminho autenticado por sessão real: o backend
  deste ambiente resolve usuário implicitamente (risco separado já registrado na auditoria T4).
- Não avaliei os demais endpoints de issue (`/usage`, `/comments`, `/runs`) quanto ao mesmo contrato.
- Nenhum valor de segredo lido.
