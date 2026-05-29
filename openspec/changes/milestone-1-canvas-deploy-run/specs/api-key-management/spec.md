## ADDED Requirements

### Requirement: BYOK Settings Page

The system SHALL provide a Settings page under the route `/settings/providers` where users enter API keys for supported providers. In v1 the page SHALL surface every provider listed in master spec §8.10: OpenAI, Anthropic, Google, AWS (covering Q CLI and Kiro CLI), Azure, Moonshot, Copilot, and OpenCode. Each provider entry SHALL display: provider name, supported models (when validated), the validation status, and a key-edit affordance. The page SHALL be structured so additional providers can be added by registering provider definitions without UI rework.

#### Scenario: User opens settings to a provider list

- **WHEN** an authenticated (or local-dev unauthenticated) user navigates to `/settings/providers`
- **THEN** the page renders one card per registered provider, each labeled with provider name and current validation status (`set`, `unset`, or `invalid`)
- **AND** all 8 v1 providers from master spec §8.10 are present

#### Scenario: Adding a provider definition does not require UI changes

- **WHEN** a developer adds a new entry to the provider registry following the documented contract
- **THEN** the new provider appears in the Settings page on next render with no additional component code

### Requirement: Per-Provider Validation by Live Call

When a user submits an API key for a provider, the system SHALL perform a live validation request against that provider's API and SHALL store the key only if validation succeeds. The validators SHALL match master spec §8.10:

- **OpenAI:** `GET https://api.openai.com/v1/models` → 200 OK
- **Anthropic:** `GET https://api.anthropic.com/v1/models` → 200 OK
- **Google:** `GET https://generativelanguage.googleapis.com/v1beta/models?key=...` → 200 OK
- **AWS:** `aws sts get-caller-identity` (signature) → 200 OK
- **Azure:** Endpoint health check at user-supplied `AZURE_OPENAI_ENDPOINT` → 200 OK
- **Moonshot:** API validation call against the Moonshot models endpoint → 200 OK
- **Copilot CLI:** Validation against the documented Copilot CLI auth endpoint → 200 OK
- **OpenCode CLI:** Validation against the configured OpenCode endpoint → 200 OK

Validation errors SHALL be surfaced verbatim to the user (status code + message body) without leaking the key in any error display, console log, or telemetry payload.

#### Scenario: Valid key persists and unlocks provider

- **WHEN** the user enters a valid key for any of the 8 providers and validation returns 200
- **THEN** the provider card transitions to status `set` and lists available models from the response
- **AND** the key value is persisted in the configured key store

#### Scenario: Invalid key is rejected and not persisted

- **WHEN** the user enters a key that fails validation (HTTP 401, 403, or unreachable endpoint)
- **THEN** the user sees an error message containing the provider's error text
- **AND** no key is written to storage
- **AND** the key value never appears in any console log, error message, or telemetry payload

#### Scenario: AWS dual-credential flow

- **WHEN** the user enters `AWS_ACCESS_KEY_ID` + `AWS_SECRET_ACCESS_KEY` together
- **THEN** the system validates by computing an STS GetCallerIdentity signed request
- **AND** the validated card lists Q CLI and Kiro CLI as the unlocked providers (CAO-side) for the AWS account

### Requirement: Key Storage Backend

In v1 the system SHALL persist API keys in the browser's IndexedDB under a dedicated `provider_keys` object store. Keys SHALL be stored as plaintext under the v1 threat model (developer-owned local browser); cloud storage with encryption-at-rest is post-launch (master spec §13). The storage layer SHALL be encapsulated behind a `KeyStore` interface so the cloud implementation can replace it without changing consumers. Keys SHALL be redacted to a masked form (`sk-…XXXX`) for any UI display after they have been entered.

#### Scenario: Plaintext-in-IndexedDB is documented

- **WHEN** a reviewer reads `docs/key-storage-v1.md`
- **THEN** the document explicitly states that v1 stores keys in plaintext IndexedDB and lists the upgrade path to encrypted Firestore

#### Scenario: Keys are masked in UI after entry

- **WHEN** the user has saved an Anthropic key and revisits the Settings page
- **THEN** the displayed value is masked (e.g., `sk-…AB12`) — the full plaintext is never re-rendered to the DOM

#### Scenario: KeyStore interface is the only persistence path

- **WHEN** code outside `src/api/key-store/` attempts to read or write provider keys directly
- **THEN** lint/architecture rules flag the violation

### Requirement: Provider Selection Gating

The Canvas Builder provider dropdown for an Agent Block SHALL list only providers whose validation status is `set`. Providers with status `unset` or `invalid` SHALL NOT appear in the dropdown. When a user opens the Canvas Builder with zero validated providers, the Builder SHALL still allow the user to create and edit draft canvases per master spec v4.2 §12 (read-only browse without keys is permitted; only Deploy is gated). A non-blocking inline notice SHALL inform the user that no providers are configured and SHALL provide a direct link to `/settings/providers`.

#### Scenario: Unconfigured providers are hidden from the dropdown

- **WHEN** only Anthropic is validated
- **THEN** the Canvas Builder provider dropdown shows exactly one option: Anthropic
- **AND** the other 7 providers are absent from the dropdown

#### Scenario: Non-blocking notice when zero providers configured

- **WHEN** zero providers are validated and the user opens the Canvas Builder
- **THEN** the canvas is interactive (drag, edit, save are all allowed)
- **AND** an inline notice reads "No providers configured — Deploy is disabled until you validate at least one in Settings"
- **AND** the notice contains a button linking to `/settings/providers`

### Requirement: Model Listing on Validation

When a provider key is validated, the system SHALL store the provider's available model list (parsed from the validation response) so it can be consumed by the Canvas Builder's model dropdown per master spec v4.2 §8.10. The stored model list SHALL be considered fresh for the duration of the browser session; users MAY trigger a refresh by re-validating the key.

The model dropdown in any consuming surface (Canvas Builder block configuration panel, Agent Studio profile editor) SHALL show every model in the stored list with NO default selection and NO recommendation indicator.

#### Scenario: Model list cached after validation

- **WHEN** the user validates an Anthropic key and the response lists 3 models
- **THEN** any subsequent open of the Canvas Builder model dropdown for an Anthropic-mapped node shows all 3 models without re-validating

#### Scenario: No default selection

- **WHEN** the model dropdown opens with 5 available models for the chosen provider
- **THEN** the dropdown's selected value is blank
- **AND** no item is visually marked as "default" or "recommended"

### Requirement: Key Removal

The user SHALL be able to remove a stored key from any provider card. Removal SHALL purge the key from `KeyStore` and SHALL transition the provider card to status `unset` immediately. If the removed provider was used by any node in the user's stored canvases, those nodes SHALL display a warning indicator on next canvas open.

#### Scenario: Removing a key purges it

- **WHEN** the user clicks "Remove key" on a validated Anthropic card
- **THEN** the next read from `KeyStore.get("anthropic")` returns null
- **AND** the provider card status becomes `unset`

#### Scenario: Canvases referencing removed provider show warnings

- **WHEN** a canvas contains nodes with `provider: "claude_code"` and the user removes the Anthropic key
- **THEN** opening that canvas next time shows a warning glyph on each affected node with hover text "Provider not configured"
