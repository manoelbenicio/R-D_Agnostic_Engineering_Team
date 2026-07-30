# Accepted Source Evidence

## Provenance

- Accepted original: `45cefdf1afe6e9e7a2cad086f4835f353415a07f`.
- Accepted isolated port: `7dd6f7df2cf1285c43ab2a0934d511189d1babb6`.
- Port parent: `edd7b932f7c44c3396fd87853c2795d746fc5134`.
- Preserved ref: `checkpoint/20260730/orq37-source-port`.

## Exact files

| Path | Mode | Blob |
| --- | --- | --- |
| `scripts/ops/umask-hardening/README.md` | `100644` | `63de05b780be56346d64108a56007d1e024f8450` |
| `scripts/ops/umask-hardening/install-umask-hardening.sh` | `100755` | `ccb65c4e1d5c125d2d588695e7a3fe93b59332d4` |
| `scripts/ops/umask-hardening/test-umask-hardening.sh` | `100755` | `74844eac553920af69ec24d350b6a55aaaef4f14` |

## Accepted validation

- `bash -n`: PASS.
- ShellCheck: PASS with zero findings.
- Git diff checks: PASS.
- Full effect-based fake-root harness: `54/54` PASS, zero skips.
- Stubbed `systemctl`: never invoked.

## Runtime truth

The accepted port has not been fast-forwarded to main, applied, deployed, reloaded, restarted, or accepted by a
production canary. No plaintext credential value was retrieved for this evidence. Owner/external gates remain
unchecked in `tasks.md`, and ORQ-58 remains separate.
