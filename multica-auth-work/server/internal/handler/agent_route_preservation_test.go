package handler

import (
	"strings"
	"testing"
)

func TestResolveRuntimeRouteUpdate(t *testing.T) {
	replacement := "openai/gpt-replacement"
	emptyReplacement := ""
	nativeReplacement := "gpt-replacement"
	tests := []struct {
		name            string
		route           string
		currentProvider string
		targetProvider  string
		replacement     *string
		wantPreserve    bool
		wantError       string
	}{
		{name: "compatible runtime preserves exact slash route", route: "agy/claude-opus-4-6-thinking", currentProvider: "claude", targetProvider: "claude", wantPreserve: true},
		{name: "incompatible CLI change requires atomic reselection", route: "agy/claude-opus-4-6-thinking", currentProvider: "claude", targetProvider: "codex", wantError: "replacement model atomically"},
		{name: "explicit model replacement owns selection", route: "agy/claude-opus-4-6-thinking", currentProvider: "claude", targetProvider: "codex", replacement: &replacement},
		{name: "explicit empty cannot erase gateway route", route: "agy/claude-opus-4-6-thinking", currentProvider: "claude", targetProvider: "codex", replacement: &emptyReplacement, wantError: "exact replacement RouteModel"},
		{name: "explicit native ID cannot downgrade gateway route", route: "agy/claude-opus-4-6-thinking", currentProvider: "claude", targetProvider: "codex", replacement: &nativeReplacement, wantError: "exact replacement RouteModel"},
		{name: "empty prior route needs no preservation", currentProvider: "claude", targetProvider: "codex"},
		{name: "native prior model keeps native policy", route: "claude-sonnet-4-6", currentProvider: "claude", targetProvider: "codex"},
		{name: "provider slash ID is preserved without reinterpretation", route: "kimi/k2.5", currentProvider: "claude", targetProvider: "claude", wantPreserve: true},
		{name: "non-exact slash ID fails closed", route: " kimi/k2.5", currentProvider: "claude", targetProvider: "claude", wantError: "exact RouteModel"},
		{name: "unknown target CLI fails closed", route: "provider/model", currentProvider: "claude", targetProvider: "custom", wantError: "replacement model atomically"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			preserve, err := resolveRuntimeRouteUpdate(test.route, test.currentProvider, test.targetProvider, test.replacement)
			if preserve != test.wantPreserve {
				t.Fatalf("preserve = %v, want %v", preserve, test.wantPreserve)
			}
			if test.wantError == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if test.wantError != "" && (err == nil || !strings.Contains(err.Error(), test.wantError)) {
				t.Fatalf("error = %v, want substring %q", err, test.wantError)
			}
		})
	}
}
