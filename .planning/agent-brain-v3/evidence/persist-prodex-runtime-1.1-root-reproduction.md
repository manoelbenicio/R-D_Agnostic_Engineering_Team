# Persist Prodex runtime task 1.1 — root supplemental reproduction

- Date: 2026-07-18T20:51:00Z
- Reviewer: Codex/root, supplemental independent reproducer
- Producer: Opus48#A
- Adjudicator: Kiro#Opus48-TL
- Exact task: `persist-prodex-runtime-integration` 1.1 — validate
  `MULTICA_L2_SIDECAR_PATH` independently from pinned
  `MULTICA_PRODEX_PATH`
- Disposition: **TECHNICAL PASS / CONTRACT INCOMPLETE / TASK REMAINS OPEN**

## Provenance and scope

This reproduction followed the interrupted Gemini-squad-design review recorded
in `persist-prodex-runtime-1.1-readiness-audit.md` (SHA-256
`ef0db321240441a1e9434d9c76c5c1a0927188d55ab64118f94aa3c541d409d2`).
That reviewer statically verified the slice but hit its quota before it could
finish the absolute-toolchain retry. Codex/root independently ran the missing
commands. Kiro#Opus48-TL remains the only task adjudicator; this artifact does
not self-accept or change a checkbox.

### Process exception

Codex/root created this supplemental artifact without first recording a
Golden Rule pre-edit check-in. This disclosure is retrospective and does not
cure that missing pre-edit step. The executable result may be used as
supplementary technical evidence, but this artifact cannot independently
satisfy the process/evidence contract or justify closing task 1.1.

## Frozen source manifest

```text
312fef692eeca53b84413f429b2cb19136974f7326a133105379d18263bdf78e  internal/daemon/prodex_runtime_integration_test.go
a54fb79dcb55e4f5928261a4a7cb200dd6377c57c528168f0983053e561139de  internal/daemon/l2_runtime.go
82035719e5a3b3a1472af973412f431e0c14ad11e04939885141893187ee38f7  internal/daemon/prodex.go
```

Paths are relative to `multica-auth-work/server`.

## Executable offline reproduction

Toolchain and boundary:

```text
GOTOOLCHAIN=local
GOPROXY=off
GOSUMDB=off
/home/dataops-lab/go-sdk/bin/go
```

Anchored test selector:

```text
^(TestLoadL2RuntimeConfigRequiresSidecarPath|TestLoadL2RuntimeConfigRejectsMissingSidecarExecutable|TestLoadL2RuntimeConfigResolvesSidecarIndependentOfProdexPath)$
```

Commands:

```text
/home/dataops-lab/go-sdk/bin/go test -v -count=20 ./internal/daemon -run "$regex"
/home/dataops-lab/go-sdk/bin/go test -race -count=1 ./internal/daemon -run "$regex"
```

Results:

```text
x20 exit: 0
top-level RUN: 60
top-level PASS: 60
FAIL: 0
SKIP: 0
race exit: 0
```

The three tests use `t.Setenv` and synthetic/missing executable paths. No DB,
network, live provider, credential, auth home, or environment-secret value was
read or used.

## Source-contract result

- `internal/daemon/prodex.go:83` reads
  `MULTICA_L2_SIDECAR_PATH` separately.
- `internal/daemon/prodex.go:85` fails closed when the sidecar path is absent.
- The tests at `prodex_runtime_integration_test.go:43`, `:59`, and `:76`
  prove required-path failure, missing-executable rejection, and independence
  from `MULTICA_PRODEX_PATH`.
- `internal/daemon/prodex.go:26` and `:150` handle the pinned Prodex path on
  its separate configuration/environment path.

## Evidence-contract boundary

- Proposed evidence identifier: `EV-PP-1.1-ROOT-REPRO` (not registered or
  accepted by this reviewer).
- The task maps exactly to OpenSpec task 1.1.
- No formal Prodex-specific `AB-REQ-*` exists in
  `.planning/agent-brain-v3/REQUIREMENTS.md`; this reviewer does not invent
  one. That missing formal mapping keeps the evidence contract incomplete.
- Producer, supplemental reviewer, and adjudicator identities are distinct.
- No product, test, OpenSpec, task checkbox, evidence index, git index,
  credential, or live-production state was changed.

## Verdict

The current source technically satisfies the bounded task-1.1 contract and its
offline tests are genuine and reproducible. Task 1.1 remains open until
Kiro#Opus48-TL confirms the complete evidence contract, including an
owner-approved formal requirement mapping if the standing bar requires one.
