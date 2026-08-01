package daemon

import (
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/rotation"
)

func TestNIMExhaustionDetectorMatcher(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name string
		text string
		want bool
	}{
		{name: "native backend 429", text: `NIM API returned 429 Too Many Requests: {"detail":"slow down"}`, want: true},
		{name: "rate limit retry", text: "NVIDIA rate limit exceeded. Please retry after 30 seconds.", want: true},
		{name: "resource exhausted", text: "Resource exhausted; try again after cooldown.", want: true},
		{name: "quota depleted reset", text: "Quota depleted. Resets at 3:15 PM.", want: true},
		{name: "limit without retry", text: "rate limit exceeded", want: false},
		{name: "documented limit", text: "The rate limit is 40 requests per minute.", want: false},
		{name: "high traffic", text: "NVIDIA is experiencing high traffic. Try again later.", want: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchesNIMScreenExhaustion(tt.text); got != tt.want {
				t.Fatalf("matchesNIMScreenExhaustion(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestNIMExhaustionDetectorViaDetect(t *testing.T) {
	now := time.Date(2026, 7, 14, 13, 0, 0, 0, time.UTC)
	detector := &NIMExhaustionDetector{now: func() time.Time { return now }}

	got := detector.Detect("nim", "Quota depleted. Resets at 3:15 PM.", 0)
	if !got.Exhausted || got.Signal != rotation.SignalScreen || got.ResetAt == nil || got.ResetAt.Hour() != 15 || got.ResetAt.Minute() != 15 {
		t.Fatalf("Detect = %+v, want screen exhaustion resetting at 15:15", got)
	}
	got = detector.Detect("nim", "", 429)
	if !got.Exhausted || got.Signal != rotation.SignalHTTP429 {
		t.Fatalf("Detect HTTP 429 = %+v", got)
	}
	got = detector.Detect("nim", "NVIDIA rate limit exceeded. Retry later.", 503)
	if got.Exhausted {
		t.Fatalf("Detect HTTP 503 = %+v, want transient", got)
	}
}
