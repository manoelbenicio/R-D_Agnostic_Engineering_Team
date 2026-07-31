# Kanban Multica via Tailscale — Guia 360° para Owner, TL e Agentes

> Documento operacional consolidado para acesso humano, diagnóstico, uso da CLI/API,
> acompanhamento de tasks e reconciliação de Kanban, Git, testes, deployment e OpenSpec.
>
> Estado verificado em 2026-07-31. Este documento não contém credenciais e não autoriza
> mutações, dispatch, login em nome de terceiros, leitura de secrets ou alteração de serviços.

## 1. Resumo executivo

O endereço canônico do Multica é:

```text
https://orq1.tail96e2c0.ts.net
```

Ele é acessível somente por dispositivos autorizados na Tailnet. Não se deve acrescentar
`:13100` ou `:18080`, usar o IP `100.118.244.61` como origem HTTPS, nem criar túnel SSH para
o uso normal pelo navegador.

Em 2026-07-31 foi observado diretamente no ORQ1:

| Componente | Evidência observada |
|---|---|
| Self Tailscale | online |
| MagicDNS/FQDN | `orq1.tail96e2c0.ts.net` |
| IP Tailscale | `100.118.244.61` |
| Tailscale Serve | ativo, `tailnet only` |
| TLS | certificado validado (`ssl_verify_result=0`) |
| Frontend interno | `127.0.0.1:13100` → HTTP 200 |
| Backend health | `127.0.0.1:18080/healthz` → HTTP 200 |
| Backend readiness | `127.0.0.1:18080/readyz` → HTTP 200 |
| HTTPS principal | `/` → HTTP 200 |
| Login | `/login` → HTTP 200 |
| Configuração pública | `/api/config` → HTTP 200 |
| API sem sessão | `/api/me` e `/api/issues` → HTTP 401 esperado |

Isso prova que não havia indisponibilidade geral do Tailscale Serve, frontend, backend ou
readiness no momento da verificação. Uma diferença entre duas telas não é, sozinha, evidência
de falha no serviço.

## 2. Modelo mental: por que duas pessoas veem telas diferentes

A tela exibida é o resultado de:

```text
dados canônicos do backend
+ identidade Multica
+ membership
+ workspace selecionado
+ rota (Issues ou My Issues)
+ scope
+ view (Board/List/Swimlane/Gantt)
+ grouping
+ filtros
+ colunas ocultas
+ ordenação
+ propriedades dos cards
+ paginação/carregamento
+ estado local do navegador
```

O board não é uma imagem global idêntica para todos. Os dados são compartilhados, mas a lente
visual é específica por usuário, dispositivo, navegador e workspace.

## 3. Arquitetura HTTPS/Tailscale

O fluxo humano é:

```text
Navegador
  → dispositivo autorizado no Tailscale
  → https://orq1.tail96e2c0.ts.net:443
  → Tailscale Serve
  → frontend e backend restritos ao loopback do ORQ1
```

Rotas verificadas do Tailscale Serve:

```text
/            → http://127.0.0.1:13100/
/ws          → http://127.0.0.1:18080/ws
/api         → http://127.0.0.1:18080/api
/uploads     → http://127.0.0.1:18080/uploads
/auth/login  → http://127.0.0.1:18080/auth/login
/auth/google → http://127.0.0.1:18080/auth/google
/auth/logout → http://127.0.0.1:18080/auth/logout
```

O frontend e o backend não precisam ficar expostos nas portas internas. A entrada humana e de
operador via Tailnet é o HTTPS na porta 443.

Não usar como URL do navegador:

```text
http://orq1.tail96e2c0.ts.net
https://100.118.244.61
http://100.118.244.61:13100
http://127.0.0.1:13100
http://127.0.0.1:18080
```

O certificado pertence ao hostname, não ao IP. Se houver alerta de certificado, não aceitar uma
exceção: confirmar primeiro o hostname e a conexão Tailscale.

## 4. Duas autenticações independentes

### 4.1 Tailscale

O Tailscale responde:

> Este dispositivo pode alcançar o servidor privado?

Ele não autentica o usuário dentro do Multica.

### 4.2 Multica

O login Multica responde:

> Qual usuário está acessando e de quais workspaces ele é membro?

É possível ter Tailscale funcionando e, ao mesmo tempo:

- não haver sessão Multica;
- usar outra conta/e-mail;
- não ter membership no workspace correto;
- estar em outro workspace;
- estar em `My Issues` com scope restrito;
- ter filtros locais ativos.

Nenhum desses casos prova indisponibilidade.

## 5. Verificação humana do Tailscale

No terminal ou PowerShell:

```bash
tailscale status
tailscale ping orq1
tailscale ping orq1.tail96e2c0.ts.net
```

No PowerShell:

```powershell
Test-NetConnection orq1.tail96e2c0.ts.net -Port 443
```

Esperado:

```text
TcpTestSucceeded : True
```

Teste HTTP sem segredo:

```powershell
curl.exe -I https://orq1.tail96e2c0.ts.net
```

Não usar `-k` para ignorar TLS.

## 6. Login, logout e perfis do navegador

Página de login:

```text
https://orq1.tail96e2c0.ts.net/login
```

O formulário atual contém e-mail, senha e `Sign in`. Credenciais são informadas somente no fluxo
autorizado. Nunca enviar senha, cookie, JWT ou token em chat.

Sessões não são compartilhadas entre:

- Chrome e Edge;
- janela normal e anônima;
- perfis diferentes do navegador;
- Windows e VM/WSL;
- dispositivos diferentes.

Logout encerra a sessão daquele navegador e limpa estado local associado, mas não desliga:

- Tailscale;
- frontend;
- backend;
- banco;
- daemon;
- fila de tasks.

## 7. Identidade e membership

Após o login, conferir no perfil o e-mail público usado. Aliases podem formar contas distintas.
Uma conta criada não recebe automaticamente acesso ao workspace compartilhado.

Se o usuário vê onboarding, criação de workspace ou board vazio:

1. confirmar o e-mail;
2. confirmar o convite/membership;
3. abrir o seletor de workspace;
4. escolher o workspace com issues `ORQ-*`;
5. não criar um workspace novo como contorno.

Cada workspace isola issues, agentes, projetos, membros, comentários, tasks e configurações.

## 8. A URL identifica o workspace

Formato:

```text
https://orq1.tail96e2c0.ts.net/<workspace-slug>/issues
```

Workspaces com slugs diferentes são contextos diferentes mesmo no mesmo servidor. Para comparar
duas telas, comparar a URL completa sem parâmetros sensíveis e confirmar o mesmo slug.

## 9. Normalização da tela humana

Para owner e TL enxergarem a mesma lente:

1. abrir o mesmo hostname HTTPS;
2. confirmar a conta correta;
3. selecionar o mesmo workspace;
4. comparar o `workspace-slug` da URL;
5. entrar em `Issues`, não `My Issues`;
6. selecionar scope `All`;
7. selecionar view `Board`;
8. selecionar grouping `Status`;
9. selecionar ordering `Manual`, direção ascendente;
10. abrir `Filter` e usar `Reset all filters`;
11. desligar o chip `agents working`;
12. restaurar todas as `Hidden columns`;
13. habilitar as mesmas propriedades de cards;
14. rolar horizontalmente;
15. executar hard refresh (`Ctrl+Shift+R` ou `Cmd+Shift+R`);
16. comparar identificadores `ORQ-*`, não apenas posição ou título.

Colunas normais do Board:

```text
Backlog
Todo
In Progress
In Review
Done
Blocked
```

`Cancelled` é válido, mas não aparece como coluna normal do Board. Usar List ou filtro para vê-lo.

## 10. Issues versus My Issues

`/<slug>/issues` apresenta o workspace conforme scope `All`, `Members` ou `Agents`.

`/<slug>/my-issues` é uma lente pessoal com scopes como:

```text
All
Assigned
Created
Agents
```

O padrão de `My Issues` pode ser `Assigned`. Se nada estiver atribuído ao TL, a tela pode ficar vazia.

## 11. Preferências locais que alteram a tela

São persistidos por workspace no navegador:

- view mode;
- grouping;
- filtros de status, prioridade, assignee, creator, projeto e label;
- inclusão de sem-assignee/sem-projeto;
- ordenação e direção;
- propriedades dos cards;
- status recolhidos;
- zoom/completos de Gantt;
- grouping, ordem e lanes recolhidas de Swimlane;
- scope All/Members/Agents.

Chaves internas incluem:

```text
multica_issues_view:<workspace-slug>
multica_issues_scope:<workspace-slug>
multica_my_issues_view:<workspace-slug>
```

Não inspecionar ou copiar storage/cookies: ele pode conter material de sessão. Resetar pela UI ou
remover somente os dados do site `orq1.tail96e2c0.ts.net` após logout.

## 12. Diagnóstico visual por camada

| Sintoma | Camada provável |
|---|---|
| DNS não resolve | Tailscale/MagicDNS |
| Porta 443 não conecta | Tailscale/ACL/Serve |
| `/login` abre | rede, TLS, Serve e frontend alcançáveis |
| `/api/me` anônimo retorna 401 | backend e auth middleware alcançáveis |
| Login funciona, workspace não aparece | conta/membership |
| Workspace correto, UI vazia | rota/scope/filtros/view |
| API mostra issue, Board não | filtros/hidden columns/cancelled/paginação |
| 502 | proxy alcançável, upstream interno indisponível |
| 5xx reproduzido por vários usuários | possível falha real de backend |

## 13. O agente não vê pixels: ele lê a API

Navegador e agente usam o mesmo backend:

```text
Frontend → GET /api/issues → Backend → PostgreSQL
CLI/API  → GET /api/issues → Backend → PostgreSQL
```

O agente recebe JSON canônico. Ele não herda filtros, grouping ou colunas ocultas do navegador.
Para reproduzir a visão humana, precisa descobrir explicitamente identidade, workspace, rota, scope,
status e paginação.

## 14. Quatro papéis operacionais

### 14.1 Humano no navegador

Usa sessão web e a interface HTTPS.

### 14.2 TL/operador pela CLI

Usa perfil humano isolado, PAT/sessão autorizada e comandos `multica`.

### 14.3 Agente dentro de uma task

Recebe token `mat_` temporário e workspace vinculado pelo daemon.

### 14.4 Coordenador Herdr

Herdr coordena e supervisiona. Trabalho executável continua no Kanban/API.

```text
Herdr = coordenação
Kanban/API = dispatch executável
```

## 15. Tipos de credencial e fronteiras

| Tipo | Uso |
|---|---|
| Sessão/cookie web | navegador humano |
| PAT `mul_...` | usuário/CLI autorizada |
| Task token `mat_...` | somente a task de agente vinculada |
| Token de daemon | claim/heartbeat/rotas internas do daemon |
| Cloud node PAT `mcn_...` | identidade de cloud node, quando aplicável |

Nunca:

- imprimir token;
- usar `echo $MULTICA_TOKEN`;
- copiar cookie/header Authorization;
- passar segredo inline na linha de comando;
- ler credential homes;
- usar credencial do daemon como usuário;
- usar PAT humano dentro de task quando um `mat_` deveria existir.

## 16. Ambiente injetado em uma task

O daemon injeta:

```text
MULTICA_SERVER_URL
MULTICA_TOKEN
MULTICA_WORKSPACE_ID
MULTICA_AGENT_NAME
MULTICA_AGENT_ID
MULTICA_TASK_ID
MULTICA_TASK_SLOT
```

Validação sem imprimir valores:

```bash
test -n "${MULTICA_SERVER_URL:-}" || exit 1
test -n "${MULTICA_WORKSPACE_ID:-}" || exit 1
test -n "${MULTICA_AGENT_ID:-}" || exit 1
test -n "${MULTICA_TASK_ID:-}" || exit 1
case "${MULTICA_TOKEN:-}" in mat_*) ;; *) exit 1 ;; esac
```

Dentro da task, a CLI falha fechado se o token não for `mat_`. Ela não usa como fallback o
`~/.multica/config.json` global e não deixa o agente mudar para outro workspace.

## 17. Ciclo completo de uma execução

```text
ativação autorizada no Kanban
→ linha em agent_task_queue
→ daemon elegível reclama a task
→ backend cria task token mat_
→ daemon injeta token/workspace/agent/task
→ processo filho executa
→ mensagens e resultado persistem
→ status terminal
→ task token revogado
→ UI atualizada por WebSocket/API
```

Estados ativos:

```text
queued
dispatched
running
waiting_local_directory
```

Estados terminais:

```text
completed
failed
cancelled
```

## 18. Preparar um TL permanente na CLI

Primeiro verificar rede e binário:

```bash
tailscale ping orq1
command -v multica
multica --version
multica issue --help
multica issue comment add --help
```

Usar perfil isolado:

```bash
multica \
  --profile novo-tl \
  --server-url https://orq1.tail96e2c0.ts.net \
  auth status
```

O login é feito pelo humano autorizado:

```bash
multica \
  --profile novo-tl \
  --server-url https://orq1.tail96e2c0.ts.net \
  login
```

Se PAT for necessário, preferir prompt interativo do humano:

```bash
multica \
  --profile novo-tl \
  --server-url https://orq1.tail96e2c0.ts.net \
  login --token
```

Não passar o valor inline. Depois, verificar somente com `auth status`.

## 19. Divergência CLI/backend

O binário inspecionado em `/home/ec2-user/bin/multica` reportou `multica dev`, commit/build unknown.
Seu help local não contém `--documentation-only`, embora o backend integrado na ancestralidade de
`edd7b932f7c44c3396fd87853c2795d746fc5134` suporte `documentation_only`.

Portanto:

- sempre ler o `--help` real;
- não inventar flags;
- não atualizar CLI sem autorização;
- distinguir contrato do backend de capacidade do cliente instalado;
- registrar versão do cliente na evidência.

## 20. Descobrir o workspace programaticamente

```bash
multica \
  --profile novo-tl \
  --server-url https://orq1.tail96e2c0.ts.net \
  workspace list \
  --output json \
  | jq '.[] | {id,name,slug}'
```

Selecionar:

```bash
multica \
  --profile novo-tl \
  --server-url https://orq1.tail96e2c0.ts.net \
  workspace switch <id-ou-slug>
```

Confirmar:

```bash
multica \
  --profile novo-tl \
  --server-url https://orq1.tail96e2c0.ts.net \
  workspace get \
  --output json \
  | jq '{id,name,slug,issue_prefix}'
```

Prioridade de resolução fora de task:

```text
--workspace-id
→ MULTICA_WORKSPACE_ID
→ workspace padrão do perfil
```

Dentro de task, somente o workspace injetado é autoridade.

## 21. Listar e paginar issues

```bash
multica \
  --profile novo-tl \
  --server-url https://orq1.tail96e2c0.ts.net \
  issue list \
  --limit 100 \
  --offset 0 \
  --output json
```

Resposta da CLI:

```json
{
  "issues": [],
  "total": 0,
  "limit": 100,
  "offset": 0,
  "has_more": false
}
```

O padrão é 50. Se `has_more=true`, continuar nos offsets seguintes. Não concluir que issues estão
ausentes olhando somente a primeira página.

## 22. Reconstruir o Board em JSON

Contagem por status:

```bash
multica \
  --profile novo-tl \
  --server-url https://orq1.tail96e2c0.ts.net \
  issue list --limit 100 --output json \
  | jq '.issues | group_by(.status) | map({status:.[0].status,count:length})'
```

Matriz resumida:

```bash
multica \
  --profile novo-tl \
  --server-url https://orq1.tail96e2c0.ts.net \
  issue list --limit 100 --output json \
  | jq -r '.issues[] | [(.identifier // .key // .id),.status,.priority,(.assignee_type // "none"),(.title // "")] | @tsv'
```

Board normal, excluindo cancelled:

```bash
multica \
  --profile novo-tl \
  --server-url https://orq1.tail96e2c0.ts.net \
  issue list --limit 100 --output json \
  | jq '.issues | map(select(.status == "backlog" or .status == "todo" or .status == "in_progress" or .status == "in_review" or .status == "done" or .status == "blocked"))'
```

## 23. Buscar e ler uma issue

```bash
multica --profile novo-tl --server-url https://orq1.tail96e2c0.ts.net \
  issue search 'ORQ-42' --include-closed --output json
```

```bash
multica --profile novo-tl --server-url https://orq1.tail96e2c0.ts.net \
  issue get ORQ-42 --output json
```

Resumo:

```bash
multica --profile novo-tl --server-url https://orq1.tail96e2c0.ts.net \
  issue get ORQ-42 --output json \
  | jq '{id,identifier,title,status,priority,assignee_type,assignee_id,project_id,parent_issue_id,created_at,updated_at}'
```

## 24. Ler comentários com limites

Raízes resumidas:

```bash
multica --profile novo-tl --server-url https://orq1.tail96e2c0.ts.net \
  issue comment list ORQ-42 --roots-only --summary --output json
```

Threads recentes:

```bash
multica --profile novo-tl --server-url https://orq1.tail96e2c0.ts.net \
  issue comment list ORQ-42 --recent 10 --summary --output json
```

Thread específica:

```bash
multica --profile novo-tl --server-url https://orq1.tail96e2c0.ts.net \
  issue comment list ORQ-42 --thread <comment-id> --tail 20 --output json
```

Usar cursores `--before` e `--before-id` quando fornecidos. Não carregar um histórico inteiro sem
necessidade.

## 25. Ler runs e mensagens

```bash
multica --profile novo-tl --server-url https://orq1.tail96e2c0.ts.net \
  issue runs ORQ-42 --output json
```

```bash
multica --profile novo-tl --server-url https://orq1.tail96e2c0.ts.net \
  issue run-messages <task-id> --issue ORQ-42 --output json
```

Incremental:

```bash
multica --profile novo-tl --server-url https://orq1.tail96e2c0.ts.net \
  issue run-messages <task-id> --issue ORQ-42 --since <seq> --output json
```

## 26. Ler agentes

```bash
multica --profile novo-tl --server-url https://orq1.tail96e2c0.ts.net \
  agent list --output json \
  | jq '.[] | {id,name,status,runtime_id,archived_at}'
```

```bash
multica --profile novo-tl --server-url https://orq1.tail96e2c0.ts.net \
  agent tasks <agent-id> --output json
```

Assignee sem runtime válido pode não ser executável. Sempre correlacionar issue, agent, runtime e task.

## 27. Rotas canônicas read-only

| Objetivo | Rota |
|---|---|
| Identidade | `GET /api/me` |
| Workspaces | `GET /api/workspaces` |
| Workspace | `GET /api/workspaces/{id}` |
| Membros | `GET /api/workspaces/{id}/members` |
| Issues | `GET /api/issues` |
| Busca | `GET /api/issues/search` |
| Issue | `GET /api/issues/{uuid}` |
| Comentários | `GET /api/issues/{uuid}/comments` |
| Timeline | `GET /api/issues/{uuid}/timeline` |
| Task ativa | `GET /api/issues/{uuid}/active-task` |
| Runs | `GET /api/issues/{uuid}/task-runs` |
| Mensagens | `GET /api/tasks/{task-id}/messages` |
| Agentes | `GET /api/agents` |
| Tasks por agente | `GET /api/agents/{id}/tasks` |
| Squads | `GET /api/squads` |
| Snapshot de tasks | `GET /api/agent-task-snapshot` |
| Projetos | `GET /api/projects` |

Contexto de workspace pode vir por slug/header/query ou UUID/header/query. Em task token, o vínculo
server-side prevalece e não pode ser ampliado.

## 28. Preferir CLI a curl autenticado

A CLI adiciona autenticação, workspace, identidade de agente/task, timeout e resolução de `ORQ-*`.
Evitar `curl -H "Authorization: Bearer ..."`, pois o valor pode aparecer no argv/process list.

Para diagnóstico autenticado de baixo nível, usar somente wrapper aprovado ou arquivo privado `0600`
preparado pelo owner. O agente não lê nem imprime esse arquivo.

## 29. Sondas anônimas seguras

```bash
curl -sS -o /dev/null -w 'http=%{http_code} tls=%{ssl_verify_result}\n' \
  --max-time 10 https://orq1.tail96e2c0.ts.net/
```

```bash
curl -sS -o /dev/null -w 'http=%{http_code}\n' \
  --max-time 10 https://orq1.tail96e2c0.ts.net/login
```

```bash
curl -sS -o /dev/null -w 'http=%{http_code}\n' \
  --max-time 10 https://orq1.tail96e2c0.ts.net/api/config
```

```bash
curl -sS -o /dev/null -w 'http=%{http_code}\n' \
  --max-time 10 https://orq1.tail96e2c0.ts.net/api/me
```

`/api/me=401` sem credencial é esperado e prova que a requisição chegou ao auth middleware.

## 30. Códigos HTTP

| Código | Interpretação inicial |
|---:|---|
| 200 | leitura realizada |
| 201 | recurso criado |
| 202 | ação aceita/enfileirada |
| 400 | parâmetro/workspace/payload inválido |
| 401 | autenticação ausente/inválida/expirada |
| 403 | identidade válida, ação proibida ou workspace fora do token |
| 404 | recurso/workspace não encontrado ou não visível |
| 409 | conflito/duplicata/estado concorrente |
| 429 | limite/capacidade |
| 502 | edge responde, upstream interno falha |
| 503 | dependência temporariamente indisponível |
| 5xx | possível falha real de backend |

## 31. Comparar UI e API

Registrar na UI, sem dados sensíveis:

```text
host
workspace slug
route
scope
view
grouping
filtros ativos
```

Depois comparar pela CLI o workspace e a contagem por status. Se a API contém a issue e a UI não,
investigar preferências visuais. Se o workspace não aparece na API, investigar identidade/membership.

## 32. Ações que podem disparar trabalho

### 32.1 Assign/reassign

```bash
multica issue assign ORQ-42 --to-id <agent-id>
```

É mutação e pode criar task. Não usar como teste visual.

### 32.2 Rerun

```bash
multica issue rerun ORQ-42
```

É explicitamente executável. Não usar como refresh.

### 32.3 Comentário normal

Pode disparar assignee, agente mencionado ou líder de squad. Não tratar como nota inerte.

### 32.4 Documentação sem execução

O backend integrado suporta `documentation_only=true`, que suprime todos os triggers, e reconhece
comentário iniciado por `/note` como nota sem execução. Ainda assim, comentar é mutação e exige
autorização, GET-back e confirmação de que nenhuma task foi criada.

### 32.5 Status

Metadados foram desacoplados de execução em vários casos, mas a primeira promoção de backlog com
assignee executável nunca despachado pode carregar intenção de execução. `cancelled` também pode
cancelar tasks. Tratar toda mudança de status como mutação operacional.

## 33. Contrato integrado relevante

Na ancestralidade do backend reconciliado `edd7b932...` está:

```text
3ee6a2e — fix(server): decouple Kanban metadata from paid execution
```

Comportamento:

- alterações comuns de título/descrição/prioridade/status não redisparam uma issue já executada;
- transição explícita de assignee permanece intenção de execução;
- primeira ativação de backlog pode executar;
- comentário normal pode disparar triggers;
- `documentation_only=true` e `/note` suprimem execução;
- rerun permanece executável.

Não usar documentação histórica como descrição integral do backend atual sem validar a linhagem live.

## 34. Uma ativação, uma task

Antes de dispatch:

1. ler issue e critérios;
2. consultar active-task;
3. consultar task-runs, inclusive failed/cancelled;
4. provar que não há task ativa cobrindo o delta;
5. verificar assignee, runtime e elegibilidade;
6. verificar branches/commits existentes;
7. verificar trabalho aceito/substituto;
8. confirmar autorização;
9. realizar uma única ativação;
10. capturar o task ID;
11. não reenviar por atraso visual.

## 35. Acompanhamento read-only

```bash
multica issue get ORQ-42 --output json
multica issue runs ORQ-42 --output json
multica issue run-messages <task-id> --issue ORQ-42 --since <seq> --output json
```

Não usar assign, rerun, cancel, status ou comentário apenas para forçar atualização.

## 36. Kanban não é prova isolada de conclusão

Reconciliar:

```text
OpenSpec
↔ issue e acceptance criteria
↔ comentários/decisões
↔ task terminal
↔ branch/ref
↔ commit e ancestralidade
↔ diff
↔ testes
↔ candidate/build
↔ deployment live
↔ comportamento observado
↔ aceite
```

Vocabulário:

- **source**: código existe em SHA exato;
- **tested**: testes nomeados passaram naquele SHA;
- **candidate**: build elegível;
- **accepted**: revisor/gate aprovou;
- **integrated**: SHA está na linhagem canônica/candidata;
- **deployed/live**: identidade implantada foi observada;
- **blocked**: condição concreta impede o próximo gate.

`completed` não significa automaticamente aceito; `Done` não significa automaticamente integrado e live.

## 37. Classificação A–F

| Classe | Significado | Próxima ação |
|---|---|---|
| A | já feito e aceito | reconciliar documentação/card |
| B | implementado, falta gate/integração/deploy | executar somente o gate faltante |
| C | parcial | despachar apenas o delta delimitado |
| D | tentativa falhou, substituto válido existe | reutilizar substituto |
| E | genuinamente não feito | menor task não sobreposta |
| F | bloqueio externo/owner/segurança | registrar ação exata do owner |

## 38. Ledger mínimo do TL

```text
issue | classification | workspace | board status | assignee | active task |
terminal tasks | reused commit | branch/ref | review | tests | candidate |
deployment | OpenSpec | missing delta | owner blocker | disposition
```

## 39. Árvore programática de diagnóstico

1. Hostname não resolve → Tailscale/MagicDNS.
2. Root HTTPS 200, CLI 401 → autenticação CLI.
3. Auth funciona, workspace ausente → conta/membership.
4. Workspace aparece, issue list vazio → workspace/paginação/filtros/status.
5. API mostra issues, UI não → view/scope/filtros/grouping/hidden columns/cache.
6. `issue get` funciona, Board não → cancelled/hidden/filter/paginação.
7. CLI dentro de task pede login → contexto `mat_`/env ausente; falhar fechado.
8. `task token is bound to a different workspace` → tentativa cross-workspace; não contornar.
9. 5xx reproduzido no mesmo contexto por vários membros → investigar serviço.

## 40. Evidência segura

Pode registrar:

```text
timestamp UTC
hostname
rota/método/status HTTP
workspace slug/UUID
issue identifier/UUID
task UUID
agent UUID
status
commit SHA
branch/ref
teste e exit code
deployment identity
```

Nunca registrar:

```text
senha
token
cookie
Authorization
CSRF
credential home
arquivo de configuração autenticada
private key
secret value
```

Exemplo:

```text
timestamp=2026-07-31T12:00:00Z
host=orq1.tail96e2c0.ts.net
route=GET /api/issues
http=200
workspace=<slug>
total=<n>
auth_secret_observed=false
mutation_performed=false
```

## 41. Checklist final do novo TL

```text
[ ] Tailscale conectado e orq1 alcançável
[ ] HTTPS canônico responde
[ ] binário e versão da CLI identificados
[ ] flags reais verificadas no --help
[ ] perfil isolado do TL
[ ] auth status sem expor credencial
[ ] workspaces descobertos pela API
[ ] workspace ORQ selecionado
[ ] id, slug e issue_prefix confirmados
[ ] issues listadas em JSON
[ ] paginação total/limit/offset/has_more tratada
[ ] contagem por status produzida
[ ] issue buscada incluindo closed
[ ] detalhe, comentários e runs lidos
[ ] task, agent e runtime correlacionados
[ ] Git, testes, deployment e OpenSpec reconciliados
[ ] classificação A–F atribuída
[ ] nenhuma duplicação de trabalho
[ ] Herdr usado somente para coordenação
[ ] nenhum assign/rerun/comment/status usado como diagnóstico
[ ] qualquer mutação autorizada seguida de GET-back
[ ] ativação produziu exatamente uma task
[ ] evidência não contém segredo
```

## 42. Conclusão

A frase operacional central é:

> A interface do owner e a consulta programática do agente usam o mesmo backend, mas podem usar
> identidades, workspaces, rotas, filtros e estados locais diferentes.

Saúde de transporte/serviço:

```text
HTTPS 200
+ login 200
+ config 200
+ /api/me anônimo 401 esperado
+ frontend 200
+ healthz 200
+ readyz 200
```

Mesma visão de dados:

```text
identidade autorizada
+ mesmo workspace
+ mesmo issue prefix
+ GET /api/issues 200
+ paginação completa
+ mesma issue por identificador
```

Conclusão de produto:

```text
issue
+ task
+ commit/ancestralidade
+ testes/review
+ integração/deployment
+ OpenSpec
+ aceite
```

Somente depois de normalizar essas camadas uma diferença restante deve ser tratada como possível
defeito. Antes disso, `401`, workspace vazio, filtro local ou CLI desatualizada não sustentam a
afirmação de que o Kanban está quebrado.
