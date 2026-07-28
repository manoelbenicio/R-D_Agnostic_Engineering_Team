# ORQ-17 Stage3B V2 — private build and self-test evidence

- Status: **PROPOSAL / build proof only**.
- Date: 2026-07-27.
- No secret reference was resolved. No provider, AWS, database, container, Serve/Funnel, queue,
  board or application-runtime command was run.
- Build root: `/home/ec2-user/.local/state/orq17-stage3b-v2-build-20260727T165000Z`, mode `0700`.
  `GOCACHE`, `GOMODCACHE` and `HOME` were private children of that directory.

## Pinned toolchain and inputs

```text
Go             go1.26.1 linux/amd64
Go executable  sha256:548e61b2d08ae52043be2f1924ed3c1d2b2c41967e360f3e317667f6fa912fc2
GOOS/GOARCH     linux/amd64
asm-exec       sha256:d55eb38ad33a5b76f584ca180f633ecc120cf39b8fd29427ffbe11a8fbf19556
helper         sha256:fabc8f8585449d2263fbc62cb7437726db9cae0c33c1607adbad17abebb59e82
tests          sha256:92f92b5a58e6efd763452dbd79eeb096d94591cd6334989fcf6b3155dc51dc97
go.mod         sha256:a5353ddad76e7bd5c5755b04520773cdef9c999d1cca67dd592e00fb1c31d48f
go.sum         sha256:919cc991383fee36687ed926f944f092ffdd93cfd948427a12a7fd7aabb60ea8
runner         sha256:c3d64f85a348fdf1d5b373b9253f0fb5bc19f5b468add7ef17072c6f253d8b42
closure list   sha256:719f9a6b44c2b6fab1d198bf7af77a00164946d68b7df9b2ce8fa9726ba3d9d1
closure files  1696 non-standard-library compiled inputs, each individually SHA-256 pinned
```

The standard library is bound by the exact Go executable hash. The closure manifest covers helper
and test files, the read-only `server-snapshot`, all non-standard-library compiled files, module
versions and embedded files. The snapshot was made read-only before compilation.

## Commands and results

Both test/build invocations used:

```text
HOME=<private>
GOWORK=off
GOFLAGS=-mod=readonly
CGO_ENABLED=0
GOCACHE=<private>
GOMODCACHE=<private>
```

Build flags were `-trimpath -buildvcs=false -ldflags=-buildid=`. Results:

```text
go test ./...  PASS
gofmt           PASS (artifact byte-identical)
go vet ./...    PASS
sh -n runner    PASS
build A        1710e01001440c82e12a4326d18d49e8905d72ea6b97dabe5d482a24934af87b
build B        1710e01001440c82e12a4326d18d49e8905d72ea6b97dabe5d482a24934af87b
closure check  1696 checked, 0 failed
F1-F5 unit path fixed-owner JSON/body and identity binding PASS
private cache   1721 MiB, below accepted 2048 MiB cap
```

Build B used a separate private build cache. `go tool nm` found no linked
`GetSecretValue`, `BatchGetSecretValue` or SMA symbol. The complete source closure necessarily
records unused AWS SDK source filenames compiled as dependencies of the existing large
`internal/handler` package; this is provenance evidence, not an invoked secret API. The helper and
runner themselves contain no such call.

This establishes a candidate expected binary hash. It is **not execution authorization**. An
independent reviewer must reproduce the build from the pinned closure and approve the package
before the binary is used.
