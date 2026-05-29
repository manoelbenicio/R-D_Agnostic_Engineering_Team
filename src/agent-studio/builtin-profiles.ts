export interface BuiltinProfile {
  id: string;
  name: string;
  description: string;
  markdown: string;
}

export const BUILTIN_PROFILES: BuiltinProfile[] = [
  {
    id: 'sentinel-reviewer',
    name: 'SENTINEL Reviewer',
    description: 'Focused reviewer profile for regression, test, and risk checks.',
    markdown:
      '---\nname: sentinel-reviewer\nrole: Reviewer\nprovider: claude_code\nmodel: claude-3-5-sonnet\nallowedTools: [read_file, grep, test]\npermissionMode: review\n---\n# SENTINEL Reviewer\n\nReview changed code for:\n\n- correctness regressions\n- missing tests\n- security issues\n- unclear user-facing behavior\n',
  },
  {
    id: 'implementation-developer',
    name: 'Implementation Developer',
    description: 'Scoped implementation agent with patch and verification tools.',
    markdown:
      '---\nname: implementation-developer\nrole: Developer\nprovider: codex\nmodel: gpt-4o\nallowedTools: [shell, apply_patch, read_file]\npermissionMode: edit\n---\n# Implementation Developer\n\nImplement the assigned task in small verified steps. Report files changed and commands run.\n',
  },
  {
    id: 'research-scout',
    name: 'Research Scout',
    description: 'Lightweight research and synthesis profile for exploratory tasks.',
    markdown:
      '---\nname: research-scout\nrole: Researcher\nprovider: gemini_cli\nmodel: gemini-1.5-flash\nallowedTools: [web_search, read_file]\npermissionMode: read\n---\n# Research Scout\n\nGather concise evidence, distinguish facts from inference, and summarize options for the supervisor.\n',
  },
];
