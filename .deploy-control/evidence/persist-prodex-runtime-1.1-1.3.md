# Evidence — persist-prodex-runtime-integration tasks 1.1-1.3

- agent: Kiro
- stream: PRODEX-RUNTIME-1.1-1.3
- finished_at: 2026-07-18T20:20:00Z
- scope: focused offline Go tests only; NO source edits (behavior already present in shared baseline)
- credentials/production: none touched; all env/tokens/paths in tests are synthetic constants and temp fixtures

## Spec ↔ implementation coverage (compared before editing)

Task 1.1 — Separate Prodex and adapter executables (spec: "Separate Prodex and adapter executables", "Executable mismatch"):
- `loadL2RuntimeConfig` reads `MULTICA_L2_SIDECAR_PATH`, requires non-empty, resolves via `exec.LookPath`
  into `L2RuntimeConfig.SidecarPath`. `MULTICA_PRODEX_PATH` is validated separately in
  `loadProdexLaunchConfig`. Independent env key + config field + lifecycle → fail-closed on each.

Task 1.2 — Adapter starts the pinned gateway (spec: "Adapter starts the pinned gateway"):
- `l2Sidecar.runLoop` execs `cfg.L2Runtime.SidecarPath` (the adapter) with
  `cmd.Env = prodexSidecarEnv(cfg)`; `prodexSidecarEnv(cfg)` injects `MULTICA_PRODEX_PATH=cfg.Prodex.Path`
  plus `MULTICA_L2_BEARER_TOKEN`, `PRODEX_PG_URL`, and forces `PRODEX_ALLOW_UNSAFE_CHILD_ENV=off`
  (Golden Rule 7). `l2SidecarArgs()` returns adapter args (default `127.0.0.1:43117`) and rejects an
  executable-path first argument.

Task 1.3 — Required-mode fail-closed (spec: "Required configuration is missing"):
- `loadProdexLaunchConfig`: `MULTICA_PRODEX_REQUIRED` && !`MULTICA_PRODEX_ENABLED` → error.
- `config.Load`: `prodexCfg.Required` && !`l2Cfg.Enabled` → error ("prodex is required but MULTICA_L2_ENABLED is disabled").
- `loadL2RuntimeConfig`: required && missing `MULTICA_L2_TENANT_ID` → error.

Conclusion: behavior for 1.1-1.3 already implemented in the working-tree shared baseline. Added focused
offline tests to prove it (per steering: accept existing impl via executable evidence, add no unnecessary code).

## New test file (disjoint ownership)

`multica-auth-work/server/internal/daemon/prodex_runtime_integration_test.go` (new; no locked source touched).

## Build (offline, go toolchain /home/dataops-lab/go-sdk/bin/go)

```
$ go version
go version go1.26.4 linux/amd64
$ go build ./internal/daemon/
BUILD_OK
$ go vet ./internal/daemon/
(no output)
```

## Focused tests (offline; go test -count=1)

```
=== RUN   TestLoadL2RuntimeConfigRequiresSidecarPath
--- PASS: TestLoadL2RuntimeConfigRequiresSidecarPath (0.00s)
=== RUN   TestLoadL2RuntimeConfigRejectsMissingSidecarExecutable
--- PASS: TestLoadL2RuntimeConfigRejectsMissingSidecarExecutable (0.00s)
=== RUN   TestLoadL2RuntimeConfigResolvesSidecarIndependentOfProdexPath
--- PASS: TestLoadL2RuntimeConfigResolvesSidecarIndependentOfProdexPath (0.00s)
=== RUN   TestProdexSidecarEnvInjectsPinnedProdexPath
--- PASS: TestProdexSidecarEnvInjectsPinnedProdexPath (0.00s)
=== RUN   TestL2SidecarArgsDefaultsToAdapterListenNotProdexPath
--- PASS: TestL2SidecarArgsDefaultsToAdapterListenNotProdexPath (0.00s)
=== RUN   TestL2SidecarArgsRejectsExecutablePathFirstArg
--- PASS: TestL2SidecarArgsRejectsExecutablePathFirstArg (0.00s)
=== RUN   TestLoadProdexLaunchConfigRequiredButDisabledFailsClosed
--- PASS: TestLoadProdexLaunchConfigRequiredButDisabledFailsClosed (0.00s)
=== RUN   TestLoadL2RuntimeConfigRequiredMissingTenantFailsClosed
--- PASS: TestLoadL2RuntimeConfigRequiredMissingTenantFailsClosed (0.00s)
=== RUN   TestLoadL2RuntimeConfigNotRequiredDefaultsTenant
--- PASS: TestLoadL2RuntimeConfigNotRequiredDefaultsTenant (0.00s)
PASS
ok  	github.com/multica-ai/multica/server/internal/daemon	0.085s
```

Regression check (existing prodex/L2 suite, no collisions):

```
$ go test ./internal/daemon/ -run 'Prodex|L2Sidecar|L2Runtime'
ok  	github.com/multica-ai/multica/server/internal/daemon	0.043s
```

Note: `go test` runs offline (unit-level loaders + env only); no Postgres, no network, no live sidecar.
