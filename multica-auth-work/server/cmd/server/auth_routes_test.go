package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/analytics"
	"github.com/multica-ai/multica/server/internal/events"
	"github.com/multica-ai/multica/server/internal/handler"
	"github.com/multica-ai/multica/server/internal/realtime"
)

type routerAuthProvider struct{}

func (routerAuthProvider) Login(_ context.Context, _, _ string) (handler.AuthIdentity, error) {
	return handler.AuthIdentity{}, handler.ErrInvalidCredentials
}
func (routerAuthProvider) Logout(context.Context) error { return nil }

type routerPasswordProvisioner struct{}

func (routerPasswordProvisioner) ProvisionPassword(context.Context, pgtype.UUID, string) error {
	return nil
}

func TestPasswordAuthRoutes(t *testing.T) {
	router, _ := NewRouterWithOptions(nil, realtime.NewHub(), events.New(), analytics.NoopClient{}, nil, RouterOptions{AuthProvider: routerAuthProvider{}})

	login := httptest.NewRecorder()
	router.ServeHTTP(login, httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"u@example.com","password":"wrong"}`)))
	if login.Code != http.StatusUnauthorized {
		t.Fatalf("login status=%d body=%s", login.Code, login.Body.String())
	}

	for _, path := range []string{"/auth/send-code", "/auth/verify-code"} {
		res := httptest.NewRecorder()
		router.ServeHTTP(res, httptest.NewRequest(http.MethodPost, path, nil))
		if res.Code != http.StatusNotFound {
			t.Fatalf("%s status=%d, want 404", path, res.Code)
		}
	}
}

func TestPasswordUpdateRouteRequiresAuthentication(t *testing.T) {
	router, _ := NewRouterWithOptions(nil, realtime.NewHub(), events.New(), analytics.NoopClient{}, nil, RouterOptions{
		AuthProvider:        routerAuthProvider{},
		PasswordProvisioner: routerPasswordProvisioner{},
	})

	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodPut, "/api/me/password", strings.NewReader(`{"new_password":"synthetic"}`)))
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("password update status=%d body=%s, want authenticated route", res.Code, res.Body.String())
	}
}
