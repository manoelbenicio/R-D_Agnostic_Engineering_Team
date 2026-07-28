# ORQ-26 - PASS registrado, push **impossivel**, e um defeito meu descoberto antes do push

- executor/owner exclusivo: **Opus48#A** - ORQ2 w6:p1 - 2026-07-27T18:42Z
- **PASS da re-review final**: commit `916111c`, evidencia
  `.deploy-control/p0/evidence/orq26-deadline-fix-916111c.md`, sha256
  **`bda51ec00919f8016139f7cd7dfc7f431505faeb16b4b0a9c1476cac9d4829a7`**
- commit adicional desta rodada, **novo, sem amend**: `20cab47`
- **push NAO executado. PR/merge NAO abertos.**

## 1. `gh` continua **nao autenticado** - push impossivel, provado sem mutar

```
$ gh auth status
You are not logged into any GitHub hosts. To log in, run: gh auth login
GH_TOKEN=<vazio>  GITHUB_TOKEN=<vazio>
credential.helper: nenhum;  ~/.git-credentials: inexistente;  ~/.config/gh/hosts.yml: inexistente
```
Prova direta, com `--dry-run` (nao muta nada) e sem prompt:
```
$ GIT_TERMINAL_PROMPT=0 git push --dry-run -u origin ci/orq26-db-gate:ci/orq26-db-gate
remote: No anonymous write access.
fatal: Authentication failed for 'https://github.com/manoelbenicio/R-D_Agnostic_Engineering_Team.git/'
exit 128
```
Leitura anonima **funciona** (o repositorio e legivel sem credencial): `git ls-remote` respondeu, e por
isso pude medir a topologia do origin. Escrita, nao.

Estado do alvo, medido: `ci/orq26-db-gate` **nao existe** no origin (seria criacao, portanto `-u`);
branch default do origin e `main` em `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f`.

## 2. Comando exato de login - **para o owner executar**, nunca por mim

Opcao A, interativa, preferida (nenhum segredo passa por agente):
```bash
gh auth login --hostname github.com --git-protocol https --web
```
Opcao B, token ja guardado no Secrets Manager, seguindo a skill (`asm-exec`, sem plaintext em contexto,
sem `get-secret-value`):
```bash
GH_PAT='{{resolve:secretsmanager:<SECRET-ID>:SecretString:<JSON-KEY>}}' \
  asm-exec -- sh -eu -c 'printf %s "$GH_PAT" | gh auth login --hostname github.com --with-token'
```
Escopo minimo necessario: `repo` (push). `workflow` **tambem** e exigido, porque o commit `d244718`
adiciona `.github/workflows/orq26-db-gate.yml` - um push que cria ou altera workflow e **rejeitado**
sem esse escopo. **Eu nao executo nenhuma das duas** e nao quero o token em contexto.

Depois do login, o comando unico ja aprovado em desenho:
```bash
git -C /home/ec2-user/workspace/worktrees/ci-orq26-db-gate push -u origin ci/orq26-db-gate:ci/orq26-db-gate
```
sem `--force`, sem `--no-verify`, sem tocar `main`, sem PR.

## 3. 🔴 Defeito **meu**, encontrado por causa do push - a guarda de range estava quebrada

A leitura anonima do origin me deixou medir o que eu **nao** tinha medido quando endureci a guarda em
`fa0c9de`. O resultado invalida aquela versao:

- a base que eu usei era `origin/${{ github.event.repository.default_branch }}` = `origin/main`;
- mas este branch **descende da linhagem de integracao**: a base real e
  `0cb8aebb5aff79cb430b3740d22fadc53c0116fd`, e `git merge-base --is-ancestor 0cb8aeb b6571299`
  responde **falso** - `0cb8aeb` **nao** e ancestral de `main`;
- consequencia medida: `origin/main..HEAD` abrange **130 commits e 1878 arquivos**. A igualdade de
  conjunto com 3 caminhos **nunca** poderia valer, e o gate **teria falhado na primeira execucao real**.

Corrigido em `20cab47`: a base passa a ser **pinada** em `GATE_BASE_SHA: 0cb8aebb…`, com **duas provas
fail-closed** antes de usar - o commit tem de existir (`rev-parse --verify "$base^{commit}"`) e tem de
ser **ancestral de HEAD** (`merge-base --is-ancestor`). Dry-run local com a base pinada:
`0cb8aeb..HEAD` = exatamente os 3 caminhos permitidos, **igualdade OK**.

Nenhum arquivo Go mudou em `20cab47`, portanto os dois hashes congelados e o conjunto de **11** folhas
permanecem.

## 4. Verificacao desta rodada

```
yaml.safe_load                                  -> OK, 16 passos
git diff --check                                -> limpo
sha256 dos .go                                  -> inalterados (70f45ebd…, b1c3c515…)
base pinada existe e e ancestral de HEAD        -> OK
igualdade de range com base pinada              -> OK (3 == 3)
git push --dry-run                              -> exit 128, "No anonymous write access"
worktree                                        -> limpo, HEAD 20cab47
```

## 5. Nao-afirmacoes

- **Nao dei push, nao abri PR, nao fiz merge, nao autentiquei nada, nao li token.** O unico comando de
  rede que executei foi leitura anonima (`ls-remote`) e um `push --dry-run` que **falhou** por
  autenticacao - nenhum ref remoto foi criado ou alterado.
- **O workflow nunca rodou.** Nao ha CI para reportar: sem push, nao existe execucao. Nao vou relatar
  "CI verde" nem "CI vermelho" - nao existe nenhum dos dois.
- Os **11 testes seguem sem execucao**, aqui ou em CI. O PASS da re-review e sobre o **codigo**, nao
  sobre resultado de teste.
- A guarda de range corrigida foi validada **localmente**, com base pinada; ainda **nao** rodou no
  runner. `fetch-depth: 0` continua necessario para o `0cb8aeb` estar presente, e nao medi o custo.
- Nao sei se o token do owner tera escopo `workflow`; se nao tiver, o push falha **por causa do arquivo
  de workflow**, nao por causa do codigo.
- O fallback de `KeyFromURL` em `internal/storage/s3.go` **permanece**; segue como card proprio.
- Nao toquei `gtl-orq26`, `gtl-orq26-consumer-tests` (congelados) nem o worktree principal.
