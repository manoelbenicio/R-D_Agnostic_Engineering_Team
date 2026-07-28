#!/usr/bin/env bash
# ORQ-13 integration script — GTM R2 canonical sequence
#
#   ORQ21  ->  ORQ13 (migration 127)  ->  ORQ12 (migration 128)
#
# STATUS: NAO EXECUTADO. Este arquivo e um artefato de revisao.
# Autor: Opus48#B (ORQ2, w6:p2). Eu escrevi os 4 commits do ORQ-13, portanto NAO
# executo o merge nem me autoaprovo. Execucao exige:
#   (1) ORQ-21 com veredito PASS registrado;
#   (2) revisao independente (nao o autor);
#   (3) autorizacao de merge do owner.
#
# Reserva R1: migration 127, expira 2026-07-29T13:21:29Z.
#
# Convencoes: falha em qualquer gate = ABORTA sem tocar o alvo. Nenhum passo usa
# `git reset --hard`, `git push`, `--amend` ou `docker prune`.

set -euo pipefail

REPO=/home/ec2-user/workspace/R-D_Agnostic_Engineering_Team
TARGET=integration/dev-transition-candidate-20260719
BASE=0cb8aeb
ORQ13_COMMITS=(11ef715 67e9a4c b1f08e3 c0e93a2)   # ordem obrigatoria
RESERVATION_EXPIRY="2026-07-29T13:21:29Z"
EVID=/home/ec2-user/.cache/orq13-integration-evidence

# Fingerprints ja provados, usados como gate de imutabilidade
FP_TRACKED7=b87bc5cae3e877af9a26ebe39cce7f0a90fff2a126e075c7f5d12799d04707ce
SHA_MIG_UP=0ea3005da0ee257618062552cf8792f9c2e6ed478dce3ba174bf08692486cac1
SHA_MIG_DOWN=74354ae28dee526c7dbc6bc6733471a59c2f3dabfe5a7fe609fe20d747e61113
PG_DIGEST=pgvector/pgvector@sha256:d2ef61f42ef767baa5a1475393303cc235bcd92febd9d7014eddb48b41f3bad0
LINT_SHA256=6b89d77b6396f81decee882f20473486bda5ef28b160f9a13b08f24848d9003c

die(){ echo "ABORT: $*" >&2; exit 1; }
ok(){  echo "GATE OK: $*"; }

install -d -m 0700 "$EVID"
cd "$REPO"

# ---------------------------------------------------------------------------
# G0  PRE-CONDICAO DE SEQUENCIA — ORQ-21 PRIMEIRO
# ---------------------------------------------------------------------------
# ORQ-13 NAO entra antes de ORQ-21 ter PASS. Este gate e humano por natureza:
# exige o arquivo de veredito no board. Sem ele, o script para aqui.
ORQ21_VERDICT=$(ls -t .deploy-control/p0/checkins/ 2>/dev/null \
  | grep -iE 'ORQ-?21' | head -1 || true)
[ -n "$ORQ21_VERDICT" ] || die "G0: nenhum check-out de ORQ-21 encontrado"
grep -qiE '"(verdict|status)"[^,]*(PASS)' \
  ".deploy-control/p0/checkins/$ORQ21_VERDICT" \
  || die "G0: ORQ-21 sem veredito PASS em $ORQ21_VERDICT — ORQ-13 NAO entra"
ok "G0 ORQ-21 PASS em $ORQ21_VERDICT"

# ORQ-21 ja integrado ao alvo?
git merge-base --is-ancestor "$(git rev-parse agent/codex-b/orq-21)" "$TARGET" \
  || die "G0b: ORQ-21 ainda nao esta contido em $TARGET — sequencia canonica violada"
ok "G0b ORQ-21 contido no alvo"

# ---------------------------------------------------------------------------
# G1  RESERVA DENTRO DA VALIDADE
# ---------------------------------------------------------------------------
[ "$(date -u +%s)" -lt "$(date -u -d "$RESERVATION_EXPIRY" +%s)" ] \
  || die "G1: reserva R1 da migration 127 expirou em $RESERVATION_EXPIRY — pedir renovacao ao registrador"
ok "G1 reserva valida ate $RESERVATION_EXPIRY"

# ---------------------------------------------------------------------------
# G2  ESTADO LIMPO E SEM DRIFT
# ---------------------------------------------------------------------------
[ -z "$(git -C "$REPO" status --porcelain)" ] || die "G2: repo principal sujo"
for b in agent/opus48-b/squad-default-leader agent/opus48-b/orq-13-thinking-level; do
  git rev-parse -q --verify "$b" >/dev/null || die "G2: branch ausente: $b"
done
[ "$(git rev-parse agent/opus48-b/squad-default-leader)" = \
  "$(git rev-parse 67e9a4c)" ] || die "G2: squad divergiu de 67e9a4c"
[ "$(git rev-parse agent/opus48-b/orq-13-thinking-level)" = \
  "$(git rev-parse c0e93a2)" ] || die "G2: orq13 divergiu de c0e93a2"
ok "G2 branches nos commits esperados"

# ---------------------------------------------------------------------------
# G3  MIGRATION 127 AINDA EXCLUSIVA  +  128 LIVRE PARA ORQ-12
# ---------------------------------------------------------------------------
CLAIM127=""
for b in $(git for-each-ref --format='%(refname:short)' refs/heads); do
  if git ls-tree -r --name-only "$b" -- multica-auth-work/server/migrations 2>/dev/null \
     | grep -q '/127_'; then
    [ "$b" = agent/opus48-b/orq-13-thinking-level ] || CLAIM127="$CLAIM127 $b"
  fi
done
[ -z "$CLAIM127" ] || die "G3: migration 127 reivindicada por outra branch:$CLAIM127"
ok "G3 migration 127 exclusiva do ORQ-13"

# ---------------------------------------------------------------------------
# G4  IMUTABILIDADE DO CONTEUDO
# ---------------------------------------------------------------------------
S=multica-auth-work/server
GOT=$(git diff "$BASE" c0e93a2 -- \
  $S/internal/daemon/daemon.go $S/internal/daemon/types.go $S/internal/handler/daemon.go \
  $S/pkg/db/generated/models.go $S/pkg/db/generated/task_message.sql.go \
  $S/pkg/db/generated/task_usage.sql.go $S/pkg/db/queries/task_usage.sql | sha256sum | cut -d' ' -f1)
[ "$GOT" = "$FP_TRACKED7" ] || die "G4: fingerprint dos 7 rastreados divergiu: $GOT"
[ "$(git show c0e93a2:$S/migrations/127_task_usage_thinking_level.up.sql   | sha256sum | cut -d' ' -f1)" = "$SHA_MIG_UP" ]   || die "G4: migration up divergiu"
[ "$(git show c0e93a2:$S/migrations/127_task_usage_thinking_level.down.sql | sha256sum | cut -d' ' -f1)" = "$SHA_MIG_DOWN" ] || die "G4: migration down divergiu"
ok "G4 conteudo identico ao provado"

# ---------------------------------------------------------------------------
# G5  DRY-RUN DE MERGE, SEM TOCAR O ALVO
# ---------------------------------------------------------------------------
git merge-tree --write-tree "$TARGET" agent/opus48-b/squad-default-leader   >/dev/null \
  || die "G5: conflito squad x alvo"
git merge-tree --write-tree "$TARGET" agent/opus48-b/orq-13-thinking-level  >/dev/null \
  || die "G5: conflito orq13 x alvo"
ok "G5 merge-tree limpo"

# ---------------------------------------------------------------------------
# G6  GATE FUNCIONAL NA ARVORE COMBINADA — banco EFEMERO, nunca producao
# ---------------------------------------------------------------------------
# Reproduz exatamente o gate ja executado em 2026-07-28T13:35Z.
WT=$(mktemp -d -p /home/ec2-user/.cache orq13-int-XXXX); chmod 700 "$WT"
cleanup(){
  set +e
  [ -n "${LPORT:-}" ] && pkill -f "L 127.0.0.1:$LPORT:127.0.0.1:47490"
  ssh -o BatchMode=yes ec2-user@100.118.244.61 \
    'docker rm -f orq13-int-pg >/dev/null 2>&1; rm -f /home/ec2-user/.orq13-int-pw'
  git -C "$REPO" worktree remove --force "$WT/wt" 2>/dev/null
  rm -rf "$WT"
  echo "teardown concluido"
}
trap cleanup EXIT

git worktree add --detach "$WT/wt" "$TARGET" >/dev/null
( cd "$WT/wt"
  for c in "${ORQ13_COMMITS[@]}"; do
    git cherry-pick -x "$c" >/dev/null 2>&1 || { git cherry-pick --abort; die "G6: conflito no cherry-pick de $c"; }
  done )
ok "G6a cherry-pick dos 4 commits sem conflito"

ssh -o BatchMode=yes ec2-user@100.118.244.61 "bash -s" <<REMOTE
set -euo pipefail
docker rm -f orq13-int-pg >/dev/null 2>&1 || true
PW=\$(head -c 18 /dev/urandom | base64 | tr -dc 'A-Za-z0-9' | head -c 20)
printf '%s' "\$PW" > /home/ec2-user/.orq13-int-pw; chmod 600 /home/ec2-user/.orq13-int-pw
docker run -d --name orq13-int-pg -e POSTGRES_USER=multica -e POSTGRES_DB=multica \
  -e POSTGRES_PASSWORD="\$PW" -p 127.0.0.1:47490:5432 "$PG_DIGEST" >/dev/null
for i in \$(seq 1 30); do docker exec orq13-int-pg pg_isready -U multica -d multica -q && break; sleep 1; done
REMOTE
LPORT=$(python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1]);s.close()')
ssh -o BatchMode=yes -o ExitOnForwardFailure=yes -f -N \
    -L 127.0.0.1:$LPORT:127.0.0.1:47490 ec2-user@100.118.244.61
sleep 2
PW=$(ssh -o BatchMode=yes ec2-user@100.118.244.61 'cat /home/ec2-user/.orq13-int-pw')
umask 077
printf 'postgres://multica:%s@127.0.0.1:%s/multica?sslmode=disable' "$PW" "$LPORT" > "$WT/dburl"
chmod 600 "$WT/dburl"

cd "$WT/wt/multica-auth-work/server"
export PATH=/home/ec2-user/goroot/go/bin:$PATH
export GOCACHE=/home/ec2-user/.cache/go-build-i03
export GOTMPDIR=/home/ec2-user/.cache/go-build-i03/tmp
export TMPDIR=$GOTMPDIR

DATABASE_URL="$(cat "$WT/dburl")" go run ./cmd/migrate up | tail -2
CHANGED=$(cd "$WT/wt" && git diff --name-only "$TARGET" HEAD | grep '\.go$' | sed "s#$S/##")
[ -z "$(gofmt -l $CHANGED)" ] || die "G6b: gofmt sujo nos arquivos alterados"
go build ./...                                              || die "G6c: build falhou"
go vet ./internal/handler/ ./internal/daemon/ ./pkg/db/generated/ || die "G6d: vet falhou"
ok "G6b-d gofmt/build/vet"

LOG="$EVID/integration-json-$(date -u +%Y%m%dT%H%M%SZ).log"
DATABASE_URL="$(cat "$WT/dburl")" go test -race -count=1 -json ./internal/handler > "$LOG" 2>&1
python3 - "$LOG" <<'PY' || die "G6e: criterio de teste nao atendido"
import json,sys,collections
c=collections.Counter(); verdict=None
for line in open(sys.argv[1]):
    line=line.strip()
    if not line.startswith('{'): continue
    try: e=json.loads(line)
    except: continue
    c[e.get('Action')]+=1
    if e.get('Action') in ('pass','fail') and not e.get('Test'): verdict=e['Action']
must=["TestCreateChatSession_Routing","TestCreateWorkspaceUsesRequestedSlug",
      "TestCreateWorkspace_DoesNotMarkOnboarded","TestThinkingLevelText",
      "TestThinkingLevelTextNeverStoresEmptyString",
      "TestTaskUsagePayloadDecodesThinkingLevel","TestTaskUsagePayloadLegacyDaemonYieldsNull"]
seen={}
for line in open(sys.argv[1]):
    line=line.strip()
    if not line.startswith('{'): continue
    try: e=json.loads(line)
    except: continue
    if e.get('Test') in must and e.get('Action') in ('pass','fail','skip'):
        seen.setdefault(e['Test'], e['Action'])
bad=[t for t in must if seen.get(t)!='pass']
print("package:",verdict,"| fail:",c['fail'],"| skip:",c['skip'])
# CRITERIO: package pass, fail 0, os 7 obrigatorios em pass.
# skip NAO e exigido zero: 37 skips externos sao divida documentada (D1/D2).
assert verdict=='pass', "pacote nao passou"
assert c['fail']==0,    "ha falhas"
assert not bad,         f"testes obrigatorios fora de pass: {bad}"
assert c['skip']<=37,   f"skips acima do baseline documentado: {c['skip']}"
PY
grep -q 'DATA RACE' "$LOG" && die "G6f: data race detectada"
grep -q 'Skipping tests' "$LOG" && die "G6g: TestMain abortou o pacote (falso-verde)"
ok "G6e-g package pass, fail 0, 7/7 obrigatorios, sem race, sem falso-verde"

# ---------------------------------------------------------------------------
# G7  LINT: nenhuma issue NOVA contra o alvo
# ---------------------------------------------------------------------------
LD="$WT/lint"; install -d -m 0700 "$LD" "$LD/c" "$LD/b"
( cd "$LD"
  A=golangci-lint-2.12.0-linux-amd64
  B=https://github.com/golangci/golangci-lint/releases/download/v2.12.0
  curl -fsSL --max-time 120 -o "$A.tar.gz" "$B/$A.tar.gz"
  echo "$LINT_SHA256  $A.tar.gz" | sha256sum -c - || die "G7: checksum do golangci-lint divergiu"
  tar -xzf "$A.tar.gz"; install -m 0700 "$A/golangci-lint" "$LD/golangci-lint" )
git worktree add --detach "$WT/wt-base" "$TARGET" >/dev/null
norm(){ grep -E '\.go:[0-9]+:[0-9]+:' "$1" | sed -E 's#^.*multica-auth-work/server/##; s#^\./##' | sort; }
( cd "$WT/wt/multica-auth-work/server"      && GOLANGCI_LINT_CACHE="$LD/c" "$LD/golangci-lint" run --timeout 10m ./... > "$LD/comb.log" 2>&1 || true )
( cd "$WT/wt-base/multica-auth-work/server" && GOLANGCI_LINT_CACHE="$LD/b" "$LD/golangci-lint" run --timeout 10m ./... > "$LD/base.log" 2>&1 || true )
NEW=$(comm -13 <(norm "$LD/base.log") <(norm "$LD/comb.log") | wc -l)
[ "$NEW" -eq 0 ] || die "G7: $NEW issues de lint NOVAS introduzidas"
cp "$LD/comb.log" "$LD/base.log" "$EVID/"
git worktree remove --force "$WT/wt-base"
ok "G7 zero issues de lint novas (caches isolados por execucao)"

# ---------------------------------------------------------------------------
# PONTO DE PARADA OBRIGATORIO
# ---------------------------------------------------------------------------
cat <<'STOP'

=========================================================================
TODOS OS GATES PASSARAM. O MERGE NAO E EXECUTADO POR ESTE SCRIPT.
Comandos exatos, para quem tiver autorizacao do owner e revisao independente:

  git -C REPO switch integration/dev-transition-candidate-20260719
  git cherry-pick -x 11ef715 67e9a4c b1f08e3 c0e93a2

  # ou, preservando a topologia das branches:
  git merge --no-ff agent/opus48-b/squad-default-leader
  git merge --no-ff agent/opus48-b/orq-13-thinking-level

Depois do merge, ANTES de liberar o ORQ-12:
  informar ao owner do ORQ-12 que ele deve REBASEAR sobre o alvo atualizado e
  RE-GERAR o sqlc com v1.31.1 em vez de resolver models.go / task_usage.sql.go
  a mao. Ver secao "colisao anunciada" na evidencia.
=========================================================================
STOP
