# ORQ-17 — Gate A preflight independent re-review

- Scope: READ-ONLY review of `gtl-orq17-gate-a-preflight.md`; no `serve`, `cert`, ACL,
  restart or other operational command executed.
- Existing card: ORQ-17.

## Verdict: PASS (preflight only; not authorization to apply)

The corrected preflight's command grammar and routing plan are consistent with the Tailscale
Serve CLI reference, subject to the explicit caveat that no live route was exercised here.
The official reference documents `tailscale serve [flags] <target>`, `--set-path=<path>`,
automatic HTTPS certificate provisioning, `serve status --json`, and `serve reset`:
<https://tailscale.com/docs/reference/tailscale-cli/serve> (validated 2026-01-26, lines 351–400,
474–495). The preflight records the installed 1.98.9 build and clean `No serve config`
state at lines 22–31.

| Check | Result | Evidence |
|---|---|---|
| Tailscale 1.98.9 syntax | PASS | Preflight lines 37–40 use `tailscale serve --bg --https=443 --set-path=<path> <target>`, matching the official grammar and flag. |
| Prefix handling | PASS, unexercised | Lines 41–45 preserve `/api/` and `/uploads/` in the target while mounting those prefixes. This is consistent with Serve's path-mount/reverse-proxy model, but remains a plan claim until a non-production harness or authorized live smoke proves the exact forwarded path. |
| Exact/subtree pairs | PASS | Lines 47–51 and commands 67–70 cover both `/api/` + `/api` and `/uploads/` + `/uploads`; Go ServeMux-like matching is the documented model (official reference lines 395–397). |
| Prefix preservation | PASS, unexercised | Targets `127.0.0.1:18080/api/` and `/uploads/` are present at lines 67–70; no target silently drops the backend prefix. |
| Exact route precedence | PASS | Explicit `/auth/login`, `/auth/google`, `/auth/logout`, `/ws` precede root catch-all in the plan (lines 47–53); route precedence remains to be verified by the eventual status/config output. |
| Automatic certificate | PASS as expectation | HTTPS mode is used at lines 62–71; Tailscale documents automatic HTTPS certificate provisioning for Serve (official reference lines 387–397). This is not a certificate issuance result. |
| Status verification | PASS | `tailscale serve status --json` at lines 74–77 is the documented status form (official reference lines 474–483). |
| Rollback reset | PASS | `tailscale serve reset` at lines 79–84 is the documented reset subcommand (official reference lines 490–495). It was not executed. |
| Funnel | PASS | The artifact contains only `tailscale serve` commands and explicitly records zero `tailscale serve apply`, `tailscale cert`, ACL changes and restarts at lines 1–7; no `tailscale funnel` command or Funnel enablement is present. |

## Gate boundary

This is **PASS for syntax/preflight readiness only**. It does not prove the forwarded path,
certificate issuance, ACL reachability, or browser behavior. Any Gate A execution still requires
separate owner authorization and must capture `tailscale serve status --json` before/after;
Funnel must remain uninvoked.
