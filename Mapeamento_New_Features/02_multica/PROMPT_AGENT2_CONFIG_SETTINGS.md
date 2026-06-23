# 🎯 AGENT 2 — Configuration: Settings + Skills + Search

> **Role:** Senior Product Mapper — Configuration & Search
> **Scope:** PHASE 3 (Skills), PHASE 4 (Settings — ALL tabs), PHASE 7 (Search)
> **Coordination:** You are Agent 2 of 3. Agent 1 handles Issues/Projects/New Issue. Agent 3 handles Inbox/My Issues/Landing/Design System.
> **Output Path:** `C:\VMs\Projetos\Automonous_Agentic\Mapeamento_New_Features\02_multica\`

---

## CONTEXT — What is Multica?

**Multica** (https://multica.ai) is an open-source project management platform for Human + Agent teams. It turns coding agents (Claude Code, Codex, Gemini CLI) into real teammates — assign issues, track progress, compound skills.

**Tech Stack:** Next.js App Router, Vercel, Tailwind CSS, shadcn/ui, Lucide icons, i18next (en/zh/ko/ja), WebSocket, next-themes (light/dark/system)

**Demo Workspace URL Base:** `https://multica.ai/navy-seals/`

If login wall appears: screenshot the login flow (Email OTP + Google OAuth), then try "Get started" or demo access from the landing page.

---

## 🛠️ RECOMMENDED SKILLS & TOOLS

Use these for maximum efficiency and quality:

### Browser & Capture
- **Chrome DevTools MCP** — Use `take_screenshot` for pixel-perfect captures, `evaluate_script` to extract form field properties and computed styles, `click` to interact with toggles/buttons, `take_snapshot` for DOM accessibility tree
- **Browser Subagent** — Use for multi-step flows: navigate to settings tab → screenshot → toggle setting → screenshot result → navigate next tab
- Use `evaluate_script` to extract form validation rules, toggle states, and configuration values

### Analysis & Documentation
- **gsd-ui-review** — Run the 6-pillar visual audit on the Settings page to document layout patterns (form fields, section separators, danger zones)
- **gsd-ui-phase** — Use to generate UI-SPEC.md for the Settings/Configuration component architecture
- **gsd-sketch** — Create annotated wireframes of the Skills system UI if complex

### Codebase Intelligence (if GitHub repo accessible)
- **gsd-map-codebase** — Multica is open-source at `https://github.com/multica-ai/multica`. If accessible, analyze `/app/[workspaceSlug]/(dashboard)/settings/` and `/configure/skills/` component trees
- Check repo for: settings schema, skill model definition, GitHub integration handlers

### Extraction Helpers
```javascript
// Extract all form field labels and placeholders from a settings tab
Array.from(document.querySelectorAll('label, input, textarea, select')).map(el => ({
  tag: el.tagName, label: el.labels?.[0]?.textContent, placeholder: el.placeholder,
  value: el.value, type: el.type, name: el.name
}));

// Extract all toggle/switch states
Array.from(document.querySelectorAll('[role="switch"], [data-state]')).map(el => ({
  label: el.closest('[class]')?.textContent?.substring(0, 80),
  state: el.dataset.state || el.getAttribute('aria-checked')
}));

// Extract all tab navigation items
Array.from(document.querySelectorAll('[role="tab"], nav a, [data-tab]')).map(t => ({
  text: t.textContent.trim(), href: t.href, active: t.getAttribute('aria-selected')
}));
```

---

## YOUR PHASES

### PHASE 3: Configure — Skills ⭐
**URL:** `https://multica.ai/navy-seals/configure/skills` (or navigate via sidebar Configure → Skills)

Skills are Multica's system for giving agents **compound abilities**. This is a KEY differentiator. Map it exhaustively:

- [ ] Screenshot the Skills listing page (empty state AND populated state)
- [ ] Map what a "Skill" consists of — what fields/properties does it have?
  - Name
  - Description
  - Instructions (prompt content?)
  - File patterns / globs
  - Associated agent(s)
  - Category/type
  - Enabled/disabled toggle
- [ ] Map skill creation flow — click "New Skill" or equivalent:
  - Screenshot the creation form/dialog
  - Document every field, placeholder, validation
  - Is there a rich text / markdown editor for instructions?
- [ ] Map skill editing flow — click an existing skill to edit
- [ ] Map skill deletion flow — screenshot confirmation dialog
- [ ] Map skill assignment — how are skills connected to specific agents?
- [ ] Look for skill templates / marketplace / library — any pre-built skills?
- [ ] Map skill categories or grouping (if any)
- [ ] Map any skill testing/preview functionality
- [ ] Document how skills interact with the issue system (does an agent use skills when working on issues?)
- [ ] Check for skill import/export functionality

---

### PHASE 4: Settings (ALL 10 TABS) ⭐⭐
**URL:** `https://multica.ai/navy-seals/settings`

Navigate to Settings. Screenshot the settings page layout first (sidebar tabs + content area). Then go through EVERY tab:

#### Tab 1: General
- [ ] Screenshot the full General settings page
- [ ] Document all fields:
  - Workspace name (input)
  - Workspace description (textarea — placeholder text?)
  - Workspace context (textarea — "Background information for AI agents")
  - Slug (input — URL path)
  - Issue prefix (input — e.g., "MUL" for MUL-123)
  - Workspace logo (upload area)
- [ ] Map the "Danger Zone" section:
  - Leave workspace button + confirmation dialog
  - Delete workspace button + confirmation dialog (type workspace name to confirm)
- [ ] Note which fields require admin/owner role
- [ ] Screenshot the save/update button states (idle, saving, saved)

#### Tab 2: Members
- [ ] Screenshot the members list
- [ ] Document member properties shown: avatar, name, email, role, join date
- [ ] Map the invite flow — click invite button, screenshot the invite dialog
- [ ] Map role management — click a member's role, screenshot the role dropdown (Owner, Admin, Member)
- [ ] **CRITICAL:** Document the visual distinction between HUMAN members and AGENT members
  - Do agents appear in the same list?
  - Is there a bot icon badge?
  - Different avatar style?
- [ ] Map member removal flow

#### Tab 3: Repositories
- [ ] Screenshot the repositories list (empty + populated)
- [ ] Map "Add repository" flow — screenshot the form
- [ ] Document fields: URL input (placeholder text), Description textarea
- [ ] Map repository editing (inline? dialog?)
- [ ] Map repository deletion (confirmation?)
- [ ] Note: these repos are used by agents to clone code

#### Tab 4: GitHub
- [ ] Screenshot the GitHub integration page
- [ ] Document the connection status (connected vs not connected)
- [ ] Map "Connect GitHub" flow — what happens on click?
- [ ] If connected, document:
  - Connected organization/account name
  - "Connected by" attribution
  - Disconnect button + confirmation dialog
- [ ] Map feature toggles (each has label + description + toggle):
  - **PR sidebar** — "Show linked pull requests inside the issue detail sidebar"
  - **Co-authored-by trailer** — "Append Co-authored-by to commits by agents"
  - **Auto-link issues and PRs** — "Match PR titles/bodies/branches against issue identifiers"
- [ ] Map the "Enable GitHub features" master toggle
- [ ] Map the repositories shortcut link

#### Tab 5: Integrations (Lark/Feishu)
- [ ] Screenshot the integrations page
- [ ] Document the Lark bot binding flow:
  - "Bind to Lark" button
  - QR code scanning dialog
  - Connected bot list
  - Disconnect flow
- [ ] Note: supports both Lark (international) and Feishu (mainland China)
- [ ] Map the bot management interface (connected bots list, status, disconnect)

#### Tab 6: Profile (Personal)
- [ ] Screenshot the profile settings page
- [ ] Document fields:
  - Avatar upload (click to change)
  - Name input
  - "About you" description — this is the field shared with agents, document:
    - Placeholder text ("e.g. Backend engineer (Go + Postgres). Prefer terse PRs...")
    - Character limit (max noted in i18n)
    - Purpose explanation text
- [ ] Map save/update flow

#### Tab 7: Preferences
- [ ] Screenshot the preferences page
- [ ] Map the Theme toggle:
  - Light / Dark / System options
  - Screenshot the selector UI
  - Switch between themes and screenshot each
- [ ] Map the Language selector:
  - Options: English, 中文, 한국어, 日本語
  - Screenshot the dropdown
- [ ] Map the Timezone selector:
  - Document the "(browser)" suffix behavior
  - Screenshot the timezone picker

#### Tab 8: Notifications
- [ ] Screenshot the notifications settings page
- [ ] Map each notification category toggle:
  - **Assignments** — "When you are assigned or unassigned from an issue"
  - **Status changes** — "When an issue you follow changes status"
  - **Comments & Mentions** — "New comments on issues you follow, or @mentions"
  - **Priority & Due date** — "When priority or due date changes"
  - **Agent activity** — "When an agent task completes or fails"
- [ ] Map System Notifications section:
  - "Show system notifications" toggle + hint text
- [ ] Map Browser Notifications section:
  - Enable button, granted/denied states

#### Tab 9: API Tokens
- [ ] Screenshot the tokens page (empty + populated)
- [ ] Map token creation:
  - Name input (placeholder: "Token name (e.g. My CLI)")
  - Expiry selector: 30 days, 90 days, 1 year, No expiry
  - Create button
  - **Token display dialog** after creation (copy token, "you won't see it again")
- [ ] Map token list display: masked prefix, created date, last used date, expiry
- [ ] Map token revocation: revoke button → confirmation dialog

#### Tab 10: Labs
- [ ] Screenshot the Labs page
- [ ] Document: currently shows "No experiments yet" placeholder (per i18n keys)
- [ ] Note the placeholder description text

---

### PHASE 7: Search
**URL:** `https://multica.ai/navy-seals/search` (or trigger via `Cmd+K` / sidebar Search icon)

- [ ] Determine search type: dedicated page? modal/overlay? command palette?
- [ ] Screenshot the search interface at rest (empty state)
- [ ] Type a search query — screenshot results as they appear
- [ ] Map result types: issues, projects, members, agents, settings?
- [ ] Map search filters / facets (if any)
- [ ] Map search result display format (icon, title, metadata per result type)
- [ ] Test keyboard navigation within search results
- [ ] Map the keyboard shortcut to invoke search (Cmd+K? /)
- [ ] Document any recent searches / search history
- [ ] Map any autocomplete / suggestions behavior

---

## OUTPUT REQUIREMENTS

### Screenshots — save to:
```
C:\VMs\Projetos\Automonous_Agentic\Mapeamento_New_Features\02_multica\screenshots\
├── 04-configure-skills/
│   ├── skills-list-empty.png
│   ├── skills-list-populated.png
│   ├── skill-creation-form.png
│   ├── skill-detail-edit.png
│   ├── skill-deletion-confirm.png
│   └── skill-assignment.png
├── 05-settings/
│   ├── settings-general.png
│   ├── settings-general-danger-zone.png
│   ├── settings-members-list.png
│   ├── settings-members-invite.png
│   ├── settings-members-role-dropdown.png
│   ├── settings-repositories.png
│   ├── settings-repositories-add.png
│   ├── settings-github-connected.png
│   ├── settings-github-features.png
│   ├── settings-integrations-lark.png
│   ├── settings-profile.png
│   ├── settings-preferences-theme.png
│   ├── settings-preferences-language.png
│   ├── settings-preferences-timezone.png
│   ├── settings-notifications.png
│   ├── settings-tokens-list.png
│   ├── settings-tokens-create.png
│   ├── settings-tokens-created-dialog.png
│   ├── settings-tokens-revoke-confirm.png
│   └── settings-labs.png
└── 08-search/
    ├── search-empty.png
    ├── search-results.png
    ├── search-filters.png
    └── search-keyboard-nav.png
```

### Deliverable — PRD Section
Write your findings into:
`C:\VMs\Projetos\Automonous_Agentic\Mapeamento_New_Features\02_multica\PRD_AGENT2_CONFIG_SETTINGS.md`

Structure:
```markdown
# PRD — Agent 2: Configuration (Skills, Settings, Search)

## 1. Skills System
### 1.1 Skills Overview
### 1.2 Skill Properties & Schema
### 1.3 Skill Creation Flow
### 1.4 Skill-Agent Assignment
### 1.5 Skill Templates/Library

## 2. Settings — General
## 3. Settings — Members
## 4. Settings — Repositories
## 5. Settings — GitHub Integration
## 6. Settings — Integrations (Lark)
## 7. Settings — Profile
## 8. Settings — Preferences
## 9. Settings — Notifications
## 10. Settings — API Tokens
## 11. Settings — Labs

## 12. Search
### 12.1 Search Interface
### 12.2 Search Result Types
### 12.3 Keyboard Shortcuts

## 13. Screenshots Inventory
```

### Quality Rules
- Screenshot at MAXIMUM resolution (1920x1080+)
- Screenshot BEFORE and AFTER clicking every toggle, dropdown, and button
- Document every field label, placeholder text, hint text, and tooltip
- Capture every dialog/modal that appears
- Note which actions require admin/owner permissions
- Capture empty states and loading states

---

**END — Execute PHASE 3 first (Skills), then PHASE 4 (Settings — all tabs), then PHASE 7 (Search)**
