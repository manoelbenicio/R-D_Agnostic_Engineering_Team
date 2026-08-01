package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/multica-ai/multica/server/internal/service/runtimeconfig"
)

type c3RuntimeManagerPlatform struct{}

func (c3RuntimeManagerPlatform) PlatformLayer(context.Context) (*runtimeconfig.Config, error) {
	return nil, nil
}

func c3AuthClass(value string) C3RuntimeManagerMiddleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-C3-Auth-Class", value)
			next.ServeHTTP(w, r)
		})
	}
}

func c3Composition(t *testing.T) *C3RuntimeManagerComposition {
	t.Helper()
	catalog := &rmFakeCatalog{capabilities: rmCapabilities()}
	platform := c3RuntimeManagerPlatform{}
	composition, err := NewC3RuntimeManagerComposition(
		NewRuntimeStandardAPI(&rmFakeStandardStore{versions: map[string]RuntimeConfigurationVersionRecord{}}, catalog, platform),
		NewRuntimeConfigurationAPI(&rmFakeConfigStore{versions: map[string]RuntimeConfigurationVersionRecord{}}, catalog, platform),
		c3AuthClass("owner-read"), c3AuthClass("owner-admin"), c3AuthClass("workspace-admin"),
	)
	if err != nil {
		t.Fatalf("NewC3RuntimeManagerComposition: %v", err)
	}
	return composition
}

func TestC3RuntimeManagerCompositionMountClassifiesAuthorization(t *testing.T) {
	router := chi.NewRouter()
	if err := c3Composition(t).Mount(router); err != nil {
		t.Fatalf("Mount: %v", err)
	}

	for _, test := range []struct{ method, path, want string }{
		{http.MethodGet, "/api/runtime-standards", "owner-read"},
		{http.MethodPost, "/api/runtime-standards", "owner-admin"},
		{http.MethodGet, "/api/workspaces/workspace-a/runtime-bindings/binding-a/configuration-versions", "workspace-admin"},
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(test.method, test.path, nil)
		router.ServeHTTP(recorder, request)
		if got := recorder.Header().Get("X-C3-Auth-Class"); got != test.want {
			t.Errorf("%s %s auth class = %q, want %q", test.method, test.path, got, test.want)
		}
		if recorder.Code == http.StatusNotFound {
			t.Errorf("%s %s was not mounted", test.method, test.path)
		}
	}
}

func TestC3RuntimeManagerCompositionFailsClosed(t *testing.T) {
	valid := c3Composition(t)
	if _, err := NewC3RuntimeManagerComposition(nil, valid.Configuration, valid.OwnerRead, valid.OwnerAdmin, valid.WorkspaceAdmin); err == nil {
		t.Fatal("nil standard API accepted")
	}
	if _, err := NewC3RuntimeManagerComposition(valid.Standards, valid.Configuration, nil, valid.OwnerAdmin, valid.WorkspaceAdmin); err == nil {
		t.Fatal("nil owner-read middleware accepted")
	}
	missingDependency := *valid.Configuration
	missingDependency.Store = nil
	if _, err := NewC3RuntimeManagerComposition(valid.Standards, &missingDependency, valid.OwnerRead, valid.OwnerAdmin, valid.WorkspaceAdmin); err == nil {
		t.Fatal("nil API dependency accepted")
	}
	if err := valid.Mount(nil); err == nil {
		t.Fatal("nil router accepted")
	}
}
