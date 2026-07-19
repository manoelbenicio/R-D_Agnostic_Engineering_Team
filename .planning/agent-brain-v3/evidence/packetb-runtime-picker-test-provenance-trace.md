# Packet B runtime-picker test provenance trace

## Golden Rule check-in / check-out

- **CHECK-IN:** 2026-07-18T21:41:23Z — Codex56#B (Codex-root), read-only provenance trace. The first user-visible commentary check-in preceded this captured timestamp; the timestamp was captured after an initial read-only lookup used the packet's shorthand path and returned `ENOENT`. No shared ledger entry was made.
- **CHECK-OUT:** 2026-07-18T21:44:40Z — trace complete; **EXCLUDE pending traceable ownership/evidence**. Kiro TL adjudicates and root integrates; this artifact does not accept a task, allocate an EV, or authorize a push.
- **Process boundary:** only this uniquely named artifact was created. Product, test, OpenSpec, task, shared planning/index/ledger/state, and git index/worktree content were not changed. No credential or environment value was read; no DB, network, service, provider, or live process was used.

## Verdict

`multica-auth-work/packages/views/agents/components/runtime-picker.test.tsx` is mechanically part of the staged Packet B file set and is technically coupled to the staged `runtime-picker.tsx` change, but it is **not covered by the Packet B producer's direct-test manifest or a non-zero execution transcript**. Its exact source producer, owning OpenSpec task, and registered EV are not provable from the current durable records.

**Current disposition: EXCLUDE.** It must not be admitted or pushed merely because it is staged or because the related product file appears in `vendor-model-visibility-ui.md`. If retained, the product/test pair must be added to a traceable, explicitly owned evidence manifest and independently reviewed with a genuine non-zero test run on a suitable POSIX-local checkout. The present trace is not that acceptance review.

## Current object identity and dependency boundary

Checkout inspected: `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f`.

| Path | Index state | Git blob | Current/index SHA-256 | Staged delta |
|---|---:|---|---|---:|
| `multica-auth-work/packages/views/agents/components/runtime-picker.test.tsx` | added | `6e5293e6dbc05ff87c952e1240c899728fed7d03` | `55ef6cafabe5b2a90dc4b94253d367f9fecffdc8820176e88a66763be44c79c0` | `+139/-0` |
| `multica-auth-work/packages/views/agents/components/runtime-picker.tsx` | modified | `2b2a6674f88eb9eab38e3acc9705068ea6333092` | `f14fac5a30dc55154f8769939ff5a0c43283c353f6653ccd64ebf9162451cddc` | `+66/-5` |

`git show :<path> | sha256sum` reproduced both filesystem SHA-256 values, and `git diff --name-only -- <both paths>` was empty. Thus the inspected working-tree bytes equal the staged bytes for both files.

The test is absent from committed `HEAD`. Committed `HEAD` also lacks `RUNTIME_PROVIDER_LABELS`, `runtimeProviderLabel`, and `RuntimeProviderMark`. The test imports `runtimeProviderLabel` at test line 26, so it cannot be admitted independently of the staged product counterpart. This is an atomic product/test dependency, not evidence that the pair is accepted.

## Exact diff purpose

The staged product change is a provider-identity presentation correction:

- product lines 18-40 add canonical human labels for 16 supported runtime provider identifiers plus trimmed unknown/blank fallback behavior;
- product lines 43-67 add provider marks, including distinct local text marks `CL` and `NIM` rather than a generic provider-logo fallback;
- product lines 170-200 expose the selected runtime's provider label alongside owner/device identity;
- product lines 239-274 expose the same provider label and mark in each picker row.

The new test verifies that correction. It does not exercise model discovery, timeout/cache behavior, a daemon/provider, credentials, runtime execution, or the existence of a native backend.

## Assertion inventory

All fixtures are synthetic (`user-1`, `workspace-1`, generated runtime IDs and host names); provider-logo and actor-avatar components are mocked at test lines 16-24.

| Test | Lines | Inventory | Expanded semantic checks |
|---|---:|---|---:|
| accessible runtime/provider identity | 92-108 | 16 provider rows; for each: runtime text exists, enclosing button exists, provider label is visible, accessible name contains runtime plus provider label | 64 |
| distinct Cline/NIM marks | 110-133 | both rows exist; `CL` and `NIM` are visible; neither generic mocked provider logo is used | 6 |
| unknown/blank forward-compatible labels | 135-138 | trimmed unknown provider is preserved; whitespace-only provider becomes `Runtime` | 2 |
| **Total** | | 3 top-level tests, 16 provider fixtures, 12 static `expect(` sites | **72 per complete execution** |

The 16 fixture identities at lines 31-48 match the 16 product-label entries at product lines 18-35. The 72 figure expands the four assertions inside the 16-row loop; it is a static assertion inventory, **not an executed assertion count**.

## Packet B and native-task trace

### Packet B

- The producer artifact `vendor-model-visibility-ui.md` (current SHA-256 `67cdb00d8a930a9c6ca02991e8b431b22f00a5fdde2dad74712b23af175b9106`) names the product file at lines 45-46, characterizes it as a “bounded prior correction,” and explicitly says the catalog round did not change its behavior further.
- Its Direct tests section, lines 63-91, lists only `models.test.tsx`, `model-dropdown.test.tsx`, and `inspector/model-picker.test.tsx`. Its recorded commands/results at lines 118-128 likewise cover only those suites (14, 6, and 3 tests). The runtime-picker test is absent from the manifest, command list, lint list, and non-zero results.
- Kiro's independent Packet B review (SHA-256 `7a5fd48c41ae61d3b952f9fb3e8bb6fbb831b02152894e896b4d437a68c55b17`) includes the product blob in its seven producer-covered files at lines 20-22, but identifies this test as an untraced coverage gap at line 54 and requires provenance or exclusion at line 75.
- That review records successful package typechecks at lines 29-31. Those are Kiro's historical results, not Codex56#B reproductions. Its attempted Vitest command included this test at line 36 but failed during worker startup with **no test executed** (lines 34-40); it provides no runtime-picker pass count.
- The staged ownership review (SHA-256 `1a4d58ddebfb2b3728d478d30497e850a0aff269a3c4b6a8666aca14800df6be`) lists the product/test at lines 31-32 only as **PENDING** staged files. Staging and that table do not cure the producer-manifest omission.

Conclusion: the test is related to Packet B by staging and behavior, but it is outside the producer-evidenced direct-test scope.

### Native runtimes onboarding

- Native task 1.1 and `agent-runtimes/spec.md:9-15` require NIM to appear/select and execute with usage; task 1.3 and the spec at lines 17-30 require Cline availability and ACP execution. The local `NIM`/`CL` marks are compatible presentation support, but this jsdom test proves none of those backend/runtime requirements.
- Native task 1.4 (`tasks.md:9`) and `model-discovery/spec.md:5-15` concern bounded, cached model-list population/error behavior. This test contains no model-list request, timeout, cache, progress, or error assertion.
- Tasks 1.5 and 1.6 (`tasks.md:10-11`) are open onboarding/design-parity lanes, but neither text specifically assigns runtime-provider identity/accessibility. No ownership is inferred from their generic frontend location.

Conclusion: task 1.4 is the closest wording because it mentions UI population, and tasks 1.1/1.3 name NIM/Cline visibility, but none is an exact contract for this 16-provider identity/accessibility correction. This test cannot be used as acceptance evidence for any native task without an owner mapping.

## Ownership and evidence provenance

| Contract field | Finding | Grade |
|---|---|---|
| Source producer | No durable artifact inspected attributes authorship of this added test. The Packet B producer artifact calls the product behavior a prior correction and omits the test. Identity must not be inferred from staging or file timestamps. | **UNPROVEN** |
| Producer evidence | Packet B evidence covers the product descriptively but omits the test from its file/test/result manifests. | **MISSING for test** |
| Independent reviewer | Kiro independently found the omission and statically reviewed Packet B, but recorded zero execution for this jsdom test. Codex56#B authors only this provenance trace and does not self-accept. | **PARTIAL, non-accepting** |
| OpenSpec task | No exact current task owns provider-identity/accessibility behavior in this picker. | **UNPROVEN** |
| EV mapping | `EVIDENCE_INDEX.md` has no `EV-VIS`, vendor-model-visibility, or runtime-picker entry at the inspected checkout. No EV is allocated here. | **MISSING** |
| Non-zero durable execution | None found for this test. Kiro's `/mnt/c` attempt stopped before loading tests. | **MISSING** |

## Required admission path

To include rather than exclude, an owner should require all of the following as one traceable unit:

1. Assign an exact task/acceptance contract and registered EV for runtime-provider identity/accessibility.
2. Identify the truthful product/test producer with durable check-in/out provenance.
3. Manifest both current files, their exact SHA-256 values and git blobs; do not manifest the test alone.
4. Record a deterministic non-zero focused execution of all 3 tests (expected 72 expanded semantic checks) on a POSIX-local checkout where the jsdom worker can start, plus the views typecheck and diff check.
5. Obtain a reviewer distinct from the producer and Kiro adjudicator; record limitations and any changed hashes.

Until all five are satisfied, exclude both runtime-picker files from any “accepted/traceable Packet B” atomic manifest. Merely appending the test filename to the existing producer artifact would not establish its producer, owning contract, or executed result.

## Commands actually run by Codex56#B

Working directory for every command: `/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team`. All commands were read-only except creation of this artifact.

```text
sha256sum <runtime-picker test> <runtime-picker product> <three cited evidence artifacts>
git ls-files --stage -- <test> <product>
git diff --name-only -- <test> <product>
git diff --cached --numstat -- <test> <product>
git show :<test> | sha256sum
git show :<product> | sha256sum
git show HEAD:<product> | rg 'runtimeProviderLabel|RUNTIME_PROVIDER_LABELS|RuntimeProviderMark'
git cat-file -e HEAD:<test>
rg/nl read-only searches over the cited evidence, EVIDENCE_INDEX, and native OpenSpec files
node <read-only static counter over runtime-picker.test.tsx>
git diff --cached --check -- <test> <product>
```

Observed deterministic outputs:

```text
topLevelTests=3
syntacticExpectSites=12
providerRows=16
expandedSemanticAssertions=72
HEAD_RUNTIME_EXPORTS=ABSENT
HEAD_TEST=absent
DIFF_CHECK_EXIT=0
UNSTAGED_PATH_DIFF=(empty)
EVIDENCE_INDEX matches for EV-VIS/vendor-model-visibility/runtime-picker=(empty)
```

No typecheck was rerun: the current Kiro review already records an offline views typecheck pass, while this assignment is provenance-only. No Node/Vitest/jsdom test was invoked, in compliance with the explicit instruction not to retry the `/mnt/c` worker. Therefore this artifact claims static structure and provenance only, not executable UI acceptance.

## Non-claims

- No task, evidence ID, checkbox, acceptance, producer identity, or push eligibility is created here.
- No native NIM/Cline backend, model discovery, accessibility behavior at runtime, browser behavior, or provider integration is accepted.
- No claim is made that a staged file is safe to integrate merely because its bytes are internally coherent.
- Kiro TL remains adjudicator; root alone decides and performs integration.
