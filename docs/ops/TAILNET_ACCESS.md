# Tailnet Access (ORQ1 / ORQ2) + OmniRoute Dashboard

> Operational runbook. Every statement below was **tested/verified on 2026-07-24**.
> Owner evidence file (owner's machine): `C:\Users\mbenicios\Downloads\EVIDENCIAS_TAILSCALE_ORQ.md`.

## Rule (non-negotiable)
**All tailnet access goes through WSL.** The Windows host is **not** on Tailscale.

## Why Windows is not on the tailnet (verified)
- Windows is blocked from joining Tailscale by a **network filter (Netskope)** and has **no IPv6**.
- Evidence: `winget install --id Tailscale.Tailscale --exact` failed downloading the MSI with
  `0x80190193 : Forbidden (403)` from `pkgs.tailscale.com` — the filter blocks the download.
- Therefore: do **not** try to reach tailnet IPs (`100.x`) directly from Windows. Use WSL.

## Nodes (tailnet `tail96e2c0.ts.net`, account `cloud.labs.brazil@gmail.com`)
| Node | Tailscale IP | Notes |
|---|---|---|
| wsl-dataops-labs (this WSL) | `100.117.245.15` | the only local tailnet entry point |
| orq1 | `100.118.244.61` | EC2 Amazon Linux, sa-east-1; WireGuard direct (~10 ms) |
| orq2 | `100.110.178.47` | EC2 Amazon Linux, sa-east-1; DERP relay São Paulo (~10 ms) |

- Transport: **IPv4 + WireGuard/DERP** (no IPv6). DNS via MagicDNS (`100.100.100.100`) while Tailscale is UP.
- Auth: **Tailscale SSH** (no key, no password). User: `ec2-user`. Use lowercase `ssh`.

## How to connect (verified)
```bash
# from WSL
ssh orq1        # -> ec2-user@100.118.244.61
ssh orq2        # -> ec2-user@100.110.178.47
```
```powershell
# from PowerShell: helper functions that shell out to `wsl ssh ...`
orq1
orq2
```
If a Windows-side attempt fails, fall back to: `wsl ssh orq1` / `wsl ssh orq2`.

## OmniRoute dashboard access from the Windows browser (verified path)
The dashboard runs on **orq1**, bound only to its tailnet IP `100.118.244.61:20128`.
Since Windows has no tailnet route, reach it via a WSL SSH tunnel, then browse `127.0.0.1`.

1. **WSL — start the tunnel (background, uses WSL's Tailscale):**
   ```bash
   ssh -f -N -L 0.0.0.0:8128:100.118.244.61:20128 ec2-user@100.110.178.47
   ```
   Verify: `curl -s -o /dev/null -w "%{http_code}\n" http://127.0.0.1:8128/login` → `200`.

2. **Windows — open a clean Edge on `127.0.0.1` (NOT the LAN IP):**
   ```powershell
   Start-Process msedge.exe -ArgumentList "--no-proxy-server","--user-data-dir=$env:TEMP\edgeclean","http://127.0.0.1:8128/login"
   ```
   - `--user-data-dir` forces a fresh Edge so `--no-proxy-server` is actually applied
     (the default Edge reuses a proxied instance and renders blank).

### Why `127.0.0.1` and NOT `192.168.1.9` (root cause, verified)
- A deployed login chunk (`/_next/static/chunks/2bj59zc08w-jg.js`) calls
  `globalThis.crypto.subtle.digest("SHA-256", ...)` **unguarded**.
- `crypto.subtle` exists **only in a secure context**: HTTPS **or** `localhost`/`127.0.0.1`.
- A LAN IP like `http://192.168.1.9:8128` is **not** a secure context → `crypto.subtle` is
  undefined → the call throws → **blank page** even though every asset returns HTTP 200.
- Verified: all `/login` HTML + `/_next/*` chunks return `200` on `127.0.0.1:8128`,
  `192.168.1.9:8128`, and direct `100.118.244.61:20128` (Host-header tested). The blank on the
  LAN IP is purely the secure-context crypto call, not connectivity/CSP/assets.

### If the canonical `https://192.168.1.9` URL is ever wanted (durable, optional)
Not required for access. Would need a real **HTTPS** origin on the Windows side (e.g. a local
TLS reverse proxy → `127.0.0.1:8128`) so the LAN IP becomes a secure context. Security-impacting
(local CA/trust) — do only with explicit owner approval. **Do not** use
`--unsafely-treat-insecure-origin-as-secure` as a durable fix (weakens browser security).

## Upstream bug to report (OmniRoute)
`crypto.subtle` is called without an `isSecureContext`/`typeof` guard, so the UI silently
blank-renders on non-secure origins instead of failing visibly or using a non-secure-compatible
hash path.
