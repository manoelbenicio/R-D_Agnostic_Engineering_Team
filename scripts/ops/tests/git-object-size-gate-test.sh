#!/usr/bin/env bash
set -euo pipefail
ROOT=$(git rev-parse --show-toplevel)
GATE=$ROOT/scripts/ops/git-object-size-gate.py
WORKFLOW=$ROOT/.github/workflows/repository-integrity.yml
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
ZERO=0000000000000000000000000000000000000000

expect_fail() { if "$@" >/dev/null 2>&1; then echo "expected failure: $*" >&2; exit 1; fi; }
expect_pass() { "$@" >/dev/null 2>&1 || { echo "expected pass: $*" >&2; exit 1; }; }

# Staged-path and exact decimal threshold fixtures.
REPO=$TMP/staged
mkdir "$REPO"
git -C "$REPO" init -q
git -C "$REPO" config user.name fixture
git -C "$REPO" config user.email fixture@example.invalid
cp "$GATE" "$REPO/gate.py"
printf '# path<TAB>bytes<TAB>blob_oid\n' > "$REPO/.git-large-files.allow"
git -C "$REPO" add gate.py .git-large-files.allow
git -C "$REPO" commit -qm baseline
run_staged() { (cd "$REPO" && python3 gate.py --staged); }
reset_case() { git -C "$REPO" reset --hard -q HEAD; git -C "$REPO" clean -qfd; }
stage_size() { truncate -s "$2" "$REPO/$1"; git -C "$REPO" add "$1"; }

printf x > "$REPO/c:small.txt"; git -C "$REPO" add 'c:small.txt'; expect_fail run_staged; reset_case
printf x > "$REPO/senior-audit-session.json"; git -C "$REPO" add senior-audit-session.json; expect_fail run_staged; reset_case
mkdir -p "$REPO/.deploy-control/p0/checkins"; printf x > "$REPO/.deploy-control/p0/checkins/INDEPENDENT-AUDIT.json"
git -C "$REPO" add .deploy-control/p0/checkins/INDEPENDENT-AUDIT.json; expect_pass run_staged; reset_case

stage_size below-warn.bin 5242879
run_staged 2>"$TMP/below-warn.stderr"; test ! -s "$TMP/below-warn.stderr"; reset_case
for spec in at-warn.bin:5242880 above-warn.bin:5242881 below-allow.bin:10485759; do
    path=${spec%%:*}; size=${spec##*:}; stage_size "$path" "$size"
    run_staged 2>"$TMP/$path.stderr"; grep -q '^WARN:' "$TMP/$path.stderr"; reset_case
done

for size in 10485760 10485761; do
    path="allow-$size.bin"; stage_size "$path" "$size"; expect_fail run_staged
    oid=$(git -C "$REPO" rev-parse ":$path")
    printf '%s\t%s\t%s\n' "$path" "$size" "$oid" >> "$REPO/.git-large-files.allow"
    git -C "$REPO" add .git-large-files.allow; expect_pass run_staged; reset_case
done

stage_size wrong-oid.bin 10485760
printf 'wrong-oid.bin\t10485760\t0000000000000000000000000000000000000000\n' >> "$REPO/.git-large-files.allow"
git -C "$REPO" add .git-large-files.allow; expect_fail run_staged; reset_case

stage_size exact-limit.bin 100000000
oid=$(git -C "$REPO" rev-parse :exact-limit.bin)
printf 'exact-limit.bin\t100000000\t%s\n' "$oid" >> "$REPO/.git-large-files.allow"
git -C "$REPO" add .git-large-files.allow; expect_pass run_staged; reset_case
stage_size over-limit.bin 100000001
oid=$(git -C "$REPO" rev-parse :over-limit.bin)
printf 'over-limit.bin\t100000001\t%s\n' "$oid" >> "$REPO/.git-large-files.allow"
git -C "$REPO" add .git-large-files.allow; expect_fail run_staged; reset_case

# Execute the denylist code path for every quarantined blob identity.
python3 - "$GATE" >"$TMP/oid-denial.stdout" 2>"$TMP/oid-denial.stderr" <<'PY'
import importlib.util,sys
spec=importlib.util.spec_from_file_location('gate',sys.argv[1]); gate=importlib.util.module_from_spec(spec); sys.modules[spec.name]=gate; spec.loader.exec_module(gate)
oids=['b0e282b997de2ff35a56886cf6a6c8b4655e1031','fa62eefd5707ffbd15e372cd82d7bd10fa6e2490','0693a65276ed5b6bb8ebf09f6d6d420100cec934','053d8a10aa84a4e2de557880847e8f337968fb1e']
for oid in oids:
    if gate.inspect([gate.Blob('fixture.bin',oid,1)],{}) != 1: raise SystemExit('denied OID accepted')
print('four omission OIDs denied')
PY
grep -Fxq 'four omission OIDs denied' "$TMP/oid-denial.stdout"

# CI range: inherited unallowlisted >=10 MiB is outside the baseline and allowed;
# a newly introduced blob of the same size is rejected.
GRAPH=$TMP/graph
mkdir "$GRAPH"; git -C "$GRAPH" init -q
git -C "$GRAPH" config user.name fixture; git -C "$GRAPH" config user.email fixture@example.invalid
cp "$GATE" "$GRAPH/gate.py"; printf '# path<TAB>bytes<TAB>blob_oid\n' > "$GRAPH/.git-large-files.allow"
truncate -s 10485760 "$GRAPH/inherited.bin"
git -C "$GRAPH" add .; git -C "$GRAPH" commit -qm inherited-large; BASE=$(git -C "$GRAPH" rev-parse HEAD)
printf small > "$GRAPH/small.txt"; git -C "$GRAPH" add small.txt; git -C "$GRAPH" commit -qm small; SMALL=$(git -C "$GRAPH" rev-parse HEAD)
expect_pass bash -c "cd '$GRAPH' && python3 gate.py --range '$BASE..$SMALL'"
git -C "$GRAPH" checkout -q -b introduced "$BASE"
truncate -s 10485760 "$GRAPH/new-large.bin"; git -C "$GRAPH" add new-large.bin; git -C "$GRAPH" commit -qm new-large
NEW_LARGE=$(git -C "$GRAPH" rev-parse HEAD)
expect_fail bash -c "cd '$GRAPH' && python3 gate.py --range '$BASE..$NEW_LARGE'"
git -C "$GRAPH" update-ref refs/remotes/origin/main "$BASE"
test "$(git -C "$GRAPH" merge-base "$NEW_LARGE" refs/remotes/origin/main)" = "$BASE"
grep -q 'baseline=$(git merge-base' "$WORKFLOW"
grep -q 'PR_BASE_SHA' "$WORKFLOW"
grep -q 'BEFORE_SHA' "$WORKFLOW"
grep -q 'default branch ref unavailable' "$WORKFLOW"

# Pre-push graph with target and non-target remotes.
PUSH=$TMP/push
mkdir "$PUSH"; git -C "$PUSH" init -q
git -C "$PUSH" config user.name fixture; git -C "$PUSH" config user.email fixture@example.invalid
cp "$GATE" "$PUSH/gate.py"; printf '# path<TAB>bytes<TAB>blob_oid\n' > "$PUSH/.git-large-files.allow"
printf base > "$PUSH/base.txt"; git -C "$PUSH" add .; git -C "$PUSH" commit -qm base; P0=$(git -C "$PUSH" rev-parse HEAD)
git init --bare -q "$TMP/target.git"; git init --bare -q "$TMP/other.git"; git init --bare -q "$TMP/empty.git"
git -C "$PUSH" remote add target "$TMP/target.git"; git -C "$PUSH" remote add other "$TMP/other.git"; git -C "$PUSH" remote add empty "$TMP/empty.git"
git -C "$PUSH" update-ref refs/remotes/target/main "$P0"; git -C "$PUSH" update-ref refs/remotes/other/main "$P0"
prepush() { input=$1; (cd "$PUSH" && printf '%s\n' "$input" | python3 gate.py --pre-push --remote-name target --remote-location "$TMP/target.git"); }

printf one > "$PUSH/one.txt"; git -C "$PUSH" add one.txt; git -C "$PUSH" commit -qm one; P1=$(git -C "$PUSH" rev-parse HEAD)
expect_pass prepush "refs/heads/main $P1 refs/heads/main $P0"
expect_pass prepush "refs/heads/new-small $P1 refs/heads/new-small $ZERO"
expect_pass prepush "(delete) $ZERO refs/heads/old $P0"
expect_fail bash -c "cd '$PUSH' && printf '%s\\n' 'refs/heads/new-small $P1 refs/heads/new-small $ZERO' | python3 gate.py --pre-push --remote-name missing --remote-location '$TMP/target.git'"
expect_fail bash -c "cd '$PUSH' && printf '%s\\n' 'refs/heads/new-small $P1 refs/heads/new-small $ZERO' | python3 gate.py --pre-push --remote-name empty --remote-location '$TMP/empty.git'"
expect_fail bash -c "cd '$PUSH' && printf '%s\\n' 'refs/heads/new-small $P1 refs/heads/new-small $ZERO' | python3 gate.py --pre-push --remote-name '$TMP/target.git' --remote-location '$TMP/target.git'"
expect_fail bash -c "cd '$PUSH' && printf '%s\\n' 'refs/heads/new-small $P1 refs/heads/new-small $ZERO' | python3 gate.py --pre-push --remote-name target --remote-location '$TMP/other.git'"
expect_fail prepush "refs/heads/task16-rc-16bfcb4 $P1 refs/heads/forbidden $ZERO"

# Ambiguous and divergent named mappings must fail before baseline exclusion.
git -C "$PUSH" remote add split "$TMP/target.git"; git -C "$PUSH" remote set-url --push split "$TMP/other.git"; git -C "$PUSH" update-ref refs/remotes/split/main "$P0"
expect_fail bash -c "cd '$PUSH' && printf '%s\\n' 'refs/heads/new-small $P1 refs/heads/new-small $ZERO' | python3 gate.py --pre-push --remote-name split --remote-location '$TMP/other.git'"
git -C "$PUSH" remote add multifetch "$TMP/target.git"; git -C "$PUSH" remote set-url --add multifetch "$TMP/other.git"; git -C "$PUSH" update-ref refs/remotes/multifetch/main "$P0"
expect_fail bash -c "cd '$PUSH' && printf '%s\\n' 'refs/heads/new-small $P1 refs/heads/new-small $ZERO' | python3 gate.py --pre-push --remote-name multifetch --remote-location '$TMP/target.git'"
git -C "$PUSH" remote add multipush "$TMP/target.git"; git -C "$PUSH" remote set-url --add --push multipush "$TMP/target.git"; git -C "$PUSH" remote set-url --add --push multipush "$TMP/other.git"; git -C "$PUSH" update-ref refs/remotes/multipush/main "$P0"
expect_fail bash -c "cd '$PUSH' && printf '%s\\n' 'refs/heads/new-small $P1 refs/heads/new-small $ZERO' | python3 gate.py --pre-push --remote-name multipush --remote-location '$TMP/target.git'"

truncate -s 10485760 "$PUSH/new-large.bin"; git -C "$PUSH" add new-large.bin; git -C "$PUSH" commit -qm large; P2=$(git -C "$PUSH" rev-parse HEAD)
expect_fail prepush "refs/heads/main $P1 refs/heads/main $P0
refs/heads/new-large $P2 refs/heads/new-large $ZERO"
git -C "$PUSH" update-ref refs/remotes/other/topic "$P2"
# This must still fail: refs/remotes/other/topic is not a target baseline.
expect_fail prepush "refs/heads/cross-remote $P2 refs/heads/cross-remote $ZERO"
expect_fail prepush "refs/heads/main $P2 refs/heads/main $P0"

# Merge topology: a large blob introduced on the merged side is still scanned.
git -C "$PUSH" checkout -q -B merge-main "$P0"
printf main > "$PUSH/main.txt"; git -C "$PUSH" add main.txt; git -C "$PUSH" commit -qm merge-main
git -C "$PUSH" checkout -q -b merge-side "$P0"
truncate -s 10485760 "$PUSH/side-large.bin"; git -C "$PUSH" add side-large.bin; git -C "$PUSH" commit -qm merge-side
git -C "$PUSH" checkout -q merge-main; git -C "$PUSH" merge -q --no-ff merge-side -m merge; MERGE=$(git -C "$PUSH" rev-parse HEAD)
expect_fail prepush "refs/heads/merge-main $MERGE refs/heads/merge-main $P0"

# Actual local push regression: fetch=A/push=B must be blocked and B unchanged.
ACT=$TMP/actual-push; A=$TMP/fetch-a.git; B=$TMP/push-b.git
mkdir "$ACT"; git -C "$ACT" init -q; git -C "$ACT" config user.name fixture; git -C "$ACT" config user.email fixture@example.invalid
mkdir -p "$ACT/scripts/ops" "$ACT/.githooks"
cp "$GATE" "$ACT/scripts/ops/git-object-size-gate.py"; cp "$ROOT/.githooks/pre-push" "$ACT/.githooks/pre-push"; chmod +x "$ACT/.githooks/pre-push"
printf '# path<TAB>bytes<TAB>blob_oid\n' > "$ACT/.git-large-files.allow"; printf base > "$ACT/base.txt"
git -C "$ACT" add .; git -C "$ACT" commit -qm base; ABASE=$(git -C "$ACT" rev-parse HEAD)
git init --bare -q "$A"; git init --bare -q "$B"
git -C "$ACT" push -q "$A" HEAD:refs/heads/main; git -C "$ACT" push -q "$B" HEAD:refs/heads/main
truncate -s 10485760 "$ACT/split-large.bin"; git -C "$ACT" add split-large.bin; git -C "$ACT" commit -qm split-large; SPLIT=$(git -C "$ACT" rev-parse HEAD); SPLIT_BLOB=$(git -C "$ACT" rev-parse HEAD:split-large.bin)
git -C "$ACT" push -q "$A" HEAD:refs/heads/main
git -C "$ACT" remote add target "$A"; git -C "$ACT" remote set-url --push target "$B"; git -C "$ACT" fetch -q target main:refs/remotes/target/main
git -C "$ACT" config core.hooksPath .githooks
set +e
git -C "$ACT" push target HEAD:refs/heads/leak >"$TMP/split-push.stdout" 2>"$TMP/split-push.stderr"; SPLIT_RC=$?
set -e
test "$SPLIT_RC" -ne 0
! git --git-dir="$B" show-ref --verify --quiet refs/heads/leak
! git --git-dir="$B" cat-file -e "$SPLIT_BLOB" 2>/dev/null
# Same fetch/push destination with a target baseline and a small delta must pass.
git -C "$ACT" checkout -q -b normal "$ABASE"; printf normal > "$ACT/normal.txt"; git -C "$ACT" add normal.txt; git -C "$ACT" commit -qm normal
git -C "$ACT" remote add same "$B"; git -C "$ACT" fetch -q same main:refs/remotes/same/main
git -C "$ACT" push same HEAD:refs/heads/normal >"$TMP/same-push.stdout" 2>"$TMP/same-push.stderr"
git --git-dir="$B" show-ref --verify --quiet refs/heads/normal
if [[ -n "${GATE_FIXTURE_EVIDENCE_DIR:-}" ]]; then
    mkdir -p "$GATE_FIXTURE_EVIDENCE_DIR"
    cp "$TMP/split-push.stdout" "$TMP/split-push.stderr" "$TMP/same-push.stdout" "$TMP/same-push.stderr" "$GATE_FIXTURE_EVIDENCE_DIR/"
    printf 'split_push_rc=%s\nsplit_destination_ref_absent=true\nsplit_blob_absent_from_destination=true\nsame_destination_push_rc=0\nsame_destination_ref_present=true\nexternal_network=false\n' "$SPLIT_RC" > "$GATE_FIXTURE_EVIDENCE_DIR/RESULT.txt"
fi

echo 'git-object-size-gate fixtures: PASS'
