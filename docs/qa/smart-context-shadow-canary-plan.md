# Smart Context Shadow / Canary Plan

Status: PRE-DEPLOY REQUIRED

## 1. Default

Initial PROD mode:

```text
PRODEX_SMART_CONTEXT_SHADOW=1
PRODEX_SMART_CONTEXT_CANARY_PERCENT=0
```

No live rewrite without owner approval and evidence.

## 2. Shadow Mode

Purpose: measure before/after and validation decisions while sending original
request upstream.

Required metrics:

- estimated tokens before;
- estimated tokens after;
- rewrite ratio;
- selected segment categories;
- fallback reasons;
- validation result;
- additional turns after shadow if measurable.

Gate to canary:

- no protocol integrity warning;
- no tool-call integrity warning;
- no secret in logs;
- event stream healthy;
- owner approval.

## 3. Canary Mode

Initial canary:

```text
PRODEX_SMART_CONTEXT_CANARY_PERCENT=1
```

Promotion requires:

- exact fallback works;
- no continuation failure;
- no tool-call corruption;
- no JSON corruption;
- no missing mandatory artifact;
- p95 rewrite overhead acceptable to owner;
- rollback and kill switch tested.

## 4. Live Mode

Live mode is allowed only after:

- shadow evidence;
- canary evidence;
- QA sign-off;
- owner approval;
- kill switch test.

## 5. Immediate Disable Conditions

Disable Smart Context if:

- previous response not found after rewrite;
- invalid tool-call continuation;
- corrupted JSON;
- repeated missing context recovery;
- secret appears;
- fallback exact fails;
- user-visible task regression linked to rewrite.
