#!/usr/bin/env bash
# P12.12.6 logs-scrubbed harness.
# Scans explicit log paths for secret/token patterns required by
# .planning/EVIDENCE_CONTRACT.md without printing matched secret values.

set -Eeuo pipefail
IFS=$'\n\t'

SCRIPT_NAME="logs-scrubbed-12-6"
PATTERN='sk-|bearer|api[_-]?key|token='
OUTPUT_FILE=""
LOG_PATHS=()

log() { printf '[%s] %s\n' "$SCRIPT_NAME" "$*" >&2; }
die() { log "ERROR: $*"; exit 1; }

usage() {
  cat <<'USAGE'
Usage: logs-scrubbed-12-6.sh --log-path PATH [--log-path PATH...] [--output FILE]

Runs the P12.12.6 scrub check:
  grep -RniE 'sk-|bearer|api[_-]?key|token=' <log path>

The harness records command shape, paths, match counts, and matching file names only.
It intentionally does not print matching line contents, because a failing match may
contain a real secret.
USAGE
}

while (($#)); do
  case "$1" in
    --log-path)
      shift
      [[ $# -gt 0 ]] || die "--log-path requires a value"
      LOG_PATHS+=("$1")
      ;;
    --output)
      shift
      [[ $# -gt 0 ]] || die "--output requires a value"
      OUTPUT_FILE="$1"
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      die "unknown argument: $1"
      ;;
  esac
  shift
done

((${#LOG_PATHS[@]} > 0)) || die "at least one --log-path is required"

for path in "${LOG_PATHS[@]}"; do
  [[ -e "$path" ]] || die "log path does not exist: $path"
done

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT
MATCH_FILES="$TMP_DIR/matching-files.txt"
MATCH_COUNTS="$TMP_DIR/match-counts.txt"
: >"$MATCH_FILES"
: >"$MATCH_COUNTS"

TOTAL_MATCHES=0
for path in "${LOG_PATHS[@]}"; do
  count="$({ grep -RniE "$PATTERN" "$path" 2>/dev/null || true; } | wc -l | tr -d ' ')"
  printf '%s\t%s\n' "$path" "$count" >>"$MATCH_COUNTS"
  TOTAL_MATCHES=$((TOTAL_MATCHES + count))
  if ((count > 0)); then
    grep -RIliE "$PATTERN" "$path" 2>/dev/null >>"$MATCH_FILES" || true
  fi
done

if [[ -n "$OUTPUT_FILE" ]]; then
  mkdir -p "$(dirname "$OUTPUT_FILE")"
  {
    printf '# P12 12.6 logs-scrubbed harness\n\n'
    printf -- '- timestamp_utc: %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    printf -- '- harness: `scripts/smoke/logs-scrubbed-12-6.sh`\n'
    printf -- '- evidence_contract: `.planning/EVIDENCE_CONTRACT.md`\n'
    printf -- '- command_shape: `grep -RniE '\\''sk-|bearer|api[_-]?key|token='\\'' <log path>`\n'
    printf -- '- value_leak_guard: `matched line contents are not printed; only counts and file names are recorded`\n'
    printf -- '- total_matches: `%s`\n\n' "$TOTAL_MATCHES"
    printf '## Log Paths\n\n'
    for path in "${LOG_PATHS[@]}"; do
      printf -- '- `%s`\n' "$path"
    done
    printf '\n## Match Counts\n\n'
    printf '```text\n'
    cat "$MATCH_COUNTS"
    printf '```\n\n'
    if ((TOTAL_MATCHES > 0)); then
      printf '## Matching Files\n\n'
      printf '```text\n'
      sort -u "$MATCH_FILES"
      printf '```\n\n'
      printf '## Verdict\n\n'
      printf 'FAIL - secret/token pattern matches were found. Inspect the matching files locally; raw matching values are intentionally omitted from evidence.\n'
    else
      printf '## Matching Files\n\n'
      printf '```text\n'
      printf '<none>\n'
      printf '```\n\n'
      printf '## Verdict\n\n'
      printf 'PASS - no `sk-`, `bearer`, `api[_-]?key`, or `token=` matches found in the scanned log paths.\n'
    fi
  } >"$OUTPUT_FILE"
fi

if ((TOTAL_MATCHES > 0)); then
  log "FAIL: found $TOTAL_MATCHES secret/token pattern match(es)"
  exit 1
fi

log "PASS: no secret/token pattern matches found"
