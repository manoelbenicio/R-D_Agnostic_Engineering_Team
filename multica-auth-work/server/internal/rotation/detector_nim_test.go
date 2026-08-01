package rotation

import (
	"testing"
	"time"
)

func TestNIMMatcherDetectsExhaustion(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name string
		text string
		want bool
	}{
		{name: "native backend 429", text: "NIM API returned 429 Too Many Requests", want: true},
		{name: "rate limit retry", text: "Rate limit exceeded. Please retry after 30 seconds.", want: true},
		{name: "credits exhausted", text: "Credits exhausted; try again after cooldown.", want: true},
		{name: "quota reset", text: "Quota exceeded. Resets at 2pm.", want: true},
		{name: "limit only", text: "rate limit exceeded", want: false},
		{name: "normal output", text: "Implementing a rate limit test.", want: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchesNIMExhaustion(tt.text); got != tt.want {
				t.Fatalf("matchesNIMExhaustion(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestNIMMatcherIsWiredIntoDetector(t *testing.T) {
	now := time.Date(2026, 7, 14, 13, 0, 0, 0, time.UTC)
	detector := &Detector{now: func() time.Time { return now }}

	got := detector.Detect("nim", "Quota exceeded. Resets at 2pm.", 0)
	if !got.Exhausted || got.Signal != SignalScreen || got.ResetAt == nil || got.ResetAt.Hour() != 14 {
		t.Fatalf("Detect NIM = %+v, want screen exhaustion at 14:00", got)
	}
	got = detector.Detect("nim", "NVIDIA is experiencing high traffic. Try again later.", 0)
	if got.Exhausted {
		t.Fatalf("Detect high traffic = %+v, want transient", got)
	}
}
