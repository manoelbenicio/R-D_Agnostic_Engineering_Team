# T4 — Auditoria READ-ONLY do Kanban/frontend e ORQ-26 (2026-07-27)

Auditor: Codex56#A (ORQ2, pane `w7:p3`) · UTC 2026-07-27T10:20Z · solicitado por Codex56#B (`w7:p4`)
Modo: somente leitura. Nada editado, nada deployado, nada reiniciado, nenhum rerun, nenhuma escrita
via API. Alvo: ORQ1 `100.118.244.61` (projeto Docker `multica-dev-transition`).

## 1. Achado principal: a tag `transition-6a2aba3` NAO e mais o artefato de 19/07

```text
docker inspect (ORQ1):
multica-web:transition-6a2aba3      -> sha256:cf8017e3d2fd...  created 2026-07-24T11:42:47Z
multica-web:20260724-web-cf8017e3d2fd -> sha256:cf8017e3d2fd... (MESMO ID: tag dupla)
multica-backend:transition-6a2aba3  -> sha256:04217fa1fdbe...  created 2026-07-20T07:32:05Z
multica-backend:t20obs-8f524054     -> sha256:994aa2284b55...  created 2026-07-24T12:19:04Z (EM USO)
```

O dossier de deploy (`docs/transition/DEV_RESTART_DOSSIER_20260719.md:269-270` e
`DOCKER_AND_REDIS_INVENTORY_20260719.md:38-39`) fixa como artefato `6a2aba3`:

```text
multica-web:transition-6a2aba3      sha256:29e4bc52c351443234ac593eefecb4a712e205f5c9602b66fc6d5add39a8952d
multica-backend:transition-6a2aba3  sha256:c8ba7dc56be057c9bff5e076d6020d7d550cf4406a344729defc1eba94057dd0
```

Verificacao dos dois IDs no host:

```text
docker inspect sha256:29e4bc52c351... -> ABSENT
docker inspect sha256:c8ba7dc56be0... -> ABSENT
```

Conclusao: **os artefatos originais de `6a2aba3` nao existem mais no ORQ1**. A tag sobreviveu e foi
reapontada para builds de 20/07 (backend) e 24/07 (web). O rollback do ORQ-26 levou ao build de
**24/07**, nao ao codigo de `6a2aba3`. A propria issue registra isso corretamente
(`sha256:cf8017e3...`), portanto nao houve engano na execucao; o risco e de **rotulo**: os runbooks de
transicao continuam prometendo um sha256 que nao existe mais, e "voltar para a transition-6a2aba3
conhecida como boa" hoje significa outra coisa. Recomendacao: nos runbooks e no ORQ-26, identificar
imagem por ID/digest, nunca por tag; e se o owner quiser o artefato real de `6a2aba3`, sera rebuild a
partir do commit `6a2aba3550aaf6b0468a37bfdf2f00c7faaae084`, nao rollback.

## 2. Estado de execucao

```text
docker ps (ORQ1):
multica-dev-transition-frontend-1 | multica-web:transition-6a2aba3   | Up 7 hours | 127.0.0.1:13100->3000
multica-dev-transition-backend-1  | multica-backend:t20obs-8f524054  | Up 2 days  | 127.0.0.1:18080->8080
multica-dev-transition-postgres-1 | pgvector/pgvector:pg17           | Up 5 days (healthy) | 127.0.0.1:15433->5432
omniroute                         | diegosouzapw/omniroute:latest    | Up 2 days (healthy) | 100.118.244.61:20128->20128

frontend: started=2026-07-27T03:11:24Z restarts=0; ultimas 200 linhas de log sem error/unhandled/failed
```

Web (11:42) e server (12:19) sao do mesmo dia 24/07 — nao ha defasagem grande de contrato entre os dois
binarios em execucao. Atencao: a porta do frontend e **13100**, nao 3100 (probe em 3100 retorna 000;
`/` e `/login` em 13100 retornam 200).

## 3. API read: funciona. API write: NAO testada (proibido).

```text
GET http://127.0.0.1:18080/health                              -> 200
GET /api/workspaces                                            -> 200 (workspace orq2-dev, issue_prefix ORQ)
GET /api/me                                                    -> 200
GET /api/issues?workspace_id=20fce817-...                      -> 200 (inclui ORQ-26)
GET /api/issues        (sem workspace_id)                      -> 400
GET /api/agents        (sem workspace_id)                       -> 400
GET /api/runtimes      (sem workspace_id)                       -> 400
GET http://127.0.0.1:13100/  e /login                          -> 200
GET http://127.0.0.1:13100/api/health                          -> 404 (rota inexistente no web, esperado)
```

Nenhum POST/PATCH/DELETE foi emitido. A verificacao de escrita foi **estatica**: as rotas de acao de
board existem no binario em execucao e casam com o bundle do frontend.

| fragmento | bundle web (`/app/apps/web/.next`) | binario backend (`/app/server`) |
|---|---:|---:|
| `api/issues/` | 56 | presente |
| `api/issues/grouped` | 4 | 3 |
| `api/issues/batch-update` | 2 | 1 |
| `api/issues/batch-delete` | 2 | 1 |
| `api/issues/quick-create` | 2 | 13 |
| `api/issues/child-progress` | 2 | 1 |
| `api/issues/children` | 4 | 42 |
| `pending-task` | 8 | 5 |
| `messages/page` | 2 | 1 |

## 4. Erros 4xx/5xx observados (ultimas 5000 linhas do log do backend)

```text
status=200 -> 3922
status=404 ->    8
status=400 ->    3
status=5xx ->    0     (nenhuma linha level=ERROR tambem)
```

Detalhe dos 404:

```text
6x GET /api/chat/sessions/cffb5ee4-24ae-4ee8-a9af-a9276488784a/messages
6x GET /api/chat/sessions/cffb5ee4-.../messages/page
6x GET /api/chat/sessions/cffb5ee4-.../pending-task
7x GET /api/issues/<uuid>/runs
1x GET /api/health
```

Classificacao com evidencia:

- Os 404 de chat **nao sao rota faltando**: `pending-task` (5 hits) e `messages/page` (1 hit) existem
  no binario, e no Postgres `select count(*) from chat_session where id::text like 'cffb5ee4%'` = **0**
  contra **190** sessoes existentes. E um cliente/aba antigo pollando uma sessao removida: 404 correto.
- Os 404 de `/api/issues/<uuid>/runs`: a rota `/runs` existe (3 hits), mas o bundle atual chama
  `/api/issues/${id}/task-runs` e `/api/issues/${id}/active-task`. Ou seja quem pede `/runs` **nao e o
  frontend em execucao** (aba antiga, CLI ou script). Observacao, nao conclusao de causa.
- Os 3 `status=400` das 10:14:12Z **sao meus proprios probes** (`/api/issues`, `/api/agents`,
  `/api/runtimes` sem `workspace_id`). Declaro para nao poluirem a analise de ninguem.

## 5. Board actions "deveriam funcionar" na imagem em execucao?

Resposta com o escopo que consigo provar: **sim, o caminho servidor esta la**. As rotas de mutacao de
board (`batch-update`, `batch-delete`, `quick-create`, `grouped`, `children`, `child-progress`) existem
no backend em execucao, o frontend as referencia, backend e web sao do mesmo dia, o log nao tem 5xx e o
frontend nao reiniciou nenhuma vez desde 03:11:24Z.

O que isso **nao** prova, e e o limite explicito: nao ha browser autenticado nesta auditoria. Nao
executei clique, drag-and-drop, nem li console do navegador. A regressao do ORQ-26 e descrita como
"nenhum botao do painel de chat responde" com backend validado a parte — exatamente a classe de falha
que **nao aparece** em log de servidor nem em `strings` de binario (erro de bundle/hidratacao/handler
no cliente). Portanto: minha auditoria nao pode confirmar nem refutar que os botoes funcionam na
imagem atual. O critério de aceite do ORQ-26 (erro de console reproduzido ou teste de browser) segue
**em aberto**.

## 6. Risco de seguranca observado (nao alterei nada)

`GET /api/me` sem qualquer credencial retorna identidade do owner
(`id 7efc68e4-...`, `owner@local.test`) e `GET /api/workspaces` / `/api/issues` devolvem dados reais.
As linhas de log dos meus probes anonimos vem com `user_id=7efc68e4-...`, isto e, **a autenticacao nao
esta sendo exigida neste deployment** — qualquer processo local ou quem tiver shell no ORQ1 tem acesso
de owner a API. Mitigacao existente: `18080` e `13100` estao publicados **apenas em 127.0.0.1** (o
OmniRoute e o unico bind em IP de rede, `100.118.244.61:20128`), portanto nao ha exposicao via
LAN/Tailscale. Registro como risco para decisao do owner; nao propus nem apliquei mudanca.

## 7. Nao-alegacoes

- Nao editei arquivo, nao fiz deploy, nao reiniciei container ou daemon, nao rodei rerun, nao commitei.
- Nao emiti nenhuma requisicao de escrita (POST/PATCH/PUT/DELETE) contra a API.
- Sem browser autenticado: zero verificacao de UI, clique, drag-and-drop ou console.
- Nao inspecionei o bundle da imagem defeituosa `reasoning-ui-20260727T013129Z` (preservada, fora de
  producao) — comparar os dois bundles e o proximo passo natural para o ORQ-26, e nao foi pedido aqui.
- Nao li credencial, token nem `secret_ref`.
- Os 3 `status=400` das 10:14:12Z sao meus.
