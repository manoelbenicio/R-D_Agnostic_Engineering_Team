# TEMA 3 — de onde o daemon DEVE tirar a associacao conta→slot (PROPOSTA, NAO APLICADA)

Autor: Codex56#A (ORQ2, pane `w7:p3`) · UTC 2026-07-26T23:12Z · escopo: agy, codex, kiro (opencode
fora de escopo por determinacao do owner). Nada editado, nada compilado, nada reiniciado.

## 1. A fonte canonica JA EXISTE no schema: migrations 123 e 124

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
    ...
CREATE TABLE assignments (
    agent_id    UUID PRIMARY KEY,
    account_id  UUID NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE credentials (
    credential_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id    UUID NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    vendor        TEXT NOT NULL,
    secret_ref    TEXT NOT NULL,
    format        TEXT NOT NULL,
    ...
CREATE TABLE rotation_events ( ... reason TEXT NOT NULL
        CHECK (reason IN ('quota_exhausted_reactive','quota_forecast_proactive','login_failed','manual')) ...);
```

`server/migrations/124_approved_accounts.up.sql`:

```sql
CREATE TABLE approved_accounts (
    approved_id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL,
    account_id     UUID NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    allowed        BOOLEAN NOT NULL DEFAULT true,
    worktype_scope TEXT CHECK (worktype_scope IN ('GENERAL','HEAVY','CHEAP','REVIEW')),
    UNIQUE (tenant_id, account_id)
);
```

Leitura direta: **`accounts.home_dir` e `accounts.config_dir` SAO o slot** que
`prepareKiroHome` (execenv.go:303), `prepareAntigravityHome` (execenv.go:314) e
`prepareCodexHomeWithOpts` (execenv.go:287) esperam em `AccountHome`. `assignments(agent_id → account_id)`
e a associacao 1 agente = 1 conta, isto e o isolamento por conta que o owner pediu.
`approved_accounts.allowed` e o gate de aprovacao do dono. `credentials.secret_ref` guarda **referencia**,
nunca plaintext (e ha `uq_credentials_active_account` unico parcial em `expires_at IS NULL`, ou seja uma
credencial ativa por conta). `rotation_events` e a auditoria de rotacao.

## 2. Quem consulta isso no codigo: NINGUEM. Provas.

```text
$ grep -rn "approved_accounts|home_dir|config_dir" server --include=*.go   (sem _test)
(zero linhas)

$ ls server/pkg/db/queries/    → 36 arquivos .sql
  nenhum accounts.sql, credentials.sql, assignments.sql nem rotation.sql
  (existe runtime.sql, runtime_profile.sql, runtime_usage.sql, task_usage.sql, ...)

ORQ1, docker exec multica-dev-transition-postgres-1 psql -U multica_transition -d multica_transition:
  COUNT accounts     | 0
  COUNT approved     | 0
  COUNT credentials  | 0
  agent              | 14 linhas
```

Conclusao: `daemon.go:3448 credentialAccountHome := ""` nao e um bug isolado — **nao existe caminho de
leitura**: sem query sqlc, sem repositorio, sem campo no payload de claim, sem resolver no daemon. As
tabelas foram criadas (123/124) e nunca populadas nem ligadas.

## 3. BLOQUEIO VERIFICADO: os slots nao estao em nenhum host que eu alcanco

```text
ORQ1 (ec2-user@100.118.244.61): ls -d ~/.agent-cred-homes → No such file or directory
ORQ1 root:  sudo -n ls -d /root/.agent-cred-homes/slots → No such file or directory
ORQ1 outros /home/*/.agent-cred-homes → vazio
ORQ1 containers (omniroute, backend-1, frontend-1, postgres-1) → none em todos
TL local (dataops-lab@100.117.245.15): ~/.agent-cred-homes/slots → No such file or directory
ORQ2 (este host): ~/.agent-cred-homes/slots → 0 diretorios
```

O daemon roda como `ec2-user` no ORQ1 (`ps -o user -p 3244391`). Um `home_dir` apontando para
`~/.agent-cred-homes/slots/slot-145/home` **falha nesse host**: `prepareAntigravityHome` recebe um
AccountHome inexistente. O slot-145 medido pelo TL esta em outra maquina. Antes de qualquer patch,
alguem precisa dizer QUAL host tem os slots — ou provisiona-los no ORQ1 (login por slot de agy, kiro e
codex), que e mudanca de credencial/auth e portanto decisao do owner.

## 4. Caminho mais rapido HOJE, sem tocar em DB, API nem sqlc

Nao existe override por env: `grep -rn "CredentialAccountHome" server --include=*.go` da apenas
execenv.go (60/73/287/301-303/312-314/321-325/341/413/416/506/519-521) e daemon.go (3448/3475).
Logo **qualquer solucao exige recompilar** — e `go build ./...` e `go vet ./...` ja passam exit 0.

Carona que ja existe e chega em todo claim: `types.go:129-143 AgentData.RuntimeConfig json.RawMessage`
(`agent.runtime_config jsonb` no banco, 14 agentes ja cadastrados), ja parseado hoje para openclaw via
`decodeOpenclawRuntimeConfig`. Isso da fase 1 sem migration, sem endpoint novo e sem sqlc.

ANTES — `daemon.go:3447-3448` (literal):

```go
	localAssignment, _ := findLocalDirectoryAssignment(task.ProjectResources, d.cfg.DaemonID)
	credentialAccountHome := ""
```

DEPOIS proposto:

```go
	localAssignment, _ := findLocalDirectoryAssignment(task.ProjectResources, d.cfg.DaemonID)
	// Per-account credential isolation. Source of truth order:
	//   1. agent.runtime_config.credential_account_home (per-agent binding,
	//      already delivered on every claim — no API/schema change);
	//   2. d.cfg.CredentialHomesRoot + "/" + slot from runtime_config.credential_slot.
	// Empty result keeps today's shared/global behavior, so this is additive
	// and fails closed to the current semantics instead of to an error.
	credentialAccountHome := d.resolveCredentialAccountHome(task, provider)
```

E o resolver (arquivo novo `internal/daemon/credential_home.go`, ~40 linhas):

```go
// resolveCredentialAccountHome returns a validated per-account HOME for the
// providers that support isolation (kiro, antigravity, codex, cline).
// It NEVER reads credential material: it resolves a directory path only.
func (d *Daemon) resolveCredentialAccountHome(task Task, provider string) string {
	switch provider {
	case "kiro", "antigravity", "codex", "cline":
	default:
		return ""
	}
	if task.Agent == nil || len(task.Agent.RuntimeConfig) == 0 {
		return ""
	}
	var cfg struct {
		CredentialAccountHome string `json:"credential_account_home"`
		CredentialSlot        string `json:"credential_slot"`
	}
	if err := json.Unmarshal(task.Agent.RuntimeConfig, &cfg); err != nil {
		return "" // broken JSON must never block dispatch
	}
	candidate := strings.TrimSpace(cfg.CredentialAccountHome)
	if candidate == "" && strings.TrimSpace(cfg.CredentialSlot) != "" && d.cfg.CredentialHomesRoot != "" {
		slot := strings.TrimSpace(cfg.CredentialSlot)
		if !strings.HasPrefix(slot, "slot-") || strings.ContainsAny(slot, "/\\.\x00") {
			return ""
		}
		candidate = filepath.Join(d.cfg.CredentialHomesRoot, slot, "home")
	}
	if candidate == "" || !filepath.IsAbs(candidate) {
		return ""
	}
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "" // slot missing on this host: fall back to shared behavior
	}
	if d.cfg.CredentialHomesRoot != "" {
		root, rerr := filepath.EvalSymlinks(d.cfg.CredentialHomesRoot)
		if rerr != nil || (resolved != root && !strings.HasPrefix(resolved, root+string(os.PathSeparator))) {
			return "" // no escape outside the configured root
		}
	}
	if info, serr := os.Stat(resolved); serr != nil || !info.IsDir() {
		return ""
	}
	d.logger.Info("credential account home resolved",
		"provider", provider, "agent_id", task.AgentID, "home_present", true)
	return resolved
}
```

Observabilidade: log so registra provider/agent_id/booleano — nunca o conteudo do home, nunca conta,
nunca segredo.

## 5. Fase 2 (duravel): ligar accounts/assignments/approved_accounts

Query proposta em `server/pkg/db/queries/accounts.sql` (nova, sqlc):

```sql
-- name: GetAssignedAccountHome :one
SELECT a.account_id, a.vendor, a.home_dir, a.config_dir, a.status
FROM assignments asg
JOIN accounts a ON a.account_id = asg.account_id
JOIN approved_accounts ap
  ON ap.account_id = a.account_id AND ap.tenant_id = a.tenant_id AND ap.allowed
WHERE asg.agent_id = $1 AND a.vendor = $2 AND a.status IN ('available','leased')
LIMIT 1;
```

O servidor passa `home_dir` no claim (campo novo `credential_account_home` em `Task`) e o daemon usa o
mesmo resolver da fase 1. Seed necessario (mudanca de dados, decisao do owner): 4 contas agy do owner
(`a2a7860c dataops.cloud.mbf`, `965277db cloud.labs.brazil`, `aebdffce mbenicios.filho82`,
`199009bc brow.brow2k22`) + kiro (Builder ID) + codex, cada uma com `home_dir` = slot real do host que
tiver os slots, mais 1 linha em `approved_accounts` por conta e 1 em `assignments` por agente.

## 6. Estimativa

Fase 1 (resolver + troca da linha 3448 + testes de path-escape): 1 arquivo novo + 1 linha alterada,
`go build`/`go vet` ja verdes, ~15 min de codigo + build. Fase 2 (query sqlc + campo no claim + seed):
~1 h, e depende do seed autorizado. **Ambas exigem recompilar; nao existe caminho sem rebuild.**
Sem resolver antes o item 3 (onde estao os slots), nem a fase 1 produz efeito no ORQ1.

## 7. Nao-alegacoes

- Nao editei codigo, nao recompilei, nao troquei binario, nao reiniciei daemon nem container, nao commitei.
- So executei leitura: `information_schema`, `count(*)`, `ls`, `grep`, `ps`. Nao li `secret_ref`, nao li
  token, nao imprimi credencial.
- Nao validei os 11 modelos exigidos (opus-4-6-thinking, sonnet-4-6, gemini-3.1-pro-high/low,
  3.5-flash e 3.6-flash high/medium/low): sem slot no ORQ1, `agy models` la responde erro de login.
- Nao provei que `/tmp/multica-auth-fixed` corresponde a este codigo.
