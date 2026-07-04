# EVIDENCE LOG — SEV-0 Stop-Ship

| Item | Evidence Type | Link / Path | Command | Output Snippet | Notes |
|------|---------------|-------------|---------|----------------|-------|
| Repo HEAD | git | `multica-auth-work` | `git log -1 --oneline` | `52cdd87 rotation-parity polyglot...` | main |
| Uncommitted | git | idem | `git status --porcelain \| wc -l` | `91` | ISSUE-005 |
| prodex source | fs | `/tmp/prodex-audit-7750da9` | `git rev-parse HEAD` | `7750da9b6a5c` | pin correto |
| prodex binário | fs | idem | `ls target/release/prodex` | (ausente) | **ISSUE-001** |
| Rust | env | fleet | `command -v cargo` | (ausente) | bloqueia build |
| Postgres/Redis | docker | fleet | `docker ps` | `deploy-postgres-1 pg17 healthy :5432` / redis :6379 | OK |
| daemon.go LOC | fs | `server/internal/daemon/daemon.go` | `wc -l` | `4626` | hotspot |
| l2 client LOC | fs | `server/internal/l2runtime/client.go` | `wc -l` | `762` | hot path |
| test files | fs | server | `find -name *_test.go \| wc -l` | `317` | cobertura % a medir |
| concorrência | grep | server/internal | `grep -l 'go func\|Mutex\|chan'` | `78` arquivos | race risk |
| auth clobber | fs | `~/.codex-*/auth.json` | `sha256sum` | 4 idênticos → após fix 4 distintos | ISSUE-002 resolvido |
| dashboard crash | run | `plan_dashboard.py` | `PYTHONIOENCODING=ascii python3 ...` | `UnicodeEncodeError` → após fix rc=0 | ISSUE-003 resolvido |
| dashboard QA | test | `scripts/dashboard/test_plan_dashboard.py` | `python3 ...` | `29/29 PASS` | evidência |
| openspec tasks | cli | change | `openspec list --json` | `0/51 in-progress` | ISSUE-004 resolvido |
| CI workflows | fs | `.github/workflows/` | `ls` | `ci.yml desktop-smoke release ...` | ISSUE-007 verificar gates Go |