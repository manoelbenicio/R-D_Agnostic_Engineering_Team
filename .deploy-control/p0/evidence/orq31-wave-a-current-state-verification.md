# ORQ-31 — verificação factual do estado atual da contenção Wave A (READ-ONLY, **auto-auditoria do autor**)

- Card: **ORQ-31** `f7e13350-c7f2-4335-8a04-01b98527d034` — "Security Wave A Containment (Permissions & Quarantine)", status `in_progress`
- Executor desta verificação: Codex56#A (`w7:p3`) · UTC 2026-07-28T16:04Z
- **Independência**: eu sou o autor da contenção original
  (`gtl-security-remediation-wave-a-execution.md:3`, executor Codex56#A, 2026-07-27T12:31–12:33Z).
  Por ruling do GTL, **não emito aceite**. Este documento é insumo medido para o revisor independente.
- Modo: somente leitura — `stat`, `find`, `grep`, `umask`. Nenhum `chmod`, `mv`, escrita, leitura de
  conteúdo de arquivo ou de segredo. Todos os itens são referenciados por **caminho e metadado**;
  nenhum valor de chave, token ou header aparece aqui.

## 1. ORQ2 `ip-172-31-30-9` — contenção in place (4 alvos)

```text
600 ec2-user:ec2-user /tmp/arch.txt
600 ec2-user:ec2-user /tmp/backend-recover.sh
600 ec2-user:ec2-user /tmp/mcp.json.bak
600 ec2-user:ec2-user /tmp/end.txt
```

Todos **mantidos em 0600** e com dono `ec2-user`, idênticos ao estado deixado pela Wave A. Nenhuma
regressão de modo, nenhum alvo ausente.

Nota de escopo: a Wave A conteve **4** arquivos in place no ORQ2, não 12. O número 12 do dispatch
corresponde ao **conjunto inspecionado** (12 alvos com dono/modo verificados por `stat`), dos quais 4
foram contidos in place e 9 foram movidos para quarentena — 13 ações sobre 12 caminhos porque
`/tmp/end.txt` era artefato novo, criado às 11:28:33Z, depois do inventário das 11:02. O revisor deve
usar essa decomposição, não o número agregado.

## 2. ORQ2 — quarentena

```text
700 ec2-user:ec2-user /home/ec2-user/.private-tmp/quarantine-20260727T123153Z
arquivos = 9
dec.txt  e2eq.txt  f.txt  f2-fulltest.log  o30.txt  ph.txt  proj.txt  sdk.txt  sec.txt
modos distintos = 600      donos distintos = ec2-user:ec2-user
```

Diretório segue `0700`, contagem **exatamente 9**, todos os nove `0600` e do mesmo dono. Confere com o
manifesto original arquivo por arquivo.

## 3. ORQ1 `ip-172-31-18-217`

```text
600 ec2-user:ec2-user /tmp/mh
600 ec2-user:ec2-user /tmp/mcp_h
644 root:root        /tmp/test_login.html                      <- intencionalmente NÃO tocado (dono root)
600 ec2-user:ec2-user /tmp/daemon.environ.20260724T110622Z.bak
600 ec2-user:ec2-user /tmp/daemon.environ.20260724T034949Z.bak
```

`mh` e `mcp_h` permanecem `0600`. O arquivo de dono `root` continua `644` e fora do escopo, como
declarado na execução (gate de parada por dono diferente) — é estado esperado, não regressão.

## 4. LOCAL `100.117.245.15` (usuário `dataops-lab`) — **os 16 alvos não existem mais**

```text
AUSENTE /tmp/handshake-token.txt   AUSENTE /tmp/rev-token.txt   AUSENTE /tmp/arch.txt
AUSENTE /tmp/sec.txt  /tmp/o30.txt  /tmp/ph.txt  /tmp/dec.txt  /tmp/sdk.txt  /tmp/proj.txt  /tmp/e2eq.txt
AUSENTE /tmp/av.sh  /tmp/raw.sh  /tmp/imp.sh  /tmp/sync.sh  /tmp/g36.sh  /tmp/idx.html
umask atual = 0022        arquivos >0600 do usuário em /tmp nível 1 = 0
```

Os 16 arquivos que a Wave A levou de `644` para `600` **desapareceram** — comportamento compatível com
limpeza de `/tmp` ou reinício do ambiente WSL2. Duas leituras que o revisor precisa separar:

1. **exposição de arquivo**: encerrada, e não há novos arquivos mundo-legíveis do usuário nesse nível;
2. **exposição de segredo**: **não** encerrada. Os artefatos que continham token cru de handshake, token
   cru `rev` e uma chave de API sumiram, mas **o segredo em si continua válido** enquanto não houver
   rotação. A Wave A nunca prometeu rotação; os itens B1, B4 e B5 do plano da Wave B seguem abertos e
   agora **sem artefato local para apontar**, o que torna a rotação mais difícil de auditar, não menos
   necessária. Ausência de arquivo não é prova de destruição segura nem de invalidação de credencial.

## 5. Regressão de umask e regeneração de artefatos

```text
ORQ2  /etc/bashrc:75  umask 002   (inalterado)      umask do shell = 0002
ORQ1  /etc/bashrc:75  umask 002   (inalterado)      umask do shell = 0002
LOCAL                              umask do shell = 0022
```

A causa raiz **não foi corrigida** — e isso é coerente com o escopo declarado da Wave A, que foi
contenção não disruptiva; a correção estrutural (`umask 077` por drop-in) ficou para a Wave B e nunca
foi autorizada.

Efeito medido da não correção, arquivos `>0600` do próprio usuário em `/tmp` nível 1:

| host | na Wave A (2026-07-27) | agora (2026-07-28) | delta |
|---|---|---|---|
| ORQ2 | 209 | **237** | +28 |
| ORQ1 | não quantificado então (varredura de nomes de chave vazia) | **69** | — |
| LOCAL | 193 | **0** | −193 (por limpeza de `/tmp`, não por contenção) |

Ou seja, no ORQ2 a população mundo-legível **cresceu** em ~28 arquivos em 24 h. Nenhum deles foi
inspecionado por nome de chave nesta verificação: isso está fora do escopo read-only que recebi e
seria um novo inventário, não uma verificação de estado.

## 6. Resumo para o revisor independente

| item do dispatch | estado medido | observação |
|---|---|---|
| modos/donos dos alvos ORQ2 | **íntegro** | 4 in place em 0600; ver §1 sobre a decomposição 4 + 9 vs "12" |
| quarentena 0700 com 9 arquivos | **íntegro** | dir 0700, 9 arquivos, todos 0600, mesmo dono |
| `mh`/`mcp_h` 0600 no ORQ1 | **íntegro** | ambos 0600; `test_login.html` root segue 644 por desenho |
| 16 alvos LOCAL | **inexistentes** | contenção de arquivo irrelevante agora; rotação de segredo continua pendente |
| regressão de umask | **causa raiz intacta** | `umask 002` em ORQ2 e ORQ1; +28 arquivos mundo-legíveis no ORQ2 |

O que eu **não** afirmo, e que pertence à decisão do revisor: se a Wave A pode ser aceita como
concluída. A contenção que eu executei está preservada onde os arquivos ainda existem, mas o objetivo
de segurança só se completa com a Wave B (rotação dos segredos nomeados e `umask` estrutural), que
segue sem autorização.

## 7. Não-alegações

- Não emiti aceite nem veredito de card; por ruling do GTL isso é de outro agente.
- Não executei `chmod`, `mv`, `rm`, escrita ou qualquer mutação; nada foi alterado em nenhum host.
- Não li conteúdo de arquivo algum: classifiquei tudo por caminho e metadado. Nenhum valor de token,
  chave ou header aparece nesta evidência.
- Não investiguei **quem** removeu os 16 arquivos do LOCAL nem se houve destruição segura; só constatei
  ausência.
- Não inventariei os 237 arquivos `>0600` do ORQ2 por nome de chave: fora do escopo desta verificação.
- Não toquei board, fila, runtime, container ou daemon.
