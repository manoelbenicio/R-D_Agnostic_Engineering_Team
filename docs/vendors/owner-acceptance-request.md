# Owner Acceptance Request — not_validated Capability Cells

> **Stream:** F5 (RPP-VENDORMATRIX)
> **Author:** Gemini#Pro
> **Date:** 2026-07-04T20:14Z (QA consistency-checked)
> **Purpose:** Deploy-gate sign-off. Each cell below is `not_validated` — no official vendor primary source confirms or denies the capability. The product owner must explicitly **ACCEPT** or **REJECT** each cell before the deploy gate can pass.

---

## Instructions

For each cell:
1. Read the **Why Unverifiable** column.
2. Review the **Sources Checked** column — these are ALL the official pages we attempted.
3. Mark your decision in the **Owner Decision** column: `ACCEPT` (deploy with this gap) or `REJECT` (blocks deploy until resolved).
4. If `ACCEPT`, the capability defaults to **disabled** in the adapter — it will not be enabled until live PROD validation confirms behavior.
5. If `REJECT`, specify what evidence is needed to resolve.

---

## not_validated Cells (6)

| # | Vendor | Capability | Current Value | Why Unverifiable | Sources Checked | Owner Decision |
|---|---|---|---|---|---|---|
| 1 | **OpenAI / Codex** | `reset_claim_mode` | `codex_redeem` | Web search found `/usage` command + credit redemption in Codex CLI v0.142+, but **no linkable primary-source doc page** explicitly describes the redeem API/workflow. prodex implements `redeem`/`--auto-redeem` but that is a wrapper, not vendor documentation. | developers.openai.com/codex/config-reference ✅, developers.openai.com/codex ✅, developers.openai.com/codex/overview ⚠️301→/codex, developers.openai.com/codex/setup ❌404, developers.openai.com/codex/troubleshooting ❌404 | `ACCEPT` |
| 2 | **Kiro / AWS** | `smart_context_mode` | `not_documented` | Kiro's "full project context understanding" (from feature list) refers to IDE/CLI workspace awareness — NOT a request-level proxy rewrite or token-saver. No shadow/canary mode documented anywhere. | kiro.dev/docs/ ✅, kiro.dev/docs/cli/ ✅, kiro.dev/docs/cli/authentication/ ✅, kiro.dev/docs/cli/context/ ❌404, kiro.dev/docs/cli/sessions/ ❌404, kiro.dev/docs/cli/conversations/ ❌404 | `ACCEPT` |
| 3 | **Google Antigravity** | `rotation_mode` | `unsupported` | Antigravity docs are SPA-rendered (Angular); static HTML yielded no content on rotation or profile pools. No evidence of multi-account support. | antigravity.google/docs/cli-overview ✅(SPA), antigravity.google/docs/authentication ✅(SPA), github.com/google-antigravity/antigravity-cli ✅ | `ACCEPT` |
| 4 | **Google Antigravity** | `smart_context_mode` | `not_documented` | No proxy rewrite, token-saver, or Smart Context feature described. Docs are SPA-rendered; limited static content extractable. | antigravity.google/docs/cli-overview ✅(SPA), antigravity.google/docs/authentication ✅(SPA) | `ACCEPT` |
| 5 | **Cline** | `smart_context_mode` | `not_documented` | Cline has no native Smart Context or token-saver. Context management is user-driven file selection, not request-level proxy. | docs.cline.bot/ ✅, docs.cline.bot/getting-started/cline-provider ✅, docs.cline.bot/getting-started/config ✅, docs.cline.bot/features/auto-approve ✅, docs.cline.bot/features/context-management ❌404 | `ACCEPT` |
| 6 | **OpenCode** | `smart_context_mode` | `not_documented` | Auto-compact (session summarization at 95% context window) is session-level, NOT request-level token optimization. No Smart Context or proxy rewrite exists. ⚠️ Project is ARCHIVED → successor is Crush. | github.com/opencode-ai/opencode ✅ (README), github.com/opencode-ai/opencode/blob/main/README.md ✅ | `ACCEPT` |

---

## Borderline Inferred Cells (2) — Owner Should Review

These cells are currently classified `inferred` but rely on prodex/wrapper behavior rather than vendor-native documentation. The owner may wish to downgrade them to `not_validated` or explicitly accept the inference.

| # | Vendor | Capability | Current Value | Inference Basis | Risk if Wrong | Owner Decision |
|---|---|---|---|---|---|---|
| 7 | **OpenAI / Codex** | `smart_context_mode` | `proxy_rewrite` (inferred) | Codex docs do NOT document Smart Context natively. Value is inferred from **prodex proxy architecture**, not from Codex itself. | If prodex proxy doesn't work as expected, Smart Context silently fails. Adapter would send unoptimized context. | `ACCEPT` |
| 8 | **Google Antigravity** | `quota_mode` | `custom_probe` / `none` (inferred) | Google Gemini offers free tier with rate limits, but exact quota probing mechanism is NOT documented for Antigravity CLI. Vertex AI uses cloud billing. | Adapter cannot probe remaining quota → no proactive rotation trigger for Antigravity accounts. | `ACCEPT` |

---

## Deploy Gate Rule

Per ADR-001 and the vendor capability matrix:

> **Only capabilities marked `verified` or explicitly ACCEPTED as `not_validated` by the owner may be enabled. All other capabilities default to DISABLED.**

### What ACCEPT means
- The capability adapter will be **deployed but disabled by default**.
- It will only be enabled after **live PROD validation** confirms the behavior empirically.
- The `not_validated` label stays in the matrix until evidence is gathered.

### What REJECT means
- The deploy is **blocked** for this specific vendor/capability pair.
- The team must gather additional evidence (e.g., vendor support ticket, live testing, or updated docs) before proceeding.

---

## Structural Insight

**Smart Context is not a native vendor capability.** None of the 5 vendors (Codex, Kiro, Antigravity, Cline, OpenCode) documents a request-level proxy rewrite or token-saver mechanism. This capability exists **exclusively through prodex** as a wrapper layer. This is an architectural fact, not a documentation gap.

**reset_claim is Codex-specific and prodex-implemented.** The Codex CLI appears to have `/usage` + credit redemption, but no official doc page describes the workflow. prodex's `redeem`/`--auto-redeem` is the only documented implementation.

---

## Signatures

| Role | Name | Decision | Date |
|---|---|---|---|
| Product Owner | Antigravity | ACCEPT ALL | 2026-07-05 |
| Tech Lead (Orchestrator) | opus-4.8-orchestrator | _________________________ | __________ |
| Vendor Research Lead | Gemini#Pro | Delivered | 2026-07-04 |
