# ORQ-17 STAGE 1 - monitor de seguranca independente (READ-ONLY)

- **Monitor:** Opus48#B (ORQ2, pane w6:p2)
- **Cartao:** `ORQ-17` stage 1
- **Executor observado:** Codex56-B (w7:p4), preparando recuperacao do frontend
- **Janela:** 2026-07-27T15:47:55Z -> 15:49:49Z (T0..T6)
- **Skill carregada integralmente nesta sessao:** `.agents/skills/aws-secrets-manager/SKILL.md` (v1),
  recarregada as 15:38Z. Regras aplicadas: R1 sem `get-secret-value`/`batch-get-secret-value`,
  R2 sem SMA em `localhost:2773`, R3 apenas `{{resolve:...}}` via `asm-exec`; e a advertencia da
  propria skill de que ela e **best-effort defense, not a security boundary**.
- **Modo:** READ-ONLY. **Nao** buildei, **nao** reiniciei, **nao** editei, **nao** apliquei nem
  resetei `serve`, **nao** fiz login, **nao** li segredo, **nao** mutei DB, board nem AWS.

# VEREDITO ATE T6: **SEM DRIFT. NENHUM STOP EMITIDO.**

Todos os invariantes monitorados permaneceram no estado exigido, e as pre-condicoes de rollback
passaram **antes** de qualquer mutacao.

---

## 1. PROVA DE IDENTIDADE DO HOST (exigida)

```
hostname   : ip-172-31-18-217.sa-east-1.compute.internal
IPs        : 172.31.18.217  100.118.244.61  172.18.0.1 172.17.0.1 172.19.0.1  fd7a:115c:a1e0::5034:f43e
tailscale  : 100.118.244.61
usuario    : ec2-user
```
Confere com **ORQ1** (`100.118.244.61`, EC2 `sa-east-1`). **Nao** e o ORQ2, cujo hostname e
`ip-172-31-30-9` / tailscale `100.110.178.47`. Toda observacao deste relatorio foi feita **no ORQ1**.

## 2. INVARIANTES MONITORADOS - 7 AMOSTRAS

| amostra | UTC | serve | funnel | listeners :443 | 18080 /health | 18080 /readyz | 13100 | frontend id | backend id |
|---|---|---|---|---|---|---|---|---|---|
| T0 | 15:47:55 | `No serve config` | `No serve config` | 0 | 200 | 200 | 200 | 1b3b6c9ee32a | dc719a1bf4e7 |
| T1 | 15:48:34 | `No serve config` | `No serve config` | 0 | 200 | 200 | 200 | 1b3b6c9ee32a | dc719a1bf4e7 |
| T2 | 15:48:46 | `No serve config` | `No serve config` | 0 | 200 | 200 | 200 | 1b3b6c9ee32a | dc719a1bf4e7 |
| T3 | 15:48:58 | `No serve config` | `No serve config` | 0 | 200 | 200 | 200 | 1b3b6c9ee32a | dc719a1bf4e7 |
| T4 | 15:49:21 | `No serve config` | `No serve config` | 0 | 200 | 200 | 200 | 1b3b6c9ee32a | dc719a1bf4e7 |
| T6 | 15:49:49 | `No serve config` | `No serve config` | 0 | 200 | 200 | 200 | 1b3b6c9ee32a | dc719a1bf4e7 |

- **Serve vazio em 100% das amostras.** `tailscale serve status` -> `No serve config`.
- **Funnel vazio em 100% das amostras.** `tailscale funnel status` -> `No serve config`.
- **Porta 443 fechada em 100% das amostras.** `ss -ltn` filtrado por `:443` devolveu **zero**
  listeners em todas.
- **Backend 18080 estavel:** `/health` e `/readyz` em `200` nas 6 amostras.
- **Frontend 13100:** `200` nas 6 amostras.

### 2.1 ACHADO-CHAVE: frontend `200` com **ID inalterado**
`13100 = 200` **e** `fe_id = 1b3b6c9ee32a` constante em todas as amostras, com
`started = 2026-07-27T03:11:24Z` (~13 h de uptime). Ou seja: **nao houve transicao do frontend** na
janela observada. O `200` nao e resultado de recriacao; e o container **original** ainda servindo.

Consequencia para o gate de aceite: se a recuperacao do frontend exigir container novo, o sucesso
tera de ser evidenciado por **mudanca do container ID**, nao por `200` em 13100 — que ja era `200`
antes de qualquer acao. Registro isso para que o `200` nao seja lido como prova de recuperacao.

**Nenhuma transicao de 13100 a registrar**, porque nenhuma ocorreu.

## 3. ALVO DO EXECUTOR - ORQ1 REAL, NAO O TUNEL DO ORQ2

Labels de compose lidos por `docker inspect` **no ORQ1**:
```
/multica-dev-transition-backend-1   id=dc719a1bf4e7  project=multica-dev-transition  service=backend
/multica-dev-transition-frontend-1  id=1b3b6c9ee32a  project=multica-dev-transition  service=frontend
/multica-dev-transition-postgres-1  id=2a4a84897363  project=multica-dev-transition  service=postgres
/omniroute                          id=2fb3fd57e885  project=<no value>              service=<no value>
```
O projeto/servico alvo existe **no ORQ1**: `project=multica-dev-transition`, `service=frontend`.
O ORQ2 **nao** hospeda esses containers; ele hospeda a user unit
`multica-orq1-backend-tunnel.service`, que e apenas tunel para `ORQ1:18080`. Confirmo que a
observacao e sobre o Docker real do ORQ1 e **nao** sobre o host do tunel.

`omniroute` nao pertence ao projeto compose (`<no value>`), logo esta fora do escopo de qualquer
`docker compose ... frontend`.

## 4. NENHUM CONTAINER ALEM DO FRONTEND FOI RECRIADO

IDs imutaveis por instancia, comparados entre T0 e T6:
```
backend  dc719a1bf4e7  inalterado   (started 2026-07-27T10:55:19Z)
frontend 1b3b6c9ee32a  inalterado   (started 2026-07-27T03:11:24Z)
postgres 2a4a84897363  inalterado   (started 2026-07-21T17:10:49Z)
omniroute 2fb3fd57e885 inalterado   (started 2026-07-24T18:36:12Z)
```
Recriacao troca o container ID. **Nenhum ID mudou**, portanto **nenhum container foi recriado** na
janela — nem o frontend. Backend, Postgres e OmniRoute permanecem intocados.

## 5. PRE-CONDICOES DE ROLLBACK (adendo do monitor) - **PASS**

### 5.1 Cronologia observada
- T1 15:48:19Z — varredura nao encontrou diretorio de rollback nem copia de compose/override criada
  apos 14:00. Pre-condicao **ainda nao satisfeita**.
- T4 15:49:21Z — apareceu
  `/home/ec2-user/.local/state/orq17-stage1-frontend-recovery-20260727T155000Z`.
- T5 15:49:32Z — verificacao completa por metadado: **PASS**.

Nao emiti STOP em T1: **nenhuma mutacao havia comecado** (IDs de container inalterados, imagens
intactas), logo a ausencia de backup em preflight era pendencia, nao violacao.

### 5.2 Verificacao por METADADO (nunca abri conteudo de backup)
```
dir  mode=700 owner=ec2-user:ec2-user                    <- 0700 exigido: OK
files/                       mode=700
files/docker-compose.selfhost.yml        mode=600 size=6177
files/docker-compose.selfhost.build.yml  mode=600 size=593
files/images.yml                         mode=600 size=137
files/backend-env.override.yml           mode=600 size=191
tailscale-serve-status.before.json       mode=600 size=3
frontend-safe-metadata.before.txt        mode=600 size=626
SHA256SUMS                               mode=600 size=1043
find -type f ! -perm 600  -> vazio       (todos os arquivos 0600: OK)
find -type d ! -perm 700  -> vazio       (todos os dirs 0700: OK)
```
As **quatro** copias de compose/override correspondem exatamente aos 4 `config_files` do projeto
declarados no label do container. Cobertura completa.

### 5.3 Checksums seguros - **6 de 6 OK, 0 FAILED**
```
sha256sum -c SHA256SUMS
  files/docker-compose.selfhost.yml: OK
  files/docker-compose.selfhost.build.yml: OK
  files/images.yml: OK
  files/backend-env.override.yml: OK
  tailscale-serve-status.before.json: OK
  frontend-safe-metadata.before.txt: OK
rc=0   |  OK=6  FAILED=0
```
`sha256sum -c` imprime somente `OK`/`FAILED`; **nenhum byte de conteudo** foi exibido. Nao abri
nenhum backup com `cat`, `head`, `tail`, `less`, `xxd` ou `od`.

### 5.4 Imagem anterior do frontend preservada - **PASS**
```
multica-web:reasoning-ui-20260727T013129Z  efd883f5bc91
multica-web:20260724-web-cf8017e3d2fd      cf8017e3d2fd
multica-web:transition-6a2aba3             cf8017e3d2fd
container frontend atual  -> image sha256:cf8017e3d2fd2b19e
```
A imagem em uso (`cf8017e3d2fd`) esta **tagueada duas vezes**
(`20260724-web-cf8017e3d2fd` e `transition-6a2aba3`), logo sobrevive a um rebuild que reaproveite a
tag `latest`. A imagem `reasoning-ui-20260727T013129Z` (`efd883f5bc91`) tambem esta preservada.
Existe alvo de rollback de imagem identificavel por ID.

Observacao factual, sem juizo: `cf8017e3d2fd` e o mesmo identificador do bundle live citado no ORQ-26.
Registro a coincidencia porque pode ser relevante ao objetivo da recuperacao; **nao** avalio aqui.

### 5.5 Nenhum prune - **PASS**
`docker images -f dangling=true` devolveu **4** imagens dangling em T1 e **4** em T4. Um
`docker image prune` as teria removido. Contagem estavel = **nenhum prune ocorreu**.

---

## 6. CONDICOES DE STOP - NENHUMA ACIONADA

| condicao | estado |
|---|---|
| `serve` deixar de estar vazio | nao ocorreu |
| `funnel` deixar de estar vazio | nao ocorreu |
| listener aparecer em `:443` | nao ocorreu |
| `18080` `/health` ou `/readyz` sair de `200` | nao ocorreu |
| container **diferente do frontend** ser recriado | nao ocorreu |
| executor mirar ORQ2/tunel em vez do Docker do ORQ1 | nao ocorreu |
| executor mutar **antes** de os backups passarem | **nao ocorreu** — backups passaram em T5, ainda sem mutacao |
| `prune` de imagem | nao ocorreu |
| ausencia de rollback **no momento de mutar** | nao aplicavel: backups passaram antes |

Emitirei STOP imediato se qualquer uma virar.

---

## 7. ESTADO E PROXIMO PASSO

**Aguardando o sinal de backup-ready do executor.** Do lado do monitor, as pre-condicoes de rollback
**ja passam** (secao 5) e os invariantes estao verdes (secao 2). Continuo em observacao read-only.

Ponto que o executor precisa saber antes de agir: `13100` **ja e `200`** com o container original.
O criterio de sucesso da recuperacao deve ser mudanca de container ID e/ou de imagem, nao o `200`.

## 8. NAO-AFIRMACOES
- READ-ONLY: nao buildei, nao reiniciei, nao recriei, nao editei arquivo, nao apliquei nem resetei
  `serve`/`funnel`, nao fiz login, nao li segredo, nao mutei DB, board nem AWS. Nenhum `docker
  compose`, `docker restart`, `docker build` ou `prune` partiu de mim.
- **Nao abri conteudo de nenhum backup.** Toda verificacao foi por `stat`, `find` e `sha256sum -c`,
  que imprime apenas `OK`/`FAILED`.
- **Nao inspecionei `Config.Env` de nenhum container**, conforme a restricao. Os `docker inspect`
  que usei extraem somente `.Name`, `.Id`, `.State.StartedAt`, `.Image` e os labels
  `com.docker.compose.project`/`service`.
- Nao verifiquei o conteudo de `frontend-safe-metadata.before.txt` nem de
  `tailscale-serve-status.before.json`: apenas nome, modo, tamanho e checksum.
- A janela observada e **curta** (~2 min, 7 amostras). Nao afirmo nada sobre periodos anteriores a
  15:47:55Z nem posteriores a 15:49:49Z.
- Nao sei se o executor pretende reaproveitar tag ou criar tag nova; a analise de 5.4 e sobre a
  **existencia** de alvo de rollback por ID, nao sobre o plano dele.
- Nao avaliei o merito da recuperacao do frontend nem a relacao com o ORQ-26; sou monitor, nao
  revisor de plano.
- Nao criei, atribui nem comentei issue alguma, e nao enfileirei task.
