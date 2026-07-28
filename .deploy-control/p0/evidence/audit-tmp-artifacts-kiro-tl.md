# Auditoria de permissao dos artefatos em /tmp (KIRO-PRINCIPAL-TL) — 2026-07-27

Auditor: Codex56#A (ORQ2, pane `w7:p3`) · UTC 2026-07-27T11:02Z
Metodo: varredura por **NOME DE CHAVE**. Nenhum valor de segredo foi lido, impresso, copiado ou
transmitido — usei `grep -l` (so nome do arquivo), `grep -o` limitado ao **nome da chave** e `grep -c`
(contagem). Nenhuma permissao alterada. Hosts alcancados: ORQ2 `ip-172-31-30-9`, ORQ1
`ip-172-31-18-217` (`100.118.244.61`), LOCAL `21LAPGLMVPJ4` (`dataops-lab@100.117.245.15`).

## 0. CORRECAO DA PREMISSA: `/tmp/daemon.env.bak` nao existe em nenhum host que eu alcanco

```text
ORQ2  : ABSENT /tmp/daemon.env.bak   ABSENT /tmp/daemon.cmd.bak
ORQ1  : ABSENT /tmp/daemon.env.bak   ABSENT /tmp/daemon.cmd.bak
LOCAL : ABSENT /tmp/daemon.env.bak   ABSENT /tmp/daemon.cmd.bak
```

O que existe no ORQ1, e esta **correto**:

```text
-rw------- 600 ec2-user:ec2-user 2449 /tmp/daemon.environ.20260724T110622Z.bak
-rw------- 600 ec2-user:ec2-user 2449 /tmp/daemon.environ.20260724T034949Z.bak
```

Ou seja: ou o `daemon.env.bak` 0664 foi removido/renomeado para esses `.environ.*.bak` 0600, ou esta em
host que eu nao alcanco. **Nao ha, hoje, backup de env de daemon com permissao frouxa nos 3 hosts.**
Isso nao anula o resto do achado: ha artefato frouxo, so nao e esse arquivo.

## 1. CAUSA RAIZ SISTEMICA: umask permissiva + /tmp mundo-legivel

```text
ORQ2  UMASK=0002  /tmp = drwxrwxrwt (1777)   -> arquivo nasce 664 (grupo+outros leem)
ORQ1  UMASK=0002  /tmp = drwxrwxrwt (1777)   -> arquivo nasce 664
LOCAL UMASK=0022  /tmp = drwxrwxrwt (1777)   -> arquivo nasce 644
```

Como `/tmp` e 1777, **qualquer usuario ou processo local do host le qualquer arquivo 0644/0664 ali**.
Volume do problema no ORQ2: `find /tmp -maxdepth 3 -type f -size -4M -perm /077 | wc -l` = **74537**
arquivos com bit de grupo/outros (a maioria e cache de build/agent, mas o default e o problema).
Corrigir arquivo por arquivo sem corrigir o `umask` do shell dos agentes so reintroduz o defeito no
proximo artefato.

## 2. ORQ2 — arquivos com modo > 0600 e nome de chave sensivel (exceto caches `client/`)

Prioridade ALTA (unicos onde **provei** padrao `CHAVE=<valor>`, por contagem, sem ler o valor):

| modo | dono | bytes | arquivo | chaves (nomes) | atribuicoes com valor |
|---|---|---:|---|---|---:|
| 664 | ec2-user:ec2-user | 2586 | `/tmp/arch.txt` | API_KEY, Bearer, **OPENAI_API_KEY=** | 1 |
| 664 | ec2-user:ec2-user | 2739 | `/tmp/backend-recover.sh` | **JWT_SECRET=**, POSTGRES_PASSWORD | 1 |

Prioridade MEDIA (nome de chave presente, zero atribuicao com valor detectada — provavelmente prosa de
auditoria, mas o TL deve confirmar porque eu **nao** li conteudo):

| modo | bytes | arquivo | chaves (nomes) |
|---|---:|---|---|
| 664 | 2409 | `/tmp/sec.txt` | DATABASE_URL, JWT_SECRET, MULTICA_TOKEN, POSTGRES_PASSWORD |
| 664 | 2256 | `/tmp/o30.txt` | JWT_SECRET |
| 664 | 2222 | `/tmp/ph.txt` | JWT_SECRET |
| 664 | 2008 | `/tmp/dec.txt` | JWT_SECRET, SECRET |
| 664 | 1818 | `/tmp/f.txt` | OPENAI_API_KEY |
| 664 | 1413 | `/tmp/sdk.txt` | API_KEY, TOKEN |
| 664 | 3590 | `/tmp/proj.txt` | TOKEN |
| 664 | 2823 | `/tmp/e2eq.txt` | SECRET |
| 664 | 5239 | `/tmp/f2-fulltest.log` | JWT_SECRET |
| 664 |  327 | `/tmp/mcp.json.bak` | (nome de arquivo sensivel; sem chave com valor) |

Prioridade BAIXA (fixture/cache de build, mesmo hash repetido em 12 diretorios `*gocache*`, chave so
`Authorization`):

```text
664 258 bytes  /tmp/{gocache,base-gocache,kgc,c3-gocache,agent-brain-C1-gocache,gocache_c9,c4-gocache,
               gocache-c8,f6-gocache,f4-gocache,gocache-f8,f1-gocache,l5-gocache}/d9/d9b41f03...-d
664 479 bytes  os mesmos diretorios .../c8/c848a2a5...-d
664 463641     /tmp/unleash-repo-schema-v1-codeium-language-server.json        [TOKEN]
```

Caches de client de agente (nome de chave presente, 664/644):

```text
664 ec2-user:ec2-user /tmp/uO-nHg60RfVLctF_qlu8N/client/{0027c7b1,15fa3396,dc843294,6f7fc400,1817badf,
                                                        344cb0ae,4eab1b0e,c4b4bc3c}
644 ec2-user:ec2-user /tmp/JTFt9721FtNoKDB1Hq9-Z/client/12fea4ce
```

## 3. ORQ1 — apenas 3 arquivos > 0600 com nome de chave

```text
664 ec2-user:ec2-user   1387 /tmp/mh       [Authorization]  bearer_with_value=0  (1 linha authorization)
664 ec2-user:ec2-user   1387 /tmp/mcp_h    [Authorization]  bearer_with_value=0  (conteudo != mh, sha256 difere)
644 root:root         477429 /tmp/test_login.html  [Authorization, Bearer, PASSWORD, TOKEN]  (07-24 18:34, dono root)
```

Boa noticia no ORQ1: os scripts `av.sh`, `raw.sh`, `imp.sh`, `sync.sh` estao **0600** e os backups de
env do daemon estao **0600** (secao 0). O total de candidatos frouxos em /tmp no ORQ1 e 127, contra
74537 no ORQ2.

## 4. LOCAL (`dataops-lab@21LAPGLMVPJ4`) — 16 arquivos 0644 com nome de chave

Prioridade ALTA:

```text
644 dataops-lab:root   24 /tmp/handshake-token.txt   mtime 2026-07-26 13:12:33 -0300
644 dataops-lab:root   24 /tmp/rev-token.txt         mtime 2026-07-26 13:56:12 -0300
```

Os dois tem **24 bytes e zero nome de chave dentro** (`grep -c '[A-Z_]{4,}='` = 0): pelo nome e pelo
tamanho, o arquivo **e o proprio valor do token**, mundo-legivel. Nao abri nenhum dos dois.

```text
644 2586 /tmp/arch.txt                [OPENAI_API_KEY= com valor, 1 atribuicao]   <- ALTA
```

Prioridade MEDIA (mesmos nomes do ORQ2, agora 644):

```text
644 2409 /tmp/sec.txt   [DATABASE_URL, JWT_SECRET, MULTICA_TOKEN, POSTGRES_PASSWORD]
644 2256 /tmp/o30.txt   [JWT_SECRET]        644 2222 /tmp/ph.txt   [JWT_SECRET]
644 2008 /tmp/dec.txt   [JWT_SECRET, SECRET] 644 1413 /tmp/sdk.txt [API_KEY, TOKEN]
644 3590 /tmp/proj.txt  [TOKEN]              644 2823 /tmp/e2eq.txt [SECRET]
644 1286 /tmp/av.sh     [PASSWORD]           644  462 /tmp/raw.sh   [PASSWORD]
644 1219 /tmp/imp.sh    [PASSWORD]           644 1402 /tmp/sync.sh  [PASSWORD]
644  599 /tmp/g36.sh    [Authorization, Bearer]
644 176102 /tmp/idx.html [Bearer]
```

Nota: `av.sh`, `raw.sh`, `imp.sh`, `sync.sh` existem nos dois hosts — **0600 no ORQ1 e 0644 no LOCAL**.
O mesmo script, dois modos. Isso e efeito direto do umask por host (secao 1).

## 5. Recomendacao para o Codex56-TL (escritor unico; eu nao executo nada disto)

1. Tratar como ALTA e primeiro: `LOCAL /tmp/handshake-token.txt`, `LOCAL /tmp/rev-token.txt`,
   `arch.txt` (ORQ2 e LOCAL, `OPENAI_API_KEY=`), `ORQ2 /tmp/backend-recover.sh` (`JWT_SECRET=`).
   Para token cru em arquivo, `chmod 600` **nao basta**: se o valor e um segredo vivo, o correto e
   rotacionar e depois remover o arquivo.
2. Corrigir a causa raiz antes do resto: `umask 077` no ambiente dos agentes (ORQ2 e ORQ1 estao em
   `0002`). Sem isso, o proximo artefato nasce 664 outra vez.
3. Ordem sugerida: rotacionar o que for segredo vivo → `chmod 600` no restante da lista →
   `umask` → remover artefatos que nao sao mais necessarios.
4. Decisao do owner: `chmod`/remocao em massa e mudanca de permissao/credencial, classe STOP-AND-WAIT
   do AUTHORITY-AMENDMENT-001. Eu so listei.

## 6. Nao-alegacoes

- Nao alterei nenhuma permissao, nao removi nem editei nenhum arquivo, nao compilei, nao reiniciei
  nada, nao toquei em `.env`, em `/tmp/daemon.env.bak` (inexistente) nem em arquivo do repo alem
  deste documento e do meu check-out.
- **Nao li o valor de nenhum segredo.** Usei `grep -l`, `grep -c` e `grep -o` restrito ao nome da
  chave. Onde afirmo "atribuicao com valor", a prova e uma **contagem**, nunca o conteudo.
- Nao consigo distinguir, sem ler valor, se os arquivos de prioridade MEDIA contem segredo real ou
  apenas o nome da chave em prosa de auditoria. O TL deve confirmar caso a caso.
- Escopo: `/tmp` com `maxdepth 3`, arquivos regulares, `<= 4 MB`, apenas nos 3 hosts citados. Nao
  varri `/var/tmp`, `/dev/shm`, HOME dos usuarios, volumes de container nem a maquina Windows do owner.
- Arquivos de dono `root` (ex.: `ORQ1 /tmp/test_login.html`) nao foram criados por voce; incluo por
  completude do risco, nao como atribuicao de autoria.
