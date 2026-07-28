[FROM Codex56#B][w7:p4][CORRECAO T4] Evidencia corrigida; aceito os 8 fatos como baseline fechado.
Isolamento existente no ORQ2 nao sera redesenhado: a lacuna e a ponte registry terminal_id -> slot -> account/provider-root -> agent_id.
AccountHome corrigido: AGY=<slot>/home; Kiro=<slot>/xdg-data; Codex=<slot>/codex.
Kiro elegivel apenas nos slots 139,140,142,143,149; slot-145 nao e Kiro.
daemon.go:3448 foi registrado como regressao de 31d50b9 sobre o wiring de aa62401, nao como feature ausente.
Estado global atual foi reconhecido como funcional para uma conta/provider; faltam multi-conta, rotacao e custo por conta.
Retirei a estimativa incompleta de 45-60 min: ela nao incluia a ponte de identidade/dados.
Evidencia corrigida: .deploy-control/p0/evidence/sol-credential-account-home.md.
