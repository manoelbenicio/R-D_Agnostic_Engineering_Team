# C3 — Chat Core API: session-create optional agent_id + direct-target (verification)

- agent: `Codex56#A` · lane: `C3` · pane: `w7:p3` · task: `C3-CHAT-CORE-API`
- lock: `multica-auth-work/packages/core/chat/mutations.ts` (would-be edit target) + this evidence file
- scope: Main Brain + chat-orchestration-standard. OmniRoute internals not touched. native-runtimes-onboarding excluded.
- MODE: **VERIFICATION-ONLY** (`implementation_authorized=false`). No product edits.

> STATUS: **BLOCKED** — a real, spec-backed gap exists (session-create requires `agent_id`; the
> untargeted/optional case is unsupported), but a **contract-correct fix cannot be isolated to
> `packages/core/chat/**`**: it also requires out-of-lock `packages/core/api/client.ts` +
> `packages/core/types/chat.ts` + the Go backend handler, and product edits are unauthorized.

---

## 0. Preflight / environment note

- HEAD `a6d50986…`; git_status_count 383; node `v22.23.1`.
- **Disk was 100% full** — the initial `p0_control check-in` failed with `No space left on device`.
  Reclaimed my own regenerable GOCACHE (`go clean -cache`, 116M under `slot-110/.cache/go-build`,
  created by my earlier verification runs) → 478M free. Non-destructive, not repo/git, not shared
  infra. Check-in then succeeded. Focused TS tests/typecheck NOT run (see §4).

## 1. Authoritative contract (spec-backed)

`openspec/changes/chat-orchestration-standard/specs/chat-orchestration/spec.md`:
- "The system SHALL, by default, route an incoming chat or task **without an explicit target** to a
  TL/Manager (squad leader)…" → an **untargeted** chat is valid ⇒ `agent_id` must be **optional** at
  session/task creation.
- "Direct-to-agent escape hatch preserved … a user SHALL address a specific agent (e.g. `@codex`) …
  routed directly to that agent" → when a target IS given, route direct.

So the required contract = **optional `agent_id`** (absent ⇒ default TL routing) **+ direct target**
(present ⇒ bypass TL).

## 2. Current state (verified by reading)

| Location | Current | In my lock? |
|---|---|---|
| `packages/core/chat/mutations.ts:15` `useCreateChatSession` mutationFn | `(data: { agent_id: string; title?: string })` — **required** | YES |
| `packages/core/chat/mutations.ts:17` | `return api.createChatSession(data)` | YES |
| `packages/core/api/client.ts:1709` `createChatSession` | `(data: { agent_id: string; title?: string })` — **required** | **NO (out of lock)** |
| `packages/core/types/chat.ts` `ChatSession.agent_id` | `agent_id: string` — **required** | **NO (out of lock)** |
| `packages/core/chat/store.ts` `selectedAgentId` / `newSessionDraftKey(id: string \| null)` | already **nullable** (client already models "no agent") | YES |

The client store already tolerates a null selected agent, but the **create-session request type**
still mandates `agent_id`. So the untargeted/default-TL case cannot be expressed at create time.

## 3. Why the fix is NOT isolatable to `packages/core/chat/**`

`useCreateChatSession.mutationFn` calls `api.createChatSession(data)`. Changing the mutation param to
`{ agent_id?: string; title?: string }` while `api.createChatSession` still requires
`{ agent_id: string; … }` is a TypeScript compile error (TS2345: an optional-`agent_id` object is not
assignable to a required-`agent_id` parameter). A contract-correct change therefore requires,
together:
1. `packages/core/chat/mutations.ts` — param `agent_id?: string` **(in lock)**;
2. `packages/core/api/client.ts:1709` — `createChatSession` accepts optional `agent_id` **(out of lock)**;
3. `packages/core/types/chat.ts` — `ChatSession.agent_id` optional/nullable, or a target discriminator **(out of lock)**;
4. Go backend chat-session create handler — accept missing `agent_id` and apply default squad-TL routing **(out of lock; server)**.

Making only (1) would break the build; fabricating a default `agent_id` client-side is forbidden
("never hard-code") and is a server routing responsibility per the spec.

## 4. Validation performed / not performed

- Static verification: read `chat/{mutations,index,store,queries}.ts`, `api/client.ts:1709`,
  `types/chat.ts`, and the spec. Divergence is definitive from the type signatures.
- Focused TS tests / `tsc` typecheck: **NOT run** — (a) the change is cross-package and cannot be
  isolated to a meaningful in-lock test, (b) root disk critical / no dependency-install, (c) no
  in-lock edit made. `gofmt`/`go vet` are N/A (TypeScript package). `git diff --check` on lock:
  CLEAN (no edits).

## 5. Blocker / routing

- **Blocker:** the optional-`agent_id` + direct-target session-create contract is spec-required but
  cannot be delivered within the `packages/core/chat/**` lock alone; it needs coordinated out-of-lock
  edits (`packages/core/api/client.ts`, `packages/core/types/chat.ts`, Go chat-session create
  handler), and product edits are gated (`implementation_authorized=false`).
- **Owners:** owner of `packages/core/api` + `packages/core/types` (frontend core API/types lane) and
  the server chat-handler owner; authorization from Principal Orchestrator (`w5:p9`).
- **Next action:** (a) Principal authorizes the change + assigns the out-of-lock api/types/server
  files to a single owner (or grants C3 an expanded lock); then (b) apply the 4 coordinated edits in
  §3 as one contract-correct unit and (c) add a focused mutation test asserting create succeeds with
  and without `agent_id`. Until then C3 is BLOCKED on scope + authorization, not on analysis.

## 6. Non-claims

- No product source edited; no `agent_id` default fabricated; no OmniRoute internals probed; no
  inference/secret/deploy; no OpenSpec checkbox closed.
- Prior assignments (F3, P0-GLM-LIVE-ACCEPTANCE) remain BLOCKED; no lock overlap with `chat/**`.
- The reclaimed GOCACHE was my own regenerable build cache, not repo/git/shared state.

---

## 7. IMPLEMENTATION (lock expansion authorized by Principal; zero-overlap confirmed)

Authoritative contract: `FULL_MAIN_BRAIN_CHAT_PROMPTS.md#C3` — "Change the session-create contract so
`agent_id` is optional for untargeted chat while preserving explicit agent direct routing … Typecheck
and focused mutation/client tests cover body with no `agent_id` and body with explicit `agent_id`. No
API compatibility regression."

Expanded lock (Principal-granted): `packages/core/chat/**` + `packages/core/api/client.ts` +
`packages/core/types/chat.ts` (+ matching tests).

### 7.1 Edits (minimal, backward-compatible: required → optional)

| File | Change |
|---|---|
| `packages/core/types/chat.ts` | Added `export interface CreateChatSessionRequest { agent_id?: string; title?: string }` (single source of truth; `agent_id` OPTIONAL). |
| `packages/core/api/client.ts` | Imported `CreateChatSessionRequest`; `createChatSession(data: CreateChatSessionRequest)` (was `{ agent_id: string; title?: string }`). Body still `JSON.stringify(data)` → omits `agent_id` when absent. |
| `packages/core/chat/mutations.ts` | Imported `CreateChatSessionRequest`; `useCreateChatSession` mutationFn param now `CreateChatSessionRequest`. |
| `packages/core/api/client.test.ts` | +2 focused tests: explicit `agent_id` present in POST body; untargeted omits `agent_id` (`body).not.toContain("agent_id")`). |
| `packages/core/chat/mutations.test.tsx` (NEW) | Mirrors `issues/mutations.test.tsx` (`setApiInstance` + `renderHook` + `QueryClientProvider`, `../hooks` mocked). Asserts the mutation forwards `{agent_id,title}` and forwards `{title}` with `agent_id` undefined. |

### 7.2 Correctness / compatibility (static, by inspection)

- Required→optional is backward compatible: existing callers passing `agent_id` still typecheck; the new capability is omitting it.
- Only core `createChatSession` consumer is `chat/mutations.ts` (updated). The mobile app (`apps/mobile/data/api.ts`, `apps/mobile/data/mutations/chat.ts`) has its **own independent** client — out of lock, unaffected.
- `setApiInstance` confirmed exported at `packages/core/api/index.ts:19` (mutation test dependency valid).
- `git diff --check` on all locked files: **CLEAN**.
- Server default-TL routing for a missing `agent_id` is a backend responsibility (chat-orchestration spec) and out of this lock; no client-side default fabricated.

### 7.3 Validation status — BLOCKED on toolchain (not on analysis or code)

- `tsc --noEmit` (typecheck) and `vitest run` (focused tests): **NOT executed** — `node_modules` is
  ABSENT (root + `packages/core`), `vitest`/`tsc` binaries unavailable, dependency-install is
  prohibited, and root disk is critical (~453M). Per contract this is REVIEWED, not VERIFIED.
- **Route:** run `pnpm --filter @multica/core typecheck` and `pnpm --filter @multica/core test`
  (or vitest on `packages/core/api/client.test.ts` + `packages/core/chat/mutations.test.tsx`) in an
  environment/lane with core JS deps installed (e.g. F2 final matrix / CI) using `/tmp` caches.
- Owner of execution: F2 runner / CI environment with installed deps. Implementation itself is complete.

---

## 8. Definitive test-execution re-verification (per manager "finish for 1.3")

Re-probed the JS toolchain thoroughly (no install):

```
pnpm -> /home/ec2-user/.nvm/.../bin/pnpm      (present)
npm  -> present   npx -> present   node -> v22.23.1 (present)
vitest -> absent   tsc -> absent
find multica-auth-work -maxdepth 3 -name node_modules -type d -> (none)
find ... -path '*/.bin/vitest' -> (none)
pnpm store path / ~/.local/share/pnpm / ~/.pnpm-store -> (none)
disk / -> 356M free (99% used)
```

**Conclusion (unchanged, now definitive):** there are **zero installed JS dependencies** in the repo.
`vitest`/`tsc` cannot run. The only way to execute the focused core tests is `pnpm install`, which is
a **prohibited dependency-install** (and would exceed the critical disk). Therefore the acceptance
tests **cannot be executed in this pane**, and a DONE that claims a test PASS would be fabricated
(forbidden by the evidence contract). This is a concrete environment blocker, not an analysis gap.

**Implementation remains COMPLETE** (§7): edits + focused tests are written, minimal, backward-compatible,
`git diff --check` CLEAN, and type-correct/backward-compatible by inspection. Only execution is blocked.

**Unblock (any ONE):**
1. Run the focused tests in an environment/lane where core JS deps are already installed (F2 final
   matrix / CI): `pnpm --filter @multica/core typecheck` and `vitest run packages/core/api/client.test.ts
   packages/core/chat/mutations.test.tsx` with `/tmp` caches; then the Principal closes Chat 1.3 on that
   PASS evidence; OR
2. Principal grants an explicit scoped `pnpm install` exception in a suitable (non-disk-critical)
   environment so this lane can execute the two focused tests.

Owner: F2 runner / CI env + Principal Orchestrator (`w5:p9`).
