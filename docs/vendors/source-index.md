# Vendor Source Index

> **Owner:** Gemini#Pro (stream F5)
> **Updated:** 2026-07-04T18:51Z (link-rot check pass 3)
> **Status:** DONE
> **Rule:** Every claim in the capability matrix MUST trace to a primary source listed here. Blog/Reddit/YouTube are NOT valid primary sources.

## Primary Sources by Vendor

### OpenAI / Codex

| Source | URL | Status | Checked-On (UTC) | Notes |
|---|---|---|---|---|
| Codex Config Reference | https://developers.openai.com/codex/config-reference | ✅ 200 | 2026-07-04T18:51Z | config.toml and requirements.toml reference |
| Codex Docs (main) | https://developers.openai.com/codex | ✅ 200 | 2026-07-04T18:51Z | Product overview, usage tiers, setup |
| Codex Overview | https://developers.openai.com/codex/overview | ⚠️ 301→/codex | 2026-07-04T18:51Z | **CHANGED:** redirects to main /codex page. No longer a separate page. |
| Codex Setup | https://developers.openai.com/codex/setup | ❌ 404 | 2026-07-04T18:51Z | Still 404 (unchanged from pass 1) |
| Codex Troubleshooting | https://developers.openai.com/codex/troubleshooting | ❌ 404 | 2026-07-04T18:51Z | Still 404 (unchanged from pass 2) |

### Kiro (AWS)

| Source | URL | Status | Checked-On (UTC) | Notes |
|---|---|---|---|---|
| Kiro Docs (main) | https://kiro.dev/docs/ | ✅ 200 | 2026-07-04T18:51Z | IDE docs; links to CLI, Web |
| Kiro CLI Docs | https://kiro.dev/docs/cli/ | ✅ 200 | 2026-07-04T18:51Z | CLI get-started page (SSR-deferred content) |
| Kiro CLI Authentication | https://kiro.dev/docs/cli/authentication/ | ✅ 200 | 2026-07-04T18:51Z | Builder ID, IAM Identity Center, Kiro account auth |
| Kiro CLI Installation | https://kiro.dev/docs/cli/installation/ | ✅ 200 | 2026-07-04T18:51Z | CLI install instructions (previously marked "Referenced from nav"; now directly confirmed 200) |
| Kiro CLI Context | https://kiro.dev/docs/cli/context/ | ❌ 404 | 2026-07-04T18:51Z | Still 404 — no such page exists |
| Kiro CLI Sessions | https://kiro.dev/docs/cli/sessions/ | ❌ 404 | 2026-07-04T18:51Z | Still 404 — no such page exists |
| Kiro CLI Conversations | https://kiro.dev/docs/cli/conversations/ | ❌ 404 | 2026-07-04T18:51Z | Still 404 — no such page exists |
| Kiro Pricing | https://kiro.dev/pricing/ | ✅ 200 | 2026-07-04T18:51Z | Credit tiers (Free/Pro/Pro+/ProMax/Power) |

### Google Antigravity

| Source | URL | Status | Checked-On (UTC) | Notes |
|---|---|---|---|---|
| Antigravity CLI Overview | https://antigravity.google/docs/cli-overview | ✅ 200 (SPA) | 2026-07-04T18:51Z | Official CLI documentation hub; SPA-rendered, limited static content |
| Antigravity Authentication | https://antigravity.google/docs/authentication | ✅ 200 (SPA) | 2026-07-04T18:51Z | Auth docs; SPA shell only in static HTML |
| GitHub Repo | https://github.com/google-antigravity/antigravity-cli | ✅ 200 | 2026-07-04T18:51Z | Source code, install instructions |

### Cline

| Source | URL | Status | Checked-On (UTC) | Notes |
|---|---|---|---|---|
| Cline Docs (main) | https://docs.cline.bot/ | ✅ 308→200 | 2026-07-04T18:51Z | Mintlify-hosted docs; 308 permanent redirect to canonical (normal for Mintlify) |
| Cline Provider Setup | https://docs.cline.bot/getting-started/cline-provider | ✅ 200 | 2026-07-04T18:51Z | ClinePass, BYO key, OpenRouter provider options |
| Cline Config | https://docs.cline.bot/getting-started/config | ✅ 200 | 2026-07-04T18:51Z | Configuration options |
| Cline Auto-Approve | https://docs.cline.bot/features/auto-approve | ✅ 200 | 2026-07-04T18:51Z | Auto-approve feature docs |
| Cline Context Mgmt | https://docs.cline.bot/features/context-management | ❌ 404 | 2026-07-04T18:51Z | Still 404 — no such page exists |
| Cline GitHub | https://github.com/cline/cline | ✅ 200 | 2026-07-04T18:51Z | VS Code extension source |

### OpenCode

| Source | URL | Status | Checked-On (UTC) | Notes |
|---|---|---|---|---|
| OpenCode GitHub | https://github.com/opencode-ai/opencode | ✅ 200 | 2026-07-04T18:51Z | ⚠️ ARCHIVED — project renamed to Crush |
| Crush (successor) | https://github.com/charmbracelet/crush | ✅ 200 | 2026-07-04T18:51Z | Active successor by same author + Charm |

### prodex (wrapper — not a vendor)

| Source | URL | Status | Checked-On (UTC) | Notes |
|---|---|---|---|---|
| prodex GitHub | https://github.com/christiandoxa/prodex | ✅ 200 | 2026-07-04T18:51Z | Apache-2.0; rotation, Smart Context, redeem |
| prodex npm (web) | https://www.npmjs.com/package/@christiandoxa/prodex | ⚠️ 403 (bot block) | 2026-07-04T18:51Z | npm web UI returns 403 (bot protection). **Registry API confirms package exists:** registry.npmjs.org returns `@christiandoxa/prodex@0.246.0`. Package is valid. |

## Out of Scope

| Product | Reason |
|---|---|
| Kimchi | Removed from scope per ADR-001 and master plan |
| Claude Code | Not in project scope; Opus runs via Kiro (AWS) |
| DeepSeek | Not in vendors scope per master plan |
| AWS Bedrock | Not in vendors scope per master plan (used as transport, not standalone vendor) |

## Evidence Labels

| Label | Definition |
|---|---|
| `verified` | Primary source directly confirms the claim |
| `inferred` | Derived from verified facts via logical deduction |
| `not_validated` | Requires live QA/PROD evidence; vendor does not document |
| `out_of_scope` | Not in this deploy |

## Verification Log

| Vendor | Pass 1 (UTC) | Pass 2 (UTC) | Pass 3 Link-Rot (UTC) | Agent | Method |
|---|---|---|---|---|---|
| OpenAI/Codex | 2026-07-04T18:16Z | 2026-07-04T18:32Z | 2026-07-04T18:51Z | Gemini#Pro | curl HEAD + follow redirects |
| Kiro/AWS | 2026-07-04T18:16Z | 2026-07-04T18:32Z | 2026-07-04T18:51Z | Gemini#Pro | curl HEAD |
| Antigravity/Google | 2026-07-04T18:16Z | 2026-07-04T18:32Z | 2026-07-04T18:51Z | Gemini#Pro | curl HEAD |
| Cline | 2026-07-04T18:16Z | 2026-07-04T18:32Z | 2026-07-04T18:51Z | Gemini#Pro | curl HEAD |
| OpenCode | 2026-07-04T18:16Z | 2026-07-04T18:32Z | 2026-07-04T18:51Z | Gemini#Pro | curl HEAD |
| prodex | 2026-07-04T18:16Z | 2026-07-04T18:32Z | 2026-07-04T18:51Z | Gemini#Pro | curl HEAD + registry API |

## Link-Rot Check Summary (Pass 3)

| Total URLs | ✅ Reachable | ⚠️ Changed | ❌ Still 404 |
|---|---|---|---|
| 23 | 16 | 2 | 5 |

### Changes Detected

| URL | Previous Status | Current Status | Impact on Matrix |
|---|---|---|---|
| `developers.openai.com/codex/overview` | ✅ Reachable (pass 2) | ⚠️ 301→/codex | **Low.** Content merged into main /codex page. No matrix cell relied solely on this URL. |
| `npmjs.com/package/@christiandoxa/prodex` | ✅ Referenced | ⚠️ 403 (bot block) | **None.** npm registry API confirms package at v0.246.0. Web UI blocked by bot protection. |

### No New Link Rot
All URLs that were ✅ in pass 2 remain reachable (200 or valid redirect). No verified matrix cells have lost their primary source.

## QA Consistency Check (Pass 4) — 2026-07-04T20:14Z

Cross-referenced `owner-acceptance-request.md` (8 cells) against `vendor-capability-matrix.md` (6 not_validated + 10 inferred).

### Result: CONSISTENT ✅ (1 minor discrepancy fixed)

| Check | Result |
|---|---|
| All 6 `not_validated` matrix cells present in acceptance request #1–#6 | ✅ Match |
| No extra `not_validated` cells in acceptance request | ✅ No extras |
| No missing `not_validated` cells from acceptance request | ✅ None missing |
| Borderline inferred #7 (Codex smart_context) matches matrix L43 | ✅ Match |
| Borderline inferred #8 (Antigravity quota) matches matrix L64 | ✅ Match |
| Vendor names consistent across both docs | ✅ Match |
| Capability names consistent across both docs | ✅ Match |
| Current Values consistent across both docs | ✅ Match |
| Sources in acceptance request match source-index URLs | ✅ Match |

### Discrepancy Found and Fixed

| File | Field | Was | Fixed To | Severity |
|---|---|---|---|---|
| `owner-acceptance-request.md` cell #1 | Sources: codex/overview status | ✅ (stale from pass 2) | ⚠️301→/codex (per link-rot pass 3) | Minor — no impact on sign-off decision |
