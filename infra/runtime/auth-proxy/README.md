# Auth proxy — reference skeleton (Option B, NOT wired)

> This is a **non-functional reference** for the "custom proxy" option in
> [`docs/cloud-runtime-auth.md`](../../../docs/cloud-runtime-auth.md) (task
> 6.5). It is **not** built into the runtime image, has **no** dependencies
> installed, and requires **no** credentials. It exists only to make the
> decision concrete. Do not deploy as-is.

If Option B is chosen, this proxy sits in front of the runtime, verifies the
Firebase ID token the SPA already sends, and forwards to the runtime on
localhost. The shape would be roughly:

```python
# proxy.py — SKETCH ONLY. Pseudocode; deps intentionally not declared.
#
# Verifies the Firebase ID token (Authorization: Bearer <jwt>), enforces a
# tenant claim, then reverse-proxies to the local runtime. WebSocket upgrade
# handling is omitted here and MUST be implemented for the terminal stream.
#
# Run target (NOT configured): uvicorn proxy:app --host 0.0.0.0 --port $PORT
#
# from fastapi import FastAPI, Request, HTTPException
# from firebase_admin import auth, initialize_app   # needs creds at runtime
#
# RUNTIME_UPSTREAM = "http://127.0.0.1:8080"
# EXPECTED_TENANT = os.environ["GO_CORE_TENANT_ID"]
# initialize_app()
# app = FastAPI()
#
# async def require_user(request: Request):
#     header = request.headers.get("authorization", "")
#     if not header.startswith("Bearer "):
#         raise HTTPException(401, "missing bearer token")
#     try:
#         claims = auth.verify_id_token(header[7:])
#     except Exception:
#         raise HTTPException(401, "invalid token")
#     if claims.get("tenant") != EXPECTED_TENANT:
#         raise HTTPException(403, "wrong tenant")
#     return claims
#
# @app.api_route("/{path:path}", methods=["GET","POST","PUT","DELETE","PATCH"])
# async def proxy(path: str, request: Request, _user=Depends(require_user)):
#     ...  # httpx reverse-proxy to RUNTIME_UPSTREAM, stream body + headers
#     ...  # WebSocket (/ws) upgrade handling required for the terminal grid
```

## Why it is not wired

- Verifying tokens is security-critical; the implementation choice (Option A
  vs B vs hybrid) is a **human decision point** still open.
- It would pull `fastapi`, `firebase-admin`, `httpx`, etc. into the image and
  add a second process to supervise — premature before the decision lands.
- Overlaps with the `validation-proxy` change (topology-guard). If Option B
  wins, the two proxy layers should be unified rather than duplicated.

## Next steps if Option B is selected

1. Pin dependencies (`fastapi`, `uvicorn`, `firebase-admin`, `httpx`).
2. Implement HTTP + **WebSocket** reverse-proxy with streaming.
3. Add the proxy as PID-1-managed second process (or a sidecar container).
4. Unit-test JWT verification (valid/expired/wrong-audience/wrong-tenant).
5. Fold in topology-guard from `validation-proxy`.
