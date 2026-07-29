package util

import "regexp"

// Mention represents a parsed @mention from markdown content.
type Mention struct {
	Type string // "member", "agent", "issue", or "all"
	ID   string // user_id, agent_id, issue_id, or "all"
}

// MentionRe matches [@Label](mention://type/id) or [Label](mention://issue/id) in markdown.
// The @ prefix is optional to support issue mentions which use [MUL-123](mention://issue/...).
// Uses .+? (non-greedy) instead of [^\]]* so labels containing square brackets
// (e.g. "David[TF]") are matched correctly — the ](mention:// anchor is specific
// enough to prevent over-matching.
var MentionRe = regexp.MustCompile(`\[@?(.+?)\]\(mention://(member|agent|squad|issue|all)/([0-9a-fA-F-]+|all)\)`)

// IsMentionAll returns true if the mention is an @all mention.
func (m Mention) IsMentionAll() bool {
	return m.Type == "all"
}

// ParseMentions extracts deduplicated mentions from markdown content.
func ParseMentions(content string) []Mention {
	matches := MentionRe.FindAllStringSubmatch(content, -1)
	seen := make(map[string]bool)
	var result []Mention
	for _, m := range matches {
		key := m[2] + ":" + m[3]
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, Mention{Type: m[2], ID: m[3]})
	}
	return result
}

// HasMentionAll returns true if any mention in the slice is an @all mention.
func HasMentionAll(mentions []Mention) bool {
	for _, m := range mentions {
		if m.IsMentionAll() {
			return true
		}
	}
	return false
}

// FirstAgentMention returns the id of the first agent addressed in the content,
// in document order.
//
// Chat uses it for the direct-to-agent escape hatch: an untargeted chat turn
// goes to the session's agent (the default Squad TL), while a message writing
// `[@Codex](mention://agent/<id>)` is addressed to that agent and must reach it
// without TL interception.
//
// Only `mention://agent/...` counts. Member, squad, issue and @all mentions are
// addressing/context markup inside a message that is still directed at the
// chat's own agent, and squad mentions would re-introduce the very TL hop this
// hatch bypasses. When several agents are mentioned the first one wins: a chat
// turn produces exactly one run, and document order matches how people address
// a message ("@codex can you …, cc @kiro").
func FirstAgentMention(content string) (string, bool) {
	for _, m := range ParseMentions(content) {
		if m.Type == "agent" {
			return m.ID, true
		}
	}
	return "", false
}
