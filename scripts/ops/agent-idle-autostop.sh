#!/bin/sh
set -u

veto() {
    printf 'agent-idle-autostop: VETO: %s\n' "$1" >&2
    exit 1
}

[ "$#" -gt 0 ] || veto 'no repositories supplied'
SYSTEMCTL=${AGENT_IDLE_SYSTEMCTL:-systemctl}

for repo in "$@"; do
    inside=$(git -C "$repo" rev-parse --is-inside-work-tree 2>/dev/null) || veto 'repository state indeterminate'
    [ "$inside" = true ] || veto 'not a worktree'
    git_dir=$(git -C "$repo" rev-parse --git-dir 2>/dev/null) || veto 'git directory indeterminate'
    common_dir=$(git -C "$repo" rev-parse --git-common-dir 2>/dev/null) || veto 'common directory indeterminate'
    [ -n "$git_dir" ] && [ -n "$common_dir" ] || veto 'worktree metadata indeterminate'

    branch=$(git -C "$repo" branch --show-current 2>/dev/null) || veto 'branch state indeterminate'
    [ -n "$branch" ] || veto 'detached HEAD'

    state=$(git -C "$repo" status --porcelain=v1 --untracked-files=all 2>/dev/null) || veto 'worktree status indeterminate'
    [ -z "$state" ] || veto 'dirty or untracked worktree'

    upstream=$(git -C "$repo" rev-parse --abbrev-ref --symbolic-full-name '@{upstream}' 2>/dev/null) || veto 'no upstream'
    [ -n "$upstream" ] || veto 'no upstream'
    counts=$(git -C "$repo" rev-list --left-right --count 'HEAD...@{upstream}' 2>/dev/null) || veto 'upstream comparison indeterminate'
    ahead=${counts%%[[:space:]]*}
    [ "$ahead" = 0 ] || veto 'unpushed commits'
done

printf 'agent-idle-autostop: all repositories are clean, attached, and fully published\n' >&2
exec "$SYSTEMCTL" poweroff
