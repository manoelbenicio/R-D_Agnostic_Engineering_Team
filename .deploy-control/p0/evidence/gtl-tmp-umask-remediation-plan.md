# GTL-04 — Plano de remediacao /tmp + umask (AUDITORIA READ-ONLY, NADA APLICADO)

Auditor: Codex56#A (ORQ2, pane `w7:p3`) · medicoes UTC 2026-07-27T11:27–11:33Z
Executor previsto: Codex56-TL (`w5:pC`), escritor unico. Eu nao mutei permissao, arquivo, servico nem
segredo. Nenhum valor de segredo foi lido: so `stat`, `find`, `grep -l/-c` e nomes de chave.

## 1. Confirmacao: o que AINDA existe (remedido agora, nao herdado do relatorio anterior)

ORQ2 `ip-172-31-30-9` — `umask=0002`, todos ainda `664 ec2-user:ec2-user`:

```text
664 2586 /tmp/arch.txt            [OPENAI_API_KEY= com valor]      <- ALTA
664 2739 /tmp/backend-recover.sh  [JWT_SECRET= com valor, POSTGRES_PASSWORD]  <- ALTA
664 2409 /tmp/sec.txt   664 2256 /tmp/o30.txt   664 2222 /tmp/ph.txt   664 2008 /tmp/dec.txt
664 1818 /tmp/f.txt     664 1413 /tmp/sdk.txt   664 3590 /tmp/proj.txt 664 2823 /tmp/e2eq.txt
664 5239 /tmp/f2-fulltest.log     664  327 /tmp/mcp.json.bak
```

ORQ1 `ip-172-31-18-217` — `umask=0002`:

```text
664 1387 ec2-user /tmp/mh          [Authorization]
664 1387 ec2-user /tmp/mcp_h       [Authorization]
644 477429 root:root /tmp/test_login.html   [Authorization, Bearer, PASSWORD, TOKEN]
600 2449 ec2-user /tmp/daemon.environ.20260724T110622Z.bak   <- correto, nao tocar
```

LOCAL `21LAPGLMVPJ4` (WSL2 Ubuntu, systemd 255) — `umask=0022`:

```text
644 dataops-lab:root   24 /tmp/handshake-token.txt   <- ALTA (valor cru)
644 dataops-lab:root   24 /tmp/rev-token.txt         <- ALTA (valor cru)
644 dataops-lab:root 2586 /tmp/arch.txt              <- ALTA (OPENAI_API_KEY=)
644 2409 /tmp/sec.txt  644 1286 /tmp/av.sh  644 462 /tmp/raw.sh  644 1219 /tmp/imp.sh
644 1402 /tmp/sync.sh  644  599 /tmp/g36.sh  644 176102 /tmp/idx.html
```

Nada foi corrigido desde a auditoria das 11:02Z: os 4 itens ALTA persistem.

## 2. Causa exata do 0002 (linha e trecho)

`/etc/bashrc` no ORQ2 e no ORQ1, linhas 70-77:

```text
70:    # By default, we want umask to get set. This sets it for non-login shell.
75:       umask 002
77:       umask 022
```

E o padrao Amazon Linux: `umask 002` quando `UID > 199` e o usuario tem grupo privado
(`/etc/login.defs:270 USERGROUPS_ENAB yes`). O `/etc/login.defs:117 UMASK 022` **nao vale** para essas
sessoes porque o `bashrc` sobrepoe depois. `pam_umask` existe, mas so em `/etc/pam.d/postlogin`
(`session optional pam_umask.so silent`), logo tambem e sobreposto pelo `bashrc`.
Consequencia: **mudar `login.defs` ou `pam_umask` NAO resolve** — o `bashrc` ganha.

Contexto de execucao medido: o daemon atual roda no ORQ2 como PID **3417665**
(`multica-auth-credential-home-v1 daemon start --foreground ... --daemon-id orq2-credential-runtime-v1`),
e **e gerenciado por systemd de USUARIO** (corrigido apos nova medicao):

```text
$ cat /proc/3417665/cgroup
0::/user.slice/user-1000.slice/user@1000.service/app.slice/multica-daemon-orq2-credential.service
$ systemctl --user list-units --all | grep multica
multica-daemon-orq2-credential.service   loaded active running  Multica credential-isolated daemon on ORQ2
multica-orq1-backend-tunnel.service      loaded active running  Multica ORQ2 to ORQ1 backend tunnel
```

Ha tambem `reap-cred-slots.service` (system scope, inactive) e `herdr.service` (inactive dead).
O PID 3244391 do ORQ1 nao existe mais. **Existe unit, e ela e de escopo de usuario** — a Camada B se
aplica direto, com drop-in de usuario (secao 3).

## 3. Mecanismo duravel para umask 077 sem quebrar build — 4 camadas, ordem de risco crescente

**Camada A (recomendada primeiro; nao depende de umask e nao quebra nada): TMPDIR privado.**
Um diretorio `0700` por agente torna irrelevante o modo do arquivo, porque ninguem atravessa o
diretorio. Resolve a exposicao inteira de `/tmp` de uma vez.

```text
install -d -m 700 "$HOME/.private-tmp"                 # dono unico, sem grupo/outros
export TMPDIR="$HOME/.private-tmp"
export GOCACHE="$HOME/.private-tmp/gocache"            # tira os 26 caches Go de /tmp
export GOTMPDIR="$HOME/.private-tmp"
export XDG_CACHE_HOME="$HOME/.private-tmp/xdg-cache"
```
Onde colocar de forma duravel: no wrapper/launcher dos panes de agente (mesmo lugar onde hoje se define
`HOME` por slot — os slots ja provam que esse mecanismo existe e funciona). Nao mexe em arquivo global.

**Camada B: drop-in systemd `UMask=0077` para servico.** systemd 252 (ORQ1/ORQ2) e 255 (LOCAL)
suportam `UMask=`. O daemon do ORQ2 roda em unit de **usuario**
(`multica-daemon-orq2-credential.service` sob `user@1000`), portanto o drop-in correto e de usuario:

```text
~/.config/systemd/user/multica-daemon-orq2-credential.service.d/10-umask.conf
[Service]
UMask=0077
```
Mesmo padrao aplicavel a `multica-orq1-backend-tunnel.service` (usuario) e a `herdr.service` (sistema,
hoje inativo). Ativacao: `systemctl --user daemon-reload` + restart da unit — **restart e STOP-AND-WAIT
e nao foi feito**.

**Camada C: `/etc/profile.d/zz-agent-umask.sh` com `umask 077`.** ATENCAO: precisa ser validado
**antes** de aplicar, porque `/etc/bashrc:75` roda depois em shell interativo non-login e pode
sobrepor. Teste de aceite proposto: `bash -lc umask` e `bash -ic umask` devem retornar `0077` nos dois
casos; se `bash -ic` retornar `0002`, a camada C nao serve sozinha.

**Camada D (nao recomendada isolada): `login.defs`/`pam_umask`.** Comprovadamente sobreposta pelo
`bashrc` (secao 2). Só faz sentido junto com edicao do `/etc/bashrc`, que e mudanca global de host.

### Riscos concretos de `umask 077` que podem quebrar build (mitigacao)

| risco | por que | mitigacao |
|---|---|---|
| bind mount Docker com UID diferente | arquivo `0600` do host fica ilegivel para o UID do container | manter artefato de build fora do umask restrito ou alinhar UID; validar `docker compose build` antes/depois |
| cache compartilhado em `/tmp` entre usuarios/CI | 26 entradas Go em 13 diretorios `*gocache*` hoje `664` | Camada A move cache para dir privado por agente: cada um passa a ter o seu |
| artefato consumido por outro usuario (ex.: `root`) | `ORQ1 /tmp/test_login.html` e `root:root` | nao aplicar chmod cego em arquivo de outro dono; escalar |
| script que grava e outro agente le | 7 agentes em paralelo no mesmo host | usar `.deploy-control/` (repo) para troca entre agentes, nunca `/tmp` |

## 4. Classificacao: ROTACAO obrigatoria vs chmod/move

**ROTACAO obrigatoria (chmod nao resolve: o valor ja esteve mundo-legivel por >21 h em `/tmp` 1777).**

| host | arquivo | por que rotacao |
|---|---|---|
| LOCAL | `/tmp/handshake-token.txt` | 24 bytes, `grep -c '[A-Z_]{4,}='` = 0 → o arquivo E o valor; mtime 2026-07-26 13:12 -03 |
| LOCAL | `/tmp/rev-token.txt` | idem; mtime 2026-07-26 13:56 -03 |
| ORQ2+LOCAL | `/tmp/arch.txt` | `OPENAI_API_KEY=` com valor (1 atribuicao) |
| ORQ2 | `/tmp/backend-recover.sh` | `JWT_SECRET=` com valor (1 atribuicao) + nome `POSTGRES_PASSWORD` |

Ordem correta para esses 4: **rotacionar primeiro, depois remover o arquivo**. `chmod 600` neles e
apenas contencao temporaria, nao remediacao.

**chmod 600 ou move para dir privado (nome de chave presente, zero atribuicao com valor detectada — o
TL confirma caso a caso, eu nao posso distinguir sem ler valor).**

```text
ORQ2 : sec.txt, o30.txt, ph.txt, dec.txt, f.txt, sdk.txt, proj.txt, e2eq.txt, f2-fulltest.log, mcp.json.bak
ORQ1 : mh, mcp_h
LOCAL: sec.txt, av.sh, raw.sh, imp.sh, sync.sh, g36.sh, idx.html
```

**Move, nao chmod (volume alto, sao caches):** 26 entradas em 13 diretorios `*gocache*` no ORQ2, o
`unleash-repo-schema-...json`, e `/tmp/uO-nHg60RfVLctF_qlu8N/client` (8) + `/tmp/JTFt9721FtNoKDB1Hq9-Z/client` (1).
Camada A resolve na origem; `chmod -R` em cache e trabalho perdido.

**Escalar, nao tocar:** `ORQ1 /tmp/test_login.html` (`root:root`) — outro dono, exige decisao separada.
**Nao tocar:** `ORQ1 /tmp/daemon.environ.*.bak` — ja estao `0600` e sao caminho de rollback.

## 5. Comandos seguros propostos (sem segredo em argv nem em stdout) — para o TL executar apos autorizacao

Inventario e verificacao pos-fix (mesma tecnica que usei; nunca `cat`, nunca `grep -o` de valor):

```text
find /tmp -maxdepth 3 -type f -size -4M -perm /077 -user "$(id -un)" -printf '%m %u:%g %s %p\n'
find /tmp -maxdepth 3 -type f -perm /077 -user "$(id -un)" -print0 \
  | xargs -0 -r grep -l -I -E 'JWT_SECRET|POSTGRES_PASSWORD|MULTICA_TOKEN|API_KEY|SECRET|TOKEN|PASSWORD'
```

Contencao em massa **sem citar nome de segredo em argv** (o `find` seleciona, o `chmod` nao ve nome):

```text
find /tmp -maxdepth 1 -type f -user "$(id -un)" -perm /077 -exec chmod 600 {} +
```

Mover para dir privado (o `mv` nao abre o conteudo, entao nao ha vazamento em stdout):

```text
install -d -m 700 "$HOME/.private-tmp/quarantine-$(date -u +%Y%m%dT%H%M%SZ)"
mv /tmp/<arquivo> "$HOME/.private-tmp/quarantine-.../"
```

Rotacao sem expor valor:

```text
# segredo gerenciado: resolver dentro do processo filho, nunca em argv/stdout
asm-exec -- <comando> '{{resolve:secretsmanager:<id>:SecretString:<json-key>}}'
# HTTP autenticado: credencial em arquivo 0600, nunca em -H na linha de comando
curl --config /caminho/0600.curlrc https://...        # o arquivo contem: header = "Authorization: ..."
```

Proibicoes explicitas para quem executar: nao usar `cat`/`head` nos arquivos ALTA; nao `echo`/`export`
de valor; nao `grep -o` que imprima valor; nao passar token em `-H`, `--data` ou variavel na linha de
comando (fica em `ps` e no history); preferir `--config`/`.netrc` `0600`.

Aceite proposto (verificavel sem ler valor): (i) `find ... -perm /077` no `/tmp` do dono retorna 0 para
os arquivos listados; (ii) `bash -lc umask` e `bash -ic umask` retornam `0077` na sessao de agente;
(iii) `echo $TMPDIR` aponta para dir `0700`; (iv) `go build ./...` e `docker compose build` seguem exit
0 depois da mudanca; (v) os 4 itens ALTA aparecem como rotacionados no registro do TL.

## 6. Nao-alegacoes

- Nada mutado: nenhuma permissao, arquivo, servico, unit, `umask`, `.env` ou segredo. Escrita unica:
  este documento + meu check-out.
- Nenhum valor de segredo lido. Onde digo "com valor", a prova e **contagem** (`grep -c`), nunca conteudo.
- Nao sei, sem ler valor, se os itens de classificacao chmod/move contem segredo real — o TL confirma.
- Nao validei a Camada C (ordem `profile.d` vs `/etc/bashrc`) empiricamente: propus o teste de aceite
  em vez de afirmar que funciona.
- Escopo: `/tmp`, `maxdepth 3`, arquivos regulares, `<=4 MB`, nos 3 hosts alcancaveis. Nao varri
  `/var/tmp`, `/dev/shm`, HOMEs, volumes de container nem a maquina Windows do owner.
