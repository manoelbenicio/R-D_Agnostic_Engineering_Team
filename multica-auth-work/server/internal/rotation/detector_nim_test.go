package rotation

import "testing"

// TestNimMatcherDetectsExhaustion covers the NIM matcher directly: OpenAI-compatible
// rate/usage/quota banners (NIM is OpenAI-compatible at integrate.api.nvidia.com/v1),
// NVIDIA/NIM-branded limit messages, and credits-exhausted — each paired with a
// reset/retry indicator. The Detector.Detect("nim", ...) contract path is exercised
// once the Kiro wiring (case "nim" in matchesVendorExhaustion, Wave 2) is applied;
// the daemon-layer NimExhaustionDetector covers the same shared flow meanwhile.
func TestNimMatcherDetectsExhaustion(t *testing.T) {
	tests := []struct {
		name       string
		screenText string
		want       bool
	}{
		{
			name:       "rate limit exceeded with try again",
			screenText: "request failed: rate limit exceeded. Please try again later.",
			want:       true,
		},
		{
			name:       "usage limit reached with reset",
			screenText: "Usage limit reached. Your limit will reset at 2pm.",
			want:       true,
		},
		{
			name:       "nim branded limit with retry",
			screenText: "You've hit your NIM limit. Retry after cooldown.",
			want:       true,
		},
		{
			name:       "nvidia branded limit with retry",
			screenText: "You've hit your NVIDIA limit. Please retry in a moment.",
			want:       true,
		},
		{
			name:       "quota exceeded with try again",
			screenText: "Quota exceeded. Please try again in 30 minutes.",
			want:       true,
		},
		{
			name:       "too many requests with back off",
			screenText: "Too many requests. Please back off and retry.",
			want:       true,
		},
		{
			name:       "credits exhausted with reset",
			screenText: "Credits exhausted. Your credits will reset at 12am.",
			want:       true,
		},
		{
			name:       "monthly usage limit reached with reset",
			screenText: "Monthly usage limit reached. Resets at 3:00 AM.",
			want:       true,
		},
		{
			name:       "limit phrase without reset indicator",
			screenText: "rate limit exceeded",
			want:       false,
		},
		{
			name:       "reset indicator without limit phrase",
			screenText: "Please try again later.",
			want:       false,
		},
		{
			name:       "normal agent output",
			screenText: "Adding tests for the NIM rotation module.",
			want:       false,
		},
		{
			name:       "bare rate limit mention is not exhaustion",
			screenText: "The endpoint rate limit is 100 requests per minute.",
			want:       false,
		},
		{
			name:       "empty screen",
			screenText: "",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesNimExhaustion(tt.screenText)
			if got != tt.want {
				t.Fatalf("matchesNimExhaustion(%q) = %v, want %v", tt.screenText, got, tt.want)
			}
		})
	}
}
