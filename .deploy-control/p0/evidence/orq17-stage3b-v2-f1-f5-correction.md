# ORQ-17 Stage3B V2 — F1–F5 correction record

- Status: **PROPOSAL / REQUEST_REVIEW**.
- Fixed login: `dataops.cloud.mbf@gmail.com`; no password is embedded.
- Gate 0: General-TL accepted local edit/build/test only. Toolchain complete; private cache
  `1721 MiB` under the `2048 MiB` cap. Runtime, AWS, secret resolution, DB, Docker/Compose and
  Tailscale remained prohibited and were not invoked.

| Fix | Executable correction |
|---|---|
| F1 | Added `rollback-breakglass`: written owner receipt + SHA gate; skips only password `Login`; keeps ORQ1/DB binding, queue guard plus same-transaction member lock, exact FK catalog, all ref checks/locks, explicit deletes and `0|0|0`. |
| F2 | `provision` now requires a private custody receipt bound to the fixed login and approved hash before Compose/helper mutation. |
| F3 | Provision returns committed user ID; guard-commit failure attempts compensation only under a fresh guard and emits distinct fixed compensated/state-unknown tokens. Rollback committed + guard failure has its own token. Runner captures aggregate `0|0|0` or `1|1|0` before rollback; manual SQL delete is forbidden. |
| F4 | Same resolved child retains credentials through contained HTTP login: JSON via helper stdin→curl stdin, private cookie jar, `/auth/login=200`, authenticated `/api/me=200`. Failure captures counts before guarded rollback; success is emitted only after the gate. |
| F5 | Runner executes `command -V printf` builtin gate before credential expansion and `ulimit -c 0`. |

Final lock ruling: the guard locks only `agent_task_queue` in `SHARE`; `member` is included in both
`beginMutation` lists in `SHARE ROW EXCLUSIVE` on the mutation connection. No cross-connection
member lock remains, eliminating the reviewed self-deadlock.

## Rebuilt package

```text
helper source   fabc8f8585449d2263fbc62cb7437726db9cae0c33c1607adbad17abebb59e82
helper tests    92f92b5a58e6efd763452dbd79eeb096d94591cd6334989fcf6b3155dc51dc97
runner          c3d64f85a348fdf1d5b373b9253f0fb5bc19f5b468add7ef17072c6f253d8b42
source closure  719f9a6b44c2b6fab1d198bf7af77a00164946d68b7df9b2ce8fa9726ba3d9d1
binary A/B      1710e01001440c82e12a4326d18d49e8905d72ea6b97dabe5d482a24934af87b
```

Local-only results: `go test ./...` PASS; `go vet ./...` PASS; gofmt byte match PASS; `sh -n`
PASS; 1,696 closure entries PASS with zero failures; two builds using separate caches produced the
same binary hash.

No secret/provider/DB/container/Serve/Funnel/board mutation or runtime command was executed.
