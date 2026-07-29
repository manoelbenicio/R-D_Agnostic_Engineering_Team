import {
  RUNTIME_PROFILE_PROTOCOL_FAMILIES,
  type RuntimeProfile,
  type RuntimeProtocolFamily,
} from "@multica/core/types";

// A single row in the runtimes catalog the management dialog renders: the
// built-in protocol families ship as read-only reference rows, while custom
// profiles are the user's editable assets.
export type RuntimeCatalogEntry =
  | {
      kind: "builtin";
      // Stable row id — the protocol family doubles as the key for built-ins.
      id: string;
      protocolFamily: RuntimeProtocolFamily;
    }
  | {
      kind: "custom";
      id: string;
      protocolFamily: RuntimeProtocolFamily;
      profile: RuntimeProfile;
    };

export interface RuntimeCatalogSections {
  customs: RuntimeCatalogEntry[];
  builtins: RuntimeCatalogEntry[];
}

// Re-export the whitelist as a typed array so callers (the family picker,
// the catalog builder) share the single source of truth.
export const PROTOCOL_FAMILIES: readonly RuntimeProtocolFamily[] =
  RUNTIME_PROFILE_PROTOCOL_FAMILIES;

// buildRuntimeCatalog keeps user-owned custom profiles separate from built-in
// protocol families. The dialog renders customs as the primary management
// surface and built-ins as a collapsed reference section.
export function buildRuntimeCatalog(
  profiles: RuntimeProfile[],
): RuntimeCatalogSections {
  const builtins: RuntimeCatalogEntry[] = PROTOCOL_FAMILIES.map((family) => ({
    kind: "builtin" as const,
    id: `builtin:${family}`,
    protocolFamily: family,
  }));

  const customs: RuntimeCatalogEntry[] = [...profiles]
    .sort((a, b) => {
      if (a.enabled !== b.enabled) return a.enabled ? -1 : 1;
      const aTime = Date.parse(a.updated_at) || 0;
      const bTime = Date.parse(b.updated_at) || 0;
      if (aTime !== bTime) return bTime - aTime;
      return a.display_name.localeCompare(b.display_name, undefined, {
        sensitivity: "base",
      });
    })
    .map((profile) => ({
      kind: "custom" as const,
      id: profile.id,
      protocolFamily: profile.protocol_family,
      profile,
    }));

  return { customs, builtins };
}

// NOTE: `fixed_args` is intentionally NOT exposed in the v1 UI. The server
// still carries the column, but the daemon does not yet splice these args into
// the agent launch command, so surfacing an input/display here would promise
// admins a behavior that does not exist. Re-introduce the parse/format helpers
// and the form field only once the daemon actually passes them to the backend
// (proven by a test). See TODO(MUL-3284) in server/internal/daemon/daemon.go.

export interface ProfileFormValues {
  displayName: string;
  commandName: string;
  description: string;
}

// Per-family create defaults used to pre-fill the "Create from this" flow so
// admins never face empty mystery fields. `commandName` MIRRORS the command
// skeleton the daemon spawns (server/pkg/agent/agent.go `launchHeaders`), with
// the trailing transport/mode parenthetical stripped so the value is a real
// command the admin can extend — e.g. launchHeaders "claude (stream-json)"
// becomes "claude", "agy -p (print mode)" becomes "agy -p". Families whose
// launch header is a native (non-CLI) backend have NO command skeleton, so the
// command is left empty rather than inventing one (`nim` = native HTTP). Keep
// this in exact sync with launchHeaders; never invent a command.
export const RUNTIME_PROFILE_FORM_DEFAULTS: Record<
  RuntimeProtocolFamily,
  { displayName: string; commandName: string }
> = {
  claude: { displayName: "Claude", commandName: "claude" },
  codebuddy: { displayName: "CodeBuddy", commandName: "codebuddy" },
  cline: { displayName: "Cline", commandName: "cline --acp" },
  codex: { displayName: "Codex", commandName: "codex app-server" },
  copilot: { displayName: "Copilot", commandName: "copilot" },
  // Native HTTP backend — no CLI launch skeleton in launchHeaders, so no
  // command is pre-filled (never invented). The admin supplies one.
  nim: { displayName: "NVIDIA NIM", commandName: "" },
  opencode: { displayName: "OpenCode", commandName: "opencode run" },
  openclaw: { displayName: "OpenClaw", commandName: "openclaw agent" },
  hermes: { displayName: "Hermes", commandName: "hermes acp" },
  gemini: { displayName: "Gemini", commandName: "gemini" },
  pi: { displayName: "Pi", commandName: "pi" },
  cursor: { displayName: "Cursor", commandName: "cursor-agent" },
  kimi: { displayName: "Kimi", commandName: "kimi acp" },
  kiro: { displayName: "Kiro", commandName: "kiro-cli acp" },
  antigravity: { displayName: "Antigravity", commandName: "agy -p" },
};

// Builds the initial create-form values for a chosen base family. Unknown
// families (should be impossible given the closed union) fall back to empty
// fields so nothing is fabricated.
export function createProfileFormDefaults(
  family: RuntimeProtocolFamily,
): ProfileFormValues {
  const defaults = RUNTIME_PROFILE_FORM_DEFAULTS[family];
  return {
    displayName: defaults?.displayName ?? "",
    commandName: defaults?.commandName ?? "",
    description: "",
  };
}

export type ProfileFormErrorField = "displayName" | "commandName";

// Pure, synchronous validation for the create/edit form. Returns the set of
// invalid fields (empty = valid). Display name and command name are the only
// hard-required fields; description and fixed args are optional.
export function validateProfileForm(
  values: ProfileFormValues,
): ProfileFormErrorField[] {
  const errors: ProfileFormErrorField[] = [];
  if (!values.displayName.trim()) errors.push("displayName");
  if (!values.commandName.trim()) errors.push("commandName");
  return errors;
}

// Returns true when the entry should be treated as a built-in (read-only).
export function isBuiltinEntry(entry: RuntimeCatalogEntry): boolean {
  return entry.kind === "builtin";
}
