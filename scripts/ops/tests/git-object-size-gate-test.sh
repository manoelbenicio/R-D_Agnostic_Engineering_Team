#!/usr/bin/env bash
set -euo pipefail
ROOT=$(git rev-parse --show-toplevel)
GATE=$ROOT/scripts/ops/git-object-size-gate.py
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
REPO=$TMP/repo
mkdir "$REPO"
git -C "$REPO" init -q
git -C "$REPO" config user.name fixture
git -C "$REPO" config user.email fixture@example.invalid
cp "$GATE" "$REPO/gate.py"
printf '# path<TAB>bytes<TAB>blob_oid\n' > "$REPO/.git-large-files.allow"
git -C "$REPO" add gate.py .git-large-files.allow
git -C "$REPO" commit -qm baseline

run_gate() { (cd "$REPO" && python3 gate.py --staged); }
expect_fail() { if "$@" >/dev/null 2>&1; then echo "expected failure" >&2; exit 1; fi; }
reset_case() { git -C "$REPO" reset --hard -q HEAD; git -C "$REPO" clean -qfd; }

printf x > "$REPO/c:small.txt"
git -C "$REPO" add 'c:small.txt'
expect_fail run_gate
reset_case

printf x > "$REPO/senior-audit-session.json"
git -C "$REPO" add senior-audit-session.json
expect_fail run_gate
reset_case

mkdir -p "$REPO/.deploy-control/p0/checkins"
printf x > "$REPO/.deploy-control/p0/checkins/INDEPENDENT-AUDIT.json"
git -C "$REPO" add .deploy-control/p0/checkins/INDEPENDENT-AUDIT.json
run_gate >/dev/null
reset_case

truncate -s 5242880 "$REPO/warn.bin"
git -C "$REPO" add warn.bin
run_gate 2>"$TMP/warn.stderr"
grep -q '^WARN:' "$TMP/warn.stderr"
reset_case

truncate -s 10485760 "$REPO/allow.bin"
git -C "$REPO" add allow.bin
expect_fail run_gate
OID=$(git -C "$REPO" rev-parse :allow.bin)
printf 'allow.bin\t10485760\t%s\n' "$OID" >> "$REPO/.git-large-files.allow"
git -C "$REPO" add .git-large-files.allow
run_gate >/dev/null
reset_case

truncate -s 100000000 "$REPO/exact-limit.bin"
git -C "$REPO" add exact-limit.bin
OID=$(git -C "$REPO" rev-parse :exact-limit.bin)
printf 'exact-limit.bin\t100000000\t%s\n' "$OID" >> "$REPO/.git-large-files.allow"
git -C "$REPO" add .git-large-files.allow
run_gate >/dev/null
reset_case

truncate -s 100000001 "$REPO/over-limit.bin"
git -C "$REPO" add over-limit.bin
OID=$(git -C "$REPO" rev-parse :over-limit.bin)
printf 'over-limit.bin\t100000001\t%s\n' "$OID" >> "$REPO/.git-large-files.allow"
git -C "$REPO" add .git-large-files.allow
expect_fail run_gate
reset_case

HEAD_OID=$(git -C "$REPO" rev-parse HEAD)
ZERO_OID=0000000000000000000000000000000000000000
expect_fail bash -c "cd '$REPO' && printf '%s\\n' 'refs/heads/task16-rc-16bfcb4 $HEAD_OID refs/heads/review $ZERO_OID' | python3 gate.py --pre-push"

grep -q 'b0e282b997de2ff35a56886cf6a6c8b4655e1031' "$GATE"
echo 'git-object-size-gate fixtures: PASS'
