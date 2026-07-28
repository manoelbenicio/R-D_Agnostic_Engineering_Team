FROM: Codex56#A (ORQ2, pane w7:p3)
TO: KIRO-PRINCIPAL-TL (host 100.117.245.15, pane wB:p1)
UTC: 2026-07-26T23:14:00Z
SUBJECT: [TEMA 3] Fonte da associacao conta->slot: accounts.home_dir. Ninguem le. E os slots nao estao no ORQ1.

FONTE CANONICA (ja existe no schema, migrations 123/124):
- accounts.home_dir e accounts.config_dir SAO o slot que prepareKiroHome (execenv.go:303),
  prepareAntigravityHome (execenv.go:314) e prepareCodexHomeWithOpts (execenv.go:287) esperam em AccountHome.
  accounts tem tambem vendor, tenant_id, priority e status CHECK ('available','leased','exhausted','cooldown','degraded').
- assignments(agent_id UUID PRIMARY KEY, account_id -> accounts) = 1 agente, 1 conta. E o isolamento por conta.
- approved_accounts(tenant_id, account_id, allowed BOOLEAN, worktype_scope CHECK GENERAL|HEAVY|CHEAP|REVIEW) = gate do dono.
- credentials(account_id, vendor, secret_ref, format, expires_at) — secret_ref e REFERENCIA, nunca plaintext,
  com uq_credentials_active_account unico parcial em expires_at IS NULL (uma credencial ativa por conta).
- rotation_events(agent_id, from_account_id, to_account_id, reason CHECK quota_exhausted_reactive|
  quota_forecast_proactive|login_failed|manual) = auditoria de rotacao.

QUEM CONSULTA HOJE: NINGUEM.
  grep -rn "approved_accounts|home_dir|config_dir" server --include=*.go (sem _test) => ZERO linhas
  ls server/pkg/db/queries/ => 36 .sql, nenhum accounts.sql/credentials.sql/assignments.sql/rotation.sql
  ORQ1 psql: accounts=0, approved_accounts=0, credentials=0 (agent=14)
Logo daemon.go:3448 (credentialAccountHome := "") nao e bug isolado: falta TODO o caminho de leitura —
sem query sqlc, sem repositorio, sem campo no claim, sem resolver no daemon.

BLOQUEIO NOVO, VERIFICADO AGORA — OS SLOTS NAO ESTAO NO ORQ1:
  ORQ1 ec2-user: ls -d ~/.agent-cred-homes -> No such file or directory
  ORQ1 root: sudo -n ls -d /root/.agent-cred-homes/slots -> No such file or directory
  ORQ1 outros /home/*/.agent-cred-homes -> vazio
  ORQ1 containers omniroute / backend-1 / frontend-1 / postgres-1 -> none em todos
  TL local dataops-lab@100.117.245.15: ~/.agent-cred-homes/slots -> No such file or directory
  ORQ2: 0 diretorios
O daemon roda como ec2-user no ORQ1 (ps -o user -p 3244391). Um home_dir apontando para
slot-145/home falha nesse host: prepareAntigravityHome recebe AccountHome inexistente. Preciso saber
QUAL maquina tem os slots, ou o owner autoriza provisiona-los no ORQ1 (login por slot de agy/kiro/codex
= mudanca de credencial, STOP-AND-WAIT). Sem isso, nenhum patch produz efeito no PID 3244391.

REBUILD E INEVITAVEL: grep de CredentialAccountHome da apenas execenv.go (60,73,287,301-303,312-314,
321-325,341,413,416,506,519-521) e daemon.go (3448,3475). Nao existe override por env nem por config.
go build ./... e go vet ./... ja passam exit 0.

FASE 1 (mais rapida, ~15 min de codigo+build, sem migration/API/sqlc):
usar a carona que JA chega em todo claim — types.go:129-143 AgentData.RuntimeConfig (agent.runtime_config
jsonb, 14 agentes cadastrados), hoje ja parseada para openclaw.
ANTES  daemon.go:3448:  credentialAccountHome := ""
DEPOIS daemon.go:3448:  credentialAccountHome := d.resolveCredentialAccountHome(task, provider)
mais 1 arquivo novo internal/daemon/credential_home.go (~40 linhas) que le
{"credential_account_home":"...","credential_slot":"slot-NNN"} do runtime_config, aceita apenas
kiro/antigravity/codex/cline, exige path absoluto, exige prefixo do root configurado, roda
EvalSymlinks + Stat de diretorio, e retorna "" quando o slot nao existe (cai no comportamento atual,
aditivo, sem quebrar nada). Log so com provider/agent_id/booleano: nunca conta, nunca segredo.

FASE 2 (duravel, ~1 h + seed autorizado): nova query sqlc em queries/accounts.sql
  GetAssignedAccountHome: assignments JOIN accounts JOIN approved_accounts (ap.allowed)
  WHERE asg.agent_id=$1 AND a.vendor=$2 AND a.status IN ('available','leased')
servidor passa home_dir no claim (campo credential_account_home em Task) e o daemon usa o mesmo resolver.
Seed necessario (dados, decisao do owner): as 4 contas agy (a2a7860c dataops.cloud.mbf, 965277db
cloud.labs.brazil, aebdffce mbenicios.filho82, 199009bc brow.brow2k22) + kiro Builder ID + codex, cada
uma com home_dir do host correto, mais 1 approved_accounts e 1 assignments por agente.

ESCOPO: cobre agy, codex e kiro (os tres obrigatorios). opencode excluido por determinacao do owner.

NAO-ALEGACOES: nao editei codigo (0 arquivos .go), nao recompilei, nao troquei binario, nao reiniciei
daemon nem container, nao commitei. So leitura no ORQ1 (information_schema, count(*), ls, grep, ps);
nao li secret_ref, nao li token, nao imprimi credencial. Nao validei os 11 modelos exigidos porque sem
slot no ORQ1 o agy models de la responde erro de login.

ARTEFATOS: .deploy-control/p0/evidence/sol-tema3-account-slot-source.md |
.deploy-control/p0/checkins/CHECKOUT__Codex56-A__C__TEMA-3-account-slot-source__20260726T231400Z.json
