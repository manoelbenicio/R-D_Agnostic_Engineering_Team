package auth

import (
	"context"
	"time"
)

type authenticationTimeContextKey struct{}

// WithAuthenticationTime records a timestamp obtained from a successfully
// verified authentication token. Callers must never populate it from request
// headers or other client-controlled data.
func WithAuthenticationTime(ctx context.Context, authenticatedAt time.Time) context.Context {
	return context.WithValue(ctx, authenticationTimeContextKey{}, authenticatedAt)
}

// HasRecentAuthentication reports whether verified authentication occurred
// within maxAge. Future timestamps are rejected rather than treated as fresh.
func HasRecentAuthentication(ctx context.Context, now time.Time, maxAge time.Duration) bool {
	authenticatedAt, ok := ctx.Value(authenticationTimeContextKey{}).(time.Time)
	if !ok || authenticatedAt.IsZero() || authenticatedAt.After(now) {
		return false
	}
	return now.Sub(authenticatedAt) <= maxAge
}
