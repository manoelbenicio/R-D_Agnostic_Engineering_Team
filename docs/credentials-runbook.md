# Provider CLI Credentials — Plan & Runbook

**Scope:** Make CAO `GET /auth/sessions` return real sessions for the three
providers in use — **Codex, Kiro, Gemini**. Claude is explicitly out of scope
(no Anthropic account). No Dockerfile consolidation (the current stack works).

---

## 1. Verified state (live, read-only checks — 2026-05-31)

| Check | Result |
|-------|--------|
| CAO `/health` | HTTP 200 (healthy) |
| `/agents/providers` → `codex` | `installed: true` |
| `/agents/providers` → `kiro_cli` | `installed: true` |
| `/agents/providers` → `gemini_cli` | `installed: true` |
| `/agents/providers` → `claude_code` | `installed: false` (expected, unused) |
| `/auth/sessions` | `[]` (no sessions discovered yet) |

**Conclusion:** the install side is fully working — the three worker CLIs are
baked into the container by `bootstrap.ps1`'s `WORKER_CLI` build arg. The empty
`/auth/sessions` is a **credentials/discovery** question, not a missing-CLI bug.

## 2. How the system is wired (from the code, not assumption)

- `bootstrap.ps1` builds `infra/runtime/Dockerfile` (image `agentverse-cao:latest`)
  with `WORKER_CLI = "npm i -g @openai/codex && curl …cli.kiro.dev/install | bash
  && curl …antigravity.google/cli/install.sh | bash"`. Antigravity installs as
  `agy` and is symlinked to `gemini` in the Dockerfile.
- `bootstrap.ps1`'s `docker run` mounts host credential dirs **read-only**:
  `~/.codex → /root/.codex`, `~/.kiro → /root/.kiro`, `~/.gemini → /root/.gemini`.
- CAO `auth_routes.py` discovers sessions by scanning those `/root/.<cli>` dirs
  for credential files (`credentials.json`, `auth.json`, `config.json`, token-
  bearing `*.json`, etc.) and extracting email / expiry / subscription.
- Architecture (already chosen, correct): **authenticate on the host once; the
  container reuses those logins via the mounts.** Browser OAuth cannot run
  headless inside Docker, so in-container login is intentionally not used.

## 3. Root-cause hypotheses for `/auth/sessions == []`

Ranked by likelihood:

1. **(Most likely) CLIs not yet logged in on the host** → `~/.codex` / `~/.kiro`
   / `~/.gemini` are empty or absent → nothing mounted → nothing discovered.
   This is expected behavior, **not a code defect.**
2. Logged in, but the credential **filenames don't match** what `auth_routes.py`
   scans for → discovery-logic gap (in the other session's file).
3. Logged in on host, but **mount path mismatch** → files not visible in the
   container → a `bootstrap.ps1` fix.

## 4. Diagnosis steps (read-only — run on Windows PowerShell, paste output)

```powershell
# A. Are the host credential dirs present and non-empty?
dir $env:USERPROFILE\.codex
dir $env:USERPROFILE\.kiro
dir $env:USERPROFILE\.gemini

# B. What is actually mounted inside the running container?
wsl docker exec agentverse-cao sh -c "ls -la /root/.codex /root/.kiro /root/.gemini 2>&1"
```

**Branch on the result:**
- **A shows empty/missing** → Hypothesis 1. Go to §5 (log in). No code change.
- **A has files, B missing** → Hypothesis 3. Mount-path fix in `bootstrap.ps1`.
- **A and B both have files, but `/auth/sessions` still `[]`** → Hypothesis 2.
  Investigate `auth_routes.py` credential-filename matching (other session's file —
  coordinate before editing).

## 5. Fix path for Hypothesis 1 (expected case) — host login, no code change

```powershell
# Authenticate each CLI once on the host (browser/device flow opens):
codex          # ChatGPT OAuth, then exit          -> writes %USERPROFILE%\.codex
gemini         # Google login, then exit            -> writes %USERPROFILE%\.gemini
# Kiro (bash installer lives in WSL):
wsl bash -lc "kiro auth login"   # or the login command its CLI documents

# Verify creds now exist on host
dir $env:USERPROFILE\.codex, $env:USERPROFILE\.kiro, $env:USERPROFILE\.gemini

# Restart the stack so the container re-mounts the now-populated dirs
.\bootstrap.ps1 stop
.\bootstrap.ps1
```

## 6. Test (end-to-end acceptance)

```powershell
curl http://127.0.0.1:9889/auth/sessions
```
- **PASS:** JSON array with one entry per logged-in provider, each showing
  `cli_provider`, `account_email`, `status`, `expires_at`.
- Then in the SPA (`http://localhost:5173` → Sessions page): the provider cards
  render with the real account email and a green/active status dot.

## 7. What is explicitly NOT being done

- No Claude Code install or `~/.claude` mount (no Anthropic account).
- No consolidation of `infra/cao/Dockerfile` and `infra/runtime/Dockerfile` —
  the running stack builds `infra/runtime/Dockerfile` and is healthy; merging
  two working files is unnecessary risk.
- No edits to any file pending the §4 diagnosis and explicit written approval.

## 8. Decision gate

This plan is **diagnosis-first**. The likely outcome (Hypothesis 1) needs **zero
code changes** — just host login. A code change is only justified if §4 lands on
Hypothesis 2 or 3, and any edit to `auth_routes.py` / `infra/cao/*` must be
coordinated with the session that owns those files. Nothing will be changed
without written approval.