## ADDED Requirements

### Requirement: Profile List

The system SHALL render a profile list at `/agent-studio` showing every CAO agent profile returned by `GET /agents/profiles`. Each list entry SHALL display: profile name, role, provider, description (first line), and an "Edit" affordance. The list SHALL be searchable by name or role and SHALL be filterable by provider.

#### Scenario: List shows installed profiles

- **WHEN** CAO has 5 profiles installed and the user opens `/agent-studio`
- **THEN** the list shows 5 entries with name, role, provider, and description

#### Scenario: Search filters by name

- **WHEN** the user types "review" into the search field
- **THEN** the list shows only profiles whose name or role contains "review" (case-insensitive)

### Requirement: Provider Availability Panel

Agent Studio SHALL display a panel listing every CAO-managed provider (from `GET /agents/providers`) with installation status (`installed: true | false`). Profiles whose `provider` field references an uninstalled provider SHALL be flagged in the list with a warning glyph.

#### Scenario: Uninstalled provider warning appears in list

- **WHEN** a profile references `provider: "kimi_cli"` and `kimi_cli` reports `installed: false`
- **THEN** that profile entry shows a warning glyph with hover text "Provider 'kimi_cli' not installed"

### Requirement: Profile Detail Viewer

Selecting a profile SHALL open a detail viewer that renders: full markdown body (parsed and styled with SENTINEL prose styles), YAML frontmatter as a key-value list (`role`, `provider`, `model`, `allowedTools`, `mcpServers`, `permissionMode`), and metadata (created/updated dates if available from CAO).

#### Scenario: Markdown body renders parsed

- **WHEN** the user opens the detail viewer for a profile whose body is markdown
- **THEN** headings, lists, and code blocks render with SENTINEL prose styling, not as raw markdown

### Requirement: Profile Editor

The system SHALL provide a Profile Editor that allows creating new profiles or editing existing ones. The editor SHALL expose: a form for YAML frontmatter fields (with provider dropdown gated by `api-key-management`'s validated set) and a Monaco-style markdown editor for the body. Saving SHALL invoke `POST /agents/profiles/install` with the assembled profile markdown. Validation errors from CAO SHALL be surfaced verbatim.

#### Scenario: Save installs profile via CAO

- **WHEN** the user fills in a valid profile and clicks Save
- **THEN** the system serializes frontmatter + body and posts to `POST /agents/profiles/install`
- **AND** on success the profile list refreshes and includes the new entry

#### Scenario: Provider dropdown gated by validated set

- **WHEN** only Anthropic is validated in `api-key-management`
- **THEN** the editor's provider dropdown shows only Anthropic-mapped providers (`claude_code`)

### Requirement: Install From Source

The system SHALL allow installing a profile from three sources per master spec §8.5: (a) the built-in store (a curated set of profile templates shipped with AgentVerse), (b) a local file (file picker that reads a `.md` file from the user's filesystem), and (c) a URL (the user pastes a URL pointing to a markdown profile, the system fetches and parses it).

#### Scenario: Install from URL fetches and previews

- **WHEN** the user pastes a valid URL pointing to a markdown profile and clicks Install
- **THEN** the system fetches the URL, parses the frontmatter, and presents a preview before installing
- **AND** the user must confirm before `POST /agents/profiles/install` is issued
