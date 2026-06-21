#!/usr/bin/env bash
# Pre-ship gate — RCA-2026-05-31-001 (A3)
# Runs: typecheck → lint → unit tests → build → contract tests (if GO_CORE_LIVE=1)
set -euo pipefail

echo "=== PRE-SHIP GATE: AgentVerse — GO Core ==="
echo "1. TypeScript..."
npx tsc --noEmit

echo "2. Lint..."
npm run lint

echo "3. Unit tests..."
npm run test

echo "4. Build..."
npm run build
node scripts/check-bundle-size.mjs

if [[ "${GO_CORE_LIVE:-0}" == "1" ]]; then
  echo "5. Contract tests (GO_CORE_LIVE=1)..."
  GO_CORE_LIVE=1 npx vitest run tests/contract/
else
  echo "5. Contract tests SKIPPED (set GO_CORE_LIVE=1 to enable)"
fi

echo "=== ALL GATES PASSED ==="
