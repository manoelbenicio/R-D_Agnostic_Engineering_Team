# Go Integration - Sidecar Lifecycle

Status: PRE-DEPLOY REQUIRED

## 1. Goal

Multica Go starts, monitors, and stops Rust/prodex L2. Go does not implement
runtime routing for in-flight requests.

## 2. Lifecycle

States:

```text
configured -> starting -> alive -> ready -> draining -> stopped -> failed
```

## 3. Start

Go must:

- generate sidecar token;
- set required env vars;
- start sidecar/process;
- call `/healthz`;
- call `/readyz`;
- apply policy;
- register approved account refs;
- open event stream;
- mark sidecar ready.

## 4. Stop

Go must:

- stop new sessions;
- apply drain if available;
- stop sidecar;
- record event;
- preserve logs.

## 5. Failure

Go fails closed if:

- sidecar cannot start;
- readiness fails;
- policy apply fails;
- event stream unavailable;
- kill switch unavailable.

## 6. Hotspot

Only Codex#5.5#C may touch daemon dispatch/execenv hotspot during this stream.
