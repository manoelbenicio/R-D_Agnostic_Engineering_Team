# 🎯 MEGA-PROMPT — Full Mapping & PRD: multica.ai

> **Orchestrator:** Principal Architect
> **Created:** 2026-06-22
> **Target URL:** https://multica.ai/navy-seals/inbox
> **Workspace:** `navy-seals` (demo workspace)
> **Objective:** 100% feature mapping, visual documentation, and PRD for internal replication

---

## 🔴 MISSION CRITICAL — READ FIRST

You are a **Senior Subject Matter Expert** performing a complete product reverse-engineering of **Multica** (https://multica.ai). Multica is an **open-source project management platform for Human + Agent teams** — it turns coding agents (Claude Code, Codex, Gemini CLI, OpenClaw, OpenCode) into real teammates that can be assigned issues, track progress, and compound skills.

### What You Must Deliver
1. **Screenshots** of EVERY single screen, menu, modal, dropdown, dialog — saved as PNG
2. **Recordings** (browser recordings) of key interaction flows
3. **Complete PRD** document (`MULTICA_PRD.md`) with every feature cataloged
4. **Technology mapping** of the entire stack
5. **Component architecture** diagram (AS-IS)
6. **All visual assets** downloaded (images, icons, SVGs)

### Rules
- **DO NOT skip any menu, submenu, button, option, or feature**
- **Screenshot BEFORE and AFTER clicking** every interactive element
- **Document every field, label, placeholder text, and tooltip**
- **Map keyboard shortcuts** if any exist
- **Note responsive behavior** at different breakpoints
- **Save ALL screenshots** to `c:\VMs\Projetos\Automonous_Agentic\docs\multica-reference\screenshots\`
- **Save ALL videos** to `c:\VMs\Projetos\Automonous_Agentic\docs\multica-reference\videos\`

---

## 🏗️ ARCHITECTURE RECON (Already Discovered)

### Product Identity
| Property | Value |
|----------|-------|
| **Product** | Multica — Project Management for Human + Agent Teams |
| **Tagline** | "Your next 10 hires won't be human." |
| **Description** | Open-source platform that turns coding agents into real teammates |
| **GitHub** | https://github.com/multica-ai/multica |
| **License** | Open-source |
| **Twitter** | @multica_hq |

### Technology Stack (Confirmed from HTML source)
| Layer | Technology | Evidence |
|-------|-----------|----------|
| **Framework** | **Next.js** (App Router) | `/_next/static/chunks/`, RSC payloads, `app/[workspaceSlug]/` routing |
| **Deployment** | **Vercel** | `dpl_9HDsZP4vaY39qN4Y1fYsD5atUBwc` deployment ID |
| **CSS** | **Tailwind CSS** | Utility classes throughout (`flex`, `items-center`, `rounded-[11px]`) |
| **UI Components** | **shadcn/ui** (Radix-based) | `data-slot="skeleton"`, Lucide icons, component patterns |
| **Icons** | **Lucide React** | `lucide lucide-bot`, `lucide lucide-menu`, etc. |
| **Fonts** | Custom (3 font families) | `__variable_6fb02a`, `__variable_9dfe4c`, `__variable_2342dc` |
| **Theme** | **next-themes** | `ThemeProvider`, localStorage `theme` key, system/light/dark |
| **i18n** | **i18next** | Full translation resources discovered (en, zh, ko, ja) |
| **Auth** | Email OTP + Google OAuth | Sign-in flow with verification codes |
| **Real-time** | **WebSocket** | Agent progress streaming |
| **State** | React Server Components + Client | RSC streaming, `ClientSegmentRoot`, `ClientPageRoot` |
| **Analytics** | (none visible) | No GA/Mixpanel detected |
| **Hosting** | Vercel Edge | Cloudflare-style edge routing |

### URL/Route Structure (from Next.js chunks)
```
app/
├── (landing)/
│   ├── layout.tsx
│   └── page.tsx                    → multica.ai/
├── [workspaceSlug]/
│   ├── layout.tsx                  → Workspace shell (sidebar + header)
│   └── (dashboard)/
│       ├── layout.tsx              → Dashboard shell
│       ├── inbox/page.tsx          → /navy-seals/inbox
│       ├── my-issues/page.tsx      → /navy-seals/my-issues (likely)
│       ├── issues/page.tsx         → /navy-seals/issues
│       ├── projects/page.tsx       → /navy-seals/projects
│       ├── search/page.tsx         → /navy-seals/search
│       ├── new-issue/page.tsx      → /navy-seals/new-issue
│       └── settings/
│           ├── page.tsx            → /navy-seals/settings
│           └── [tab]/page.tsx      → /navy-seals/settings/[tab]
├── login/page.tsx
├── global-error.tsx
├── layout.tsx
└── not-found.tsx
```

### Confirmed Features (from i18n translation keys)
These features exist in the codebase — the agent MUST find and document each one:

1. **Issues** — Full issue tracker with status, priority, assignee, labels, due dates
2. **Projects** — Project-level grouping of issues
3. **Inbox** — Notification inbox with filtering (assignments, status changes, comments, agent activity)
4. **My Issues** — Personal filtered view of assigned issues
5. **Search** — Global search across workspace
6. **New Issue** — Issue creation form
7. **Settings** — Multi-tab settings:
   - Profile (name, avatar, description for agents)
   - Preferences (theme, language, timezone)
   - Notifications (inbox notification controls)
   - API Tokens (PAT creation/revocation)
   - General (workspace name, description, context, slug, issue prefix)
   - Repositories (git repo management for agents)
   - GitHub (GitHub App integration, PR sidebar, co-author, auto-link)
   - Integrations / Lark (Lark/Feishu bot binding)
   - Labs (experimental features)
   - Members (workspace member management)
8. **Skills** — Agent skill configuration (compound skills system)
9. **Agents** — Agent profiles, status monitoring, task assignment
10. **Configure** — Workspace configuration area

### Sidebar Navigation (Expected Structure)
```
┌─────────────────────────┐
│ [Logo] multica           │
│ [Workspace: navy-seals]  │
│ ─────────────────────── │
│ 🔍 Search               │
│ 📥 Inbox                │
│ 📋 My Issues            │
│ ─────────────────────── │
│ 📌 Issues               │  ← PRIORITY 1
│ 📁 Projects             │  ← PRIORITY 2
│ ─────────────────────── │
│ ⚙️ Configure            │  ← PRIORITY 3
│   ├── Skills            │
│   └── Settings          │
│ ─────────────────────── │
│ [+ New Issue]           │
└─────────────────────────┘
```

---

## 📋 EXECUTION PLAN — Priority Order

### PHASE 1: Issues (MOST CRITICAL)
**URL:** `https://multica.ai/navy-seals/issues`

Map EVERY aspect:
- [ ] Screenshot the full issues list view (default state)
- [ ] Document the column headers (status, priority, assignee, title, etc.)
- [ ] Map ALL filter options (status filters, priority filters, assignee filters)
- [ ] Map ALL sort options
- [ ] Map ALL grouping options (group by status, priority, assignee, project, etc.)
- [ ] Map view modes: List view vs Board/Kanban view (if exists)
- [ ] Click on an individual issue — screenshot the **issue detail view**
- [ ] Map the issue detail sidebar (Properties panel):
  - Status field (all possible values: Backlog, Todo, In Progress, In Review, Done, Cancelled)
  - Priority field (all values: No priority, Low, Medium, High, Urgent)
  - Assignee field (show the assignee picker — human vs agent distinction)
  - Labels/Tags field
  - Due date field
  - Project field
  - Cycle/Sprint field (if exists)
  - Parent issue / sub-issues (if exists)
- [ ] Map the Activity timeline on issue detail:
  - Comment input
  - Status change events
  - Assignment events
  - Agent activity events (tool calls, thinking, working status)
- [ ] Map the "Agent is working" live panel (spinner, timer, tool call list)
- [ ] Screenshot the issue creation flow (via "New Issue" button)
- [ ] Map all right-click / context menu options on issues
- [ ] Map bulk selection and bulk actions
- [ ] Map keyboard shortcuts (Cmd+K? etc.)
- [ ] Check for linked Pull Requests panel
- [ ] Document the breadcrumb navigation pattern

### PHASE 2: Projects
**URL:** `https://multica.ai/navy-seals/projects`

- [ ] Screenshot the projects list view
- [ ] Map project creation flow
- [ ] Map project detail view (clicking into a project)
- [ ] Document project properties (name, description, status, lead, members)
- [ ] Map project-scoped issue filtering
- [ ] Document project views/layouts

### PHASE 3: Configure — Skills
**URL:** `https://multica.ai/navy-seals/configure/skills` (or similar)

- [ ] Screenshot the skills configuration page
- [ ] Map what "Skills" means in Multica context (compound skills for agents)
- [ ] Document skill creation flow
- [ ] Map skill properties (name, description, instructions, file patterns, etc.)
- [ ] Map skill assignment to agents
- [ ] Document the skill templating system
- [ ] Map any skill marketplace or library

### PHASE 4: Configure — Settings (ALL TABS)
**URL:** `https://multica.ai/navy-seals/settings`

Map EVERY tab:

#### Tab: General
- [ ] Workspace name, description, context fields
- [ ] Slug configuration
- [ ] Issue prefix configuration
- [ ] Logo upload
- [ ] Danger zone (leave/delete workspace)

#### Tab: Members
- [ ] Member list view
- [ ] Invite flow
- [ ] Role management (Owner, Admin, Member)
- [ ] Agent vs Human distinction in member list

#### Tab: Repositories
- [ ] Repository list
- [ ] Add repository flow
- [ ] Repository properties (URL, description)

#### Tab: GitHub
- [ ] GitHub App connection status
- [ ] Feature toggles (PR sidebar, co-author, auto-link)
- [ ] Repository linking

#### Tab: Integrations (Lark)
- [ ] Lark bot binding flow
- [ ] QR code scanning interface
- [ ] Bot management

#### Tab: Profile
- [ ] Avatar upload
- [ ] Name field
- [ ] "About you" description (shared with agents)

#### Tab: Preferences
- [ ] Theme toggle (Light/Dark/System)
- [ ] Language selector (en, zh, ko, ja)
- [ ] Timezone selector

#### Tab: Notifications
- [ ] Notification category toggles
- [ ] System notification toggle
- [ ] Browser notification toggle

#### Tab: API Tokens
- [ ] Token list
- [ ] Token creation flow (name, expiry)
- [ ] Token display (masked prefix, created date, last used)
- [ ] Token revocation flow

#### Tab: Labs
- [ ] Any experimental features

### PHASE 5: Inbox
**URL:** `https://multica.ai/navy-seals/inbox`

- [ ] Screenshot empty inbox state
- [ ] Screenshot inbox with notifications
- [ ] Map notification types (assignment, status change, comment, agent activity)
- [ ] Map read/unread states
- [ ] Map notification actions (mark as read, navigate to issue)
- [ ] Map inbox filters/tabs if any
- [ ] Map notification settings link

### PHASE 6: My Issues
**URL:** `https://multica.ai/navy-seals/my-issues`

- [ ] Screenshot the "My Issues" view
- [ ] Document filtering (assigned to me, created by me, subscribed)
- [ ] Map view options
- [ ] Compare with main Issues view — document differences

### PHASE 7: Search
**URL:** `https://multica.ai/navy-seals/search` (or Cmd+K modal)

- [ ] Map the search interface (modal? dedicated page?)
- [ ] Test search across issues, projects, people
- [ ] Document search filters and facets
- [ ] Map search result types and their display
- [ ] Document keyboard shortcut to invoke search
- [ ] Map search suggestions/autocomplete

### PHASE 8: New Issue
**URL:** `https://multica.ai/navy-seals/new-issue`

- [ ] Screenshot the full issue creation form
- [ ] Map every field: title, description (markdown editor?), status, priority, assignee, labels, project, due date
- [ ] Map the assignee picker showing humans AND agents
- [ ] Document any AI-assisted issue creation features
- [ ] Map the description editor capabilities (markdown, mentions, attachments?)

### PHASE 9: Homepage / Landing Page
**URL:** `https://multica.ai`

- [ ] Screenshot the full landing page (hero, features, CTA)
- [ ] Map the hero section content and design
- [ ] Document the features section (Teammates, Autonomous, Skills, Runtimes)
- [ ] Map the "Works with" agent list (Claude Code, Codex, Gemini CLI, OpenClaw, OpenCode)
- [ ] Map all CTA buttons and navigation
- [ ] Download the hero image and feature images

---

## 🎨 DESIGN SYSTEM EXTRACTION

For EVERY screen, extract and document:

### Colors
- [ ] Primary background color (light & dark mode)
- [ ] Surface/card colors
- [ ] Border colors
- [ ] Text colors (primary, secondary, muted)
- [ ] Accent/brand color
- [ ] Status colors (info/blue, warning/amber, success/green, error/red)
- [ ] Agent-specific colors (bot icon badge color)

### Typography
- [ ] Font families (3 detected — identify each: sans, serif, mono)
- [ ] Font sizes for each element type
- [ ] Font weights used
- [ ] Letter-spacing values

### Components (shadcn/ui based)
- [ ] Button variants (primary, secondary, ghost, outline, destructive)
- [ ] Input fields
- [ ] Select/dropdown
- [ ] Dialog/modal
- [ ] Toast notifications
- [ ] Avatar component (with initials fallback + bot icon variant)
- [ ] Badge/tag component
- [ ] Skeleton loading states
- [ ] Breadcrumb navigation
- [ ] Sidebar navigation
- [ ] Tables/data grids
- [ ] Status icons (custom SVG pie-chart status indicators)
- [ ] Priority icons (bar chart style)

---

## 📁 OUTPUT STRUCTURE

```
docs/multica-reference/
├── MULTICA_PRD.md                 ← Complete PRD (the main deliverable)
├── AGENT_MAPPING_PROMPT.md        ← This file (reference)
├── screenshots/
│   ├── 01-landing/                ← Homepage screenshots
│   ├── 02-issues/                 ← Issues list + detail + modals
│   ├── 03-projects/               ← Projects views
│   ├── 04-configure-skills/       ← Skills configuration
│   ├── 05-settings/               ← All settings tabs
│   ├── 06-inbox/                  ← Inbox views
│   ├── 07-my-issues/              ← My Issues view
│   ├── 08-search/                 ← Search interface
│   ├── 09-new-issue/              ← Issue creation
│   └── 10-misc/                   ← Any other screens
├── videos/                        ← Browser recordings of flows
├── assets/                        ← Downloaded images, SVGs, icons
└── design-tokens/                 ← Extracted CSS/design tokens
```

---

## 🔑 AUTHENTICATION NOTE

The URL `https://multica.ai/navy-seals/inbox` loads as a **demo workspace** with pre-populated data. The `navy-seals` slug is a public demo. If you encounter a login wall:

1. Try the URL directly — it may be a public demo
2. If login required, screenshot the login flow itself (it's part of the mapping)
3. Check if there's a "Try demo" or "Get started" button on the landing page
4. Document the auth flow: Email OTP + Google OAuth (discovered in i18n keys)

---

## ⚡ AGENT COORDINATION NOTES

### For the Lead Agent
- Start with PHASE 1 (Issues) — it's the core feature
- Don't move to the next phase until the current one is 100% documented
- After each phase, update the PRD immediately
- Take screenshots at MAXIMUM resolution (1920x1080+)

### For Support Agents
- One agent per phase is ideal for parallelism
- Each agent should create their screenshots in the numbered subfolder
- Merge all findings into the single `MULTICA_PRD.md`

### Quality Checklist Per Screen
- [ ] Full-page screenshot (no cropping)
- [ ] All interactive elements clicked and states documented
- [ ] Hover states noted where visible
- [ ] Empty states documented (what shows when there's no data?)
- [ ] Error states documented (if accessible)
- [ ] Loading states documented (skeleton screens)
- [ ] Mobile responsive view (resize to 375px width) — screenshot

---

## 🏛️ PRD TEMPLATE — Fill This Structure

The final `MULTICA_PRD.md` should follow this structure:

```markdown
# PRD — Multica.ai — Complete Product Mapping

## 1. Product Overview
## 2. Technology Stack (AS-IS)
## 3. Architecture Diagram (Mermaid)
## 4. Design System
   ### 4.1 Colors
   ### 4.2 Typography
   ### 4.3 Components
   ### 4.4 Icons
## 5. Navigation & Information Architecture
## 6. Feature Mapping
   ### 6.1 Issues (screenshots + behavior)
   ### 6.2 Projects
   ### 6.3 Skills
   ### 6.4 Settings
   ### 6.5 Inbox
   ### 6.6 My Issues
   ### 6.7 Search
   ### 6.8 New Issue
   ### 6.9 Landing Page
## 7. Agent-Specific Features
   ### 7.1 Agent Profiles
   ### 7.2 Agent Assignment
   ### 7.3 Agent Activity Monitoring
   ### 7.4 Agent Skills System
   ### 7.5 Agent Task Lifecycle
## 8. Integrations
   ### 8.1 GitHub Integration
   ### 8.2 Lark/Feishu Integration
   ### 8.3 API & CLI
## 9. Real-time Features
   ### 9.1 WebSocket Streams
   ### 9.2 Live Agent Status
   ### 9.3 SSE/Polling
## 10. i18n & Accessibility
## 11. TO-BE Architecture Recommendations
## 12. Asset Inventory
## 13. Design Replication Checklist
```

---

## 🎯 SUCCESS CRITERIA

The PRD is DONE when:
1. ✅ Every menu item has been clicked and screenshotted
2. ✅ Every button, dropdown, and modal has been documented
3. ✅ Every settings tab has been fully mapped
4. ✅ The issue detail view is documented field-by-field
5. ✅ The agent-specific features (assignment, monitoring, skills) are documented
6. ✅ The design system is extracted (colors, fonts, components)
7. ✅ Technology stack is confirmed and documented
8. ✅ Architecture diagram is created
9. ✅ All screenshots are saved in organized folders
10. ✅ The PRD document is complete and self-contained
11. ✅ An engineering team could replicate the product from the PRD alone

---

**END OF PROMPT — EXECUTE WITH MAXIMUM PRECISION**
