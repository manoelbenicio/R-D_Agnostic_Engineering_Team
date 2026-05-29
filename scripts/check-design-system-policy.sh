#!/usr/bin/env bash
# task 3.7: any PR that touches src/design-system/ must carry the
# `design-system-approved` label (supervisor approval).
set -euo pipefail

BASE_REF="${BASE_REF:-main}"
LABELS="${PR_LABELS:-}"

# Make sure we have the base for diffing.
git fetch --no-tags --depth=1 origin "$BASE_REF" >/dev/null 2>&1 || true

CHANGED=$(git diff --name-only "origin/$BASE_REF...HEAD" -- 'src/design-system/' || true)

if [[ -z "$CHANGED" ]]; then
  echo "No design-system changes; policy not applicable."
  exit 0
fi

echo "Design-system files changed:"
echo "$CHANGED" | sed 's/^/  /'

if echo "$LABELS" | tr ',' '\n' | grep -qx 'design-system-approved'; then
  echo "Supervisor approval label present — change permitted."
  exit 0
fi

cat <<EOF
Locked-files policy violation (task 3.7 / design-system-sentinel/spec.md):
src/design-system/ changes require the 'design-system-approved' supervisor label.
Apply the label and re-run CI.
EOF
exit 1
