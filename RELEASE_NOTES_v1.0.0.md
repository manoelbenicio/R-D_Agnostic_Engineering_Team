# AgentVerse v1.0.0 — first GA release

**Release date:** 2026-05-29

AgentVerse v1 is now generally available. The complete master specification
v4.2 is implemented end-to-end, plus six post-v4.2 tech-debt remediations.

---

## What you can do with v1

### Build agent canvases visually
Drag agent blocks onto a canvas, connect them with `handoff`, `assign`, or
`send_message` edges, configure `system_prompt` in the embedded Monaco editor,
and click **Deploy**. Behind the scenes AgentVerse generates the CAO profiles,
spins up a tmux session, and provisions one terminal per node — all visible in
real time.

### Author canvases by voice
Press `Ctrl+Shift+V`, describe the team you want in either Portuguese or
English, and AgentVerse runs your speech through Web Speech API → your BYOK LLM
for NLU → an auto-laid-out canvas — with an intent-preview confirm step
before anything is built.

### Run runtime commands by voice
Once a canvas is deployed, voice commands like `kill terminal 2`, `pause the
reviewer`, `focus on the supervisor`, `status`, `cost`, and `stop everything`
go through a regex+keyword matcher with bilingual coverage and ≤100 ms match
latency.

### Watch every agent stream live
The terminal grid renders 12+ live PTY streams concurrently over WebSocket
binary frames at the master-spec §6.3 zero-lag profile. Tab view, focused
view, full-screen view, and a chat-bubble alternative are all available; the
default flips to chat view on viewports ≤768 px.

### Bring your own keys for 8 providers
OpenAI, Anthropic, Google, AWS (Q + Kiro), Azure, Moonshot, GitHub Copilot CLI,
and OpenCode CLI. Every key is validated live against the real provider
endpoint before it is accepted; nightly `KEYSTORE_LIVE=1` contract tests keep
those validators honest.

### See cost in real time, with the right caveats
A wall-clock × `PROVIDER_COST_PER_HOUR` estimator surfaces MTD spend, budget
utilization, cost-by-provider, and top-10 cost-by-canvas. Every cost surface
carries a mandatory ⚠️ "rough estimate" label; per-token billing arrives in
Tier 2.

### Edit a deployed canvas without tearing it down
A diff-based reconciler compares your edited canvas against the deployed
state, applies only the delta (add node / remove node / change profile / edge
update), and blocks entry-point changes behind an explicit Tear Down dialog.
Per-node profile snapshots are captured at deploy time so the diff is exact.

---

## Quality bar

| Gate | Result |
|---|---|
| `npm run lint` | 0 errors |
| `npm run typecheck` | clean |
| `npm test` | 50 files · **367 passed** · 8 skipped (`KEYSTORE_LIVE`-gated) |
| `npm run test:smoke` | green against MSW-mocked CAO |
| `src/voice/` coverage | **87.80 %** statements (target ≥ 70 %) |
| Bundle budget | within 1.5 MB gzipped |

Nightly CI runs the live `CAO_LIVE=1` and `KEYSTORE_LIVE=1` contract suites and
opens a labeled GitHub issue on shape drift.

---

## Tech-debt remediations bundled in this release

Six post-v4.2 issues were closed before tag:

1. **Keystore validator contract coverage** — 8 live contract tests + nightly
   workflow + helper docs. MSW unit tests remain the deterministic CI gate.
2. **`SCHEMA_VERSION` lives in `src/shared/`** — fixes a design-D9 violation
   accepted during the parallel build. A backwards-compatible re-export shim
   stays at the old path.
3. **React-refresh dev-console warnings cleaned up** — barrel files collapsed
   into `src/canvas-builder/index.ts`; `CostWarning` converted to a function
   declaration.
4. **Voice module coverage 40.78 % → 87.80 %** — extracted
   `src/voice/command-executor.ts` as a pure async function with deps
   injection; `VoicePanel.tsx` shrunk 803 → 586 lines (-25 %).
5. **`CanvasCommandBus` in place of cross-capability imports** — interface in
   `src/shared/`, adapter in `src/shell/`, all three
   `eslint-disable agentverse/no-sideways-capability-imports` directives
   removed from voice modules.
6. **Smoke test exercises the real voice pipeline** — animation-disable
   stylesheet replaces `{ force: true }`; a `SpeechRecognition` polyfill
   replaces the direct `useVoiceStore.setState()` workaround. The smoke now
   drives STT → NLU → command-executor end-to-end in headless Chromium.

---

## Out of scope (already filed as follow-up changes)

- `validation-proxy` — server-side edge enforcement middleware.
- `cloud-runtime-deployment` — Cloud Run / GKE / user-hosted CAO economics.
- `finops-tier2-token-parsing` — per-token cost attribution.
- Autonomous Copilot (persistent autonomous meta-agent).

---

## Upgrade path

This is the first release; there is no upgrade path. Fresh installs use
`npm ci` against the committed `package-lock.json`. Node ≥ 20.10 / npm ≥ 10.2
required.

```bash
npm ci
cp .env.example .env.local        # configure VITE_CAO_BASE_URL if needed
npm run dev                       # http://localhost:5173
```

A reachable CAO server is required (default `http://127.0.0.1:9889`). See
`docs/cao-cors.md` for the CAO environment variables that allow this SPA.

---

## Acknowledgements

AgentVerse v1 was built by a multi-agent squad (1 Supervisor + Canvas Dev +
Terminal Dev + Dashboard Dev + Studio Dev + Voice Dev + Infra Dev) per master
spec §14, all working in parallel from day one against a single shared branch.

The v4.2 master specification, tech-debt remediations, and this release are
authored under the OpenSpec workflow — see `openspec/changes/` for the full
artifact set.
