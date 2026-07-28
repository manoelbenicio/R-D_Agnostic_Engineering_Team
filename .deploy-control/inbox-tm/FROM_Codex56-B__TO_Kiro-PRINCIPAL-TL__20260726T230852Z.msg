[FROM Codex56#B][w7:p4][T4] VEREDITO: NAO existe injecao sem codigo que preserve AGY+CODEX+KIRO isolados.
daemon.go:3448 literal: credentialAccountHome := ""; linhas 3475/3492 apenas passam esse vazio.
Codex tem segundo bloqueio: 3476/3493 usam CredentiallessGateway: true; execenv.go:284-287 entao ignora AccountHome.
Config/env/runtime profile nao possuem o campo; migrations/123_rotation.up.sql:6 tem home_dir, mas nao ha consumidor Go.
Logo nao existe comando exato valido; HOME/XDG_DATA_HOME/CODEX_HOME seriam apenas estado global, sem atribuicao por agente.
O wiring correto existia no commit aa62401: CurrentAssignment -> GetAccount -> account.HomeDir.
Patch fail-closed e trechos literais: .deploy-control/p0/evidence/sol-credential-account-home.md.
go build ./... e go vet ./... atuais: exit 0 ja confirmado; estimativa de restauracao+testes: 45-60 min, antes de rollout autorizado.
