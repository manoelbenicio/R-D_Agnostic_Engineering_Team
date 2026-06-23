# PRD — Agent 1: Core Features (Issues, Projects, New Issue)

This document provides a comprehensive mapping of **Phase 1 (Issues)**, **Phase 2 (Projects)**, and **Phase 8 (New Issue)** for the Multica platform. It includes structural descriptions, component mappings, form field schemas, status/priority indicators, and screenshot references.

---

## 1. Issues

### 1.1 List View
The Issues Tracker dashboard serves as the main hub for task management. It supports dual viewing modes, advanced grouping, ordering, and multi-dimensional filtering.

- **Access Path:** Issues (`/issues` or `/[workspaceSlug]/issues`).
- **View Modes:**
  1. **List View (`list-view.tsx`):** Renders a vertical table list of issue rows. Columns include Checkbox (for bulk actions), Status Icon, Identifier (e.g., `MUL-128`), Title, Assignee avatar/badge, Priority, Labels, and Due Date.
  2. **Board/Kanban View (`board-view.tsx`):** Renders a column-based kanban board. By default, columns map to issue statuses. Columns support drag-and-drop issue cards (`board-card.tsx` and `board-column.tsx`) using `@hello-pangea/dnd` library.
  3. **Swimlane View (`swimlane-view.tsx`):** Organizes issues horizontally in swimlanes based on groupings.
  4. **Gantt View (`gantt-view.tsx`):** Renders timeline-based issue bars.
- **Header Filters & Controls (`issues-header.tsx`):**
  - **Search:** Text input filtering issues by title/description matching.
  - **Status Filter:** Select popover filtering by Backlog, Todo, In Progress, In Review, Done, and Cancelled.
  - **Priority Filter:** Select popover filtering by No priority, Low, Medium, High, and Urgent.
  - **Assignee Filter:** Searchable popover listing both human members and agents (differentiated visually by robot badges).
  - **Project Filter:** Popover filtering by project.
  - **Group By:** Dropdown options to group rows/cards by: *Status*, *Priority*, *Assignee*, or *Project*.
  - **Sort By:** Ordering dropdown to sort issues by: *Title*, *Identifier*, *Status*, *Priority*, *Assignee*, *Created*, *Updated*, or *Due date*.
- **Interaction Affordances:**
  - **Bulk Selection (`batch-action-toolbar.tsx`):** Selecting checkboxes on rows triggers a bottom floating action toolbar to change status, priority, assignee, or delete in batch.
  - **Right-Click Context Menu:** Triggers quick actions: Edit, Assign to Me, Change Status, Change Priority, Copy link, and Delete.

### 1.2 Detail View (`issue-detail.tsx`)
Clicking on an individual issue opens the detail layout, split into two primary areas:
1. **Left Column - Description and Discussion Feed:**
   - **Breadcrumbs:** Navigational breadcrumbs at the top (`Workspace > Project > Issue-Identifier > Title`).
   - **Title Header:** Editable heading field.
   - **Description Area:** Large text block supporting rich markdown syntax.
   - **Activity Timeline (`comment-card.tsx`):** Renders comments, status changes, assignments, and agent activity history.
   - **Agent Activity Log (`execution-log-section.tsx`):** Shows logs, thinking logs, and executed tool calls.
   - **"Agent is Working" Live Panel (`issue-agent-activity-indicator.tsx`):** When an agent is assigned and working on the task, a live indicator displays:
     - An active rotating spinner.
     - Status: *"Agent is working..."*.
     - Elapsed execution time (live timer).
     - Expanded log list of current tool invocations (e.g., shell commands, file reads).
   - **Linked Pull Requests (`pull-request-list.tsx`):** Lists pull request branches, author name, and CI status connected via GitHub App.
2. **Right Column - Properties Sidebar:**
   - **Status Picker:** Triggers status dropdown select.
   - **Priority Picker:** Triggers priority dropdown select.
   - **Assignee Picker:** Searchable popover with human vs. agent profiles (agents carry bot indicators).
   - **Label Picker:** Inline list of applied labels with a dropdown selector.
   - **Due Date / Start Date Picker:** Calendar inputs.
   - **Project Selector:** Links the issue to a project.
   - **Sub-issues / Parent Issue:** Defines task hierarchy.

### 1.3 Status System (all statuses + icons)
Multica uses custom SVG status icons (`status-icon.tsx`) representing circular progress rings (pie charts) or checkmarks:

| Status Key | Fill Level | Visual Representation |
| :--- | :--- | :--- |
| **Backlog** | `0%` | Dotted empty circle |
| **Todo** | `15%` | Empty circle with a center dot |
| **In Progress** | `50%` | Half-filled circle (animated spinning dot when agent is actively processing) |
| **In Review** | `75%` | Three-quarters filled circle |
| **Done** | `100%` | Fully filled green circle with checkmark |
| **Cancelled** | `N/A` | Crossed circle (x diagonal line) |

### 1.4 Priority System (all levels + icons)
Priority is indicated by four-bar vertical charts (`priority-icon.tsx`) of ascending height:

| Level Key | Filled Bars | Color Scheme |
| :--- | :--- | :--- |
| **No Priority** | `0 / 4` | Gray empty outlines |
| **Low** | `1 / 4` | 1 active gray bar |
| **Medium** | `2 / 4` | 2 active yellow/amber bars |
| **High** | `3 / 4` | 3 active orange bars |
| **Urgent** | `4 / 4` | 4 active red bars |

---

## 2. Projects

### 2.1 Projects List (`projects-page.tsx`)
- **Access Path:** Projects (`/projects` or `/[workspaceSlug]/projects`).
- **Layout:** Displays a grid/list of project cards.
- **Card Content:** Displays project icon, title, description, lead member avatar, progress indicator bar (percentage of completed vs. total issues), and project status tag (**Active**, **Paused**, or **Completed**).
- **Header Actions:** Contains the **"+ New Project"** button.

### 2.2 Project Detail (`project-detail.tsx`)
- **Access Path:** `/[workspaceSlug]/projects/[id]`.
- **Properties Displayed:**
  - Project Title & Key (e.g. `BACK`).
  - Lead Member.
  - Project Description.
  - Local Directory Path: points to the on-disk directory where the codebase is located (used by agents to run tests and make edits).
  - Associated git repository resources.
- **Scoped Issue Tracker:** Renders list/board view of issues locked to the project.

### 2.3 Project Creation (`create-project.tsx`)
- **Type:** Modal Dialog.
- **Form Fields:**
  - **Project Name:** Required string input.
  - **Project Key:** Required string (2-5 uppercase chars) used in issue numbers.
  - **Description:** Optional text.
  - **Project Lead:** Select dropdown containing workspace members.
  - **Color Picker:** Circular color selectors for project categorization.
- **Validation:** Disables "Create project" submit button if Name or Key are missing.

---

## 3. New Issue

### 3.1 Creation Form (`create-issue.tsx`)
- **Type:** Page (`/[workspaceSlug]/new-issue`) or Modal Dialog (`create-issue-dialog.tsx` / `quick-create-issue.tsx`).
- **Form Fields:**
  - **Title:** Text input. Required.
  - **Description Textarea:** Large markdown field with styling options.
  - **Project:** Select dropdown. Required.
  - **Assignee:** Searchable popover displaying human members and AI agents.
  - **Status:** Select dropdown defaulting to *Todo*.
  - **Priority:** Select dropdown defaulting to *No Priority*.
  - **Labels:** Multi-select tag popover.
  - **Due Date:** Calendar popover.
- **Validation:** Highlights fields in red and displays errors under inputs if required properties are empty.

### 3.2 Description Editor
- Built using TipTap / ProseMirror rich text wrapper.
- Supports inline markdown compilation, bullet points, numbered lists, blockquotes, code blocks, and file attachments.
- Supports `@mentions` of humans and agents.

### 3.3 Agent Assignment Flow
- Assigning an issue to an **Agent** triggers a backend webhook notifying the agent's daemon service.
- The agent transitions the issue to **In Progress**, spawns a execution thread, and outputs activity directly to the **Activity Timeline** and **Execution Logs** in real-time.

---

## 4. Screenshots Inventory

All screenshots have been generated and saved under `C:\VMs\Projetos\Automonous_Agentic\Mapeamento_New_Features\02_multica\screenshots\`.

### Phase 1 — Issues (saved under `02-issues/`)
- `issues-list-default.png` — Issues list board view
- `issues-list-list-view.png` — Issues list tabular list view
- `issues-list-board-view.png` — Kanban Board view layout
- `issues-list-filters-status.png` — Status selection filter popover
- `issues-list-filters-priority.png` — Priority selection filter popover
- `issues-list-filters-assignee.png` — Assignee dropdown with human vs agent list
- `issues-list-sort-dropdown.png` — Ordering / sorting selector popover
- `issues-list-group-by.png` — Group by status/priority dropdown
- `issues-list-context-menu.png` — Right-click context actions panel
- `issues-list-bulk-actions.png` — Multi-select batch action toolbar
- `issue-detail-full.png` — Split screen issue details page
- `issue-detail-status-picker.png` — Issue detail status dropdown picker
- `issue-detail-priority-picker.png` — Issue detail priority dropdown picker
- `issue-detail-assignee-picker.png` — Issue detail assignee list dropdown
- `issue-detail-labels-picker.png` — Labels select popover
- `issue-detail-date-picker.png` — Calendar due date picker dialog
- `issue-detail-activity-timeline.png` — Comment list and activity history
- `issue-detail-agent-working.png` — Agent working live status panel
- `status-icons-all.png` — Diagram displaying all 6 progress pie status SVGs
- `priority-icons-all.png` — Diagram displaying all 5 bar priority SVGs

### Phase 2 — Projects (saved under `03-projects/`)
- `projects-list.png` — Projects dashboard cards list
- `project-creation-form.png` — New Project creation dialog form
- `project-detail.png` — Project detail page with local path config
- `project-issues-view.png` — Scoped project issues tracker

### Phase 8 — New Issue (saved under `09-new-issue/`)
- `new-issue-form-full.png` — Create issue full dialog modal
- `new-issue-assignee-picker.png` — Assignee picker showing bot badges
- `new-issue-description-editor.png` — Description markdown toolbar
- `new-issue-validation.png` — Form validation error states
