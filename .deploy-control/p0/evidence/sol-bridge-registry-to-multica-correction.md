# CORRECAO DE EVIDENCIA + contrato da PONTE registry(ORQ2) -> Multica

Agente: Opus48#B (ORQ2, pane w6:p2) · UTC 2026-07-26T23:20Z
Modo: PREPARAR. Nao editei .go, nao compilei, nao reiniciei, nao commitei. `git status` em
`multica-auth-work/`: 0 arquivos `.go` modificados.

## 1. CORRECAO DA MINHA EVIDENCIA ANTERIOR

Arquivo corrigido: `.deploy-control/p0/evidence/sol-tema2-execenv-accounthome-layout.md`, secao 7.

O que eu escrevi: "Se o daemon roda no ORQ1, os slots precisam existir la — ponto que exige decisao".
Isso esta MAL ENQUADRADO e eu retiro o enquadramento. Nao contradiz o FATO 1: eu nunca afirmei
ausencia no ORQ2. Confirmo o FATO 1 com medicao propria como `ec2-user`:

```
# ORQ2 (ip-172-31-30-9)
$ ls -ld /home/ec2-user/.agent-cred-homes
drwx------. 5 ec2-user ec2-user 164 Jul 26 13:38 /home/ec2-user/.agent-cred-homes
$ ls -l /home/ec2-user/.agent-cred-homes/
drwx------. 20 codex-logins
drwx------.  2 fallback-terminals
-rw-------.  1 13183 registry.json
-rw-------.  1 33009 registry.json.pre-orphan-cleanup.20260722T023208Z
-rw-rw-r--.  1     0 registry.lock
drwx------. 24 slots

# ORQ1 (100.118.244.61), sessao confirmada como ec2-user (id -un -> ec2-user)
$ ls -ld /home/ec2-user/.agent-cred-homes
ls: cannot access '/home/ec2-user/.agent-cred-homes': No such file or directory
```

Enquadramento CORRETO, que e o achado util para a ponte: **o isolamento esta no ORQ2 e o daemon
esta no ORQ1**. Logo a ponte e CROSS-HOST. Nao e "criar slots no ORQ1" nem "decidir onde ficam":
e resolver como o daemon do ORQ1 obtem um caminho de slot que existe no ORQ2. Isso nao redesenha
isolamento; e exatamente a ponte pedida.

## 2. PRECISAO SOBRE O FATO 6 (evidencia nova, nao contradicao)

O FATO 6 diz que `31d50b9` removeu o wiring. Medido: `31d50b9` **nao removeu, GATEOU**.
`git show 31d50b9 -- internal/daemon/daemon.go` (literal):
```diff
-	credentialAccountHome := d.credentialAccountHomeForTask(ctx, task, provider, taskLog)
+	credentialAccountHome := ""
+	if !d.cfg.L2Runtime.Enabled {
+		credentialAccountHome = d.credentialAccountHomeForTask(ctx, task, provider, taskLog)
+	}
```
O motivo do gate esta no mesmo commit: `"runtime_router_owner", runtimeRouterOwnerRustL2` e
`"rotation_noop_reason", rotationNoopReasonL2RouterOwn` — o router L2 assumiu a rotacao.

A remocao efetiva veio DEPOIS, em `9ab80a6` ("sync(integration): observability e2e 7-hop tracing +
tier-20 capacity gate + fixes"):
```
$ git log --oneline -S'credentialAccountHomeForTask' -- .../internal/daemon/daemon.go
9ab80a6  <- remove
aa62401  <- introduz
```
No HEAD: `grep -rn 'credentialAccountHomeForTask' --include='*.go' .` -> **zero ocorrencias**.
`grep -rn 'L2Runtime' internal/daemon` -> **zero**. O gate tambem foi embora.

Conclusao que reforca o FATO 6 sem alterar seu sentido: e regressao em DUAS etapas, gate em
`31d50b9` e delecao em `9ab80a6`.

## 3. ACHADO QUE MUDA O TAMANHO DA PONTE

`internal/daemon/rotation` **NAO EXISTE** no HEAD:
```
$ ls -d internal/daemon/rotation
ls: cannot access 'internal/daemon/rotation': No such file or directory
$ grep -rIn '/rotation"' --include='*.go' .
(vazio)
```
Ou seja `d.rotationStore`, `CurrentAssignment`, `GetAccount`, `rotation.Account` e
`rotation.ErrNoAssignment` sumiram do lado Go. A ponte NAO e "religar uma chamada": e reintroduzir
um cliente de store. O lado SQL sobreviveu (secao 4).

Wiring original, recuperavel de `aa62401` como referencia (literal):
```go
func (d *Daemon) credentialAccountHomeForTask(ctx context.Context, task Task, provider string, taskLog *slog.Logger) string {
	if d.rotationStore == nil || task.AgentID == "" || provider == "" {
		return ""
	}
	accountID, err := d.rotationStore.CurrentAssignment(ctx, task.AgentID)
	if err != nil {
		if !errors.Is(err, rotation.ErrNoAssignment) {
			taskLog.Debug("rotation: current assignment unavailable; using shared credentials", "error", err)
		}
		return ""
	}
	account, err := d.rotationStore.GetAccount(ctx, accountID)
	if err != nil {
		taskLog.Debug("rotation: assigned account unavailable; using shared credentials", "error", err)
		return ""
	}
	if !strings.EqualFold(account.Vendor, provider) {
		return ""
	}
	return account.HomeDir
}
```

**ATENCAO, e o ponto critico da ponte:** essa versao devolve `account.HomeDir`, UM caminho unico
para TODOS os providers. Restaurar isso AS-IS reintroduz o defeito do FATO 3: `HomeDir` serviria o
`agy` e faria o `kiro` rodar sem credencial em silencio (`kiro_home.go:73-75`). O wiring de 02/07
foi escrito antes do layout de slot com raizes por provider. **A ponte tem de mapear
`slot_root -> subdiretorio por provider`, nao devolver um HomeDir cru.**

## 4. CONTRATO SQL QUE SOBREVIVEU (base da ponte)

`server/migrations/123_rotation.up.sql` (literal, colunas relevantes):
```sql
CREATE TABLE accounts (
    account_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vendor            TEXT NOT NULL,
    tenant_id         UUID NOT NULL,
    priority          INT NOT NULL DEFAULT 0,
    home_dir          TEXT NOT NULL DEFAULT '',
    config_dir        TEXT NOT NULL DEFAULT '',
    status            TEXT NOT NULL DEFAULT 'available'
        CHECK (status IN ('available', 'leased', 'exhausted', 'cooldown', 'degraded')),
    tokens_per_window BIGINT NOT NULL DEFAULT 0,
    tokens_used       BIGINT NOT NULL DEFAULT 0,
    window_start      TIMESTAMPTZ,
    cooldown_until    TIMESTAMPTZ,
    ...
);
CREATE TABLE credentials (
    credential_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id    UUID NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    vendor        TEXT NOT NULL,
    secret_ref    TEXT NOT NULL,
    format        TEXT NOT NULL,
    ...
);
CREATE TABLE assignments (
    agent_id    UUID PRIMARY KEY,
    account_id  UUID NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE rotation_events (
    ... reason TEXT NOT NULL CHECK (reason IN ('quota_exhausted_reactive','quota_forecast_proactive','login_failed','manual')), ...
);
```
`server/migrations/124_approved_accounts.up.sql` (literal):
```sql
CREATE TABLE approved_accounts (
    approved_id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL,
    account_id     UUID NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    allowed        BOOLEAN NOT NULL DEFAULT true,
    worktype_scope TEXT CHECK (worktype_scope IN ('GENERAL','HEAVY','CHEAP','REVIEW')),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, account_id)
);
```

Leituras diretas para a ponte:
- `accounts.vendor` e comparado com `strings.EqualFold(account.Vendor, provider)`. Os valores tem de
  ser exatamente os providers do daemon: **`antigravity`, `kiro`, `codex`**. Nao `agy`, nao `kiro-cli`.
- Existem `home_dir` e `config_dir`, mas **nao existe coluna para `xdg-data` (kiro) nem para
  `codex`**. Essa e a lacuna de schema da ponte.
- `assignments.agent_id` e PRIMARY KEY: **uma conta por agente por vez**. Combina com rotacao, nao
  com multi-conta simultanea do mesmo agente.
- `approved_accounts.worktype_scope` (GENERAL/HEAVY/CHEAP/REVIEW) e o gancho pronto para "Opus e
  Gemini Pro sao HEAVY", sem inventar schema novo.
- `credentials.secret_ref` e referencia, nao valor: coerente com nunca trazer segredo ao daemon.

## 5. PONTE PROPOSTA (aditiva, sem migration)

Preencher `accounts.home_dir` com o **SLOT ROOT** (`/home/ec2-user/.agent-cred-homes/slots/slot-NNN`)
e derivar o subdiretorio por provider em Go, no unico ponto que hoje esta vazio:

```go
// daemon.go:3448  ANTES
	credentialAccountHome := ""

// daemon.go:3448  DEPOIS
	credentialAccountHome := d.credentialAccountHomeForTask(ctx, task, provider, taskLog)

// reintroduzido, mas com a correcao do FATO 3: nunca devolver o root cru.
func (d *Daemon) credentialAccountHomeForTask(ctx context.Context, task Task, provider string, taskLog *slog.Logger) string {
	if d.accountStore == nil || task.AgentID == "" || provider == "" {
		return "" // preserva o comportamento de hoje (FATO 8): uma conta por provider
	}
	accountID, err := d.accountStore.CurrentAssignment(ctx, task.AgentID)
	if err != nil { return "" }
	account, err := d.accountStore.GetAccount(ctx, accountID)
	if err != nil || !strings.EqualFold(account.Vendor, provider) { return "" }
	return providerSlotRoot(provider, account.HomeDir) // account.HomeDir = SLOT ROOT
}

func providerSlotRoot(provider, slotRoot string) string {
	if slotRoot == "" { return "" }
	switch provider {
	case "antigravity": return filepath.Join(slotRoot, "home")     // -> .gemini/antigravity-cli
	case "kiro":        return filepath.Join(slotRoot, "xdg-data") // -> kiro-cli/data.sqlite3
	case "codex":       return filepath.Join(slotRoot, "codex")
	case "cline":       return filepath.Join(slotRoot, "cline")
	default:            return ""
	}
}
```
Vantagens: nenhuma migration, nenhum preparer alterado (`execenv.go:301`, `:312` ja fazem o resto),
e `""` continua significando comportamento global — o FATO 8 nao regride.

Validacao obrigatoria antes de devolver o caminho: exigir que a credencial exista
(`<root>/xdg-data/kiro-cli/data.sqlite3` para kiro, `<root>/home/.gemini/antigravity-cli` para agy)
e, se nao existir, devolver `""` COM log explicito em vez de seguir e falhar a task sem causa.

## 6. LACUNAS QUE EXIGEM DECISAO (nao acao minha)

1. **Cross-host.** Registry e slots no ORQ2; daemon no ORQ1. Um caminho de slot do ORQ2 nao existe no
   filesystem do ORQ1. Opcoes: mover o daemon para o ORQ2, montar/replicar os slots, ou o daemon
   passar a consultar um servico. Decisao do owner.
2. **Identidade (FATO 5).** `registry.json` indexa por `terminal_id`; `assignments` indexa por
   `agent_id` (UUID). Falta a tabela/coluna de correspondencia `terminal_id <-> agent_id`.
3. **Seed.** `accounts`, `approved_accounts` e `assignments` com ZERO linhas: mesmo com o wiring
   restaurado, `CurrentAssignment` devolve sem-atribuicao e o resultado e `""`, ou seja o
   comportamento de hoje. Sem popular as tabelas a ponte nao muda nada.
4. **Kiro tem so 5 slots** com `xdg-data/kiro-cli/data.sqlite3` (139, 140, 142, 143, 149) contra
   22 com agy. A selecao de conta para kiro precisa ser restrita a esses ou os outros 17 logados.
5. **Codex ignora AccountHome hoje:** `daemon.go:3476` passa `CredentiallessGateway: true` e
   `execenv.go:284-287` desvia para `prepareCredentiallessCodexHome`, sem usar `AccountHome`. Com
   codex obrigatorio (FATO 7), preencher o campo nao basta para ele.

## 7. NAO-AFIRMACOES
- Nao reabri nenhum dos 8 fatos. A secao 2 e precisao com evidencia de `git show`/`git log`, e a
  secao 1 e retirada de enquadramento meu, nao do FATO 1.
- Nao editei `.go`, nao rodei build/vet/test, nao substitui binario, nao reiniciei daemon, nao commitei.
- Nao li conteudo de `registry.json` nem de credencial: apenas `ls -l` de nome, tamanho e permissao.
- Nao consultei o Postgres do ORQ1: as ZERO linhas sao o FATO 5, medido pelo TL, nao por mim.
- opencode fora de escopo (FATO 7); nao aparece em nenhuma proposta acima.
