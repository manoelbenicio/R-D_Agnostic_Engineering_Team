#!/usr/bin/env bash
# enroll_account.sh — idempotent enrollment of one vendor account into the
# rotation pool (real Postgres) with per-account ISOLATED credentials.
#
# Usage:
#   bash scripts/staging/enroll_account.sh \
#       <vendor> <alias> <priority> <source-credential-path> <tokens_per_window>
#
#   vendor                 : codex | kiro | opus | antigravity
#   alias                  : human handle (e.g. stg-codex-real). If already a UUID
#                            it is used as account_id verbatim; otherwise a
#                            deterministic UUIDv5 is derived from it (stable across
#                            re-runs => idempotent).
#   priority               : integer rotation priority (lower = picked first).
#   source-credential-path : host path of the real credential to isolate. A FILE
#                            for codex/kiro/opus (auth.json / data.sqlite3); a
#                            DIRECTORY for antigravity (.gemini/antigravity-cli).
#   tokens_per_window      : integer token budget per window for this account.
#
# Env overrides (optional):
#   ENROLL_TENANT_ID         : tenant uuid (default staging tenant
#                              20000000-0000-4000-8000-000000000001, matches seed).
#   ENROLL_CREDS_EXT4_BASE   : ext4 dir holding the isolated credentials physically
#                              (default /home/dataops-lab/multica-auth-creds).
#                              Used because scripts/staging is on /mnt/c (9p drvfs
#                              without the `metadata` option) where chmod is not
#                              reflected in ls -l. The ext4 copy is exposed at
#                              scripts/staging/creds/<alias> via a symlink so the
#                              required 0600 mode is real and verifiable.
#   ENROLL_STATUS            : account status (default available).
#   ENROLL_TOKENS_USED       : starting tokens_used (default 0).
#
# Steps: (1) validate args; (2) create isolated home_dir at scripts/staging/creds/
# <alias> (symlink to ext4-backed dir) and copy the source credential to the exact
# per-vendor relative path the daemon restores from (codex=auth.json,
# kiro|opus=kiro-cli/data.sqlite3, antigravity=.gemini/antigravity-cli), chmod 0600
# on files / 0700 on dirs; (3) UPSERT accounts + credentials via enroll_account.sql.
#
# Idempotent: re-running with the same alias copies over, re-chmods, re-links, and
# upserts in place — no duplication, no error, exit 0.
#
# SECURITY: credential file CONTENTS are never printed/echoed. Only paths, ids,
# sizes, and modes are reported.

set -euo pipefail

log()  { printf '%s\n' "$*"; }
err()  { printf 'ERROR: %s\n' "$*" >&2; }
die()  { err "$*"; exit "${2:-1}"; }

is_int()  { [[ "${1:-}" =~ ^-?[0-9]+$ ]]; }
is_uuid() { [[ "${1:-}" =~ ^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$ ]]; }

resolve_path() {
    local p="$1" dir base
    if   [ -d "$p" ]; then (cd "$p" && pwd)
    elif [ -f "$p" ]; then
        dir="$(cd "$(dirname "$p")" && pwd)"
        base="$(basename "$p")"
        printf '%s/%s' "$dir" "$base"
    else printf '%s' "$p"
    fi
}

derive_account_id() {
    local alias="$1" ns="${2:-00000000-0000-4000-8000-000000000000}"
    if command -v uuidgen >/dev/null 2>&1; then
        uuidgen -s -n "$ns" -N "$alias"
    elif command -v python3 >/dev/null 2>&1; then
        python3 -c 'import uuid,sys; print(uuid.uuid5(uuid.UUID(sys.argv[1]), sys.argv[2]))' "$ns" "$alias"
    else
        die "cannot derive account_id: need uuidgen or python3 on PATH" 3
    fi
}
###############################################################################
# Args + validation
###############################################################################
if [ "$#" -ne 5 ]; then
    die "usage: $0 <vendor> <alias> <priority> <source-credential-path> <tokens_per_window>" 64
fi

VENDOR="$1"; ALIAS="$2"; PRIORITY="$3"; SRC_RAW="$4"; TOKENS_PER_WINDOW="$5"

case "$VENDOR" in
    codex)        REL_PATH="auth.json";               FORMAT="codex_auth_json_ref"; IS_DIR=0 ;;
    kiro|opus)    REL_PATH="kiro-cli/data.sqlite3";   FORMAT="kiro_sqlite_ref";     IS_DIR=0 ;;
    antigravity)  REL_PATH=".gemini/antigravity-cli"; FORMAT="antigravity_dir_ref"; IS_DIR=1 ;;
    *)            die "vendor must be one of codex|kiro|opus|antigravity (got '$VENDOR')" 64 ;;
esac

is_int "$PRIORITY"          || die "priority must be an integer (got '$PRIORITY')" 64
is_int "$TOKENS_PER_WINDOW" || die "tokens_per_window must be an integer (got '$TOKENS_PER_WINDOW')" 64

SRC="$(resolve_path "$SRC_RAW")"
if [ "$IS_DIR" -eq 1 ]; then
    [ -d "$SRC" ] || die "source-credential-path for antigravity must be an existing directory: $SRC" 66
else
    [ -f "$SRC" ] || die "source-credential-path for $VENDOR must be an existing file: $SRC" 66
fi
[ -s "$SRC" ] || die "source-credential-path is empty (zero-size): $SRC" 66

TENANT_ID="${ENROLL_TENANT_ID:-20000000-0000-4000-8000-000000000001}"
is_uuid "$TENANT_ID" || die "ENROLL_TENANT_ID must be a UUID (got '$TENANT_ID')" 64

STATUS="${ENROLL_STATUS:-available}"
TOKENS_USED="${ENROLL_TOKENS_USED:-0}"
is_int "$TOKENS_USED" || die "ENROLL_TOKENS_USED must be an integer (got '$TOKENS_USED')" 64
case "$STATUS" in
    available|leased|exhausted|cooldown|degraded) ;;
    *) die "ENROLL_STATUS must be one of available|leased|exhausted|cooldown|degraded (got '$STATUS')" 64 ;;
esac

###############################################################################
# account_id (deterministic) + paths
###############################################################################
if is_uuid "$ALIAS"; then
    ACCOUNT_ID="$ALIAS"
else
    ACCOUNT_ID="$(derive_account_id "$ALIAS")"
fi
is_uuid "$ACCOUNT_ID" || die "derived/invalid account_id is not a UUID: '$ACCOUNT_ID'" 70

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CREDS_DIR="$SCRIPT_DIR/creds"
SQL_FILE="$SCRIPT_DIR/enroll_account.sql"

HOME_DIR="$CREDS_DIR/$ALIAS"        # stored in DB; matches daemon/seed precedent
CONFIG_DIR="$HOME_DIR"              # SourceRoot == HomeRoot (credential lives in home)
SECRET_REF="file://${HOME_DIR}/${REL_PATH}"

EXT4_BASE="${ENROLL_CREDS_EXT4_BASE:-/home/dataops-lab/multica-auth-creds}"
EXT4_HOME="$EXT4_BASE/$ALIAS"
EXT4_CRED="$EXT4_HOME/$REL_PATH"    # physical credential location (ext4, real 0600)

[ -f "$SQL_FILE" ] || die "missing SQL companion: $SQL_FILE" 65
###############################################################################
# (2)+(3) Isolated home_dir + credential copy (ext4-backed), 0600 on files
###############################################################################
mkdir -p "$EXT4_BASE"; chmod 700 "$EXT4_BASE" 2>/dev/null || true
mkdir -p "$EXT4_HOME";  chmod 700 "$EXT4_HOME"  2>/dev/null || true
mkdir -p "$(dirname "$EXT4_CRED")"

if [ "$IS_DIR" -eq 1 ]; then
    rm -rf "$EXT4_CRED"
    cp -r "$SRC" "$EXT4_CRED"
    find "$EXT4_CRED" -type d -exec chmod 700 {} + 2>/dev/null || true
    find "$EXT4_CRED" -type f -exec chmod 600 {} + 2>/dev/null || true
else
    cp -f "$SRC" "$EXT4_CRED"
    chmod 600 "$EXT4_CRED" 2>/dev/null || true
fi

# Expose the ext4-backed home at the required scripts/staging/creds/<alias> path
# via a symlink (so the on-disk path matches the daemon/seed convention AND the
# 0600 mode is real/verifiable through the link).
LINK="$HOME_DIR"
if [ -e "$LINK" ] || [ -L "$LINK" ]; then
    if [ -L "$LINK" ]; then
        : # existing symlink; `ln -sfn` replaces it below.
    elif [ -d "$LINK" ]; then
        if rmdir "$LINK" 2>/dev/null; then
            : # was an empty real dir; removed so we can symlink below.
        else
            die "$LINK exists as a non-empty directory; refusing to clobber. Use a different alias or remove it." 73
        fi
    else
        die "$LINK exists as a regular file; refusing to replace with symlink. Use a different alias or remove it." 73
    fi
fi
ln -sfn "$EXT4_HOME" "$LINK"

# Verify the credential is reachable through the canonical path with 0600.
CRED_MODE="$(stat -c '%a' "$HOME_DIR/$REL_PATH" 2>/dev/null || echo 'n/a')"
if [ "$IS_DIR" -eq 1 ]; then
    CRED_SIZE="$(find "$HOME_DIR/$REL_PATH" -type f 2>/dev/null | wc -l | tr -d ' ')"
    CRED_SIZE_DESC="${CRED_SIZE} file(s)"
else
    CRED_SIZE="$(wc -c < "$HOME_DIR/$REL_PATH" 2>/dev/null | tr -d ' ' || echo 0)"
    CRED_SIZE_DESC="${CRED_SIZE} bytes"
fi

###############################################################################
# (4) UPSERT accounts + credentials via enroll_account.sql (psql vars)
###############################################################################
PSQL=(docker exec -i multica-postgres-1 psql -U multica -d multica -v ON_ERROR_STOP=1)

"${PSQL[@]}" \
    -v vendor="$VENDOR" \
    -v account_id="$ACCOUNT_ID" \
    -v tenant_id="$TENANT_ID" \
    -v priority="$PRIORITY" \
    -v home_dir="$HOME_DIR" \
    -v config_dir="$CONFIG_DIR" \
    -v status="$STATUS" \
    -v tokens_per_window="$TOKENS_PER_WINDOW" \
    -v tokens_used="$TOKENS_USED" \
    -v secret_ref="$SECRET_REF" \
    -v format="$FORMAT" \
    < "$SQL_FILE"

###############################################################################
# Masked summary (NEVER prints credential contents)
###############################################################################
log "---- enroll_account ----"
log "account_id        : $ACCOUNT_ID"
log "vendor            : $VENDOR"
log "alias             : $ALIAS"
log "priority          : $PRIORITY"
log "status            : $STATUS"
log "tenant_id         : $TENANT_ID"
log "home_dir          : $HOME_DIR -> $(readlink -f "$HOME_DIR")"
log "secret_ref        : $SECRET_REF"
log "credential path   : $HOME_DIR/$REL_PATH"
log "credential mode   : $CRED_MODE (0600 expected on files)"
log "credential size   : $CRED_SIZE_DESC (contents not printed)"
log "tokens_per_window : $TOKENS_PER_WINDOW | tokens_used : $TOKENS_USED"
log "result            : enrolled (idempotent; re-run upserts in place)"
log "------------------------"