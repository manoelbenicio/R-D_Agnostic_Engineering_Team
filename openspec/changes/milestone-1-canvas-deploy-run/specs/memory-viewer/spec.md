## ADDED Requirements

### Requirement: Memory List View

The system SHALL render a memory list at `/memory` that shows entries from CAO's memory wiki (master spec §7.7). The list SHALL be filterable by scope (`global`, `project`, `session`, `agent`), by memory type (`project`, `user`, `feedback`, `reference`), and by tag. Each entry SHALL display: title, scope, type, tags (as badges), and last-updated timestamp.

Because CAO does not currently expose a list-memories REST endpoint, the v1 implementation SHALL read memories via the per-terminal context API (`GET /terminals/{id}/memory-context`) when a terminal is selected, and SHALL read filesystem-style listings via the agent-dirs settings (`GET /settings/agent-dirs`) for the global/project/session/agent views. Where direct listing is not possible, the UI SHALL surface a clear empty state explaining the v1 limitation.

#### Scenario: Filter by global scope

- **WHEN** the user selects "Global" in the scope filter
- **THEN** the list shows only memories with `scope: global`

#### Scenario: Empty state when CAO has no list endpoint reachable

- **WHEN** CAO does not return memories for a given scope (because no endpoint serves them)
- **THEN** the list shows an empty state with the text "No memories visible in this view (v1 limitation)"

### Requirement: Memory Detail Viewer

Selecting a memory entry SHALL open a detail viewer rendering the memory content (markdown), its scope and type metadata, tags, retention info (when available), and the location path (`memory/global/wiki/global/...`). Markdown SHALL render with SENTINEL prose styling.

#### Scenario: Markdown content renders with SENTINEL prose

- **WHEN** the user opens a memory whose content is markdown
- **THEN** headings, lists, and code blocks render with SENTINEL prose styling

### Requirement: Search Across Memories

The system SHALL provide a full-text search box that filters the visible list by content match. Search SHALL be case-insensitive and SHALL match across title, tags, and visible content.

#### Scenario: Search narrows the list

- **WHEN** the user types "deployment" into the search field
- **THEN** the list shows only memories whose content or tags contain "deployment"

### Requirement: Manual Memory Creation

The system SHALL provide a "New Memory" form that creates a memory entry by composing the CAO memory format and writing it via whatever CAO endpoint is available (or, if no direct write endpoint is exposed in v1, via instructing the user to author the memory through an agent's memory tools — with a documented limitation message). The form SHALL collect: title, scope, type, tags, content (markdown).

#### Scenario: Manual creation form validates required fields

- **WHEN** the user submits with title and content empty
- **THEN** inline errors appear and the submit is blocked

### Requirement: Retention Info Display

When the memory entry includes retention metadata (e.g., session-scoped memories that expire when the session ends), the detail viewer SHALL display the retention rule (e.g., "Persists until session `cao-588213f3` ends").

#### Scenario: Session-scoped memory shows retention notice

- **WHEN** the user opens a memory with `scope: session` for a known active session
- **THEN** the detail viewer displays "Persists until session `<session_name>` ends"
