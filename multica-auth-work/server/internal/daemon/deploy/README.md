# Agent Brain deployment and operations specifications

This package is specification-only for G2D. It does not activate a service,
read a secret, change authentication, run a canary, remove Prodex, or wire the
active daemon.

## Frozen configuration references

| Purpose | Frozen name / value |
|---|---|
| gateway required | `AGENT_BRAIN_GATEWAY_REQUIRED` |
| gateway base URL | `AGENT_BRAIN_GATEWAY_BASE_URL` |
| secret-file reference | `AGENT_BRAIN_GATEWAY_SECRET_FILE` |
| readiness policy | `AGENT_BRAIN_GATEWAY_READINESS_POLICY` (`strict`) |
| capacity tier | `AGENT_BRAIN_TASK_CAPACITY_TIER` (`20` only in the authorized canary scope) |
| host/WSL gateway | `http://127.0.0.1:20128` |
| future same-network container gateway | `http://omniroute:20128` |

The host daemon must never receive the container DNS default. The container
default is future-only and requires explicit deployment topology selection.

## Restricted secret reference

The default reference is `/etc/agent-brain/secrets/omniroute-inference-key`.
The directory contract is `root:agent-brain` mode `0750`; the installed file
contract is `root:agent-brain` mode `0440`. A later runtime must reject
symlinks, non-regular files, unexpected ownership, unsafe modes, empty or
oversized content, and any read error. Error and evidence records contain only
safe metadata/outcome codes.

Provisioning and rotation are operator actions outside the repository:

1. Stage the externally derived value through the approved secret-management
   path; never use repository files, image layers, command arguments, task
   homes, screenshots, or logs.
2. Install a restricted sibling target, validate metadata without emitting
   content, and atomically rename it to the configured reference.
3. Reload or restart through the approved service procedure, hold admissions
   until authenticated readiness passes, then revoke the prior generation.
4. On read/authentication failure, fail readiness closed. Do not restore
   provider-native, Prodex, or legacy router fallback.

Ordinary backups exclude plaintext. Restore uses the audited secret escrow and
repeats metadata validation and authenticated readiness.

## Start and recreate procedure

1. Verify the immutable image digest, versioned configuration, state backup
   checkpoint, topology, secret-reference metadata, and rollback target.
2. Start the approved OmniRoute service without placing credential values on a
   command line or in committed configuration.
3. Wait for process liveness, then authenticated readiness for the selected
   model/protocol. Keep Agent Brain admissions closed until both pass.
4. Start Agent Brain with the frozen gateway-required, topology-specific base
   URL, secret-file reference, strict readiness, and tier-20 configuration.
5. For recreate/upgrade, hold or drain new work, checkpoint state, change one
   component/revision at a time, and repeat authenticated readiness.
6. On a rollback trigger, restore the last accepted image/config/state
   generation while keeping direct-provider and Prodex routing disabled.

The typed catalogs in this package define backup/restore, account and route hot
changes, key rotation, upgrade, rollback, incident classification, escalation,
feature gates, cohorts, triggers, and evidence destinations.
