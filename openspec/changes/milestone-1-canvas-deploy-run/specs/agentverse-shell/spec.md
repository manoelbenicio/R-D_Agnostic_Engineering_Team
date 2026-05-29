## ADDED Requirements

### Requirement: Vite + TypeScript SPA Bootstrap

The system SHALL be packaged as a Vite + TypeScript single-page application using React. The build SHALL produce a static asset bundle suitable for hosting on a CDN (Firebase Hosting in later milestones). Local development SHALL run via `npm run dev` and SHALL serve on `http://localhost:5173` to match the documented CAO CORS allow-list.

#### Scenario: Fresh checkout boots locally

- **WHEN** a developer runs `npm install` followed by `npm run dev` on a fresh checkout
- **THEN** the dev server starts on port 5173 and the SPA loads in a browser without console errors
- **AND** the SENTINEL design tokens are present at `:root`

#### Scenario: Production build emits static assets

- **WHEN** `npm run build` completes
- **THEN** the `dist/` directory contains an `index.html` referencing hashed JS and CSS bundles
- **AND** the bundle is loadable directly from a static file server with no runtime backend other than CAO

### Requirement: Application Routing

The shell SHALL provide client-side routing with the following v1 routes (master spec §8.1):

- `/` → Canvas list (entry point; lists existing canvases or shows empty-state).
- `/dashboard` → Dashboard / Central de Comando (KPIs, fleet status, activity feed).
- `/canvas/:id` → Canvas Builder for a specific canvas.
- `/canvas/:id/terminal/:terminalId` → Terminal Grid bound to a deployed canvas (delegates to the `terminal-grid` capability for layout; supports tab/grid/full-screen/chat modes).
- `/agent-studio` → Agent Studio (profile management).
- `/flows` → Flows (cron-scheduled agent sessions).
- `/finops` → FinOps Tier 1 (cost estimates, budget utilization, per-provider breakdown).
- `/memory` → Memory viewer.
- `/settings/providers` → BYOK provider settings.
- `/settings/appearance` → Settings → Appearance (font selection, theme).
- `/settings/general` → Settings → General (CAO base URL, default provider, default working directory).
- `/health` → Health status page (CAO + provider availability + browser capabilities).
- `*` → 404 page using SENTINEL design.

#### Scenario: Direct deep-link to a canvas works

- **WHEN** a user visits `http://localhost:5173/canvas/<existing-id>` directly
- **THEN** the Canvas Builder loads with the requested canvas without navigating through `/`

#### Scenario: Unknown route renders 404

- **WHEN** the user visits a path not in the routing table
- **THEN** a 404 page renders, styled with SENTINEL components, with a link back to `/`

### Requirement: Application Layout

The shell SHALL render a consistent layout across all routes consisting of: (a) a fixed top NavBar containing the AgentVerse wordmark, primary navigation, and a CAO health indicator; (b) a main content region; and (c) a bottom-right toast region for transient notifications. The layout SHALL be responsive and SHALL render usably on viewports as narrow as 1024×768. Mobile and tablet support is deferred to Milestone 2 (Chat View).

#### Scenario: Health indicator reflects CAO status

- **WHEN** CAO is healthy
- **THEN** the NavBar shows a green dot labeled "CAO ONLINE"
- **WHEN** CAO becomes unreachable
- **THEN** within one health-poll interval the NavBar shows a red dot labeled "CAO UNREACHABLE"

#### Scenario: Toasts display and auto-dismiss

- **WHEN** the application calls the toast API to show an info toast
- **THEN** a toast appears in the bottom-right region for the documented default duration and then disappears
- **AND** clicking the toast dismisses it immediately

### Requirement: Auth-Aware Fetch Boundary

All outbound HTTP traffic to AgentVerse-managed services (not provider APIs and not CAO direct) SHALL flow through a single `appFetch` wrapper. In Milestone 1 the wrapper is a thin pass-through (no auth header is attached because cloud auth is deferred to M2), but every consumer SHALL go through it so M2 can introduce Firebase JWT attachment without a project-wide refactor.

#### Scenario: Direct fetch is forbidden for AgentVerse services

- **WHEN** a developer attempts to call `fetch("/api/agentverse/...")` directly from a component
- **THEN** lint or architecture rules flag the call as a violation

#### Scenario: appFetch is the only entry to auth-bearing requests

- **WHEN** M2 introduces auth header attachment inside `appFetch`
- **THEN** every previously-written consumer automatically sends the auth header without modification

### Requirement: Global Error Boundary

The shell SHALL wrap the routed view in a top-level error boundary that catches unhandled render errors, logs them to the browser console with a structured object, and renders a SENTINEL-styled fallback offering "Reload" and "Report" actions. The error boundary SHALL NOT catch async errors (those are the responsibility of the originating capability) but SHALL catch any synchronous render exception.

#### Scenario: Render exception is caught

- **WHEN** a component throws during render
- **THEN** the error boundary fallback renders in place of the broken subtree
- **AND** the rest of the application (NavBar, toasts) remains interactive
