# POSIX FS Validation — Credential Storage Security

> **Phase:** P4 (State/Security) — Task 4.11
> **REQ:** REQ-41 (POSIX FS validation: credenciais em ext4/xfs, NUNCA drvfs)
> **Author:** Gemini#Pro
> **Date:** 2026-07-05
> **Status:** ACTIVE
> **Source:** docs/rotation-parity-polyglot/03_PLATFORM_PLAN_360.md §4.1

## Invariant

> **Credenciais DEVEM estar em FS POSIX real (ext4/xfs), NUNCA drvfs/9p/CIFS.**
> Validar `stat -c '%a' == 600` no deploy; senão **abortar**.
> Vale também para `$PRODEX_HOME/profiles/<name>`.

## Why This Matters

On WSL (Windows Subsystem for Linux), the Windows-mounted filesystem (`/mnt/c/`, `/mnt/d/`) uses `drvfs` (or `9p` on WSL2), which:
- **Does NOT support POSIX file permissions** (`chmod` has no effect)
- **Cannot enforce `0600`** — all files appear as `0777` or `0755`
- **Leaks credentials** to any process that can read the Windows filesystem
- **CIFS/SMB mounts** have the same problem (network-mounted shares)

## Affected Paths

| Path | Contains | Required Permissions | FS Requirement |
|---|---|---|---|
| `$PRODEX_HOME/` | prodex config, state | `0700` (dir) | ext4/xfs |
| `$PRODEX_HOME/profiles/<name>/` | Per-profile credentials | `0700` (dir) | ext4/xfs |
| `$PRODEX_HOME/profiles/<name>/auth.json` | API keys, tokens | `0600` (file) | ext4/xfs |
| `$PRODEX_HOME/profiles/<name>/cookies/` | Cookie jars | `0700` (dir) | ext4/xfs |
| `~/.codex/auth.json` | Codex credentials | `0600` (file) | ext4/xfs |
| `~/.kiro/credentials` | Kiro credentials | `0600` (file) | ext4/xfs |
| `~/.config/antigravity/` | Antigravity config | `0600` (files) | ext4/xfs |

## Validation Script

```bash
#!/usr/bin/env bash
# posix-fs-check.sh — GATE P4: POSIX FS validation for credential paths
# Exits non-zero if any credential path is on a non-POSIX filesystem or has wrong permissions.

set -euo pipefail

ERRORS=0
PRODEX_HOME="${PRODEX_HOME:-$HOME/.prodex}"

check_path() {
    local path="$1"
    local expected_perm="$2"
    local label="$3"

    if [ ! -e "$path" ]; then
        echo "[SKIP] $label: $path does not exist"
        return
    fi

    # Check filesystem type
    local fs_type
    fs_type=$(stat -f -c '%T' "$path" 2>/dev/null || echo "unknown")

    case "$fs_type" in
        ext2/ext3|xfs|btrfs|tmpfs)
            echo "[OK]   $label: FS=$fs_type (POSIX-compliant)"
            ;;
        ""|unknown|fuseblk)
            # fuseblk could be NTFS via fuse — check mount point
            local mount_fs
            mount_fs=$(df --output=fstype "$path" 2>/dev/null | tail -1)
            case "$mount_fs" in
                ext4|xfs|btrfs|tmpfs)
                    echo "[OK]   $label: FS=$mount_fs (POSIX-compliant via df)"
                    ;;
                drvfs|9p|cifs|smb*|ntfs*)
                    echo "[FAIL] $label: FS=$mount_fs — FORBIDDEN (not POSIX, chmod ineffective)"
                    ERRORS=$((ERRORS + 1))
                    ;;
                *)
                    echo "[WARN] $label: FS=$mount_fs — UNKNOWN, verify manually"
                    ;;
            esac
            ;;
        drvfs|9p|cifs|smb*|ntfs*)
            echo "[FAIL] $label: FS=$fs_type — FORBIDDEN (not POSIX, chmod ineffective)"
            ERRORS=$((ERRORS + 1))
            ;;
        *)
            echo "[WARN] $label: FS=$fs_type — UNKNOWN, verify manually"
            ;;
    esac

    # Check permissions
    local actual_perm
    actual_perm=$(stat -c '%a' "$path" 2>/dev/null || echo "???")
    if [ "$actual_perm" = "$expected_perm" ]; then
        echo "[OK]   $label: permissions=$actual_perm (expected $expected_perm)"
    else
        echo "[FAIL] $label: permissions=$actual_perm (expected $expected_perm)"
        ERRORS=$((ERRORS + 1))
    fi
}

echo "=== POSIX FS Validation (GATE P4 / REQ-41) ==="
echo ""

check_path "$PRODEX_HOME"                           "700" "PRODEX_HOME"
check_path "$PRODEX_HOME/profiles"                  "700" "PRODEX_HOME/profiles"
check_path "$HOME/.codex/auth.json"                 "600" "Codex auth"
check_path "$HOME/.kiro/credentials"                "600" "Kiro credentials"
check_path "$HOME/.config/antigravity"              "700" "Antigravity config"

# Check all profile dirs dynamically
if [ -d "$PRODEX_HOME/profiles" ]; then
    for profile in "$PRODEX_HOME/profiles"/*/; do
        [ -d "$profile" ] && check_path "$profile" "700" "Profile: $(basename "$profile")"
        [ -f "${profile}auth.json" ] && check_path "${profile}auth.json" "600" "Profile auth: $(basename "$profile")"
    done
fi

echo ""
if [ "$ERRORS" -gt 0 ]; then
    echo "❌ FAILED: $ERRORS errors. ABORTING — credential paths are NOT secure."
    echo "   FIX: Move PRODEX_HOME to a POSIX filesystem (ext4/xfs) and chmod 600/700."
    exit 1
else
    echo "✅ PASSED: All credential paths on POSIX FS with correct permissions."
    exit 0
fi
```

## Deployment Integration

### Pre-deploy check (mandatory)
```bash
# In runbook step 1 (before any prodex operations):
bash docs/security/posix-fs-validation.sh || { echo "ABORT: POSIX FS check failed"; exit 1; }
```

### Docker/container context
- Inside containers (Alpine/Debian), filesystem is always ext4/overlay2 → **always passes**
- The check is critical for **WSL/hybrid deployments** where `/mnt/c/` is drvfs

### WSL remediation
If credentials are on drvfs:
```bash
# Move to WSL-native filesystem
mkdir -p ~/prodex-secure
chmod 700 ~/prodex-secure
mv /mnt/c/path/to/.prodex/* ~/prodex-secure/
export PRODEX_HOME=~/prodex-secure
```

## Evidence

> **⚠️ NOTE:** Live validation requires prodex profiles to exist (post-P0). Script is ready to run. Evidence will be recorded in `.deploy-control/evidence/p4-posix-fs.md`.

## Gate P4 Checklist (REQ-41)
- [ ] Validation script runs without error on target deployment
- [ ] No credential path on drvfs/9p/CIFS
- [ ] All credential files have `0600` permissions
- [ ] All credential directories have `0700` permissions
- [ ] Script integrated into pre-deploy runbook
