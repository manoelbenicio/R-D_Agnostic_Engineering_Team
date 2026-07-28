# ORQ-17 Stage3B V2 — independent review request

- Status: **PROPOSAL / REQUEST_REVIEW**.
- Requested decision: `PASS` or `BLOCK`; all C1–C8 requirements are conjunctive.
- Reviewer must be independent of the V1/V2 author and must not resolve a secret or mutate AWS,
  provider, DB, container, Serve/Funnel, queue or board state.

Review these artifacts and their exact hashes:

```text
orq17-stage3b-v2-bootstrap-helper.go.txt
  fabc8f8585449d2263fbc62cb7437726db9cae0c33c1607adbad17abebb59e82
orq17-stage3b-v2-bootstrap-helper_test.go.txt
  92f92b5a58e6efd763452dbd79eeb096d94591cd6334989fcf6b3155dc51dc97
orq17-stage3b-v2-helper-go.mod.txt
  a5353ddad76e7bd5c5755b04520773cdef9c999d1cca67dd592e00fb1c31d48f
orq17-stage3b-v2-helper-go.sum.txt
  919cc991383fee36687ed926f944f092ffdd93cfd948427a12a7fd7aabb60ea8
orq17-stage3b-v2-sealed-runner.sh.txt
  c3d64f85a348fdf1d5b373b9253f0fb5bc19f5b468add7ef17072c6f253d8b42
orq17-stage3b-v2-source-closure.sha256
  719f9a6b44c2b6fab1d198bf7af77a00164946d68b7df9b2ce8fa9726ba3d9d1
asm-exec
  d55eb38ad33a5b76f584ca180f633ecc120cf39b8fd29427ffbe11a8fbf19556
candidate helper binary
  1710e01001440c82e12a4326d18d49e8905d72ea6b97dabe5d482a24934af87b
```

The reviewer should reproduce the binary twice from a read-only closure, re-run unit tests,
`gofmt`, `go vet`, shell syntax and closure checks, then audit:

1. unresolved-reference child-environment/NUL-stdin transport and residual `/proc` risk;
2. exact ORQ1/Compose/DB tuple and reversible guard-transaction admission freeze;
3. cleanup/rollback result propagation with the sole `os.Exit` at the outer boundary;
4. zero-member checks and the deliberately limited `HTTP_LOGIN=NOT_CALLED` claim;
5. exact live 16-FK/15-table contract, noting migration 029 dropped
   `daemon_pairing_session` and any live residual must BLOCK;
6. fixed private output reducer;
7. full ARN, `sa-east-1`, one `AWSCURRENT` version stability and consumed one-time authorization.

No credential provision or rollback is authorized until the independent decision is durably
recorded against these hashes.
