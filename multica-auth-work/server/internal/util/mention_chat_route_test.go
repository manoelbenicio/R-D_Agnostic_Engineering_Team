package util

import "testing"

// TestFirstAgentMention pins the parsing half of the chat direct-to-agent
// escape hatch. It is deliberately in util (no database, no HTTP) so the
// routing rule stays executable even where the DB-backed handler suite cannot
// run; the handler tests cover the workspace/access half.
func TestFirstAgentMention(t *testing.T) {
	const (
		codex = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
		kiro  = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	)

	tests := []struct {
		name    string
		content string
		want    string
		wantOK  bool
	}{
		{
			name:    "untargeted message keeps default routing",
			content: "please review the deploy wrapper",
			want:    "",
			wantOK:  false,
		},
		{
			name:    "direct agent mention",
			content: "[@Codex](mention://agent/" + codex + ") take this one",
			want:    codex,
			wantOK:  true,
		},
		{
			name:    "agent mention mid-sentence",
			content: "hey [@Codex](mention://agent/" + codex + "), can you look?",
			want:    codex,
			wantOK:  true,
		},
		{
			name:    "first agent wins over later agents",
			content: "[@Codex](mention://agent/" + codex + ") please do it, cc [@Kiro](mention://agent/" + kiro + ")",
			want:    codex,
			wantOK:  true,
		},
		{
			name:    "agent chosen regardless of position among other mention types",
			content: "[MUL-54](mention://issue/" + kiro + ") ping [@Team](mention://squad/" + kiro + ") and [@Codex](mention://agent/" + codex + ")",
			want:    codex,
			wantOK:  true,
		},
		{
			name:    "member mention is not a routing target",
			content: "[@Bob](mention://member/" + codex + ") please confirm",
			want:    "",
			wantOK:  false,
		},
		{
			name:    "squad mention does not bypass the TL",
			content: "[@Workspace Team](mention://squad/" + codex + ") please pick this up",
			want:    "",
			wantOK:  false,
		},
		{
			name:    "issue mention is context, not a target",
			content: "context: [MUL-54](mention://issue/" + codex + ")",
			want:    "",
			wantOK:  false,
		},
		{
			name:    "all mention does not target an agent",
			content: "[@All members](mention://all/all) standup in 5",
			want:    "",
			wantOK:  false,
		},
		{
			name:    "plain text @codex is not a mention link",
			content: "@codex can you take this?",
			want:    "",
			wantOK:  false,
		},
		{
			name:    "agent label with square brackets",
			content: "[@Codex[5.6]](mention://agent/" + codex + ") go",
			want:    codex,
			wantOK:  true,
		},
		{
			// The editor escapes brackets in labels when it serializes a
			// mention node ("Codex[5.6]" → "Codex\[5.6\]"), so this is the
			// exact string the chat composer sends for such an agent.
			name:    "agent label with escaped brackets as the editor emits it",
			content: `[@Codex\[5.6\]](mention://agent/` + codex + `) go`,
			want:    codex,
			wantOK:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := FirstAgentMention(tt.content)
			if ok != tt.wantOK || got != tt.want {
				t.Fatalf("FirstAgentMention(%q) = (%q, %v), want (%q, %v)", tt.content, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
