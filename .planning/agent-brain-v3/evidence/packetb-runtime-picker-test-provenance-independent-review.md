# Packet B runtime-picker test provenance — independent review (GLM52#B)

- Reviewer: **GLM52#B** (independent; distinct from producer/auditor `Codex56#B` at Herdr pane `w6:p2`, distinct from prior Kiro principal reviewers, distinct from the Packet B implementation producer `Codex3` at `w3:p9`).
- Review date: 2026-07-18T21:46:08Z
- Mode: **READ-ONLY**. No product/test/spec/task/shared-ledger/git-index/credential/env/network/service change. Deterministic static inspection + hash verification only; **no jsdom on `/mnt/c`** (per dispatch); no typecheck rerun (this is a provenance review and the producer trace already cites Kiro's historical offline views typecheck pass — not re-executed here).
- Subject under review: `.planning/agent-brain-v3/evidence/packetb-runtime-picker-test-provenance-trace.md` (producer/auditor trace by `Codex56#B`/`w6:p2`) and its target `multica-auth-work/packages/views/agents/components/runtime-picker.test.tsx` (+ its product counterpart `runtime-picker.tsx`).
- Kiro TL adjudicates; this review does not self-accept, does not allocate an EV, does not authorize a push, and does not edit any shared register.

## Golden Rule check-IN / check-OUT

- **CHECK-IN** 2026-07-18T21:42:23Z — GLM52#B — READ-ONLY provenance validation. Claimed: read-only static inspection + hash verification + this single artifact `packetb-runtime-picker-test-provenance-independent-review.md` only. Confirmed not pre-existing (no collision). Began inspection before the producer trace checked out; **finalized only after** the producer trace appeared and its SHA-256 was confirmed stable across two checks (see Provenance below).
- Excluded (honored): no product/test/spec/`tasks.md`/shared-ledger/`EVIDENCE_INDEX`/`STATE`/OpenSpec/git-index edit; no `git add/restore/commit/push`; no DB/network/credential/env-value/live-service access; no jsdom/Vitest run on `/mnt/c`; no typecheck rerun.
- **CHECK-OUT** 2026-07-18T21:46:08Z — DONE. Verdict: **the producer trace is independently reproduced; its EXCLUDE disposition is corroborated.** `tasks.md` 1.5/1.6 (the closest frontend lanes) remain `[ ]` (OPEN/BLOCKED); no checkbox changed. Kiro TL adjudicates.

## Provenance

- **Reviewer identity:** GLM52#B (the opencode assistant, distinct identity from `Codex56#B`/`w6:p2` who authored the producer trace, and from `Codex3`/`w3:p9` who authored the Packet B implementation). Identity basis: the dispatch explicitly routed this to GLM52#B for an independent review of the `Codex56#B`/`w6:p2` trace; GLM52#B is a different pane/identity and did not produce or adjudicate the trace under review.
- **Host:** WSL2 linux/amd64 (the opencode execution environment).
- **Toolchain:** static inspection only — `grep`, `Read`, `sha256sum`, `git ls-files`, `git show`, `git cat-file`, `git diff --cached --numstat`. No node/pnpm/Go toolchain invocation (provenance-only, no typecheck/test run).
- **Repository HEAD:** `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f` (matches the producer trace's "Checkout inspected" at its line 17).
- **Producer trace stability:** the producer trace `packetb-runtime-picker-test-provenance-trace.md` SHA-256 was measured twice and is stable:
  - `a8e08801aed3a03748c36836a319c4afdf1425f4153fb16b1b08951c60214572` (check #1)
  - `a8e08801aed3a03748c36836a319c4afdf1425f4153fb16b1b08951c60214572` (check #2)
  - File mtime `2026-07-18T18:45:04Z`; the trace records its own CHECK-OUT at `2026-07-18T21:42:48Z` (trace line 6). The trace is complete and stable; this review finalized after stability was confirmed.
- **Review window:** 2026-07-18T21:42:23Z through 2026-07-18T21:46:08Z UTC.
- **No credential, auth home, session file, token, environment secret, database, network, live provider/daemon/CLI, or multi-node state was read or used.** Only repository source/spec/evidence files were inspected.

## Independent reproduction of producer-trace claims

Every material claim in the producer trace was independently verified against the current tree. All reproduce exactly:

| Producer-trace claim (line) | Independent verification | Result |
|---|---|---|
| Test current/index SHA-256 `55ef6caf…` (L21) | `sha256sum runtime-picker.test.tsx` | `55ef6cafabe5b2a90dc4b94253d367f9fecffdc8820176e88a66763be44c79c0` ✓ |
| Test git blob `6e5293e6…` (L21) | `git ls-files -s` | `6e5293e6dbc05ff87c952e1240c899728fed7d03` ✓ |
| Product current/index SHA-256 `f14fac5a…` (L22) | `sha256sum runtime-picker.tsx` | `f14fac5a30dc55154f8769939ff5a0c43283c353f6653ccd64ebf9162451cddc` ✓ |
| Product git blob `2b2a6674…` (L22) | `git ls-files -s` | `2b2a6674f88eb9eab38e3acc9705068ea6333092` ✓ |
| Test staged delta `+139/-0` (L21) | `git diff --cached --numstat` | `139 0` ✓ |
| Product staged delta `+66/-5` (L22) | `git diff --cached --numstat` | `66 5` ✓ |
| Working-tree bytes == staged bytes (L24) | `git diff --name-only -- <both>` | empty ✓ |
| HEAD lacks `RUNTIME_PROVIDER_LABELS`/`runtimeProviderLabel`/`RuntimeProviderMark` (L26, L120) | `git show HEAD:runtime-picker.tsx \| grep -c` | `0` (absent) ✓ |
| HEAD test absent (L121) | `git cat-file -e HEAD:runtime-picker.test.tsx` | exit 128 "exists on disk, but not in HEAD" ✓ |
| Producer evidence SHA `67cdb00d…` (L56) | `sha256sum vendor-model-visibility-ui.md` | `67cdb00d8a930a9c6ca02991e8b431b22f00a5fdde2dad74712b23af175b9106` ✓ |
| Kiro independent review SHA `7a5fd48c…` (L58) | `sha256sum packetb-vendor-model-visibility-independent-review.md` | `7a5fd48c41ae61d3b952f9fb3e8bb6fbb831b02152894e896b4d437a68c55b17` ✓ |
| Staged-ownership review SHA `1a4d58dd…` (L60) | `sha256sum packetb-staged-frontend-push-ownership-review.md` | `1a4d58ddebfb2b3728d478d30497e850a0aff269a3c4b6a8666aca14800df6be` ✓ |
| EVIDENCE_INDEX has no EV-VIS/vendor-model/runtime-picker entry (L80, L124) | `grep -n` over EVIDENCE_INDEX.md | empty ✓ |
| 3 top-level tests (L48, L116) | `grep -c '^  it('` | `3` ✓ |
| 12 static `expect(` sites (L48, L117) | `grep -c 'expect('` | `12` ✓ (note: `expect.stringMatch(` is `expect.` not `expect(`, so it does not increment the count — the loop has 4 assertion sites, each running 16×) |
| 16 provider fixtures (L48, L118) | `grep -c '^\s*\["'` | `16` ✓ |
| 72 expanded semantic assertions (L48, L119) | manual: 4×16 (loop) + 6 (test 2) + 2 (test 3) = 72 | ✓ (static inventory, **not** an executed count — the producer trace explicitly states this at L50) |

**Conclusion:** the producer trace is internally consistent and every verifiable claim reproduces against the current tree. No factual discrepancy found.

## Independent findings (corroborating + additive)

### A. Task/EV ownership — independently confirmed UNPROVEN / MISSING

- **AGENT_LEDGER Packet B rows (71, 92)** explicitly list the Packet B file set as `packages/core/runtimes/models.ts, packages/views/agents/components/{inspector/model-picker,model-dropdown}.tsx (+ *.test.tsx)`. The named product files are **`model-picker` and `model-dropdown` — `runtime-picker` is NOT explicitly named**; the `(+ *.test.tsx)` is generic and does not specifically attribute `runtime-picker.test.tsx`. The ledger references `EV-VIS (packet B)` but states "Formal Packet-B acceptance trails EV-G4-03" — i.e., **not yet accepted**.
- **FILE_OWNERSHIP.md** lists `model-picker.tsx` and `model-dropdown.tsx` as Codex3-owned (lines 60-61) but **`runtime-picker.tsx` is NOT in the FILE_OWNERSHIP table**. So `runtime-picker.tsx` has no recorded single-owner lock.
- **EVIDENCE_INDEX.md** has **no `EV-VIS`, `vendor-model-visibility`, or `runtime-picker` entry** — confirmed by grep. So `EV-VIS` as referenced in the ledger is a proposed/working name, not a registered evidence ID.
- **native-runtimes-onboarding tasks:** 1.4 (`[x]`) is backend model discovery (timeout/cache/error, `pkg/agent` + `internal/daemon`); 1.5 (`[ ]`, BLOCKED) is frontend onboarding — marketing/landing/sponsors/email-code removal + `AuthService`/login UI, **not the picker**; 1.6 (`[ ]`, BLOCKED) is design parity/i18n/web-build/QA, **not provider-identity/accessibility**. None is an exact contract for the 16-provider identity/accessibility correction the test verifies. This matches the producer trace's conclusion at L70.

**Independent verdict on ownership:** no registered EV, no exact owning OpenSpec task, no FILE_OWNERSHIP lock for `runtime-picker.tsx`. The test is **outside the traceable Packet B producer-evidenced direct-test scope** and outside any registered EV. Corroborates the producer trace's "UNPROVEN / MISSING" grades (L76-81).

### B. Producer/reviewer identity — independently confirmed UNPROVEN for the test

- The Packet B implementation producer is `Codex3` at `w3:p9` (AGENT_LEDGER rows 71, 92), but the ledger's file list names only `model-picker`/`model-dropdown`, not `runtime-picker`. The producer evidence (`vendor-model-visibility-ui.md` L45-46) describes `runtime-picker.tsx` as a "bounded prior correction" and says "This catalog round did not change its behavior further" — yet the staged delta is `+66/-5` (the product WAS modified). This apparent contradiction is resolved by reading the producer evidence carefully: the "bounded prior correction" is a **prior** round's change now staged, and "this catalog round did not change its behavior further" means the current Packet B round added no *further* behavior change on top. Either way, the **test** (`runtime-picker.test.tsx`, `+139/-0`, absent from HEAD) is not listed in the producer evidence's "Direct tests" (L65-91) nor in its recorded commands/results (L118-128). So no durable artifact attributes authorship of the added test.
- The producer/auditor of the **trace** under review is `Codex56#B` at `w6:p2` (trace L5). The reviewer of this independent validation is `GLM52#B` (distinct). The Packet B implementation producer is `Codex3`/`w3:p9` (distinct). The prior Kiro principal reviewers are distinct. **Independence chain: producer (Codex3) ≠ trace author (Codex56#B) ≠ this reviewer (GLM52#B) ≠ adjudicator (Kiro TL).**

**Independent verdict on identity:** the test's source producer is UNPROVEN (no durable artifact attributes it); the trace author and this reviewer are distinct from the implementation producer and from each other. Corroborates the producer trace's "UNPROVEN" grade (L76).

### C. Current hash — independently confirmed

Test: git blob `6e5293e6dbc05ff87c952e1240c899728fed7d03`, filesystem SHA-256 `55ef6cafabe5b2a90dc4b94253d367f9fecffdc8820176e88a66763be44c79c0`, staged `+139/-0`, absent from HEAD.
Product: git blob `2b2a6674f88eb9eab38e3acc9705068ea6333092`, filesystem SHA-256 `f14fac5a30dc55154f8769939ff5a0c43283c353f6653ccd64ebf9162451cddc`, staged `+66/-5`, tracked at HEAD `aa62401` but with the `runtimeProviderLabel`/`RUNTIME_PROVIDER_LABELS`/`RuntimeProviderMark` exports **absent** at HEAD (0 matches).
Working-tree bytes == staged bytes for both files (`git diff --name-only` empty).

### D. Assertion inventory — independently confirmed (static, NOT executed)

3 top-level tests under `describe("RuntimePicker provider identity")`:
1. **"renders accessible runtime and provider identity for all supported runtime types"** (L92-108) — iterates 16 PROVIDERS; per iteration: runtime text exists, enclosing button exists, provider label visible, accessible name matches `Runtime <provider>.*<label>`. 4 assertion sites × 16 = 64 expanded.
2. **"uses distinct local marks for cline and nim instead of the generic provider-logo fallback"** (L110-133) — `cline` shows `CL`, `nim` shows `NIM`; neither uses the mocked `provider-logo-cline`/`provider-logo-nim` testid. 6 assertion sites.
3. **"keeps unknown future runtime providers identifiable"** (L135-138) — `runtimeProviderLabel(" future-provider ")` → `"future-provider"`; `runtimeProviderLabel("   ")` → `"Runtime"`. 2 assertion sites.

Total: **3 tests, 12 static `expect(` sites, 16 provider fixtures, 72 expanded semantic checks** (static inventory). All fixtures synthetic (`user-1`, `workspace-1`, generated runtime IDs/hostnames); `ProviderLogo` and `ActorAvatar` mocked (L16-24). **This is a static assertion inventory, NOT an executed count** — no jsdom/Vitest run was performed on `/mnt/c` (per dispatch), and the producer trace explicitly disclaims execution (L127). The 72 figure is the expected count if the suite were to run on a POSIX-local checkout where the jsdom worker can start.

### E. Relationship to runtime-picker.tsx — independently confirmed atomic product/test dependency

The test imports `RuntimePicker` and `runtimeProviderLabel` from `./runtime-picker` (test L26). These symbols are:
- `runtimeProviderLabel` (product L38-41): trims/lowercases, returns `RUNTIME_PROVIDER_LABELS[normalized] ?? (provider.trim() || "Runtime")`. Tested by test 3.
- `RuntimeProviderMark` (product L43-67): for `cline`/`nim` renders distinct `CL`/`NIM` text marks; otherwise renders `ProviderLogo`. Tested by test 2.
- `RuntimePicker` (product L69-291): renders the picker with provider labels/marks per row. Tested by test 1.

All three exports are **ABSENT from HEAD** (`git show HEAD:runtime-picker.tsx | grep -c` = 0). The test therefore **cannot be admitted independently of the staged product change** — it is an atomic product/test dependency (corroborates producer trace L26). The 16 provider labels in the test (L31-48) exactly match the 16 entries in `RUNTIME_PROVIDER_LABELS` (product L18-35). The test does NOT exercise model discovery, timeout/cache, daemon/provider, credentials, runtime execution, or any native backend — it is purely a provider-identity/accessibility presentation test (corroborates producer trace L37).

### F. Whether it must be excluded or added to an authorized manifest — independently confirmed EXCLUDE

The producer trace's verdict is **EXCLUDE pending traceable ownership/evidence** (L6, L13). My independent findings corroborate:
1. No registered EV (`EVIDENCE_INDEX.md` has no `EV-VIS`/vendor-model/runtime-picker entry).
2. No exact owning OpenSpec task (1.4 backend; 1.5/1.6 BLOCKED and about onboarding/QA, not picker identity).
3. No FILE_OWNERSHIP lock for `runtime-picker.tsx`.
4. The AGENT_LEDGER Packet B rows name `model-picker`/`model-dropdown`, not `runtime-picker`.
5. The producer evidence omits the test from its "Direct tests" manifest and recorded results.
6. No non-zero durable execution (the `/mnt/c` jsdom worker-start timeout blocks execution here and blocked Kiro's prior attempt; the producer trace did not execute it either — L127, L59).
7. The test's source producer is UNPROVEN (no durable artifact attributes authorship).

**Independent verdict: EXCLUDE.** The test must not be admitted to an "accepted/traceable Packet B" atomic manifest merely because it is staged or because the related product file appears in `vendor-model-visibility-ui.md`. To include rather than exclude, the 5-step admission path the producer trace lists (L85-91) is necessary and sufficient:
1. Assign an exact task/acceptance contract + registered EV for runtime-provider identity/accessibility.
2. Identify the truthful product/test producer with durable check-in/out provenance.
3. Manifest both current files with exact SHA-256 + git blobs; do not manifest the test alone.
4. Record a deterministic non-zero focused execution of all 3 tests (expected 72 expanded semantic checks) on a POSIX-local checkout where the jsdom worker can start, plus the views typecheck and diff check.
5. Obtain a reviewer distinct from the producer and the Kiro adjudicator; record limitations and any changed hashes.

Merely appending the test filename to the existing producer evidence would not establish its producer, owning contract, or executed result.

## Verdict (advisory; Kiro TL adjudicates)

- **Producer trace integrity:** PASS — every material claim reproduces against the current tree; the trace is internally consistent, honestly scoped, and its SHA-256 is stable (checked out at 21:42:48Z, hash `a8e08801…`).
- **Runtime-picker.test.tsx disposition:** **EXCLUDE** — independently corroborated. Not covered by the Packet B producer's direct-test manifest; no registered EV; no exact owning OpenSpec task; no FILE_OWNERSHIP lock; source producer UNPROVEN; non-zero durable execution MISSING. It is an atomic product/test dependency with `runtime-picker.tsx` (the tested exports are absent from HEAD), but that dependency does not by itself authorize admission.
- **Independence chain:** producer (Codex3/w3:p9) ≠ trace author (Codex56#B/w6:p2) ≠ this reviewer (GLM52#B) ≠ adjudicator (Kiro TL). Reviewer≠producer≠adjudicator preserved.
- **No acceptance, no EV, no checkbox, no push authorization** is created by this review. Kiro TL adjudicates; root alone decides and performs integration.

## Source SHA-256 manifest (read-only; files inspected this review)

| SHA-256 | Source |
|---|---|
| `a8e08801aed3a03748c36836a319c4afdf1425f4153fb16b1b08951c60214572` | `.planning/agent-brain-v3/evidence/packetb-runtime-picker-test-provenance-trace.md` (producer trace; stable across two checks) |
| `55ef6cafabe5b2a90dc4b94253d367f9fecffdc8820176e88a66763be44c79c0` | `multica-auth-work/packages/views/agents/components/runtime-picker.test.tsx` (filesystem; git blob `6e5293e6…`; staged `+139/-0`; absent from HEAD) |
| `f14fac5a30dc55154f8769939ff5a0c43283c353f6653ccd64ebf9162451cddc` | `multica-auth-work/packages/views/agents/components/runtime-picker.tsx` (filesystem; git blob `2b2a6674…`; staged `+66/-5`; tracked at HEAD `aa62401` but exports absent at HEAD) |
| `67cdb00d8a930a9c6ca02991e8b431b22f00a5fdde2dad74712b23af175b9106` | `.planning/agent-brain-v3/evidence/vendor-model-visibility-ui.md` (Packet B producer evidence) |
| `7a5fd48c41ae61d3b952f9fb3e8bb6fbb831b02152894e896b4d437a68c55b17` | `.planning/agent-brain-v3/evidence/packetb-vendor-model-visibility-independent-review.md` (Kiro independent Packet B review) |
| `1a4d58ddebfb2b3728d478d30497e850a0aff269a3c4b6a8666aca14800df6be` | `.planning/agent-brain-v3/evidence/packetb-staged-frontend-push-ownership-review.md` (Kiro staged-ownership review) |

Repository HEAD: `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f` (matches producer trace L17).

## Explicit non-claims

- This is a **read-only provenance validation**, not a verification lane. No jsdom/Vitest run on `/mnt/c` (per dispatch); no typecheck rerun (provenance-only; the producer trace already cites Kiro's historical views typecheck pass, not reproduced here).
- No claim that the test's 72 expanded semantic assertions were **executed** — the figure is a static inventory; no non-zero durable execution exists for this test on `/mnt/c` (the jsdom worker-start timeout blocks it; the producer trace and Kiro's prior attempt both record zero execution).
- No claim that `runtime-picker.tsx`'s `+66/-5` change is or is not from the "bounded prior correction" vs the current Packet B round — the producer evidence's framing ("prior correction … did not change its behavior further") is taken at face value; the staged delta is a fact, its round-attribution is a producer claim not re-adjudicated here.
- No claim that EXCLUDE is the **final** disposition — Kiro TL adjudicates; an owner may satisfy the 5-step admission path and reclose.
- No claim of live WebSocket delivery, live vendor behavior, PostgreSQL execution, production deployment, cutover, or tier acceptance.
- No edits to: OpenSpec (`tasks.md`/`proposal.md`/`design.md`/`specs/`), `STATE.md`, `AGENT_LEDGER.md`, `EVIDENCE_INDEX.md`, `FILE_OWNERSHIP.md`, any product/test file, any checkbox, the git index. `tasks.md` 1.5/1.6 confirmed `[ ]` (OPEN) before and after.
- No credential, auth home, session file, token, environment secret, database, network, live provider/daemon/CLI, or multi-node state was read or used.
