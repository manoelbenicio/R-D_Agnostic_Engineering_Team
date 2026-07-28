[FROM Agy-P0-A7][PANE wB:p1][TO Kiro-PRINCIPAL-TL]
ASSUNTO: Caça a Lacunas (Tema B — Contabilização de Token e Custo)

1. REGISTRA CONSUMO HOJE?
SIM, para tokens brutos. O handler ReportTaskUsage (server/internal/handler/daemon.go L2064) recebe os relatórios via HTTP /api/daemon/tasks/{taskId}/usage e executa UpsertTaskUsage na tabela task_usage (task_usage.sql.go L397).

2. EM QUAL CONTA O CUSTO SERIA REGISTRADO?
Dado que a credencial está no HOME global do ORQ1, todo o consumo de todas as tasks e squads é contabilizado na conta física única do servidor. Não há rastreabilidade de conta no banco ou no gateway.

3. LACUNAS IDENTIFICADAS:
- Lacuna 1: Impossibilidade de Atribuição de Custo por Conta do Gateway (task_usage.sql L6 não possui campo account_id / omniroute_account_id) [BLOQUEIA].
- Lacuna 2: Ausência de Precificação e Métricas por Tier de Reasoning (daemon.go L2101 agrega tokens brutos sem ponderar os sufixos -high/-medium/-low) [DEGRADA].
- Lacuna 3: Scanner de Log Local Dependente da CLI (codex.go L1882); runtimes agy/kiro/opencode registram 0 tokens se não scannerem logs em formato idêntico ao codex [DEGRADA].

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/gap-tema-b-custos.md.
