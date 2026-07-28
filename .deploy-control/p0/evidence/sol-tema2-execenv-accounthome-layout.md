# TEMA 2 - o que cada prepareXHome espera dentro de AccountHome vs a estrutura real do slot

Agente: Opus48#B (ORQ2, pane w6:p2) · UTC 2026-07-26T23:12Z
Modo: PREPARAR. Nao editei .go, nao compilei, nao substitui binario, nao reiniciei daemon, nao commitei.
Escopo ajustado pela determinacao do owner: **opencode fora de escopo**. AGY, CODEX e KIRO obrigatorios.

## RESPOSTA CURTA

| provider | o que `prepareXHome` espera DENTRO de `AccountHome` | tipo | onde isso esta no slot REAL | `AccountHome` correto | casa? |
|---|---|---|---|---|---|
| antigravity | `.gemini/antigravity-cli/` | diretorio | `slot-NNN/home/.gemini/antigravity-cli/` | `slot-NNN/home` | **CASA EXATO** |
| kiro | `kiro-cli/data.sqlite3` | arquivo regular | `slot-NNN/xdg-data/kiro-cli/data.sqlite3` | `slot-NNN/xdg-data` | **NAO casa com `slot/home`; casa com `slot/xdg-data`** |

O formato do slot esta certo. O erro seria passar um unico `slot-NNN/home` para os dois: `agy`
funciona e `kiro` falha em silencio, porque `prepareKiroHome` trata `AccountHome` como um
**XDG_DATA_HOME**, nao como um HOME.

## 1. KIRO - `execenv/kiro_home.go`

`kiro_home.go:11` (literal):
```go
const kiroCredentialRelPath = "kiro-cli/data.sqlite3"
```
`kiro_home.go:15-17` (literal, comentario que define a semantica):
```go
	// AccountHome, when non-empty, is the per-account XDG_DATA_HOME source
	// directory. Kiro is an Amazon Q fork and ignores KIRO_HOME; its native
	// credential store is data.sqlite3 under XDG_DATA_HOME/kiro-cli/.
```
`kiro_home.go:30-53` (literal, o que ele faz):
```go
func prepareKiroHome(home string, opts KiroHomeOptions, logger *slog.Logger) error {
	if opts.AccountHome == "" {
		return nil
	}
	if home == "" {
		return fmt.Errorf("kiro xdg data home is empty")
	}

	if err := os.MkdirAll(filepath.Join(home, "kiro-cli"), 0o700); err != nil {
		return fmt.Errorf("create kiro data dir: %w", err)
	}
	if err := os.Chmod(home, 0o700); err != nil { ... }
	if err := os.Chmod(filepath.Join(home, "kiro-cli"), 0o700); err != nil { ... }

	src := filepath.Join(opts.AccountHome, kiroCredentialRelPath)
	dst := filepath.Join(home, kiroCredentialRelPath)
	if err := syncCredentialFile(src, dst); err != nil {
		return fmt.Errorf("seed per-account kiro data.sqlite3: %w", err)
	}
```
ESPERA: `<AccountHome>/kiro-cli/data.sqlite3`, **arquivo regular** (`kiro_home.go:63-65`:
`"src %s is not a regular file"`). Permissao de origem e **preservada** na copia
(`kiro_home.go:90`: `os.OpenFile(dst, ..., srcInfo.Mode().Perm())`); o destino recebe `0700` no
diretorio. `mtime` e restaurado (`kiro_home.go:101`).

FALHA SILENCIOSA: se `src` nao existir, `syncCredentialFile` (`kiro_home.go:73-75`) faz
`if srcMissing { return nil }` — **retorna sucesso sem semear nada**. Ou seja, `AccountHome` apontado
para o diretorio errado nao gera erro: gera um kiro sem credencial. Esse e exatamente o risco de
passar `slot-NNN/home`.

## 2. ANTIGRAVITY - `execenv/antigravity_home.go`

`antigravity_home.go:10` (literal):
```go
const antigravityCredentialRelDir = ".gemini/antigravity-cli"
```
`antigravity_home.go:14-17` (literal):
```go
	// AccountHome, when non-empty, is the per-account HOME source directory.
	// The Antigravity CLI reads token files under
	// HOME/.gemini/antigravity-cli/, so isolation is driven by HOME rather than
	// a vendor-specific environment variable.
```
`antigravity_home.go:24-44` (literal):
```go
func prepareAntigravityHome(home string, opts AntigravityHomeOptions, logger *slog.Logger) error {
	if opts.AccountHome == "" {
		return nil
	}
	if home == "" {
		return fmt.Errorf("antigravity home is empty")
	}
	if err := os.MkdirAll(home, 0o700); err != nil { ... }
	if err := os.Chmod(home, 0o700); err != nil { ... }

	src := filepath.Join(opts.AccountHome, antigravityCredentialRelDir)
	dst := filepath.Join(home, antigravityCredentialRelDir)
	if err := syncCredentialDir(src, dst); err != nil {
		return fmt.Errorf("seed per-account antigravity token dir: %w", err)
	}
```
ESPERA: `<AccountHome>/.gemini/antigravity-cli/`, **diretorio** (`antigravity_home.go:54-56`:
`"src %s is not a directory"`). Copia recursiva com perms e mtime preservados
(`antigravity_home.go:70-79`, arquivos via `copyCredentialFile`). Mesma falha-silenciosa quando
ausente (`antigravity_home.go:64-66`).

## 3. ESTRUTURA REAL DO SLOT, MEDIDA AGORA NO ORQ2

```
$ ls -la /home/ec2-user/.agent-cred-homes/slots/slot-140/
drwx------.  2 ec2-user ec2-user     6 Jul 25 13:17 cline
drwx------.  2 ec2-user ec2-user     6 Jul 25 13:17 cline-sandbox
drwx------.  2 ec2-user ec2-user     6 Jul 25 13:17 codex
drwx------.  7 ec2-user ec2-user    86 Jul 26 23:06 home
drwx------.  3 ec2-user ec2-user    22 Jul 25 13:17 xdg-config
drwx------.  5 ec2-user ec2-user    48 Jul 25 16:13 xdg-data
```
```
$ ls -l .../slot-140/xdg-data/kiro-cli/data.sqlite3
-rw-------. 1 ec2-user ec2-user 69632 Jul 26 22:44

$ ls -l .../slot-145/home/.gemini/antigravity-cli/antigravity-oauth-token
-rw-------. 1 ec2-user ec2-user  1677 Jul 26 22:25
```

O slot NAO e um HOME unico: e um container com **raizes separadas por provider**
(`home`, `xdg-data`, `xdg-config`, `codex`, `cline`, `cline-sandbox`).

Inventario dos 22 slots (`data.sqlite3` sob `xdg-data/kiro-cli` vs `home/.gemini/antigravity-cli`):
```
slot-104 -    AGY     slot-123 -    AGY
slot-105 -    AGY     slot-139 KIRO AGY
slot-106 -    AGY     slot-140 KIRO AGY
slot-107 -    AGY     slot-141 -    AGY
slot-109 -    AGY     slot-142 KIRO AGY
slot-110 -    AGY     slot-143 KIRO AGY
slot-111 -    AGY     slot-145 -    AGY
slot-112 -    AGY     slot-146 -    AGY
slot-115 -    AGY     slot-149 KIRO AGY
slot-120 -    AGY     slot-150 -    AGY
slot-122 -    AGY     slot-152 -    AGY
```
**22/22 slots tem credencial AGY. Apenas 5 tem KIRO: 139, 140, 142, 143, 149.**
O `slot-145` que voce mediu tem `home/.gemini` (por isso `agy models` funcionou) e **nao tem**
`xdg-data/kiro-cli/data.sqlite3`. Para kiro e obrigatorio usar um dos 5 slots acima.

## 4. PERMISSOES: JA CASAM, NADA A AJUSTAR

- Slot: diretorios `0700`, credenciais `0600`.
- Preparers: `chmod 0700` nos destinos; `copyCredentialFile` (`kiro_home.go:90`) cria o destino com
  `srcInfo.Mode().Perm()`, preservando `0600`; `copyCredentialDir` (`antigravity_home.go:71-76`)
  preserva as perms do diretorio de origem.
Nenhum ajuste de permissao e necessario.

## 5. O QUE PRECISA AJUSTAR: `CredentialAccountHome` E UM CAMPO SO, MAS PRECISA DE RAIZ POR PROVIDER

`daemon.go:3448` (literal, a origem do vazio):
```go
	credentialAccountHome := ""
```
usado em `daemon.go:3475` (Reuse) e `daemon.go:3492` (Prepare). E a **unica** fonte do valor:
`grep -rIn CredentialAccountHome --include=*.go` nao mostra nenhuma leitura de env nem de config.

PATCH PROPOSTO (aditivo, nao toca nenhum preparer):
```go
// ANTES  daemon.go:3448
	credentialAccountHome := ""

// DEPOIS daemon.go:3448
	credentialAccountHome := resolveCredentialAccountHome(provider, d.credentialSlotRoot(task))

// NOVO helper, no mesmo pacote. Cada provider tem uma raiz DIFERENTE dentro do slot,
// porque prepareKiroHome trata AccountHome como XDG_DATA_HOME e prepareAntigravityHome
// trata como HOME. Passar um unico slot/home faz kiro falhar EM SILENCIO
// (kiro_home.go:73-75 retorna nil quando a origem nao existe).
func resolveCredentialAccountHome(provider, slotRoot string) string {
	if slotRoot == "" {
		return "" // preserva o comportamento global/compartilhado atual
	}
	switch provider {
	case "antigravity":
		return filepath.Join(slotRoot, "home")      // -> <AH>/.gemini/antigravity-cli
	case "kiro":
		return filepath.Join(slotRoot, "xdg-data")  // -> <AH>/kiro-cli/data.sqlite3
	case "codex":
		return filepath.Join(slotRoot, "codex")     // a confirmar pelo dono do pacote codex
	case "cline":
		return filepath.Join(slotRoot, "cline")
	default:
		return ""
	}
}
```
JUSTIFICATIVA: `execenv.go:301` e `:312` ja fazem o resto (`prepareKiroHome` na 303,
`prepareAntigravityHome` na 314) e ja tratam `""` como global. O unico defeito e a raiz.

VALIDACAO PROPOSTA (obrigatoria, para nao repetir a falha silenciosa): antes de retornar, exigir que
a credencial exista, e falhar a task com erro explicito se nao existir. `agy models` sem credencial
devolve "Please sign in", que hoje viraria task falhada sem causa legivel.

## 6. RESPOSTA AOS PONTOS DA ATUALIZACAO DE ESCOPO

- **Caminho sem recompilar: NAO EXISTE.** `daemon.go:3448` e literal `""`, sem env e sem config.
  `grep` confirma que nenhuma outra linha escreve `credentialAccountHome`.
- **Recompilar:** e a unica via. Voce reportou `go build` e `go vet` exit 0. O patch acima e ~25
  linhas em 1 arquivo (`daemon.go`) mais o helper; nenhum preparer muda. Estimativa de escrita e
  build: minutos, nao dias. Nao executei nada disso.
- **KIRO so tem 5 slots com credencial.** Se o roteamento sortear um slot sem `xdg-data/kiro-cli`,
  kiro roda sem credencial e falha. A alocacao de slot para kiro precisa ser restrita a
  {139, 140, 142, 143, 149} ou os outros slots precisam ser logados.
- **CODEX, atencao:** `daemon.go:3476` passa `CredentiallessGateway: true`, e `execenv.go:284-287`
  desvia para `prepareCredentiallessCodexHome(codexHome)` **ignorando `AccountHome`**. Com codex
  obrigatorio, isso e ponto de decisao de outro pacote, nao meu: preencher `CredentialAccountHome`
  sozinho nao muda o caminho do codex.
- **Modelos exigidos** (claude-opus-4-6-thinking, claude-sonnet-4-6, gemini-3.1-pro-high/low,
  gemini-3.5-flash-*, gemini-3.6-flash-*) sao servidos pelo `agy`, cujo formato de slot **JA CASA**.
  Nao ha ajuste de layout para eles: falta so o valor chegar preenchido.

## 7. NAO-AFIRMACOES
- Nao editei `.go`, nao rodei build/vet/test, nao substitui binario, nao reiniciei daemon, nao commitei.
- Nao li conteudo de credencial: apenas `ls -l` de caminho, tamanho e permissao.
- Nao validei o caminho do codex nem do cline por medicao; `slot-140/codex` e `slot-140/cline` estao
  **vazios** no ORQ2, o que e sinal de alerta para o dono desses pacotes.
- opencode excluido por determinacao do owner; nao analisei `opencode_home.go` nesta entrega.
- Slots medidos no ORQ2. Nao existem em `/home/ec2-user/.agent-cred-homes/slots` no ORQ1: la as
  credenciais estao no HOME global e em `cred-bak-20260726/`. Se o daemon roda no ORQ1, os slots
  precisam existir la — ponto que exige decisao, nao acao minha.
