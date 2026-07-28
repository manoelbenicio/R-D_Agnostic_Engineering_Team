# ORQ-26 - push tentado, **impossivel**: o `gh` nao esta autenticado **na visao deste agente**

- executor: **Opus48#A** - ORQ2 w6:p1 - 2026-07-27T19:58Z
- alvo: `ci/orq26-db-gate`, HEAD **`20cab47`**, worktree
  `/home/ec2-user/workspace/worktrees/ci-orq26-db-gate`
- **push NAO executado. PR/merge NAO abertos. Nenhum ref remoto criado ou alterado.**

## 1. O que medi, na ordem pedida

```
$ gh auth status --hostname github.com
You are not logged into any GitHub hosts. To log in, run: gh auth login

$ gh auth refresh --hostname github.com --scopes workflow
not logged in to any hosts. Use 'gh auth login' to authenticate with a host
```
O `refresh` **nao pode** rodar: ele exige um login previo. Nao havia device flow para reportar.

Confirmacao **funcional**, sem mutar nada:
```
$ GIT_TERMINAL_PROMPT=0 git -C <worktree> push --dry-run -u origin ci/orq26-db-gate:ci/orq26-db-gate
remote: No anonymous write access.
fatal: Authentication failed for 'https://github.com/manoelbenicio/R-D_Agnostic_Engineering_Team.git/'
exit 128
```
Sem escopo para inspecionar: `GH_TOKEN` e `GITHUB_TOKEN` vazios, nenhum `credential.helper`, e
**nenhum** `hosts.yml`.

## 2. Causa provavel - e ela e estrutural, nao um erro do owner

Este agente roda num **slot de credencial**:
```
HOME=/home/ec2-user/.agent-cred-homes/slots/slot-149/home
XDG_CONFIG_HOME=/home/ec2-user/.agent-cred-homes/slots/slot-149/xdg-config
GH_CONFIG_DIR=<vazio>
```
O `gh` procura o `hosts.yml` em `$GH_CONFIG_DIR`, senao em `$XDG_CONFIG_HOME/gh`, senao em
`$HOME/.config/gh`. Verifiquei os tres candidatos e **`/home/ec2-user/.config/gh/hosts.yml`**: nenhum
existe. Ou seja: **se o owner autenticou numa sessao com o `HOME` real do `ec2-user`, ou noutra maquina,
este processo nao ve** - os caminhos sao diferentes. Tambem confirmei que **o `gh` nao existe no ORQ1**,
portanto o login nao esta la.

## 3. Como desbloquear - tres opcoes, todas do owner

**Opcao 1 (mais simples): autenticar no `HOME` que este agente usa.**
```bash
GH_CONFIG_DIR=/home/ec2-user/.agent-cred-homes/slots/slot-149/xdg-config/gh \
  gh auth login --hostname github.com --git-protocol https --web
GH_CONFIG_DIR=/home/ec2-user/.agent-cred-homes/slots/slot-149/xdg-config/gh \
  gh auth refresh --hostname github.com --scopes workflow
```

**Opcao 2: apontar este agente para o login que ja existe.** Se o owner autenticou no `HOME` real,
basta me informar o caminho do `hosts.yml` e eu passo a usar `GH_CONFIG_DIR` apontando para ele -
**nao** vou procurar nem copiar arquivo de credencial por conta propria.

**Opcao 3: token via Secrets Manager, sem plaintext em contexto**, seguindo a skill:
```bash
GH_PAT='{{resolve:secretsmanager:<SECRET-ID>:SecretString:<JSON-KEY>}}' \
  asm-exec -- sh -eu -c 'printf %s "$GH_PAT" | gh auth login --hostname github.com --with-token'
```

**Escopos obrigatorios: `repo` e `workflow`.** O `workflow` nao e opcional: o commit `d244718` cria
`.github/workflows/orq26-db-gate.yml`, e o GitHub **rejeita** push que cria ou altera workflow sem esse
escopo. Sem ele, a falha sera **pelo arquivo de workflow**, nao pelo codigo.

## 4. O que executo no instante em que ficar verde

```bash
gh auth status --hostname github.com          # confirmar login e escopo workflow
git -C /home/ec2-user/workspace/worktrees/ci-orq26-db-gate \
  push -u origin ci/orq26-db-gate:ci/orq26-db-gate
git ls-remote origin refs/heads/ci/orq26-db-gate    # confirmar SHA remoto == 20cab47
gh run list --branch ci/orq26-db-gate --workflow orq26-db-gate.yml --limit 5
gh run watch <ID>                                    # monitorar ate terminal
gh run view <ID> --log                               # reportar gates, sem segredo
```
Sem `--force`, sem `--no-verify`, sem tocar `main`, **sem PR e sem merge**.

## 5. Nao-afirmacoes

- **Nao dei push**, nao abri PR, nao fiz merge, nao criei nem alterei ref remoto. O `--dry-run`
  **falhou** por autenticacao.
- **Nao executei `gh auth token`** e nao vou: ele imprimiria o token no meu contexto.
- **Nao procurei nem copiei arquivo de credencial de outro `HOME`**; apenas testei a **existencia** dos
  caminhos padrao do `gh`.
- **Nao existe run de CI para reportar**, porque nao houve push. Nao vou chamar de verde nem de
  vermelho.
- Nao sei **onde** o owner autenticou; a hipotese do slot de credencial e a explicacao mais provavel do
  desencontro, medida pelos caminhos, nao confirmada com ele.
- O trabalho da ORQ-42 Q-H/Q-I ficou **interrompido e salvo** no worktree
  `orq42-secret-tools` (nao commitado ainda), sem nada pendente em produto.
