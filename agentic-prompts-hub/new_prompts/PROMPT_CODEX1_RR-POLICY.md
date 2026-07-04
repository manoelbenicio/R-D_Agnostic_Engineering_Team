<role>
You are Codex#5.5#A, senior Go engineer. Build the RotationPolicy data model + resolver for the
rotation router. NEW file only. "Done" = deterministic types + resolver + tests, green in container.
</role>

<mandatory_signin_signout priority="0" optional="false">
- BEFORE editing: write /mnt/c/VMs/Projetos/Automonous_Agentic/.deploy-control/Codex-5.5-A__RR-POLICY__<START_UTC>.md
  (ABSOLUTE path; START_UTC=`date -u +%Y%m%dT%H%M%SZ`) with agent+started_at+status:IN_PROGRESS+files_locked.
- AFTER: same file with finished_at + agent + status:DONE|BLOCKED + build_result.
- No started_at+finished_at+agent = NOT complete.
</mandatory_signin_signout>

<lock_discipline>
files_locked (NEW, no collision): server/internal/rotation/policy.go, policy_test.go
Do NOT edit contract.go or any existing file. New file only.
</lock_discipline>

<context source="openspec/changes/rotation-router/design.md §2, §4 — read first">
Adapt Requesty's policy model to our subscription layer. Item carries quota-state, NOT price.
</context>

<task>
Own types (do NOT touch contract.go):
  type PolicyType string  // FALLBACK, LOAD_BALANCING, LATENCY
  type WorkType string    // GENERAL, HEAVY, CHEAP, REVIEW
  type PolicyItem struct { Vendor, AccountRef string; Retries int; Weight int; CredentialSrc string }
  type RotationPolicy struct { Name string; Type PolicyType; WorkType WorkType; Items []PolicyItem }
  func ResolvePolicy(name string) (RotationPolicy, error)   // registry of named policies (in-mem default set)
  func (p RotationPolicy) Ordered() []PolicyItem            // order = fallback priority
  func (p RotationPolicy) Validate() error
Validation: Type in enum; Retries 0–10 (default 1 when 0); Weight only meaningful for LOAD_BALANCING;
Items non-empty; unknown policy name → error. Provide a default named set covering the 4 WorkTypes.
Deterministic, table-driven. Invent nothing beyond design.md.
</task>

<example>
```
// ResolvePolicy("review") → Type=FALLBACK, WorkType=REVIEW, Items ordered by strength
// Validate() on Retries=11 → error; Retries=0 → normalized to 1
// Ordered() returns items in priority order
// ResolvePolicy("nope") → error
```
</example>

<verification>
cd /mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work
docker run --rm -v "$PWD":/src -w /src/server golang:1.26-alpine sh -c "go build ./internal/rotation/... && go vet ./internal/rotation/... && go test ./internal/rotation/ -run Policy -v" 2>&1 | tail -15
Paste tail. DONE only on green.
</verification>

<persistence>
Finish fully; fix-and-rerun on red; never DONE on red; BLOCKED only on true blocker.
</persistence>

<output>Sign-out: agent Codex#5.5#A, started_at, finished_at (UTC), status DONE, green tail in build_result.</output>
