# 🎯 AGENT 1 — Core Features: Issues + Projects + New Issue

> **Role:** Senior Product Mapper — Core Features
> **Scope:** PHASE 1 (Issues), PHASE 2 (Projects), PHASE 8 (New Issue)
> **Coordination:** You are Agent 1 of 3. Agent 2 handles Settings/Skills/Search. Agent 3 handles Inbox/My Issues/Landing/Design System.
> **Output Path:** `C:\VMs\Projetos\Automonous_Agentic\Mapeamento_New_Features\02_multica\`

---

## CONTEXT — What is Multica?

**Multica** (https://multica.ai) is an open-source project management platform for Human + Agent teams. It turns coding agents (Claude Code, Codex, Gemini CLI) into real teammates — assign issues, track progress, compound skills.

**Tech Stack:** Next.js App Router, Vercel, Tailwind CSS, shadcn/ui, Lucide icons, i18next, WebSocket, next-themes

**Demo Workspace URL Base:** `https://multica.ai/navy-seals/`

If login wall appears: screenshot the login flow (Email OTP + Google OAuth), then try "Get started" or demo access from the landing page.

---

## 🛠️ RECOMMENDED SKILLS & TOOLS

Use these for maximum efficiency and quality:

### Browser & Capture
- **Chrome DevTools MCP** — Use `take_screenshot` for pixel-perfect captures, `evaluate_script` to extract DOM structure and computed styles, `click` to interact with elements, `take_snapshot` for full DOM accessibility tree
- **Browser Subagent** — Use for multi-step click flows: open dropdown → screenshot → select option → screenshot result
- Use `evaluate_script` to extract issue data, status values, and component structures programmatically

### Analysis & Documentation
- **gsd-ui-review** — After capturing the Issues list and detail views, run the 6-pillar visual audit (spacing, color, typography, hierarchy, contrast, consistency) to document design patterns
- **gsd-ui-phase** — Use to generate a UI-SPEC.md for the issue tracker component architecture
- **gsd-sketch** — If you need to create annotated wireframes of complex layouts (issue detail view, board view), sketch throwaway HTML mockups for the PRD

### Codebase Intelligence (if GitHub repo accessible)
- **gsd-map-codebase** — Multica is open-source at `https://github.com/multica-ai/multica`. If accessible, use mapper agents to analyze the `/app/[workspaceSlug]/(dashboard)/issues/` component tree
- Check the repo for: component source files, Tailwind config, shadcn component definitions

### Extraction Helpers
```javascript
// Run in DevTools to extract all issue statuses from the UI
document.querySelectorAll('[data-status]').forEach(el => console.log(el.dataset.status, el.textContent));

// Extract all SVG icons on page
Array.from(document.querySelectorAll('svg')).map(s => ({class: s.className.baseVal, viewBox: s.getAttribute('viewBox'), html: s.outerHTML.substring(0, 200)}));

// Extract grid/table column headers
Array.from(document.querySelectorAll('th, [role="columnheader"]')).map(h => h.textContent.trim());
```

---

## YOUR PHASES

### PHASE 1: Issues (MOST CRITICAL) ⭐
**URL:** `https://multica.ai/navy-seals/issues`

You MUST map EVERY aspect of the issue tracker:

#### 1.1 Issues List View
- [ ] Screenshot the full issues list view (default state)
- [ ] Document ALL column headers (status icon, priority icon, title, assignee, labels, due date, etc.)
- [ ] Map ALL filter options — click every filter dropdown, screenshot each one:
  - Status filters (Backlog, Todo, In Progress, In Review, Done, Cancelled)
  - Priority filters (No priority, Low, Medium, High, Urgent)
  - Assignee filters (humans AND agents)
  - Label filters
  - Project filters
- [ ] Map ALL sort options — screenshot the sort dropdown
- [ ] Map ALL grouping options (group by status, priority, assignee, project)
- [ ] Map view modes: List view vs Board/Kanban view (if toggle exists — screenshot both)
- [ ] Map right-click / context menu on issue rows — screenshot it
- [ ] Map bulk selection — select multiple issues, screenshot bulk action bar
- [ ] Map keyboard shortcuts — press `?` or `Cmd+K` and screenshot any shortcut panel
- [ ] Document the issue count and any pagination

#### 1.2 Issue Detail View
- [ ] Click on an individual issue — screenshot the **full issue detail view**
- [ ] Map the **Properties sidebar** (right panel) — click each property to show its picker:
  - **Status** — click it, screenshot ALL possible values with their icons
  - **Priority** — click it, screenshot ALL levels with their bar-chart icons
  - **Assignee** — click it, screenshot the picker showing BOTH humans and agents (critical — document the human vs agent visual distinction)
  - **Labels/Tags** — click it, screenshot the label selector
  - **Due date** — click it, screenshot the date picker
  - **Project** — click it, screenshot the project selector
  - **Cycle/Sprint** — if exists, document it
  - **Parent issue / Sub-issues** — if exists, document the hierarchy
  - **Pull Requests** — if GitHub integration shows linked PRs, screenshot it
- [ ] Map the **Activity timeline**:
  - Screenshot comment input area
  - Screenshot status change events
  - Screenshot assignment events  
  - Screenshot **agent activity events** (tool calls, thinking, "Agent is working" live panel with spinner + timer + tool call list)
- [ ] Map the **breadcrumb navigation** at top (`Workspace > Project > Issue-Key > Title`)
- [ ] Map any "Subscribe" / "Watch" toggle
- [ ] Check for issue editing (inline editing vs edit mode)

#### 1.3 Issue Status Icons (CRITICAL)
Multica uses **custom SVG pie-chart style status icons** — each status has a different fill level:
- Backlog = empty circle
- Todo = empty circle with dot
- In Progress = half-filled circle (with animation?)
- In Review = ¾ filled
- Done = fully filled (green checkmark?)
- Cancelled = crossed out

Screenshot each status icon close-up and document the SVG pattern.

#### 1.4 Priority Icons (CRITICAL)
Multica uses **bar-chart style priority icons** — ascending bars:
- No priority = dotted/empty bars
- Low = 1 bar filled
- Medium = 2 bars filled (amber)
- High = 3 bars filled (orange)
- Urgent = 4 bars filled (red)

Screenshot each priority icon.

---

### PHASE 2: Projects
**URL:** `https://multica.ai/navy-seals/projects`

- [ ] Screenshot the projects list/grid view
- [ ] Map project creation flow — click "New Project" or equivalent, screenshot the form
- [ ] Click into a project — screenshot the project detail view
- [ ] Document project properties: name, description, status, lead, members, color/icon
- [ ] Map project-scoped issue view (filtered issues within a project)
- [ ] Document any project-level settings or configuration
- [ ] Map project status options (Active, Paused, Completed, etc.)

---

### PHASE 8: New Issue
**URL:** `https://multica.ai/navy-seals/new-issue` (or via sidebar button)

- [ ] Screenshot the full issue creation form/page
- [ ] Map EVERY field:
  - Title input (placeholder text, character limits?)
  - Description editor — is it markdown? WYSIWYG? What toolbar options?
  - Status selector
  - Priority selector
  - Assignee picker — screenshot showing BOTH humans and agents
  - Labels/tags picker
  - Project selector
  - Due date picker
  - Any additional custom fields
- [ ] Test creating an issue assigned to an AGENT — screenshot the flow
- [ ] Map any AI-assisted features (auto-suggestion, AI fill, etc.)
- [ ] Map the description editor capabilities: markdown, @mentions, file attachments, code blocks?
- [ ] Document form validation behaviors (required fields, error messages)

---

## OUTPUT REQUIREMENTS

### Screenshots — save to:
```
C:\VMs\Projetos\Automonous_Agentic\Mapeamento_New_Features\02_multica\screenshots\
├── 02-issues/
│   ├── issues-list-default.png
│   ├── issues-list-filters-status.png
│   ├── issues-list-filters-priority.png
│   ├── issues-list-filters-assignee.png
│   ├── issues-list-sort-dropdown.png
│   ├── issues-list-group-by.png
│   ├── issues-list-board-view.png (if exists)
│   ├── issues-list-context-menu.png
│   ├── issues-list-bulk-actions.png
│   ├── issue-detail-full.png
│   ├── issue-detail-status-picker.png
│   ├── issue-detail-priority-picker.png
│   ├── issue-detail-assignee-picker.png
│   ├── issue-detail-labels-picker.png
│   ├── issue-detail-date-picker.png
│   ├── issue-detail-activity-timeline.png
│   ├── issue-detail-agent-working.png
│   ├── status-icons-all.png
│   └── priority-icons-all.png
├── 03-projects/
│   ├── projects-list.png
│   ├── project-creation-form.png
│   ├── project-detail.png
│   └── project-issues-view.png
└── 09-new-issue/
    ├── new-issue-form-full.png
    ├── new-issue-assignee-picker.png
    ├── new-issue-description-editor.png
    └── new-issue-validation.png
```

### Deliverable — PRD Section
Write your findings into a file called:
`C:\VMs\Projetos\Automonous_Agentic\Mapeamento_New_Features\02_multica\PRD_AGENT1_CORE_FEATURES.md`

Structure:
```markdown
# PRD — Agent 1: Core Features (Issues, Projects, New Issue)

## 1. Issues
### 1.1 List View
### 1.2 Detail View  
### 1.3 Status System (all statuses + icons)
### 1.4 Priority System (all levels + icons)
### 1.5 Assignee System (humans vs agents)
### 1.6 Filters, Sort, Group
### 1.7 Bulk Actions
### 1.8 Keyboard Shortcuts

## 2. Projects
### 2.1 Projects List
### 2.2 Project Detail
### 2.3 Project Creation

## 3. New Issue
### 3.1 Creation Form
### 3.2 Description Editor
### 3.3 Agent Assignment Flow

## 4. Screenshots Inventory
```

### Quality Rules
- Screenshot at MAXIMUM resolution (1920x1080+)
- Screenshot BEFORE and AFTER clicking every interactive element
- Document every field label, placeholder text, and tooltip
- Note any animations or transitions
- Capture empty states and loading states (skeleton screens)

---

**END — Execute PHASE 1 first (Issues), then PHASE 2 (Projects), then PHASE 8 (New Issue)**
