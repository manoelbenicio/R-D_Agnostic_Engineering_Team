# ORQ-17 STAGE 2 - VERIFICACAO POS-CUTOVER (READ-ONLY)

- **Monitor:** Opus48#B (ORQ2, pane w6:p2) · **Cartao:** `ORQ-17` stage 2
- **Host:** ORQ1 — `ip-172-31-18-217.sa-east-1.compute.internal`, tailscale `100.118.244.61`
- **Janela:** 2026-07-27T15:57:43Z -> 15:58:48Z (S2-POST, amostras P1..P4)
- **Modo:** READ-ONLY. Nao buildei, nao reiniciei, nao recriei, nao editei, nao apliquei/resetei
  `serve`, nao fiz login, nao li segredo, nao mutei DB, board nem AWS.

# VEREDITO: **PASS**

Onze verificacoes exigidas, **onze conformes**. Nenhuma condicao de STOP acionada.

---

## 1. BACKEND RECRIADO - ID E START TIME NOVOS

```
antes  : id=dc719a1bf4e7  started=2026-07-27T10:55:19.715949687Z
depois : id=75e4416f06e9  started=2026-07-27T15:56:31.478491626Z
project=multica-dev-transition  service=backend
```
ID **mudou** e `started` avancou para 15:56:31Z. Recriacao confirmada por evidencia imutavel, nao por
declaracao do executor. **PASS.**

## 2. CONFIG FILES - OS 4 ORIGINAIS + 1 OVERRIDE DE AUTH

Label do container **novo**:
```
1 /home/ec2-user/R-D_Agnostic_Engineering_Team/multica-auth-work/docker-compose.selfhost.yml
2 /home/ec2-user/R-D_Agnostic_Engineering_Team/multica-auth-work/docker-compose.selfhost.build.yml
3 /home/ec2-user/.config/multica-transition/images.yml
4 /home/ec2-user/.config/multica-transition/backend-env.override.yml
5 /home/ec2-user/.config/multica-transition/orq17-auth-cutover.override.yml   <- NOVO
```
Os itens 1 a 4 sao **byte a byte identicos** aos do label anterior, na mesma grafia absoluta. O item 5
e **exatamente um** arquivo novo, e e o override de auth. `project` e `service` inalterados.
**PASS**, conforme a regra que eu havia fixado: 4 conhecidos + 1 override, nada mais.

## 3. MESMA IMAGEM DE BACKEND - SEM REBUILD

```
imagem do container : sha256:60133934d8f9
docker image inspect 60133934d8f9
  created = 2026-07-27T10:30:51.172941259Z
  tags    = [multica-backend:agy-status-20260727T102815Z,
             multica-backend:orq17-stage2-rollback-60133934d8f9]
```
A imagem foi criada as **10:30:51Z**, mais de 5 h **antes** da recriacao das 15:56:31Z. Portanto o
executor **reusou** a imagem existente e **nao** buildou nada. **PASS.**

Observacao favoravel: a mesma imagem tem a tag `orq17-stage2-rollback-60133934d8f9`, isto e existe
alvo de rollback de imagem nomeado explicitamente. Imagens anteriores tambem preservadas
(`994aa2284b55`, `04217fa1fdbe`).

## 4. HEALTH / READY - 200 EM MULTIPLAS AMOSTRAS

| amostra | UTC | /health | /readyz |
|---|---|---|---|
| P1 | 15:58:00 | 200 | 200 |
| P2 | 15:58:10 | 200 | 200 |
| P3 | 15:58:20 | 200 | 200 |
| P4 | 15:58:48 | 200 | 200 |

Backend recuperou e **manteve** `200` em `/health` e `/readyz` nas quatro amostras. **PASS.**

## 5. POSTURA DE AUTH - MUDANCA INTENCIONAL CONFIRMADA

| amostra | anon `GET /api/me` | anon `GET /api/issues` |
|---|---|---|
| P1 | **401** | **401** |
| P2 | **401** | **401** |
| P3 | **401** | **401** |
| P4 | **401** | **401** |

Antes do cutover eu havia medido `anon /api/me = 200` em duas amostras (S2-T0 e S2-T1). Agora e `401`
em quatro amostras, e `/api/issues` tambem `401`. A transicao **200 -> 401** e exatamente a mudanca
declarada, e e **esperada, nao drift**. **PASS.**

Isso tambem fecha o ponto que eu havia levantado: o criterio "anon `/api/me` deve falhar 401/403"
**passou a ser satisfazivel** porque o stage 2 desligou o bypass. Minha objecao anterior era valida
para o estado **pre-cutover** e deixa de valer agora.

## 6. CONTAINERS NAO-BACKEND - IDs INALTERADOS

```
frontend  1b3b6c9ee32a  started=2026-07-27T03:11:24.477348893Z   inalterado
postgres  2a4a84897363  started=2026-07-21T17:10:49.731230109Z   inalterado
omniroute 2fb3fd57e885  started=2026-07-24T18:36:12.135713768Z   inalterado
```
Nem o ID nem o `started` mudaram em nenhum dos tres. **Somente o backend foi recriado.** **PASS.**
`frontend` em `13100` continuou `200` durante toda a janela.

## 7. SERVE / FUNNEL / 443

| amostra | serve | funnel | listeners `:443` |
|---|---|---|---|
| P1..P3 | `No serve config` | `No serve config` | 0 |
| reconferencia 15:58:5x | `No serve config` | `No serve config` | 0 |

**Serve vazio, Funnel vazio, zero listeners em 443** em todas as leituras. **PASS.**

## 8. FILA AINDA ZERO

```
SELECT count(*) FROM agent_task_queue
WHERE status IN ('queued','dispatched','running','waiting_local_directory');  -> 0
```
Terceira leitura agregada da sessao, apos o cutover: **0**. Somadas as duas de pre-cutover, sao
**3 leituras independentes, todas `0`**. **PASS.**

## 9. OVERRIDE DE AUTH - CAMINHO, MODO E APENAS VALORES ALLOWLISTED

```
/home/ec2-user/.config/multica-transition/orq17-auth-cutover.override.yml
  mode=600  owner=ec2-user:ec2-user  size=546  type=regular file
  mtime=2026-07-27 15:55:37Z   (56 s antes do start do container: coerente)
```
Modo **0600**, dono correto. **PASS.**

Chaves presentes — **nomes apenas**, extraidos por regex, sem imprimir valores:
```
APP_ENV, AUTH_TOKEN_TTL, COOKIE_DOMAIN, CORS_ALLOWED_ORIGINS, FRONTEND_ORIGIN,
GOOGLE_REDIRECT_URI, MULTICA_APP_URL, MULTICA_LOCAL_AUTH_BYPASS,
MULTICA_LOCAL_AUTH_EMAIL, MULTICA_PUBLIC_URL, MULTICA_TRUSTED_PROXIES
```
Varredura de chave sensivel (`secret|password|passwd|token|api_key|private|credential`) casou **uma**
linha. Investiguei extraindo **somente o nome da chave**: e `AUTH_TOKEN_TTL` — casou por conter
"token", e e um **TTL**, nao um segredo. Nenhuma chave de segredo, senha, credencial ou chave privada
no arquivo. **PASS.**

Valores lidos — **exclusivamente** as duas chaves allowlisted que explicam a mudanca de postura:
```
MULTICA_LOCAL_AUTH_BYPASS: "false"
FRONTEND_ORIGIN: "https://orq1.tail96e2c0.ts.net"
```
Ambos coerentes com o `401`: pela logica de `internal/middleware/auth.go:38-55`, o bypass exige o flag
`true` **e** origem loopback; aqui o flag e `false` (primeira guarda ja barra) e a origem deixou de ser
loopback (segunda guarda barraria tambem). Dupla razao para o bypass estar desligado. **PASS.**

**Nao li valor de nenhuma outra chave.** `COOKIE_DOMAIN`, `CORS_ALLOWED_ORIGINS`,
`GOOGLE_REDIRECT_URI`, `MULTICA_TRUSTED_PROXIES`, `MULTICA_APP_URL`, `MULTICA_PUBLIC_URL`,
`AUTH_TOKEN_TTL`, `APP_ENV` e `MULTICA_LOCAL_AUTH_EMAIL` tiveram **apenas o nome** observado.

## 10. OBSERVACAO QUE NAO E DRIFT, MAS PEDE DECISAO

`FRONTEND_ORIGIN` agora aponta para `https://orq1.tail96e2c0.ts.net`, um nome de tailnet em **HTTPS**.
Ao mesmo tempo, por exigencia deste stage, **Serve e Funnel permanecem vazios e a 443 permanece
fechada** — o que confirmei em todas as amostras. Consequencia factual: **nao existe nada servindo
essa origem hoje**. Isso e coerente com o objetivo de desligar o bypass, e e o que produz o `401`.

Registro, sem tratar como drift porque nenhuma condicao de STOP foi violada:
1. a origem declarada e **aspiracional** enquanto 443 estiver fechada e Serve vazio;
2. se alguem futuramente habilitar Serve/Funnel para satisfazer essa origem, isso e **outra** mudanca,
   com gate proprio, e passaria a expor 443 — hoje explicitamente proibido;
3. fluxos que dependam de cookie/CORS casando com a origem podem falhar por essa incoerencia. Nao
   testei nenhum fluxo autenticado, e nao e escopo do monitor.

## 11. CONDICOES DE STOP - NENHUMA ACIONADA

| condicao | estado |
|---|---|
| `serve` ou `funnel` nao vazio | nao ocorreu |
| listener em `:443` | nao ocorreu (0 em todas) |
| ID de `frontend`, `postgres` ou `omniroute` mudar | nao ocorreu |
| mais de um container recriado | nao ocorreu (somente backend) |
| caminho de compose fora dos 4 conhecidos + 1 override | nao ocorreu |
| `project` ou `service` divergente | nao ocorreu |
| mais de um arquivo novo, ou novo que nao seja o override de auth | nao ocorreu |
| rebuild de imagem | nao ocorreu (imagem de 10:30:51Z reusada) |
| backend nao recuperar `200` | nao ocorreu (4 amostras 200/200) |
| fila sair de `0` | nao ocorreu |
| segredo em claro no override | nao ocorreu (unico match foi `AUTH_TOKEN_TTL`) |

## 12. NAO-AFIRMACOES
- READ-ONLY: nenhuma mutacao partiu de mim; nao recriei, nao reiniciei, nao editei, nao buildei.
- **Nao inspecionei `Config.Env` por completo** em nenhum momento desta rodada.
- **Nao li conteudo de segredo.** No override, li valor de **exatamente duas** chaves allowlisted
  (`MULTICA_LOCAL_AUTH_BYPASS`, `FRONTEND_ORIGIN`); de todas as outras observei **somente o nome**.
  A linha que casou o filtro sensivel foi identificada extraindo apenas o **nome** da chave.
- Nao abri conteudo de backup; nao verifiquei backup nesta rodada alem do que ja constava do stage 1.
- Nao testei nenhum fluxo **autenticado**: as verificacoes de auth foram todas anonimas, por codigo
  HTTP. Nao fabriquei token e nao fiz login.
- Nao verifiquei se `401` e o codigo correto **por rota** alem de `/api/me` e `/api/issues`.
- Nao avaliei se a nova `FRONTEND_ORIGIN` quebra algum fluxo de cookie/CORS: ver secao 10, item 3.
- A janela e curta (~65 s, 4 amostras) e comeca **apos** o cutover. Nao observei o instante da
  recriacao em si, nem a eventual janela de indisponibilidade do backend entre 15:56:31Z e 15:58:00Z.
- Nao criei, atribui nem comentei issue alguma, e nao enfileirei task.
