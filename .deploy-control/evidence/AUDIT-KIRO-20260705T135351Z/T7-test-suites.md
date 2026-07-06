# T7 Test Suites Evidence

- Timestamp UTC: 2026-07-05T14:18:05Z

## go-race-container

Command:

```bash
docker run --rm -v "/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work/server:/work" -w /work golang:1.24-alpine go test -race ./internal/l2runtime ./internal/daemon
```

Exit code: 2

Output:

```text
go: -race requires cgo; enable cgo by setting CGO_ENABLED=1

```

## cargo-test-prodex-sidecar

Command:

```bash
cd multica-auth-work/prodex-sidecar && cargo test
```

Exit code: 0

Output:

```text
   Compiling prodex-sidecar v0.1.0 (/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work/prodex-sidecar)
    Finished `test` profile [unoptimized + debuginfo] target(s) in 3.68s
     Running unittests src/main.rs (target/debug/deps/prodex_sidecar-c4103c4013007992)

running 0 tests

test result: ok. 0 passed; 0 failed; 0 ignored; 0 measured; 0 filtered out; finished in 0.00s


```
