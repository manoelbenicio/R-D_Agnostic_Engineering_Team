# vendor-model-visibility

## ADDED Requirements

### Requirement: Runtime-specific model catalogs SHALL remain visible offline

The client SHALL retain the selected runtime's exact model-catalog query identity
when discovery is disabled offline and SHALL render bounded last-known catalog data
without initiating a discovery request.

#### Scenario: Runtime becomes offline with cached models

- **WHEN** the selected runtime is offline and its exact runtime-specific catalog
  is cached
- **THEN** the picker SHALL keep that query identity, disable fetching, and display
  the bounded cached catalog

#### Scenario: Runtime is offline without cached models

- **WHEN** the selected runtime is offline and no exact cached catalog exists
- **THEN** the picker SHALL NOT initiate model discovery or substitute another
  runtime's catalog

### Requirement: Model discovery cancellation SHALL reach every owned HTTP request

The core model-discovery flow SHALL pass one query `AbortSignal` to both
`ApiClient.initiateListModels` and `ApiClient.getListModelsResult`, including every
polling request, and SHALL stop before issuing another request after cancellation.

#### Scenario: Discovery is cancelled during initiation or polling

- **WHEN** the query signal is aborted before or during model discovery
- **THEN** the same signal SHALL be present on the owned HTTP request and no later
  polling request SHALL be initiated after cancellation is observed

#### Scenario: Discovery remains active

- **WHEN** initiation returns a non-terminal request and the signal remains active
- **THEN** polling SHALL use the same signal until a terminal result is returned

### Requirement: Provider identity SHALL be visible and conservatively derived

The create and inspector model pickers SHALL group models consistently by provider,
include provider names in search, prefer a non-empty model provider, and use only
an exact authoritative or built-in runtime identity as fallback.

#### Scenario: Model provides an explicit provider

- **WHEN** a model row contains a non-empty provider
- **THEN** that provider SHALL win over runtime-derived fallback identity

#### Scenario: Provider is absent

- **WHEN** a model has no provider and no exact authoritative or built-in runtime
  identity is available
- **THEN** the picker SHALL classify it conservatively as unknown/custom and SHALL
  NOT infer a provider from substrings

### Requirement: Retained offline state SHALL be bounded to the client session

The client SHALL bound retained model catalogs, rows, and remembered provider
identities and SHALL clear that state when the QueryClient session is cleared.

#### Scenario: Retained catalog limits are reached

- **WHEN** a new retained catalog or row would exceed the configured bound
- **THEN** the least-recent bounded entry SHALL be evicted rather than growing
  retained state without limit

#### Scenario: Session is reset

- **WHEN** the QueryClient is cleared during logout or session reset
- **THEN** retained Packet B catalog and identity state SHALL be removed

### Requirement: Packet B ownership in the shared API client SHALL be method-bounded

Packet B SHALL own only the optional `AbortSignal` parameters and request
passthrough behavior of `ApiClient.initiateListModels` and
`ApiClient.getListModelsResult` within `packages/core/api/client.ts`.

#### Scenario: Packet B changes the shared API client

- **WHEN** a future Packet B change requires API-client modification
- **THEN** it SHALL be limited to the two list-model methods or require a new
  independently reviewed ownership decision

#### Scenario: Auth or another API concern changes

- **WHEN** login, authentication, runtime-manager, or another API-client concern is
  modified
- **THEN** that change SHALL remain outside Packet B ownership and SHALL NOT be
  bundled under this capability
