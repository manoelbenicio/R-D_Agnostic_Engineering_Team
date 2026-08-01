#!/usr/bin/env bash
set -euo pipefail
ROOT=$(git rev-parse --show-toplevel)
SCRIPT=$ROOT/scripts/ops/agent-idle-autostop.sh
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
REMOTE=$TMP/remote.git
SEED=$TMP/seed
REPO=$TMP/repo
MARKER=$TMP/poweroff.called

git init --bare -q "$REMOTE"
git init -q "$SEED"
git -C "$SEED" config user.name fixture
git -C "$SEED" config user.email fixture@example.invalid
printf baseline > "$SEED/tracked"
git -C "$SEED" add tracked
git -C "$SEED" commit -qm baseline
git -C "$SEED" branch -M main
git -C "$SEED" remote add origin "$REMOTE"
git -C "$SEED" push -q -u origin main
git --git-dir="$REMOTE" symbolic-ref HEAD refs/heads/main
git clone -q "$REMOTE" "$REPO"
cat > "$TMP/systemctl" <<EOF
#!/bin/sh
[ "\$1" = poweroff ] || exit 2
printf called > "$MARKER"
EOF
chmod +x "$TMP/systemctl"

allow() {
    rm -f "$MARKER"
    AGENT_IDLE_SYSTEMCTL="$TMP/systemctl" "$SCRIPT" "$1" >/dev/null 2>&1
    test -f "$MARKER"
}
veto() {
    rm -f "$MARKER"
    if AGENT_IDLE_SYSTEMCTL="$TMP/systemctl" "$SCRIPT" "$1" >/dev/null 2>&1; then
        echo "expected veto: $1" >&2; exit 1
    fi
    test ! -e "$MARKER"
}

allow "$REPO"
printf dirty >> "$REPO/tracked"
veto "$REPO"
git -C "$REPO" checkout -q -- tracked
printf untracked > "$REPO/untracked"
veto "$REPO"
rm "$REPO/untracked"
git -C "$REPO" checkout -q --detach
veto "$REPO"
git -C "$REPO" checkout -q main
git -C "$REPO" checkout -qb local-only
veto "$REPO"
git -C "$REPO" checkout -q main
veto "$TMP/not-a-repository"

git -C "$REPO" branch linked main
git -C "$REPO" branch --set-upstream-to=origin/main linked >/dev/null
git -C "$REPO" worktree add -q "$TMP/linked" linked
allow "$TMP/linked"

echo 'agent-idle-autostop harness: PASS'
