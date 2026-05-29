## ADDED Requirements

### Requirement: Built-in Template Library

The system SHALL ship 10 built-in canvas templates per master spec §4.8. The set SHALL include exactly: **Code Review Pipeline**, **Bug Triage**, **Documentation Sprint**, **Full Stack Dev**, **Data Pipeline**, **Security Audit**, **DevOps Pipeline**, **Research Team**, **Enterprise Squad**, and **Blank Canvas**. Each template SHALL be a fully-formed `CanvasDocument` definition (nodes, edges, default config) plus metadata: `id`, `name`, `description`, `agent_count`, `primary_edge_type`, and `est_cost_per_hour_usd` (the displayed estimate from §4.8).

#### Scenario: Library exposes 10 templates

- **WHEN** an importer loads the templates module
- **THEN** the exported `TEMPLATES` array contains exactly 10 entries
- **AND** every entry has all 6 metadata fields populated

#### Scenario: Blank Canvas template

- **WHEN** the user instantiates the "Blank Canvas" template
- **THEN** the produced `CanvasDocument` has zero nodes, zero edges, default config, and `est_cost_per_hour_usd: 0`

### Requirement: Template Cost Display Contract

Every template entry surfaced in the UI SHALL display its cost-per-hour estimate alongside the mandatory ⚠️ warning label per `finops-tier1`. Templates SHALL NOT be shown without the warning label. Cost estimates SHALL be expressed in USD and represent the wall-clock cost of running every agent on Opus-class providers as documented in master spec §4.8.

#### Scenario: Templates picker shows ⚠️ on every entry

- **WHEN** the Templates picker renders the 10 entries
- **THEN** every entry displays its cost-per-hour estimate next to a ⚠️ glyph and the text "rough estimate"

### Requirement: Template Instantiation

The capability SHALL expose `instantiateTemplate(templateId): CanvasDocument` which returns a fresh `CanvasDocument` with:

- a newly generated `id` (UUID)
- `name` set to `"<template name> (copy)"`
- `version: 1`, `created_at` and `updated_at` set to now
- `nodes` and `edges` cloned from the template definition with regenerated UUIDs (so they don't collide with other instances)
- `deploy_state: { status: "draft" }`

The instantiated canvas SHALL NOT share any IDs with the template definition or other instances.

#### Scenario: Two instantiations have no shared IDs

- **WHEN** the user instantiates "Code Review Pipeline" twice
- **THEN** the two resulting canvases have different `id`s and disjoint node/edge id sets
