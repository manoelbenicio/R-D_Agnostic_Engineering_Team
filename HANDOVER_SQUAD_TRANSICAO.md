# Handover / Transition Document — Cashless + OmniRoute Merge Program

**To:** New Squad (incoming agents & tech leads)
**From:** Outgoing engineering (dataops-lab environment)
**Date:** 2026-07-24
**Subject:** Full operational handover — where we are, what's done, what's pending, how to operate
**Format:** Living document (`.md`). Keep updated.

---

## 0. TL;DR (read this first)
- Two parallel programs are in flight: **(A) Cashless Arraiá 2026** (spec-driven, OpenSpec) and **(B) OmniRoute** (AI gateway, product/integration merge) running on **orq1**.
- **Access to servers is via Tailscale, from WSL.** Windows on this host is NOT on Tailscale (blocked by Netskope + no IPv6). Other machines with Tailscale (e.g., `msi-laptop-1`) reach servers directly.
- **OmniRoute is healthy on orq1** after reverting a broken custom image back to the official image.
- Sovereign spec for Cashless lives in OpenSpec at `/home/dataops-lab/openspec/changes/cashless-arraia-2026`.
- All handover docs are under `C:\Users\mbenicios\Downloads\` (WSL: `/mnt/c/Users/mbenicios/Downloads/`).

---

## 1. Environments & Access

### 1.1 Tailscale (VPN overlay) — PRIMARY access path
- Tailnet: **`tail96e2c0.ts.net`** · Account: **cloud.labs.brazil@gmail.com** · MagicDNS: **on**.
- This workstation connects **from WSL** (Ubuntu 24.04), node **`wsl-dataops-labs` = `100.117.245.15`**.
- **Windows (this laptop) is BLOCKED from Tailscale**: DNS for `*.tailscale.com` is hijacked to `192.200.0.x` (Netskope endpoint filter, HTTP 403) and there is **no IPv6 route** on this network. Do not waste time re-installing Tailscale on this Windows host — use WSL, or a machine that already has Tailscale.
- Machines WITH Tailscale (verified reachable): `msi-laptop-1` (Windows), `orq1`, `orq2`, `manoelneto-laptop`.

### 1.2 Key nodes (tailnet IPs)
| Node | Tailnet IP | Role |
|---|---|---|
| wsl-dataops-labs | 100.117.245.15 | This workstation (WSL) — jump point |
| orq1 | 100.118.244.61 | Orchestrator server — **runs OmniRoute** |
| orq2 | 100.110.178.47 | Orchestrator server |
| ec2-jump-box | 100.94.211.42 | offers exit node |
| msi-laptop-1 | 100.112.85.91 | Windows w/ Tailscale (direct access works) |

### 1.3 SSH access (simple, from WSL or PowerShell)
- WSL: `ssh orq1` · `ssh orq2` (user **ec2-user**, **Tailscale SSH**, no key/password).
- PowerShell (this host): functions `orq1` / `orq2` (proxy to `wsl ssh …`).
- Config: `~/.ssh/config` (aliases orq1/orq2 → tailnet IPs). PowerShell profiles updated (v5.1 + v7) with `orq1`/`orq2` functions and ExecutionPolicy RemoteSigned (CurrentUser).

### 1.4 EC2 inventory (region **sa-east-1**, AZ sa-east-1b)
| Tailnet | Instance ID | Type | Internal | Public | Root disk |
|---|---|---|---|---|---|
| **orq1** (100.118.244.61) | i-0d9d441dd364039f9 | m7i-flex.large | ip-172-31-18-217 | 15.228.84.165 | gp3 24 GiB (~71%) |
| **orq2** (100.110.178.47) | i-0af937456e125143d | m7i-flex.xlarge | ip-172-31-30-9 | 15.229.146.145 | gp3 **expanded 24→48 GiB** |
> Note: the disk expansion (root EBS 24→48 GiB, done online via `modify-volume` + `growpart` + `xfs_growfs`) was performed on **i-0af937456e125143d** (internal ip-172-31-30-9), which currently carries the Tailscale name **orq2**. During that task it was referred to loosely as "orq1" before the tailnet mapping was confirmed — the instance ID above is authoritative.

---

## 2. OmniRoute (Program B — product/integration merge)

### 2.1 What it is
AI gateway ("route any LLM through one endpoint"). Runs as Docker on **orq1**.
- Dashboard/UI: port **20128**; API (OpenAI-compatible): port **20129**.
- Container binds to the **Tailscale IP** `100.118.244.61:20128`.
- Local dev instance also runs in this WSL (ports 20128/20129/20132) with containers `omniroute` + `omniroute-redis`.

### 2.2 Access URLs
- From a Tailscale machine (e.g., msi-laptop-1): **`http://orq1:20128`** or **`http://100.118.244.61:20128`** (verified HTTP 200, full 477 KB page).
- From this Windows host (no Tailscale): via SSH tunnel → **`http://localhost:20130`** (systemd service `omniroute-orq1.service` on WSL keeps it up; auto-reconnect, survives reboot). Tunnel script: `/mnt/c/Users/mbenicios/Downloads/omniroute-orq1.sh`.

### 2.3 Incident resolved (root cause + fix)
- **Symptom:** blank dashboard page.
- **Root cause:** orq1 was running a broken custom build `omniroute:3.8.48-affinity-fix2` (built 2026-07-21) that served a 4,123-byte empty Next.js shell.
- **Fix:** swapped the running container back to the official image **`diegosouzapw/omniroute:latest`** (2026-07-13), preserving the `omniroute-data` volume, port binding, restart policy and `INITIAL_PASSWORD`. The broken container was **renamed, not deleted**: `omniroute-broken-3.8.48-affinity-fix2` (rollback available). Now healthy, serves full 477 KB page.
- **Open decision:** the `affinity-fix` builds were a deliberate local patch (CPU/process affinity) and are **newer** than `latest`. The fix, whatever it changed, is currently NOT in the running container. See §5 pending.

### 2.4 Repo & source
- Remote: **https://github.com/diegosouzapw/OmniRoute**
- Local: `/mnt/c/VMs/Projetos/Omini_Router` (branch `main`, in sync with `origin/main` commit-wise).
- **Uncommitted local changes** (the real "fixes" in play, NOT pushed): `docker-compose.yml`, `docker-compose.prod.yml`; plus untracked `HANDOFF_REMEDIACAO_REDE_MANOELNETO.md`. These must be reviewed and either committed/pushed or discarded.
- Config/env: `.env` (large, ~116 KB) and `.env.example`. Changelog fragments: `changelog.d/`.
- **No OpenSpec exists inside the OmniRoute repo** — its fixes are tracked via git + changelog, not OpenSpec.

---

## 3. Cashless Arraiá 2026 (Program A — spec-driven)

### 3.1 Sovereign source (OpenSpec)
`/home/dataops-lab/openspec/changes/cashless-arraia-2026/`
- `proposal.md`, `design.md`, `tasks.md`
- `specs/cashless-core/spec.md`, `specs/cashless-credentials/spec.md`, `specs/cashless-payments/spec.md`
- OpenSpec CLI: **@fission-ai/openspec v1.6.0**.

### 3.2 Product summary (AS IS decisions)
- Cashless for a private event (≤400 guests, 6–10 barracas), **QR-first**, physical NFC cards only as exception (**50–100 units**, NTAG213/215/216 13.56 MHz for guests without a phone).
- On-prem Docker core: API + **PostgreSQL** (ledger, source of truth) + **Redis** (cache/idempotency/nonce/offline queue) + TLS proxy + dashboard.
- Atomic debit (CAS), Mercado Pago Point/Orders + PIX webhook (Lambda), reconciliation (3 anchors), PIX refunds.
- DR: **EC2 + Docker warm** + S3 snapshots (2–5 min); no managed services on hot path.
- Anti-fraud: rotating QR (HMAC ~20s + single-use nonce in Redis), PIN (argon2), magic link, device binding.

### 3.3 Deliverables produced
- **Architecture theater (interactive HTML):** `C:\Users\mbenicios\Downloads\HTML\cashless_bastidores_arquitetura.html` — 11 scenarios, Macro/Intermediária/Micro views, On-Prem/Cloud, deterministic playback. QA-gated (Playwright/axe/html-validate). Masters: `..._2K_QHD.png`, `..._4K_UHD.png`, `..._MACRO_2K_QHD.png`, `..._MACRO_4K_UHD.png`.
- **Agent prompt library:** `/home/dataops-lab/openspec/changes/cashless-arraia-2026/agents/` — TL Lead, Senior Tech Manager, A1–A8 workers, `_shared/` (card-contract.schema.json, kanban-workflow.md, guardrails.md), README.
- **Design system (sovereign):** `C:\VMs\Projetos\Cashless\UI_UX_VillaPrime` (Fredoka/Inter/JetBrains Mono + hex tokens).

### 3.4 Agent operating model
- Orchestration: **Multica (Kanban)** at `127.0.0.1:13100` (workspace `orq2-dev`). Card = task contract (check-in/check-out on disk).
- Credential/model layer 3: **OmniRoute** (OAuth CLI broker) — agents never use raw keys.
- Roles: 1 TL Lead + 1 Senior Tech Manager + 8 workers (A1-core-api, A2-data, A3-pwa-pos, A4-observability, A5-payments, A6-credentials-security, A7-redis-resilience, A8-sre-dr).

---

## 4. Project documentation index (all paths)
| Doc | Path (Windows / WSL) |
|---|---|
| This handover | `C:\Users\mbenicios\Downloads\HANDOVER_SQUAD_TRANSICAO.md` |
| Tailscale evidence dossier | `C:\Users\mbenicios\Downloads\EVIDENCIAS_TAILSCALE_ORQ.md` (+ `.txt`) |
| Tailscale agent summary | `C:\Users\mbenicios\Downloads\CONEXAO_TAILSCALE_RESUMO_AGENTES.md` |
| OmniRoute tunnel script | `C:\Users\mbenicios\Downloads\omniroute-orq1.sh` |
| Cashless theater HTML + masters | `C:\Users\mbenicios\Downloads\HTML\` |
| KIRO-OPUS48 handoff (HTML fixes) | `C:\Users\mbenicios\Downloads\HTML\HANDOFF_KIRO-OPUS48.md` |
| Skills & prompts catalog | `C:\Users\mbenicios\Downloads\HTML\SKILLS_AND_PROMPTS_CATALOG.md` |
| Cashless OpenSpec | `\\wsl$\...\home\dataops-lab\openspec\changes\cashless-arraia-2026\` |
| OmniRoute repo (local) | `C:\VMs\Projetos\Omini_Router\` |
| Villa Prime design system | `C:\VMs\Projetos\Cashless\UI_UX_VillaPrime\` |
| Premium skills | `C:\VMs\Projetos\Consolidacao_Ofertas\super-skills\` |

---

## 5. PENDING / OPEN ITEMS (action for new squad)
1. **OmniRoute affinity fix** — decide whether the CPU/process-affinity patch (lost by reverting to `latest`) is needed. If yes, rebuild the image applying the patch **without breaking the Next.js frontend** (the previous build served a blank shell). Do NOT just re-tag.
2. **OmniRoute local changes** — review/commit or discard uncommitted `docker-compose.yml` + `docker-compose.prod.yml` in `/mnt/c/VMs/Projetos/Omini_Router`; push to remote if intended.
3. **Cleanup broken images on orq1** (after confirmation): `omniroute-broken-3.8.48-affinity-fix2`, `omniroute-canary`, `omniroute-affinity-FAILED-*`.
4. **Agent Toolkit for AWS** — setup PAUSED at Step 3 (needs region confirmation + `aws login` browser flow). AWS CLI v2 already installed; region currently `us-east-1`.
5. **Cashless HTML final review** — the KIRO-OPUS48 handoff (§4) lists remaining polish items (Macro view redesign, real components) if that thread is resumed.
6. **NFC hardware purchase** — buy NTAG213/215/216 (50–100 units) for Cashless; verify chip before purchase.
7. **"OpenSpec for the merge"** — user indicated an OpenSpec exists for the product/integration merge; it is **not** inside `Omini_Router`. New squad should confirm its location (candidates with OpenSpec: `Automonous_Agentic`, `RD_Agnostic_Engineering_Team`, `Consolidacao_Ofertas`) and reconcile OpenSpec × source × remote.

---

## 6. Runbook (how to operate)
**Connect to a server:**
```
# from WSL
ssh orq1        # or: ssh orq2
```
**Open OmniRoute dashboard:**
```
# machine WITH Tailscale (e.g. msi-laptop-1):
http://orq1:20128
# this Windows host (no Tailscale) — tunnel is auto-up via systemd:
http://localhost:20130      # (open in incognito first time to bypass PWA cache)
```
**Tunnel service (WSL):**
```
systemctl status omniroute-orq1.service      # 127.0.0.1:20130 -> orq1:20128
sudo systemctl restart omniroute-orq1.service
```
**Restart OmniRoute on orq1 (if needed):**
```
ssh orq1 'docker restart omniroute'
```
**Rollback OmniRoute image (if latest misbehaves):**
```
ssh orq1 'docker stop omniroute && docker rename omniroute omniroute-latest-bad && docker rename omniroute-broken-3.8.48-affinity-fix2 omniroute && docker start omniroute'
```

---

## 7. Known issues & gotchas
- **Windows here can't reach `100.x` tailnet IPs** (no Tailscale interface). Use `localhost:20130` tunnel or a Tailscale machine.
- **Netskope** on this Windows host hijacks `*.tailscale.com` DNS → 403. It's a corporate security agent; do not attempt to disable without IT/Security approval.
- **IPv6:** no internet route on this network; keep it as-is (disabling it broke Tailscale-dependent flows earlier — reverted).
- **OmniRoute blank page** = almost always a stale **PWA/service-worker cache** in the browser; open in incognito or clear site data.
- **`ssh` is case-sensitive** — use lowercase.
- OmniRoute secrets (`INITIAL_PASSWORD`, `OMNIROUTE_API_KEY`, Redis password) live in container env / `.env`; do not commit or echo them.

---

## 8. AS IS → TO BE (summary)
- **AS IS:** OmniRoute healthy on orq1 (official image) reachable via Tailscale/tunnel; Cashless fully specified (OpenSpec) with interactive architecture theater and agent prompt library; access standardized (`ssh orq1/orq2`, `localhost:20130`).
- **TO BE:** OmniRoute merge finalized with the affinity fix correctly rebuilt and committed/pushed; Cashless moved from spec/prototype into the F0–F7 build via the Multica-orchestrated agent squad; AWS Agent Toolkit completed; full CI/observability (trace E2E incl. Mercado Pago + Redis hotspots).

---

*End of handover. New squad: start at §0, then §1 (access) and §6 (runbook). Escalate open items in §5.*
