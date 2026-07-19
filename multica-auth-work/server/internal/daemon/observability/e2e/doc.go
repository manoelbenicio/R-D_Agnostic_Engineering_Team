// Package e2e is the End-to-End Observability correlation library for Agent
// Brain v3 (Wave B, lane W5; OBS-1/OBS-9/OBS-10).
//
// Every task produces exactly one continuous, metadata-only trace spanning the
// eight hops documented in docs/observability/e2e-metadata-span.md. Each hop
// owner (W1/W2/W3/W4/W6/W7) CALLS this library to emit its span; no lane
// co-edits this package.
//
// Hard invariant (AB-REQ-40, PD-08): spans, labels, counters, and logs carry
// ONLY correlation identifiers, bounded classification codes, numeric counters,
// and latencies. No span may carry the OmniRoute secret, provider secrets,
// authorization headers, cookies, raw prompts, raw tool payloads, repository
// content, opaque reasoning, account emails, or connection strings. CLI argv is
// redacted structurally (shape only, never values). Every span asserts
// SecretsPresent == false and is refused (fail-closed) if it would carry
// content.
//
// The package depends only on the standard library so that it stays a stable,
// dependency-light contract for every calling lane.
package e2e
