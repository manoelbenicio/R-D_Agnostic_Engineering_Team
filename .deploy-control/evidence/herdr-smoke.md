# Herdr Coordination Smoke Evidence

Status: GREEN - live Herdr primitives exercised
Timestamp: 2026-07-05T06:27:39Z
Executor: Codex-A
Task: 6.7 Herdr coordination smoke
Secrets present: false

## Environment

```text
HERDR_ENV=1
herdr binary: /home/dataops-lab/.local/bin/herdr
herdr server: running
herdr version: 0.7.1
protocol: 14, compatible=yes
```

## Discovery

Commands:

```text
herdr agent list
herdr pane list
herdr status server
```

Result:

```text
agents discovered: 9
panes discovered: 10
server status: running
```

## Agent Send

Command:

```text
herdr agent send Gemini#PRO#31 '[06-06 smoke] non-action coordination message; no response required'
```

Result:

```text
code=0
{"id":"cli:agent:send","result":{"type":"ok"}}
```

## Notification

Command:

```text
herdr notification show '06-06 smoke' --body 'Herdr coordination smoke notification' --position top-right --sound none
```

Result:

```text
code=0
{"id":"cli:notification:show","result":{"reason":"shown","shown":true,"type":"notification_show"}}
```

## Event / Wait

Command:

```text
herdr wait agent-status w3:pN --status idle --timeout 1000
```

Result:

```text
code=0
{"event":"pane.agent_status_changed","data":{"pane_id":"w3:pN","workspace_id":"w3","agent_status":"idle","agent":"agy"}}
```

## Verdict

Herdr discovery, agent send, notification, and agent-status event/wait all
returned successful live local results. No raw credentials, tokens, or provider
payloads were captured.
