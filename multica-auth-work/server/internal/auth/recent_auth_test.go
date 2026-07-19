package auth

import (
	"context"
	"testing"
	"time"
)

func TestHasRecentAuthentication(t *testing.T) {
	now := time.Unix(1_000, 0)
	window := 5 * time.Minute
	for _, tc := range []struct {
		name string
		ctx  context.Context
		want bool
	}{
		{name: "missing", ctx: context.Background()},
		{name: "fresh", ctx: WithAuthenticationTime(context.Background(), now.Add(-window)), want: true},
		{name: "expired", ctx: WithAuthenticationTime(context.Background(), now.Add(-window-time.Second))},
		{name: "future", ctx: WithAuthenticationTime(context.Background(), now.Add(time.Second))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := HasRecentAuthentication(tc.ctx, now, window); got != tc.want {
				t.Fatalf("HasRecentAuthentication = %v, want %v", got, tc.want)
			}
		})
	}
}
