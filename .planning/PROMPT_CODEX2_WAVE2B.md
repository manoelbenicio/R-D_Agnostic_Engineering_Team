You are CODEX-2-LEAD returning for Wave 2B — Canvas UX improvements for large monitors.

PROJECT: C:/VMs/Projetos/Automonous_Agentic
LEDGER: C:/VMs/Projetos/Automonous_Agentic/.planning/AGENT_LEDGER.md

The owner has a 32" monitor and has specifically requested the canvas expand to use maximum available space. The current layout constrains the canvas area and does not scale well on large displays.

═══════════════════════════════════════════════════════════════
MANDATORY: CHECK-IN / CHECK-OUT PROTOCOL

Before touching ANY source file:
1. Read: .planning/AGENT_LEDGER.md
2. Verify no other agent has the file checked-in (STOP if locked)
3. Add CHECK-IN row: | <UTC timestamp> | <agent-name> | CHECK-IN | <file> | 🔵 IN PROGRESS | <note> |
4. Update File Lock Table: add file entry with agent as owner, 🔴 Locked

After completing:
5. Add CHECK-OUT row with ✅ DONE
6. Update File Lock Table: clear owner, set 🟢 Available
═══════════════════════════════════════════════════════════════


══════════════════════════════════════
TASK C2-D: Canvas Full-Screen Expansion
Agent name for ledger: C2-D
══════════════════════════════════════

FILES:
- C:/VMs/Projetos/Automonous_Agentic/src/canvas-builder/canvas-builder.css
- C:/VMs/Projetos/Automonous_Agentic/src/canvas-builder/CanvasBuilderPage.tsx

Read both files FULLY first. Also read:
- C:/VMs/Projetos/Automonous_Agentic/src/index.css (layout variables)
- C:/VMs/Projetos/Automonous_Agentic/src/shell/NavBar.tsx (navbar height reference)

Changes:
1. Make the canvas area fill 100% of the viewport below the navbar
2. Remove any max-width constraints on the canvas container
3. Use calc(100vh - navbar_height) for the canvas wrapper height
4. Ensure the React Flow canvas container uses width: 100% and height: 100%
5. Add a CSS class for fullscreen toggle mode that removes sidebar padding
6. Make the Block Configuration Panel an overlay/floating panel rather than a fixed-width sidebar that steals canvas space
7. Ensure canvas zoom/pan works smoothly at large viewport sizes (2560x1440 and above)
8. Test that the minimap scales properly on large canvases


══════════════════════════════════════
TASK C2-E: Canvas Toolbar Enhancements
Agent name for ledger: C2-E
══════════════════════════════════════

FILES:
- C:/VMs/Projetos/Automonous_Agentic/src/canvas-builder/CanvasBuilderPage.tsx
- C:/VMs/Projetos/Automonous_Agentic/src/canvas-builder/canvas-builder.css

Read CanvasBuilderPage.tsx FULLY. Then add:

1. A floating toolbar at the top of the canvas with:
   - [Fit View] button — calls reactFlowInstance.fitView()
   - [Zoom In] / [Zoom Out] buttons
   - [Fullscreen] toggle — expands canvas to cover entire viewport (hides sidebar)
   - Zoom percentage display (e.g., "125%")
   
2. CSS for the floating toolbar:
   - Position: absolute, top-right corner of the canvas
   - Glassmorphism style (backdrop-filter: blur, semi-transparent)
   - Matches existing dark theme
   - z-index above canvas but below modals
   - Compact pill-shaped buttons

3. Keyboard shortcuts:
   - Ctrl+Shift+F → toggle fullscreen canvas
   - Ctrl+0 → fit view
   - Ctrl+= → zoom in
   - Ctrl+- → zoom out


══════════════════════════════════════
TASK C2-F: Add Session Dialog
Agent name for ledger: C2-F
══════════════════════════════════════

CREATE NEW FILES:
- C:/VMs/Projetos/Automonous_Agentic/src/sessions/AddSessionDialog.tsx
- C:/VMs/Projetos/Automonous_Agentic/src/sessions/add-session-dialog.css

Read first:
- C:/VMs/Projetos/Automonous_Agentic/src/sessions/SessionsPage.tsx (to understand the [+ Add Session] button)
- C:/VMs/Projetos/Automonous_Agentic/src/api/session-store.ts (addSession action)
- C:/VMs/Projetos/Automonous_Agentic/src/index.css (design tokens)

Create a modal dialog for adding a new session:

AddSessionDialog.tsx:
- Props: { isOpen: boolean; onClose: () => void; defaultProvider?: string }
- Provider selector: dropdown with Claude Code, Codex, Gemini CLI, Kiro CLI
- Config directory input (optional): text field for custom config dir path
- Billing label input (optional): friendly name like "Team A Account"
- [Start OAuth Login] button — calls useSessionStore().addSession(provider, configDir)
- [Cancel] button — closes dialog
- Loading state during login flow
- Success/error feedback

add-session-dialog.css:
- Modal overlay with backdrop blur
- Centered card with glassmorphism
- Form fields matching app design system
- Smooth open/close animation (opacity + scale transform)

Then update SessionsPage.tsx:
- Import AddSessionDialog
- Add state: const [showAddDialog, setShowAddDialog] = useState(false)
- Wire the existing [+ Add Session] buttons to open the dialog
- Render <AddSessionDialog isOpen={showAddDialog} onClose={() => setShowAddDialog(false)} />

NOTE: Check the ledger — if C2-A or another agent has SessionsPage.tsx locked, WAIT.


══════════════════════════════════════
QUALITY GATE
Agent name for ledger: CODEX-2-LEAD
══════════════════════════════════════

After all 3 tasks complete:
1. Run: npx tsc --noEmit → 0 errors
2. Run: npx vitest run → all tests pass
3. CHECK-OUT as CODEX-2-LEAD with ✅ DONE and "Wave 2B Gate PASSED"