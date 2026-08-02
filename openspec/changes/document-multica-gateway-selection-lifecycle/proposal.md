# Change: Document preserved Multica Gateway selection and lifecycle primitives

## Authority

The project owner has directed that the current Multica implementation remain exactly
AS-IS and that OmniRouter is excluded from this solution.

This change documents existing candidate behavior only. It authorizes no code change,
runtime wiring, credential work, deployment, production acceptance, or publication.

## Provenance

Candidate: `a5aa53e8e89d2845cacfbc82ca851fbd18f9a505`

Covered commits:

- `64fef29deeb320f52878529cf6a54ef58f19c31f`
- `5471eda25682ef12f32fa2c3c3c5f241b548d887`
- `b608539afe37dbb174ebdcce5170eddce69d015e`
- `93d4a255710e0531c1b576033b1beb59b67ffeab`
- `1fd7f94b9bdfb4d8524d705b9e2179ca8a55f937`
- `653b5846b02eb07936b6b0d3f8311d52520ef238`

## AS-IS

Multica contains Gateway selection, lifecycle, classification, retry/deduplication, and
execution primitives with deterministic in-process tests. Production Gateway
client/readiness wiring exists, but no production constructor or caller of Selector,
Executor, or the Gateway dispatch Coordinator is evidenced in the candidate.

Existing live and archived OpenSpec assigns account selection elsewhere and therefore
contradicts the preserved implementation.

## TO-BE documentation state

The repository records the exact behavior and provenance of the preserved Multica
primitives, explicitly distinguishes implemented library behavior from evidenced live
wiring, and leaves contradictory external-router authority for a separately authorized
documentation reconciliation.

## Non-claims

- No production selection/lifecycle constructor is claimed.
- No credential or account source is defined.
- No runtime behavior beyond the candidate implementation is authorized.
- No code, test, deployment, production, publication, or integration acceptance is granted.
