# CORRECAO DE EVIDENCIA - Opus48#A - contra os FATOS ESTABELECIDOS

- agente: Opus48#A - ORQ2 - pane w6:p1 - 2026-07-26T23:16Z
- corrige: `.deploy-control/p0/evidence/sol-tema1-credentialAccountHome.md`
- estado: nenhum arquivo de codigo editado, sem build, sem vet, sem restart, sem commit.

## 1. ERREI. "O resolver nunca foi escrito" esta FALSO. FATO 6 confirmado.

O que eu escrevi: *"O proprio codigo documenta o resolver que nunca foi escrito"*. Errado.
O resolver existiu, com exatamente os componentes que o FATO 6 cita. Literal, `aa62401`:

```go
3838 func (d *Daemon) credentialAccountHomeForTask(ctx context.Context, task Task, provider string, taskLog *slog.Logger) string {
3839 	if d.rotationStore == nil || task.AgentID == "" || provider == "" {
3840 		return ""
3841 	}
3842 	accountID, err := d.rotationStore.CurrentAssignment(ctx, task.AgentID)
...
3849 	account, err := d.rotationStore.GetAccount(ctx, accountID)
...
3854 	if !strings.EqualFold(account.Vendor, provider) {
3855 		return ""
3856 	}
3857 	return account.HomeDir
3858 }
```
E era chamado direto: `aa62401:daemon.go:3264 credentialAccountHome := d.credentialAccountHomeForTask(ctx, task, provider, taskLog)`.

CAUSA DA MINHA FALHA: meu grep foi so no worktree em HEAD, nao no historico. Regra que adoto:
antes de afirmar "nunca existiu", rodar `git log -S <simbolo>`.

## 2. REFINAMENTO do FATO 6, com evidencia nova: foram DUAS remocoes, nao uma

`git log -S credentialAccountHomeForTask -- server/internal/daemon/daemon.go` retorna 2 commits:
`aa62401` (criacao) e `9ab80a6` (remocao). Detalhe medido:

- `31d50b9` (Sun Jul 5 21:24:27 2026) NAO apagou a funcao: ele **degradou a chamada para
  condicional**. Diff literal: `- credentialAccountHome := d.credentialAccountHomeForTask(...)`
  para `+ credentialAccountHome := ""` MAIS `+ credentialAccountHome = d.credentialAccountHomeForTask(...)`
  dentro de um ramo. Ou seja em 05/07 o wiring ainda existia, so passou a ser condicional.
- `9ab80a6` (Fri Jul 24 03:54:01 2026, "sync(integration): observability e2e 7-hop tracing +
  tier-20 capacity gate + fixes") **apagou a chamada E a funcao**. Diff literal:
  `- credentialAccountHome, err = d.credentialAccountHomeForTask(ctx, task, provider, taskLog)` e
  `- func (d *Daemon) credentialAccountHomeForTask(ctx context.Context, task Task, provider string, taskLog *slog.Logger) (string, error) {`

Por isso HOJE nao ha nenhuma atribuicao e meu grep achou so 3 linhas. A conclusao do FATO 6 esta
certa - e regressao, nao feature nova - e a data da regressao FINAL e 24/07, dois dias atras.

## 3. RETIRO minha recomendacao de usar AgentData.RuntimeConfig

Eu propus carregar o valor via `AgentData.RuntimeConfig`. Retirado. A fonte correta e a que ja
existia: assignment de conta -> `account.HomeDir`. Minha medicao de que
`~/.agent-cred-homes` nao aparece em nenhum `.go` continua verdadeira, mas a CONCLUSAO que tirei
dela estava errada: o daemon nunca precisou conhecer o caminho do slot, ele recebia o `HomeDir`
gravado no registro da conta.

## 4. EVIDENCIA NOVA E SUPERIOR para a ponte: a camada rotation NAO EXISTE em HEAD

Medido agora no worktree:
- `grep -rln "package rotation" --include=*.go .` -> ZERO arquivos.
- `grep -rn "rotationStore" --include=*.go . | grep -v _test` -> ZERO ocorrencias.
- nao existe diretorio `internal/daemon/rotation`.

CONSEQUENCIA PARA O PROXIMO TRABALHO: nao da para "so restaurar a chamada". `9ab80a6` levou a
dependencia inteira (`d.rotationStore`, `rotation.Account`, `rotation.ErrNoAssignment`). A ponte
precisa de um provedor de assignment de novo, e e exatamente aqui que o FATO 5 morde: o registry do
ORQ2 indexa por `terminal_id` e o resolver antigo indexava por `task.AgentID`. A ponte tem de
traduzir `AgentID` -> conta -> `HomeDir`, e as tabelas accounts/approved_accounts/assignments estao
vazias, entao nao ha o que traduzir ainda.

## 5. CONFIRMO o FATO 3 e explico o mecanismo exato do silencio

`execenv/kiro_home.go:11`: `const kiroCredentialRelPath = "kiro-cli/data.sqlite3"`.
`:48`: `src := filepath.Join(opts.AccountHome, kiroCredentialRelPath)`.
Logo AccountHome do kiro **tem** de ser `slot/xdg-data`, porque o arquivo real e
`slot/xdg-data/kiro-cli/data.sqlite3`. Se passarem `slot/home`, o `src` aponta para
`slot/home/kiro-cli/data.sqlite3`, que nao existe, e `syncCredentialFile` faz:
```go
73 	if srcMissing {
74 		return nil
75 	}
```
Retorna nil, `prepareKiroHome` retorna nil, o task roda com XDG_DATA_HOME isolado e VAZIO. Falha
silenciosa, exatamente como o FATO 3 afirma. Comprovado.

## 6. CONFIRMO o FATO 4 e retiro uma contradicao que eu quase reportei

Contagem por DIRETORIO daria 10 slots com `xdg-data/kiro-cli` (139, 140, 141, 142, 143, 145, 146,
149, 150, 152) - e eu quase reportei isso como contradicao ao FATO 4. Errado da minha parte:
diretorio nao e credencial. Contando o arquivo que `prepareKiroHome` realmente copia:

```
slot-139 69632 bytes | slot-140 69632 | slot-142 28672 | slot-143 69632 | slot-149 69632
total com credencial kiro: 5
```
Slots com `kiro-cli/` VAZIO: 141, 145, 146, 150, 152. Total de slots: 22.
FATO 4 esta correto, incluindo "o slot-145 NAO tem kiro": o diretorio existe, o `data.sqlite3` nao.

## 7. NOTA OPERACIONAL, segunda causa de falso "No such file"

Alem da permissao 0700 do FATO 1, existe um segundo motivo para uma sonda falhar: agentes da frota
rodam DENTRO de um slot. Meu `$HOME` medido agora e
`/home/ec2-user/.agent-cred-homes/slots/slot-149/home`, entao `~/.agent-cred-homes` expande para
`.../slot-149/home/.agent-cred-homes`, que nao existe. Sempre usar o caminho ABSOLUTO
`/home/ec2-user/.agent-cred-homes`. Com absoluto eu li tudo: `drwx------ 5 ec2-user ec2-user`,
com `registry.json`, `registry.lock`, `registry.json.pre-orphan-cleanup.20260722T023208Z`,
`slots/`, `codex-logins/`, `fallback-terminals/`. FATOS 1 e 2 confirmados por leitura direta.
Estrutura do slot-140 confirmada: `cline, cline-sandbox, codex, home, xdg-config, xdg-data`.

## 8. NADA MUTADO

Nenhum arquivo de codigo editado, sem go build, sem go vet, sem binario, sem restart de daemon ou
container, sem commit, push, reset, clean ou checkout --. Nenhum segredo lido: contei tamanho de
`data.sqlite3`, nunca abri o conteudo.
