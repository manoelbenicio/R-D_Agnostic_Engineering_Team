# Wave A — Execução da contenção de segurança autorizada (/tmp + quarentena)

Executor: Codex56#A (`w7:p3`) · UTC 2026-07-27T12:31–12:33Z
Autorização: OWNER APPROVED SECURITY REMEDIATION — WAVE A CONTAINMENT (não disruptiva)
Base: `.deploy-control/p0/evidence/gtl-tmp-umask-remediation-plan.md` + peer review PASS
`gtl-umask-security-peer-review.md:8` ("PASS, aprovado com recomendação de sequência segura")

**Nenhum valor de segredo foi lido, impresso, copiado ou transmitido.** Toda classificação usa
**nome de chave** e contagem (`grep -o` restrito ao nome, `grep -c`). Nada foi deletado, nada
rotacionado, nenhum serviço reiniciado, nenhum arquivo de outro dono tocado.

## 1. Pré-execução read-only (gate de parada)

| Verificação | Comando | Resultado |
|---|---|---|
| dono/modo dos alvos | `stat -c '%a %U:%G %s %y %n'` | 12 alvos ORQ2 todos `664 ec2-user:ec2-user`; usuário executor `ec2-user` ⇒ **dono igual, sem parada** |
| consumidor ativo (ORQ2) | `lsof <12 alvos>` | `LSOF_EXIT=1` = **nenhum descritor aberto** |
| consumidor por fd (ORQ2) | varredura `/proc/*/fd` | `open_fds=0` nos 12 |
| falso positivo de cmdline | `grep -l <nome> /proc/*/cmdline` | os 4 "hits" eram **o meu próprio shell** (pid=self); descartados |
| consumidor ativo (ORQ1) | `lsof /tmp/mh /tmp/mcp_h` | `LSOF_EXIT=1` |
| consumidor ativo (LOCAL) | `lsof` nos 16 alvos | `LSOF_EXIT=1` |
| root-owned | `stat /tmp/test_login.html` | `644 root:root` ⇒ **NÃO TOCADO** (gate de parada respeitado, escalado) |
| backups de rollback | `stat /tmp/daemon.environ.*.bak` | já `600 ec2-user:ec2-user` ⇒ **não alterados** |

## 2. Quarentena durável criada

```text
install -d -m 700 /home/ec2-user/.private-tmp
install -d -m 700 /home/ec2-user/.private-tmp/quarantine-20260727T123153Z
700 ec2-user:ec2-user /home/ec2-user/.private-tmp
700 ec2-user:ec2-user /home/ec2-user/.private-tmp/quarantine-20260727T123153Z
```

**Decisão técnica registrada:** a quarentena **não** foi criada sob `$HOME`, porque `$HOME` do pane é
`/home/ec2-user/.agent-cred-homes/slots/slot-139/home` (um slot de isolamento) e existe
`reap-cred-slots.service` ("Reap login credential slots older than 24h (not in use)"). Colocar
quarentena dentro de um slot a tornaria reapável, o que contraria "durável". Usei o home real
`/home/ec2-user`, que já é `drwx------`.

## 3. Manifest de execução — ORQ2 `ip-172-31-30-9`

### 3.1 Contenção in place (chmod 0600), mantidos em `/tmp`

| arquivo | modo antes → depois | por que ficou em `/tmp` | chaves (NOMES) |
|---|---|---|---|
| `/tmp/arch.txt` | `664 → 600` | ALTA com rotação pendente: mover não substitui rotação e mantê-lo no caminho conhecido facilita a Wave B | `OPENAI_API_KEY` (1 atribuição com valor) |
| `/tmp/backend-recover.sh` | `664 → 600` | ALTA com rotação pendente **e papel operacional** (script de recuperação citado em runbook): mover poderia quebrar procedimento de terceiro | `JWT_SECRET` (1 atribuição), `POSTGRES_PASSWORD` (nome) |
| `/tmp/mcp.json.bak` | `664 → 600` | artefato de rollback de configuração MCP; contenção sem deslocar rollback | — (nome de arquivo sensível) |
| `/tmp/end.txt` | `664 → 600` | **novo artefato**, criado 11:28:33Z (depois do inventário de 11:02): contido ao ser detectado | `DATABASE_URL` (0 atribuição com valor) |

### 3.2 Movidos para quarentena (zero consumidor, sem papel operacional)

Origem → destino (todos `-rw------- ec2-user:ec2-user` no destino):

```text
/tmp/sec.txt          -> /home/ec2-user/.private-tmp/quarantine-20260727T123153Z/sec.txt          (2409 B)
/tmp/o30.txt          -> .../quarantine-20260727T123153Z/o30.txt                                   (2256 B)
/tmp/ph.txt           -> .../quarantine-20260727T123153Z/ph.txt                                    (2222 B)
/tmp/dec.txt          -> .../quarantine-20260727T123153Z/dec.txt                                   (2008 B)
/tmp/f.txt            -> .../quarantine-20260727T123153Z/f.txt                                     (1818 B)
/tmp/sdk.txt          -> .../quarantine-20260727T123153Z/sdk.txt                                   (1413 B)
/tmp/proj.txt         -> .../quarantine-20260727T123153Z/proj.txt                                  (3590 B)
/tmp/e2eq.txt         -> .../quarantine-20260727T123153Z/e2eq.txt                                  (2823 B)
/tmp/f2-fulltest.log  -> .../quarantine-20260727T123153Z/f2-fulltest.log                           (5239 B)
```

`mv -n` (never overwrite) e `chmod 0600` aplicado **antes** do move, de modo que em nenhum instante o
arquivo existiu no destino com modo permissivo. Tamanhos e mtimes preservados (verificados por `ls -l`).

## 4. Manifest — ORQ1 `ip-172-31-18-217`

```text
/tmp/mh      664 -> 600   [Authorization]   (lsof: nenhum consumidor)
/tmp/mcp_h   664 -> 600   [Authorization]   (lsof: nenhum consumidor)
/tmp/test_login.html      644 root:root — NÃO TOCADO (dono diferente; gate de parada)
/tmp/daemon.environ.20260724T110622Z.bak  600 — inalterado (rollback path)
/tmp/daemon.environ.20260724T034949Z.bak  600 — inalterado (rollback path)
```

Nenhum move no ORQ1: os dois arquivos são configuração de header consumível por ferramenta local; a
contenção por modo é suficiente e não desloca caminho de terceiro.

## 5. Manifest — LOCAL `21LAPGLMVPJ4` (usuário `dataops-lab`)

Todos `644 → 600`, todos verificados sem consumidor (`lsof` exit 1), **nenhum move** (host de outro
operador; deslocar caminho poderia quebrar tooling dele):

```text
/tmp/handshake-token.txt  644 -> 600   <- ALTA (token cru; rotação obrigatória na Wave B)
/tmp/rev-token.txt        644 -> 600   <- ALTA (token cru; rotação obrigatória na Wave B)
/tmp/arch.txt             644 -> 600   <- ALTA (OPENAI_API_KEY)
/tmp/sec.txt  /tmp/o30.txt  /tmp/ph.txt  /tmp/dec.txt  /tmp/sdk.txt  /tmp/proj.txt  /tmp/e2eq.txt  644 -> 600
/tmp/av.sh  /tmp/raw.sh  /tmp/imp.sh  /tmp/sync.sh  /tmp/g36.sh  /tmp/idx.html                     644 -> 600
```

## 6. Verificação pós-execução

```text
ORQ2: find /tmp -maxdepth 1 -user ec2-user -perm /077 | xargs grep -l <NOMES DE CHAVE>
      -> antes: 12 alvos + novos;  depois: 1 residual = /tmp/unleash-repo-schema-v1-codeium-language-server.json (664)
ORQ1: mesma varredura -> (vazio)
ORQ2: total de arquivos >0600 em /tmp nível 1 = 209 (cache/ruído sem nome de chave)
LOCAL: total de arquivos >0600 do usuário em /tmp nível 1 = 193 (idem)
```

Residual declarado e **fora do escopo autorizado**: o `unleash-repo-schema-...json` é catálogo de
language server classificado BAIXA no plano (a recomendação para caches é **mover a origem via
TMPDIR privado**, não `chmod` item a item). Os 209/193 restantes não casam com nome de chave.

**Fato novo relevante:** `/tmp/end.txt` nasceu `664` às 11:28:33Z, isto é, **depois** do inventário
inicial e **antes** desta contenção. Isso é prova direta de que a contenção é temporária enquanto o
`umask 0002` (`/etc/bashrc:75`) permanecer: cada novo artefato renasce mundo-legível. Wave B/umask
continua sendo o único fechamento estrutural.

## 7. Plano da Wave B — por segredo, com dependências, gate de fila zero, rollback e referências

Nada da Wave B foi executado. Regra transversal: **nenhum `GetSecretValue`/`BatchGetSecretValue`**, e
nenhum valor em `argv`, `stdout`, log, diff ou evidência. Resolução somente dentro do processo filho
via `asm-exec` + referência dinâmica `{{resolve:secretsmanager:<id>:SecretString:<json-key>}}`
(`.kiro/steering/aws-agent-rules.md`, seção Secret Safety). Se o segredo não for AWS-managed, usar
mecanismo handle-only equivalente aprovado — não criar dependência AWS por inferência.

| # | Segredo (NOME) | Onde aparece | Dependências | Gate de fila zero | Procedimento (fechado) | Rollback |
|---|---|---|---|---|---|---|
| B1 | `OPENAI_API_KEY` | `/tmp/arch.txt` (ORQ2 + LOCAL) | nenhum consumidor de produto identificado; confirmar se alguma automação usa a chave antes de invalidar | não aplicável (não toca serviço) | rotacionar no provedor → atualizar o store (handle) → validar por chamada mínima do consumidor real → só então remover os dois arquivos | manter a chave antiga válida até validação; se falhar, revogar a nova e reter a antiga |
| B2 | `JWT_SECRET` | `/tmp/backend-recover.sh` (ORQ2) e `.env` do backend | **alto acoplamento**: trocar invalida todo token de sessão emitido; exige recreate do backend | `SELECT count(*) FROM agent_task_queue WHERE status IN ('queued','dispatched','running','waiting_local_directory')` = **0 em duas leituras** | gerar novo valor fora de contexto → gravar no store → recreate do backend consumindo referência → validar `/health` + login → remover marcador do script | restaurar referência anterior no store e recreate; `.env` 0600 atual é o ponto de retorno |
| B3 | `POSTGRES_PASSWORD` / `DATABASE_URL` | `/tmp/backend-recover.sh`, `/tmp/end.txt`, `/tmp/sec.txt` (quarentena) | **o mais disruptivo**: exige `ALTER ROLE` + recreate de backend e daemon; owner já registrou que expôs `DATABASE_URL` no histórico | mesmo gate de fila zero + zero conexão ativa de aplicação | rotacionar senha do role → atualizar store → recreate backend → recreate daemon → validar `/health` e uma task sintética por runtime obrigatório (AGY/Codex/Kiro) | manter role antigo válido durante a janela; reverter store e recreate; migrations não são tocadas |
| B4 | token cru `handshake` | `/tmp/handshake-token.txt` (LOCAL) | escopo: pareamento/pane; confirmar emissor antes de invalidar | não aplicável | invalidar no emissor → reemitir sob `0600` fora de `/tmp` → remover arquivo | reemitir token anterior se o novo não parear |
| B5 | token cru `rev` | `/tmp/rev-token.txt` (LOCAL) | idem B4 | não aplicável | idem B4 | idem B4 |
| B6 | `MULTICA_TOKEN` (PAT task-scoped) | nome citado em `/tmp/sec.txt` (quarentena) | PATs são task-scoped e de curta vida; verificar se algum ainda é válido antes de agir | não aplicável | se algum PAT ainda válido: revogar por `id` no backend; nenhum valor é lido | PAT é reemitível por task; sem rollback necessário |
| B7 | `Authorization`/`Bearer` de MCP | `/tmp/mh`, `/tmp/mcp_h` (ORQ1) | ferramenta MCP local | não aplicável | mover para arquivo `0600` fora de `/tmp`, apontar a ferramenta por caminho, e rotacionar o token no provedor MCP | manter arquivo antigo até a ferramenta validar |

Ordem recomendada: **B4/B5 → B1 → B2 → B3**, do menor para o maior blast radius, e **B3 apenas em
janela com fila zero comprovada**. Estrutural em paralelo (não é segredo): `umask 077` por drop-in de
unit de usuário e `TMPDIR`/`GOCACHE`/`GOTMPDIR` privados — sem isso, novos `end.txt` continuam nascendo.

Cada item da Wave B é STOP-AND-WAIT: exige autorização própria do owner por janela, porque envolve
credencial e recreate de serviço.

## 8. Não-alegações

- Não deletei arquivo algum; não rotacionei nenhuma credencial; não reiniciei serviço ou container;
  não alterei `.env`, unit systemd, `umask` de sistema ou config de ferramenta.
- Não li conteúdo de segredo: apenas `stat`, `lsof`, `grep -l`, `grep -c` e `grep -o` restrito a
  **nomes de chave**. Nenhuma linha de conteúdo foi impressa.
- Não toquei arquivo de outro dono (`/tmp/test_login.html`, root) nem os backups `0600` de rollback.
- Não contive os 209 (ORQ2) / 193 (LOCAL) arquivos `>0600` restantes sem nome de chave: fora do
  escopo "artefatos sensíveis confirmados" desta autorização.
- Não afirmo que os valores contidos sejam segredos vivos: a classificação é por **marcador de nome**
  e por contagem de atribuição, como no plano original.
- Não executei nada da Wave B.
