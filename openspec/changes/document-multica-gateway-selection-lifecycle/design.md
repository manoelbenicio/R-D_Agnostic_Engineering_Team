# Design: Preserved Multica Gateway selection/lifecycle primitives

## Components

- `Selector`: in-memory bounded account order, eligibility, and continuation bindings.
- `Coordinator`: bounded pre-output retry, request deduplication, and cancellation.
- `ClassifyFailure`: deterministic content-free failure taxonomy.
- `Executor`: composition of selection, dispatch, classification, successful-response
  affinity binding, and pseudonymous selection recording.

## Selection

Independent requests use one concurrency-safe cursor and monotonic sequence. Stateful
continuations use explicit bindings created only after successful output. Affinity hits
do not advance the independent-request cursor.

## Lifecycle

The Selector represents eligible, quarantined, cooldown, and removed states; supports
bounded addition, re-entry, and removal; and removes bindings owned by a removed account.

The candidate does not evidence the production source that calls these lifecycle methods.
Documentation MUST NOT invent that source.

## Failure and dispatch

Failure classification separates cancellation, timeout, local overload, authentication,
authorization, account/model/provider rate limits, quota exhaustion, malformed responses,
and upstream failures.

Dispatch retries only within configured bounds and before committed output or tool action,
deduplicates request IDs, and releases slots deterministically.

## Production wiring boundary

Production code constructs Gateway client, registry, readiness, runtime-profile, policy,
and credential-source components. No non-test constructor or caller of `Selector`,
`Executor`, or the Gateway dispatch `Coordinator` is evidenced at the candidate revision.
Implemented library behavior therefore MUST NOT be represented as active production
selection or lifecycle wiring.

## Evidence boundary

All behavioral evidence is candidate-local and synthetic/in-process. The package does not
convert that evidence into live-provider, credential, production, deployment, publication,
or product-code proof.
