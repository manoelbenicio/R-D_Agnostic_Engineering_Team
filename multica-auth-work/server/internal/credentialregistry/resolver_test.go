package credentialregistry

import "testing"

func TestCanonicalProvider(t *testing.T) {
	for input, want := range map[string]string{
		"agy":         "antigravity",
		"ANTIGRAVITY": "antigravity",
		" kiro ":      "kiro",
		"codex":       "codex",
		"openclaw":    "openclaw",
	} {
		if got := CanonicalProvider(input); got != want {
			t.Fatalf("CanonicalProvider(%q)=%q, want %q", input, got, want)
		}
	}
}

func TestRequiresApprovedAssignment(t *testing.T) {
	for _, provider := range []string{"agy", "antigravity", "codex", "kiro"} {
		if !RequiresApprovedAssignment(provider) {
			t.Fatalf("%s must require an approved assignment", provider)
		}
	}
	for _, provider := range []string{"claude", "openclaw", "custom"} {
		if RequiresApprovedAssignment(provider) {
			t.Fatalf("%s must not be enrolled implicitly", provider)
		}
	}
}

func TestValidAbsoluteMetadataPath(t *testing.T) {
	for _, path := range []string{"/home/ec2-user/.agent-cred-homes/slots/slot-139/home", "/var/lib/multica/account"} {
		if !validAbsoluteMetadataPath(path) {
			t.Fatalf("expected valid path %q", path)
		}
	}
	for _, path := range []string{"", ".", "relative/path", "/", "/safe/../escape"} {
		if validAbsoluteMetadataPath(path) {
			t.Fatalf("expected invalid path %q", path)
		}
	}
}
