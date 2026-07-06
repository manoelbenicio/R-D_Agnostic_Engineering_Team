# C6 Triple Isolation Evidence

Status: PARTIAL - static/synthetic isolation green; live triple run blocked
Timestamp: 2026-07-05T02:52:42Z
Executor: Codex

## Scope

PLAN 06-02 asks for C6 triple isolation across:

- `CODEX_HOME`;
- prodex `PRODEX_HOME`;
- Herdr-managed agent panes.

Live proof is blocked because prodex currently has zero profiles, no providers,
runtime policy disabled, and no active runtime. This evidence therefore proves
local file/env isolation mechanics only; it does not close the live C6 gate.

## Static Source Evidence

Relevant source observations:

- `multica-auth-work/server/internal/daemon/execenv/codex_home.go` copies
  `AccountHome/auth.json` into per-task `CODEX_HOME` when `AccountHome` is set.
- `multica-auth-work/server/internal/daemon/execenv/execenv.go` maps Codex onto
  a per-task `CODEX_HOME`.
- `multica-auth-work/server/internal/daemon/prodex.go` sets prodex env keys such
  as `PRODEX_HOME`, `PRODEX_SMART_CONTEXT_SHADOW`, and `MULTICA_PRODEX_COMMIT`
  but does not set `CODEX_HOME`.
- `prodex-core` reads `PRODEX_HOME` independently from shared Codex home state.

## Synthetic Isolation Probe

The probe used temporary directories and fake auth markers only. No real
credential material was read or written.

Result:

```text
synthetic_isolation=pass
```

Before simulated task-A refresh:

```text
1559115cf6da798643604111f11bfa17e82bd93424556a032059e0e9ac47b310  <tmp>/account-a/auth.json
8235cf48d8276847660e36e67467a8bce163ab4805595ed7b2df8e36862b78f6  <tmp>/account-b/auth.json
1559115cf6da798643604111f11bfa17e82bd93424556a032059e0e9ac47b310  <tmp>/task-a/auth.json
8235cf48d8276847660e36e67467a8bce163ab4805595ed7b2df8e36862b78f6  <tmp>/task-b/auth.json
884b8a00698c053d053514e2bd45ef54df7e804cc52f3b1f02cbf9ea92dd80e3  <tmp>/prodex/profiles/profile-a/auth.json
1065b12ce91a55be7aab8dfde1bad77a1626f7d45101541761209177b172150c  <tmp>/prodex/profiles/profile-b/auth.json
```

After simulated task-A refresh:

```text
1559115cf6da798643604111f11bfa17e82bd93424556a032059e0e9ac47b310  <tmp>/account-a/auth.json
8235cf48d8276847660e36e67467a8bce163ab4805595ed7b2df8e36862b78f6  <tmp>/account-b/auth.json
7a5a6079420bd381402bde63fa2c8302aee91682fb8864601cb0da8fce73d339  <tmp>/task-a/auth.json
8235cf48d8276847660e36e67467a8bce163ab4805595ed7b2df8e36862b78f6  <tmp>/task-b/auth.json
884b8a00698c053d053514e2bd45ef54df7e804cc52f3b1f02cbf9ea92dd80e3  <tmp>/prodex/profiles/profile-a/auth.json
1065b12ce91a55be7aab8dfde1bad77a1626f7d45101541761209177b172150c  <tmp>/prodex/profiles/profile-b/auth.json
```

This proves the expected copy/isolation model in a local synthetic probe:
task-A refresh mutates only task-A, not account-A, task-B, account-B, or prodex
profile homes.

## Live Blocker

Live C6 remains BLOCKED until at least one working prodex profile/provider is
configured and a controlled Herdr/prodex/CODEX_HOME session can be run without
real secret disclosure.

## Verdict

- Static source isolation: GREEN.
- Synthetic filesystem isolation: GREEN.
- Live CODEX_HOME x prodex x Herdr coexistence: BLOCKED, not green.
