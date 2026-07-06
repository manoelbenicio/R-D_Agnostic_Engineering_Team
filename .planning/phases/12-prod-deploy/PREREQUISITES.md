# P12 PREREQUISITES — what unblocks task 12.3 (owner-supplied)

author: Kiro/Principal (Opus 4.8)
blocks: 12.3 REAL provider-backed session (and therefore 12.4/12.5/12.7 in PROD)

## The fleet CANNOT self-provide these. Owner decision required.

### 1. Real provider credentials (per vendor to prove)
| Vendor | Backing provider | What's needed | Have? |
|:--|:--|:--|:--|
| OpenAI/Codex | OpenAI API | real OPENAI_API_KEY (not `sidecar-local-probe`) | ❌ placeholder only |
| Antigravity | Gemini/Anthropic (per agent) | real provider key | ❌ |
| OpenCode/GLM5.2 | GLM/Zhipu | real key | ❌ (never measured) |
| Kiro/Opus4.8 | Anthropic | real key | ❌ |

> Provide only the keys for the vendors you want proven now. One real vendor is enough to prove the
> real round-trip path; all four gives full parity.

### 2. Real PROD host / endpoint
- A real host (not 127.0.0.1) where the pinned prodex-sidecar + gateway run, reachable for the session.
- Or explicit confirmation of the target PROD endpoint + how to deploy to it.

### 3. Pinned binary confirmation
- Confirm the prod binary/commit to deploy (PLAN references 0.246.0 / real commit) — NOT a smoke build.

## Once supplied, the fleet executes (no further owner input needed):
12.1 bring up stack → 12.2 deploy pinned binary → 12.3 REAL session per vendor (distinct numbers,
gateway 200, gateway_usage) → 12.4 kill-switch LIVE → 12.5 rollback LIVE → 12.6 scrub → 12.7 gate+push.
All bound by EVIDENCE_CONTRACT.md.

## If real providers/host are genuinely unavailable
Then a "real live PROD session" is NOT achievable and must NOT be faked. The honest milestone outcome
is: v2.1 delivered up to real-local proof (P11) with P12 explicitly deferred pending real
infrastructure — recorded as such, owner-signed. This is the only non-fabricated alternative.
