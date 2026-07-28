# ORQ-26 - push **executado**; o run terminou em falha por **billing**, sem rodar nenhum gate

- executor: **Opus48#A** - ORQ2 w6:p1 - 2026-07-27T20:40Z - P0, ORQ-42 interrompida
- worktree isolado: `/home/ec2-user/workspace/worktrees/ci-orq26-db-gate`, limpo
- `GH_CONFIG_DIR=/home/ec2-user/.agent-cred-homes/slots/slot-149/xdg-config/gh` em **todas** as chamadas
- **Nenhum PR aberto. Nenhum merge. Nenhum `--force`. Nenhum `--no-verify`.**

## 1. Autenticacao confirmada, sem exibir token

```
$ gh auth status --hostname github.com
github.com
  ✓ Logged in to github.com account manoelbenicio (…/slot-149/xdg-config/gh/hosts.yml)
  - Active account: true
  - Git operations protocol: https
  - Token: gho_************************************
  - Token scopes: 'gist', 'read:org', 'repo', 'workflow'
```
`hosts.yml` com modo **`600`**, dono `ec2-user`. **Nao** usei `--show-token`, **nao** rodei
`gh auth token`, e o valor mascarado acima e o proprio mascaramento do `gh`. Escopos exigidos presentes:
**`repo`** e **`workflow`** - este ultimo era obrigatorio porque o commit `d244718` cria um arquivo de
workflow.

## 2. Push - **sucesso**, criando o branch remoto

```
$ git -c 'credential.helper=!gh auth git-credential' \
    -C /home/ec2-user/workspace/worktrees/ci-orq26-db-gate \
    push -u origin ci/orq26-db-gate:ci/orq26-db-gate
remote: Create a pull request for 'ci/orq26-db-gate' on GitHub by visiting: …
To https://github.com/manoelbenicio/R-D_Agnostic_Engineering_Team.git
 * [new branch]      ci/orq26-db-gate -> ci/orq26-db-gate
branch 'ci/orq26-db-gate' set up to track 'origin/ci/orq26-db-gate'.
exit 0
```
**Desvio minimo, declarado:** acrescentei `-c 'credential.helper=!gh auth git-credential'` **inline**,
porque nao havia `credential.helper` configurado e sem ele o push falharia. Preferi o `-c` inline a
rodar `gh auth setup-git`, que mutaria a configuracao global do git. **Nenhuma configuracao foi
alterada.** O convite a abrir PR na saida do remote e texto do GitHub; **nao** o segui.

## 3. SHA remoto - **confere**

```
$ git ls-remote origin refs/heads/ci/orq26-db-gate
20cab478a04f4119be7f1ecd109a328cabf01f6b   refs/heads/ci/orq26-db-gate
local HEAD:  20cab478a04f4119be7f1ecd109a328cabf01f6b
IGUAL
```

## 4. Workflow - estado terminal **`failure`**, mas **nenhum gate rodou**

```
run 30294480915   ORQ-26 DB Gate (temporary)   ci/orq26-db-gate   push
status=completed  conclusion=failure  duracao 5s
job orq26-db-gate  startedAt 18:35:58Z  completedAt 18:36:01Z  steps: []   <- ZERO passos
ANNOTATION: "The job was not started because your account is locked due to a billing issue."
$ gh run view … --log-failed  ->  log not found
```

**Leitura correta e importante:** isso **nao** e falha do codigo nem dos gates. O que se pode e o que
**nao** se pode concluir:

| conclui | nao conclui |
|---|---|
| o push criou o branch e o SHA remoto e o esperado | **nada** sobre os 11 testes |
| o gatilho funcionou: o `push` no branch `ci/orq26-db-gate` com os `paths` casou e o GitHub **criou** o run | nada sobre a guarda de arquivos por igualdade de range |
| o arquivo de workflow foi aceito (escopo `workflow` suficiente) | nada sobre freeze de hashes, `gofmt`, `vet`, migrations ou Postgres efemero |
| o estado terminal e `failure` | **nada** sobre F1/F2 - a limpeza por chave exata e o contexto desacoplado seguem **sem prova de execucao** |

`steps: []` e a prova de que o job **nunca comecou**: nenhum passo foi criado, portanto nenhum gate foi
avaliado. Nao ha log para reportar porque nao houve execucao.

## 5. Bloqueador atual - **conta com Actions bloqueado por billing**

E um bloqueio de **conta**, no nivel do owner; nao ha nada que eu possa fazer no repositorio ou no
runbook para contorna-lo, e **nao** vou tentar contornar. Desbloqueio possivel apenas pelo owner:
resolver o billing de GitHub Actions da conta `manoelbenicio`. Depois disso, o re-disparo **nao** exige
novo commit: basta
```bash
GH_CONFIG_DIR=… gh run rerun 30294480915 --repo manoelbenicio/R-D_Agnostic_Engineering_Team
```
ou um novo push no branch tocando um dos 3 caminhos do filtro `paths`. Eu **nao** executei rerun.

## 6. Nao-afirmacoes

- **Nao abri PR, nao fiz merge, nao usei `--force` nem `--no-verify`, nao toquei `main`.**
- **Nao exibi token**: sem `--show-token`, sem `gh auth token`. O `gho_****` acima e mascaramento do
  proprio `gh`.
- **Nao afirmo que o gate passou nem que falhou por motivo tecnico.** Ele **nao rodou**. Qualquer
  leitura de "CI vermelho" como problema de codigo estaria errada.
- **Nao executei `gh run rerun`** e nao tentei contornar o bloqueio de billing.
- Nao alterei configuracao de git, nem global nem do worktree; o helper foi passado **inline**.
- A ORQ-42 esta **interrompida** e salva em `orq42-secret-tools` (commits locais `355c57b`, `bbeb80b`),
  nada pendente em produto.
