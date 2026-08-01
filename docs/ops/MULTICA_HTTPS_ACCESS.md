# Multica canonical HTTPS access guide

> **Audience:** humans, operators, coding agents, browser-automation agents, and remote collaborators.
>
> **Status:** current production behavior verified on 2026-07-31. This document contains secret metadata only, never secret values.

## 1. The rule to remember

Use this URL for normal Multica access:

```text
https://orq1.tail96e2c0.ts.net
```

Direct login URL:

```text
https://orq1.tail96e2c0.ts.net/login
```

**Do not create an SSH tunnel for normal Multica frontend or API access.** Do not append internal ports, replace the hostname with an IP address, or use localhost from another machine.

| Correct | Incorrect for normal access |
|---|---|
| `https://orq1.tail96e2c0.ts.net` | `http://orq1.tail96e2c0.ts.net` |
| `https://orq1.tail96e2c0.ts.net/login` | `https://orq1.tail96e2c0.ts.net:13100` |
| HTTPS port 443 through Tailscale | `http://100.118.244.61:13100` |
| Same HTTPS origin for UI and API | `http://127.0.0.1:18080` from a client |

The internal ports are implementation details on ORQ1:

- `127.0.0.1:13100`: Multica frontend;
- `127.0.0.1:18080`: Multica backend;
- `443`: canonical Tailscale Serve HTTPS entry point.

## 2. How the URL works

```text
Browser or authorized client
  -> local Tailscale client and tailnet ACL
  -> MagicDNS resolves orq1.tail96e2c0.ts.net
  -> TLS on port 443
  -> Tailscale Serve on ORQ1
       /, frontend assets       -> 127.0.0.1:13100
       /auth/*, /api/*, /ws     -> 127.0.0.1:18080
  -> Multica authenticates the application user
  -> PostgreSQL supplies workspaces, memberships, issues, and tasks
```

There are two independent access gates:

1. **Tailscale gate:** can this device reach the private ORQ1 service?
2. **Multica gate:** is this a valid Multica user, and which workspaces may that user access?

Tailscale access does not automatically log a user into Multica. A successful Multica login does not remove the requirement for tailnet connectivity.

### Why the hostname matters

The TLS certificate and browser cookies are bound to the HTTPS hostname. The Tailscale IP is not the canonical browser origin. Using an IP, HTTP, a different hostname, or an SSH-forwarded origin can break TLS, secure cookies, CSRF, WebSocket origin checks, and redirects even when the internal services are healthy.

## 3. Prerequisites

Before opening the URL:

1. The same device that runs the browser must have a working route into the authorized tailnet.
2. MagicDNS must resolve `orq1.tail96e2c0.ts.net`.
3. Tailnet ACLs must permit HTTPS to ORQ1.
4. The user must have an active Multica identity and membership in the intended workspace.

Secret-free checks:

```bash
tailscale status
tailscale ping orq1
tailscale ping orq1.tail96e2c0.ts.net
curl -fsSI https://orq1.tail96e2c0.ts.net/login | head -1
```

PowerShell connectivity check:

```powershell
Test-NetConnection orq1.tail96e2c0.ts.net -Port 443
curl.exe -I https://orq1.tail96e2c0.ts.net/login
```

Do not use `curl -k` and do not accept a browser certificate exception.

> Some older host-specific documentation says a Windows host was blocked from joining Tailscale by Netskope and used WSL as its tailnet entry point. That does not turn an SSH tunnel into the canonical Multica architecture. Fix or approve the client-to-tailnet path for the browser device; use SSH only for a separately authorized administrative task.

## 4. Current login contract

The deployed web page presents **email** and **password** fields and submits:

```text
POST /auth/login
Content-Type: application/json

{"email":"<email>","password":"<password>"}
```

The backend:

1. normalizes the email to lowercase and trims whitespace;
2. finds the password credential by email;
3. verifies the password using its stored bcrypt hash;
4. issues a JWT and returns HTTP 200 JSON;
5. sets `multica_auth` as an HttpOnly cookie;
6. sets the `multica_csrf` cookie for mutation protection.

After the successful login response, the **frontend/client** navigates to the requested route, first
workspace, or onboarding as applicable; callers of the API should expect 200 JSON, not a backend 3xx
redirect.

An anonymous request to `/api/me` returning `401` is expected and proves the protected API is reachable. It is not an outage.

### OpenSpec terminology

The active specification at `openspec/changes/native-runtimes-onboarding/specs/onboarding/spec.md` says “simple username/password login.” The implemented and deployed API field is currently `email`, not a separate username. For the current system, interpret that requirement as **email/password**. The same OpenSpec change still lists frontend rebuild/UAT tasks as open, so agents must not infer deployment state from the proposal alone.

## 5. Canonical privileged credential authority

The privileged bootstrap/owner login used for authenticated production validation is stored in AWS Secrets Manager.

| Field | Current authority |
|---|---|
| AWS account | `809809509961` |
| AWS Region | `sa-east-1` |
| Secret name | `prod/multica/bootstrap-owner` |
| Confirmed full ARN | `arn:aws:secretsmanager:sa-east-1:809809509961:secret:prod/multica/bootstrap-owner-3BIwjo` |
| JSON identity key | `email` |
| JSON password key | `password` |
| Version stage | `AWSCURRENT` |
| Access level | privileged owner login; effective frontend/workspace access still depends on application roles and memberships |

This is sometimes described operationally as the “default” login, but it is **not** a hardcoded
default, anonymous bypass, Tailscale identity, daemon credential, or task token. It is a privileged
application credential whose authority is Secrets Manager. The secret description explicitly states
that it creates no workspace or membership grant: successful authentication alone does not make a
workspace visible. The application user must already have the required role/membership.

### Dynamic references

Authorized automation refers to the fields without retrieving them into agent context:

```text
{{resolve:secretsmanager:arn:aws:secretsmanager:sa-east-1:809809509961:secret:prod/multica/bootstrap-owner-3BIwjo:SecretString:email:AWSCURRENT}}

{{resolve:secretsmanager:arn:aws:secretsmanager:sa-east-1:809809509961:secret:prod/multica/bootstrap-owner-3BIwjo:SecretString:password:AWSCURRENT}}
```

## 6. How humans should log in

1. Connect the browser device to the authorized tailnet.
2. Open `https://orq1.tail96e2c0.ts.net/login`.
3. An authorized owner/security operator obtains the `email` and `password` through the approved owner-controlled secret workflow, outside agent chat and logs.
4. Enter the values only into the Multica HTTPS login form.
5. Confirm the intended workspace and the full URL after login.
6. Do not send the password, cookie, JWT, CSRF value, screenshot of credentials, or browser storage to an agent.

If the page loads but login fails, the network path is already working. Investigate credential authorization, version stage, owner provisioning, or application authentication; do not build an SSH route.

## 7. How authorized agents may use the credential

### Non-negotiable safety rules

Agents must never:

- call `secretsmanager:GetSecretValue` or `BatchGetSecretValue` through CLI, SDK, MCP, curl, or another mechanism that returns plaintext to agent context;
- access the Secrets Manager Agent daemon directly;
- print, echo, log, paste, summarize, or commit the email/password values;
- put plaintext credentials in command arguments, scripts, screenshots, evidence, issue comments, or environment dumps;
- inspect browser cookies, `multica_auth`, `multica_csrf`, Authorization headers, or credential homes;
- use the owner credential from an ordinary Multica task.

For explicitly authorized browser/API automation, pass unresolved references in the incoming environment of `asm-exec`, and let only the approved child runner receive the resolved values:

```bash
MULTICA_OWNER_EMAIL='{{resolve:secretsmanager:arn:aws:secretsmanager:sa-east-1:809809509961:secret:prod/multica/bootstrap-owner-3BIwjo:SecretString:email:AWSCURRENT}}' \
MULTICA_OWNER_PASSWORD='{{resolve:secretsmanager:arn:aws:secretsmanager:sa-east-1:809809509961:secret:prod/multica/bootstrap-owner-3BIwjo:SecretString:password:AWSCURRENT}}' \
asm-exec -- /path/to/approved-secret-safe-login-runner
```

The runner path above is intentionally a placeholder, not permission to invent a new login script. Use only a reviewed runner that:

- reads the two values from its environment without printing them;
- uses the canonical HTTPS origin;
- keeps cookies and response bodies in a mode-`0700` temporary directory with mode-`0600` files;
- disables shell tracing, core dumps, screenshots, video, HAR, request logging, and debug dumps;
- deletes all temporary authentication artifacts with a trap;
- emits only fixed status messages or HTTP status codes;
- is executed in a bounded owner-authorized window, because same-UID/root processes may inspect child environments.

Do not use command substitution such as `PASSWORD=$(asm-exec ...)`. Do not run `asm-exec -- env`, `printenv`, `set`, `docker inspect`, or any command that could expose the resolved child environment.

### Ordinary agents inside Multica tasks

An agent launched by Main Brain receives a task-scoped `mat_` token and workspace binding. That agent must use the injected task identity. It must not retrieve or reuse the privileged browser owner credential.

### CLI operators

A persistent human/TL CLI profile uses the normal `multica login` or owner-approved PAT flow described in `MULTICA_KANBAN_TL_360.md`. Daemon tokens, task tokens, and browser owner credentials are not interchangeable.

## 8. When SSH is and is not appropriate

| Situation | SSH tunnel? |
|---|---|
| Open Multica UI | **No** |
| Log into Multica | **No** |
| Use Multica API through canonical HTTPS | **No** |
| Diagnose user membership, workspace, filters | **No** |
| ORQ1 host administration with explicit authorization | Possibly |
| Inspect loopback-only service health as an operator | Possibly |
| Legacy OmniRoute dashboard access documented for a host without tailnet browser routing | Separate case; not the Multica URL |
| Change Tailscale Serve, containers, systemd, or network configuration | High-impact operation requiring explicit authorization |

The SSH tunnel in `docs/ops/TAILNET_ACCESS.md` is for the separately deployed OmniRoute dashboard under a specific host constraint. It must not be copied as the Multica frontend access pattern.

## 9. Troubleshooting decision tree

### A. DNS does not resolve

- Confirm Tailscale is running on the client device.
- Confirm MagicDNS and the expected tailnet.
- Run `tailscale ping orq1`.
- Do not replace the hostname with the Tailscale IP for browser login.

### B. Port 443 is unreachable

- Check tailnet ACL/device authorization.
- Confirm the client, not merely another shell or VM, has the route.
- Escalate Tailscale Serve availability to the operator.
- Do not open ports `13100` or `18080` and do not create an ad hoc SSH tunnel.

### C. TLS warning

- Confirm the exact hostname is `orq1.tail96e2c0.ts.net`.
- Do not bypass certificate validation.
- Treat a certificate mismatch as a routing/origin error.

### D. `/login` returns 200 but credentials fail

- Connectivity, TLS, Serve, and frontend are already functioning.
- Confirm owner authorization to use `prod/multica/bootstrap-owner`.
- Confirm account `809809509961`, Region `sa-east-1`, full ARN, keys `email,password`, and stage `AWSCURRENT` without reading values into agent context.
- If automation is authorized, confirm the approved runner uses `asm-exec` and does not log requests or environments.
- Escalate provisioning/rotation issues to the owner; do not attempt direct database updates.

### E. Login succeeds but the expected board is absent

- Confirm the logged-in email.
- Confirm workspace membership and workspace selector.
- Compare the complete `/<workspace-slug>/issues` URL.
- Use `Issues`, scope `All`, Board grouped by Status, and reset filters.
- Do not create another workspace as a workaround.

### F. Anonymous `/api/me` returns 401

Expected. Authenticate through the login page.

### G. 502 or repeated 5xx

The HTTPS edge is reachable but an upstream may be unhealthy. Escalate to an authorized operator with timestamp, URL path, and HTTP status only. Do not include cookies, headers, or response bodies containing private data.

## 10. Current secret-free verification

Verified from the current host on 2026-07-31:

```text
/            -> 200
/login       -> 200
/api/config  -> 200
/api/me      -> 401 (expected while anonymous)
TLS verify   -> 0 (success)
```

This check proves the current canonical HTTPS edge, login page, public config route, protected API behavior, and TLS certificate. It does not retrieve credentials and does not perform a login.

## 11. Copy/paste instruction for other agents

```text
Multica canonical access contract:
- URL: https://orq1.tail96e2c0.ts.net
- Login: https://orq1.tail96e2c0.ts.net/login
- Normal UI/API access uses Tailscale Serve HTTPS on 443. Do not create SSH tunnels and do not use 13100/18080/IP/localhost as the browser origin.
- Current deployed login fields are email + password via POST /auth/login.
- Privileged owner credential authority is AWS Secrets Manager, account 809809509961, Region sa-east-1, secret prod/multica/bootstrap-owner, keys email and password, AWSCURRENT.
- Never retrieve or expose secret values. Authorized automation must use unresolved dynamic references through asm-exec into an approved secret-safe child runner.
- An ordinary Multica task uses its injected mat_ task token, never the owner browser credential.
- If /login opens but login fails, troubleshoot auth/authorization, not SSH/network routing.
- Read docs/ops/MULTICA_HTTPS_ACCESS.md before changing any route, service, or credential flow.
```

## 12. Sources of truth

- `docs/ops/MULTICA_KANBAN_TL_360.md`: complete Kanban, workspace, CLI, API, and UI operations.
- `docs/ops/TAILNET_ACCESS.md`: host-specific tailnet and separate OmniRoute dashboard notes.
- `openspec/changes/native-runtimes-onboarding/specs/onboarding/spec.md`: active onboarding/login requirement.
- `multica-auth-work/server/internal/handler/auth.go`: `/auth/login` request and session issuance.
- `multica-auth-work/server/internal/handler/auth_provider.go`: email normalization and bcrypt verification.
- `multica-auth-work/server/internal/auth/cookie.go`: session/CSRF cookie behavior.
- `multica-auth-work/packages/core/api/client.ts`: frontend login request.
- `.deploy-control/p0/evidence/orq17-stage3b-execution-blockers-20260727.md`: confirmed secret metadata and owner bootstrap history.
- `.deploy-control/p0/evidence/gemini-selector-p0-production-repair-20260727.md`: authenticated production proof using the Secrets Manager-backed owner credential without exposing it.
- `.agents/skills/aws-secrets-manager/SKILL.md`: mandatory secret-safe agent workflow.
