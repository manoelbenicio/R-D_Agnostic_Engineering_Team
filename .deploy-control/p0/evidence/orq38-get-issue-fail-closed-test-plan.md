# ORQ-38 — Plano read-only de testes fail-closed para `GET issue`

Data: 2026-07-27T15:05Z  
Modo desta entrega: **READ-ONLY sobre produto, testes, runtime e board**. Somente este plano é criado.  
Base congelada: `0cb8aebb5aff79cb430b3740d22fadc53c0116fd`.

## 1. Decisão executiva

O servidor já possui o comportamento de segurança correto e **não requer patch funcional**:

- `GET /api/issues/{id}` está sob `middleware.RequireWorkspaceMember`;
- ausência de `workspace_id` e `workspace_slug` termina em `400` com `workspace_id or workspace_slug is required` antes do handler;
- `workspace_id` não UUID termina em `400` com `invalid workspace_id`;
- `loadIssueForUser` consulta `GetIssueInWorkspace(ID, WorkspaceID)`;
- UUID existente em outro workspace e UUID inexistente têm a mesma forma externa: `404 issue not found`;
- sucesso retorna `200` e `IssueResponse` contém `id`, `identifier` e `number`.

A lacuna está em dois lugares: cobertura de regressão desse caminho e verificação cliente. Hoje `APIClient.GetJSON` rejeita apenas status `>=400`; portanto aceita qualquer `2xx`, e `fetchIssueRef`/`runIssueGet` não validam conjuntamente `id`, `identifier` e `number`. O plano preserva o handler e endurece somente o contrato de leitura do cliente.

## 2. Evidência de contrato atual

| Superfície | Comportamento observado no código |
|---|---|
| `server/cmd/server/router.go:725-735` | grupo de issues aplica `RequireWorkspaceMember` antes de `h.GetIssue` |
| `server/internal/middleware/workspace.go:195-233` | ausente → 400; UUID inválido → 400; não membro/workspace desconhecido → 404 |
| `server/internal/handler/issue.go:1583-1619` | `GetIssue` só serializa após `loadIssueForUser` retornar sucesso |
| `server/internal/handler/handler.go:568-606` | exige workspace e usa `GetIssueInWorkspace` para UUID |
| `server/internal/cli/client.go:208-229` | `GetJSON` considera erro somente status `>=400`; não exige 200 |
| `server/cmd/multica/cmd_id_resolver.go:130-157` | resolução aceita objeto sem provar os três campos obrigatórios |
| `server/cmd/multica/cmd_issue.go:611-658` | leitura final imprime JSON/tabela sem validar identidade completa |

A auditoria anterior `.deploy-control/p0/evidence/orq38-issue-get-workspace-contract-audit.md` já retratou a hipótese de “objeto nulo”: o corpo era um erro 400 interpretado por um verificador ingênuo. Este plano não reabre essa hipótese.

## 3. Invariantes de segurança

1. **Escopo obrigatório:** nenhuma leitura de issue pode alcançar o handler sem workspace resolvido e membership validada.
2. **Anti-enumeração:** uma issue real de B consultada sob A deve ser indistinguível de UUID inexistente em A: `404 {"error":"issue not found"}`.
3. **Sem vazamento:** toda resposta não-200 deve omitir ao menos `id`, `identifier`, `number`, `title`, `description` e `workspace_id`.
4. **Status exato:** o verificador cliente aceita somente HTTP `200`; `201`, `202`, `204`, redirecionamento não seguido e qualquer `4xx/5xx` falham.
5. **Identidade completa:** `id` e `identifier` são strings não vazias; `number` é número JSON inteiro e positivo.
6. **Consistência da segunda leitura:** quando o resolvedor produziu um UUID, o `id` da leitura final deve ser exatamente o UUID resolvido.
7. **Falha silenciosamente segura:** em erro, não imprimir tabela/JSON de sucesso nem converter campo ausente em valor vazio/nulo aceitável.
8. **Sem live:** fixtures usam `httptest` e banco de teste dedicado; nenhum gate consulta backend, board ou banco de produção.

## 4. Arquitetura dos testes

### 4.1 Servidor

Criar um teste novo no pacote `internal/handler` que envolva `testHandler.GetIssue` com o middleware real `RequireWorkspaceMember`. Não chamar apenas `GetIssue` para os casos de workspace ausente/inválido, pois isso ignoraria a precedência real do router.

A fixture cross-tenant deve criar dois workspaces únicos A/B e uma issue real em B. O mesmo usuário de teste será membro dos dois workspaces para que o 404 sob A prove o predicado `issue.workspace_id = request.workspace_id`, não uma falha anterior de membership. Cada linha criada terá UUID próprio e `t.Cleanup` explícito em ordem referencial inversa.

Para cada resposta, decodificar primeiro como `map[string]json.RawMessage` e aplicar o assert comum `assertNoIssueLeak`; nunca usar getters tolerantes que transformem ausência em zero value.

### 4.2 Cliente/verificador

Adicionar ao cliente HTTP uma operação **nova e opt-in**, por exemplo:

```go
GetJSONExpectedStatus(ctx, path, http.StatusOK, &out)
```

Contrato:

- `>=400`: manter `*HTTPError` e semântica atual;
- status `<400`, mas diferente do esperado: erro tipado/estruturado com `expected` e `actual`, sem imprimir o corpo como sucesso;
- status esperado: decodificar exatamente um objeto JSON e propagar erro de decode;
- não alterar a semântica global de `GetJSON`, evitando regressão em outros comandos.

Centralizar em `cmd_id_resolver.go` um helper puro que valide a identidade:

```text
requireIssueIdentity(map) -> {ID, Identifier, Number} ou erro
```

O helper rejeita campo ausente, `null`, string vazia/espaços, tipo incorreto, `number` fracionário, zero ou negativo. `fetchIssueRef` e a segunda leitura de `runIssueGet` usam status esperado 200 e esse mesmo helper. A segunda leitura também compara `identity.ID == issueRef.ID`.

## 5. Matriz fail-closed — servidor

| ID | Requisição/fixture | Resultado obrigatório | Assert de segurança |
|---|---|---|---|
| S01 | issue válida, sem query/header de workspace | `400`, erro exato `workspace_id or workspace_slug is required` | nenhum campo de issue; middleware encerra a cadeia |
| S02 | issue válida, `workspace_id=not-a-uuid` | `400`, erro exato `invalid workspace_id` | nenhum campo de issue |
| S03 | issue real em B, request autenticado sob A | `404`, erro exato `issue not found` | nenhum dado de B no status, corpo ou headers |
| S04 | controle pareado de S03: mesma issue sob B | `200` | `id`, `identifier`, `number` iguais à fixture; prova que S03 não é falso positivo |
| S05 | UUID sintaticamente válido e inexistente sob A | `404`, mesma forma de S03 | corpo normalizado idêntico ao cross-tenant |
| S06 | issue real em A por UUID | `200` | identidade completa e workspace correto |
| S07 | identifier real, sem workspace | `400`, mensagem do middleware | não degradar para 404 nem resolver fora de escopo |
| S08 | identifier real com workspace A | `200` | identidade igual à leitura por UUID |

Não adicionar expectativa de `403` para cross-tenant: o contrato deliberado é anti-enumeração por `404`.

## 6. Matriz fail-closed — cliente

Usar `httptest.Server`, duas leituras observáveis (resolução e leitura final), e capturar stdout. Toda falha deve retornar erro e stdout vazio.

| ID | Resposta simulada | Resultado obrigatório |
|---|---|---|
| C01 | ambas as leituras `200` com `id`, `identifier`, `number` válidos | sucesso; saída contém identidade correta |
| C02 | `201` com objeto aparentemente válido | falha por status, sem saída de sucesso |
| C03 | `204` sem corpo | falha por status antes de decode |
| C04 | `400`, `404` ou `500` com JSON de erro | `*HTTPError` preservado; sem saída de sucesso |
| C05 | `200` com JSON inválido, array ou múltiplos objetos | falha de decode/shape |
| C06 | `200` sem `id`, sem `identifier` ou sem `number` (subtestes) | falha indicando o campo, sem inferir zero value |
| C07 | `200` com qualquer campo `null` | falha |
| C08 | `200` com `id`/`identifier` vazio ou só espaços | falha |
| C09 | `200` com tipos errados: id numérico, identifier numérico, number string/bool | falha |
| C10 | `200` com number `0`, negativo, fracionário, NaN/overflow quando aplicável | falha |
| C11 | resolução válida; segunda leitura devolve outro `id` | falha de identidade/TOCTOU |
| C12 | servidor registra headers das duas leituras | ambas recebem `X-Workspace-ID` igual ao workspace do cliente |
| C13 | primeira leitura malformada, segunda preparada para sucesso | segunda leitura não ocorre; fail-fast |

## 7. `FILES_LOCKED`

Um único owner deve reservar o conjunto inteiro; não dividir cliente HTTP e comando CLI entre agentes concorrentes sem congelar primeiro a assinatura do helper.

### Arquivos existentes a modificar futuramente

1. `multica-auth-work/server/internal/cli/client.go`  
   Baseline SHA-256: `10c9dab5280d9a8b315eaf89eaa2f3f45447906ece7578e09997902e613a6166`  
   Única mudança permitida: operação opt-in de status esperado; `GetJSON` existente permanece compatível.
2. `multica-auth-work/server/cmd/multica/cmd_id_resolver.go`  
   Baseline SHA-256: `dc8601d386ae1f0366258ff003f34c127e5e6129c6dee92c1e55e490537298aa`  
   Adicionar validação central de identidade e aplicá-la em `fetchIssueRef`.
3. `multica-auth-work/server/cmd/multica/cmd_issue.go`  
   Baseline SHA-256: `69cdef805114e75a9cfe209a471ebc223bac0c6800ad0fb9a900a68f59da280e`  
   Aplicar status 200, identidade completa e igualdade do ID na leitura final.

### Arquivos novos de teste

4. `multica-auth-work/server/internal/cli/client_get_expected_status_test.go`
5. `multica-auth-work/server/cmd/multica/cmd_issue_get_contract_test.go`
6. `multica-auth-work/server/internal/handler/issue_get_workspace_scope_test.go`

Os três arquivos novos estavam ausentes na base congelada. O `git status` escopado dos seis paths estava vazio no preflight.

### Explicitamente fora do lock

- `server/cmd/server/router.go`
- `server/internal/middleware/workspace.go`
- `server/internal/handler/handler.go`
- `server/internal/handler/issue.go`
- `server/pkg/db/queries/issue.sql` e gerados SQLC
- migrations, frontend/mobile, deploy, workflows e board

Qualquer necessidade de tocar um item fora do lock é **STOP**, revisão do plano e nova autorização; não ampliar o escopo durante a implementação.

## 8. Sequência futura de implementação

1. Confirmar base/hashes e exclusividade dos seis `FILES_LOCKED`.
2. Escrever primeiro os testes do cliente HTTP e do comando; provar RED apenas para `201` e campos inválidos.
3. Implementar a operação opt-in de status esperado sem alterar `GetJSON`.
4. Implementar `requireIssueIdentity`; usar na resolução e leitura final.
5. Escrever o teste de servidor sem mudar produção; todos os casos devem nascer GREEN se o contrato atual estiver preservado.
6. Rodar gates direcionados, race e pacotes completos.
7. Solicitar revisão independente focada em anti-enumeração, falso positivo por skip e ausência de vazamento.

Nenhum passo dessa sequência foi executado nesta entrega.

## 9. Gates verificáveis

### G0 — Preflight e não sobreposição

**PASS somente se:** HEAD/base esperada ou rebase explicitamente revisado; hashes dos três arquivos existentes conferem; três arquivos novos continuam ausentes; nenhum `FILES_LOCKED` pertence a outra wave. Divergência é STOP, nunca overwrite.

### G1 — Escopo do diff

```text
git diff --name-only <base>...HEAD
```

**PASS somente se:** conjunto é subconjunto exato dos seis `FILES_LOCKED`; produção do servidor (`router`, middleware, handler, query) permanece byte a byte intacta.

### G2 — Cliente HTTP: status exato

A partir de `multica-auth-work/server`:

```text
go test ./internal/cli -run '^TestGetJSONExpectedStatus$' -count=1
go test -race ./internal/cli -run '^TestGetJSONExpectedStatus$' -count=1
```

**PASS:** 200 decodifica; 201/204 falham por status; 4xx preserva `HTTPError`; JSON inválido falha; corpo não vira saída de sucesso.

### G3 — Verificador `issue get`

```text
go test ./cmd/multica -run '^(TestRequireIssueIdentity|TestRunIssueGetFailClosed)$' -count=1
go test -race ./cmd/multica -run '^(TestRequireIssueIdentity|TestRunIssueGetFailClosed)$' -count=1
```

**PASS:** todos C01–C13 passam, incluindo workspace nas duas leituras, ID final igual ao resolvido e stdout vazio em toda falha.

### G4 — Servidor workspace/cross-tenant

O package `internal/handler` encerra com código zero quando o PostgreSQL não está disponível; portanto **exit 0 sozinho não vale como evidência**. Capturar `go test -json` e exigir evento `pass` do teste nominal:

```text
set -o pipefail
go test -json ./internal/handler -run '^TestGetIssueWorkspaceFailClosed$' -count=1 | tee /tmp/orq38-handler.json
grep -q '"Action":"pass".*"Test":"TestGetIssueWorkspaceFailClosed"' /tmp/orq38-handler.json
```

**PASS:** teste realmente executado, S01–S08 passam, S03 e S05 têm a mesma forma externa e S04 prova a existência da issue estrangeira. Ausência do evento nominal é FAIL/SKIP, não PASS.

### G5 — Regressão de pacotes afetados

```text
go test ./internal/cli ./cmd/multica -count=1
go test -json ./internal/handler -count=1 | tee /tmp/orq38-handler-full.json
```

**PASS:** pacotes cliente/CLI verdes; handler verde e com testes nominais executados. Nenhum skip por banco indisponível pode sustentar aceite.

### G6 — Qualidade estática

```text
gofmt -d <seis FILES_LOCKED existentes/criados>
go vet ./internal/cli ./cmd/multica ./internal/handler
```

**PASS:** `gofmt -d` vazio e `go vet` zero. Não usar `gofmt -w` no gate de revisão.

### G7 — Revisão independente

Revisor diferente confirma:

- status **exatamente** 200, não apenas `<400` ou `2xx`;
- `id`/`identifier`/`number` validados por presença, tipo e domínio;
- cross-tenant usa issue comprovadamente existente em B e request sob A;
- erro não imprime objeto/tabela de sucesso;
- nenhum teste consulta live, board ou credenciais;
- nenhum arquivo fora de `FILES_LOCKED` mudou.

## 10. Critérios de aceite

- S01–S08 e C01–C13 implementados e nominalmente executados.
- Falta de workspace = 400; workspace UUID inválido = 400; cross-tenant = 404 sem vazamento.
- Controle positivo cross-tenant = 200 com identidade completa.
- Cliente aceita somente HTTP 200 e corpo com os três campos válidos.
- Nenhuma alteração funcional em router/middleware/handler/query/migration.
- Gates G0–G7 com evidência anexada e revisão independente ACCEPT.

## 11. Stop conditions e rollback

STOP imediato para: colisão de lock; necessidade de migration/query/handler; teste que depende de backend live; 404 cross-tenant contendo dados; 201/204 aceito; campo ausente convertido em vazio; handler suite “verde” sem evento de teste executado.

Rollback futuro é somente reverter o commit dos seis arquivos. Não há migration, dado, serviço, deploy ou board para desfazer. Não executar rollback, teste ou implementação a partir deste plano sem task nova e autorização explícita.

## 12. Atestação desta entrega

- Nenhum código de produto ou teste foi editado.
- Nenhum teste, build, formatter, vet, deploy, restart, request HTTP ou acesso ao board foi executado.
- Nenhum status, comentário, assignee ou metadata foi alterado.
- Somente este documento de plano foi criado.
