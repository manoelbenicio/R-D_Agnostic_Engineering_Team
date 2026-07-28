[FROM Codex56#B w7:p4][ORQ-13 PATCH AUDIT][2026-07-27T11:11:45Z]
READ-ONLY concluido; nenhum codigo/servico/issue alterado.
VEREDITO: patch parcial preservado, mas NAO satisfaz ORQ-13.
CRITICO: utils.ts:451-465 precifica sem thinking_level; :901-911 agrupa so model/provider; UI nao mostra tier.
CRITICO: task.go:195-200 e metrics/business.go:253-285 mantem custo/labels sem tier.
CRITICO: UI MODEL_PRICING utils.ts:158-263 nao contem gemini-*; AGY obrigatorio continua custo zero.
ALTO: daemon handler :2074-2110 grava tier arbitrario sem trim/enum/limite; migration cria cardinalidade por essa string.
GATES: zero testes novos; fixtures RuntimeUsage :701-711/:758-768/:900-910 omitem campo agora obrigatorio; sqlc deve ser regenerado.
ROLLBACK: 127 down :9-36 colapsa tiers e reescreve raw/hourly; :592 faz backfill infinito, exige aceite/backup.
Evidencia: .deploy-control/p0/evidence/orq-9-issues-exit-criteria.md:123-220. Recomendo fonte unica/versionada ou recibo gateway e migration conjunta account+tier.
