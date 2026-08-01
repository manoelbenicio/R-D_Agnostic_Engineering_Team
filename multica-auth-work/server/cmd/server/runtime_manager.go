package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/multica-ai/multica/server/internal/handler"
	"github.com/multica-ai/multica/server/internal/middleware"
	"github.com/multica-ai/multica/server/internal/service/runtimeconfig"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

const (
	runtimeManagerEnabledEnv       = "MULTICA_RUNTIME_MANAGER_ENABLED"
	runtimeManagerAuthorityFileEnv = "MULTICA_RUNTIME_MANAGER_AUTHORITY_FILE"
	maxRuntimeManagerAuthoritySize = 1 << 20
)

// RuntimeManagerOptions is the validated, immutable authority used to compose
// the production Runtime Manager. A nil option disables the still-unreleased
// surface. Once explicitly enabled, every missing/invalid dependency is a
// startup error; the server never mounts a partial or unclassified API.
type RuntimeManagerOptions struct {
	Capabilities map[string]runtimeconfig.ProviderCapabilities
	Platform     *runtimeconfig.Config
}

type runtimeManagerAuthorityDocument struct {
	Providers map[string]runtimeManagerProviderDocument `json:"providers"`
	Platform  *runtimeManagerPlatformDocument           `json:"platform,omitempty"`
}

type runtimeManagerProviderDocument struct {
	Version  string                                 `json:"version"`
	Provider string                                 `json:"provider"`
	Models   map[string]runtimeManagerModelDocument `json:"models"`
}

type runtimeManagerModelDocument struct {
	ReasoningEfforts    []string `json:"reasoning_efforts"`
	MaxInputTokens      int64    `json:"max_input_tokens"`
	MaxOutputTokens     int64    `json:"max_output_tokens"`
	ContextWindowTokens int64    `json:"context_window_tokens"`
	MaxToolCalls        int64    `json:"max_tool_calls"`
}

type runtimeManagerPlatformDocument struct {
	Version      string               `json:"version"`
	Values       runtimeconfig.Values `json:"values"`
	Delegability map[string]bool      `json:"delegability,omitempty"`
}

func runtimeManagerOptionsFromEnv() (*RuntimeManagerOptions, error) {
	if !strings.EqualFold(strings.TrimSpace(os.Getenv(runtimeManagerEnabledEnv)), "true") {
		return nil, nil
	}
	path := strings.TrimSpace(os.Getenv(runtimeManagerAuthorityFileEnv))
	if path == "" {
		return nil, fmt.Errorf("%s=true requires %s", runtimeManagerEnabledEnv, runtimeManagerAuthorityFileEnv)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open runtime manager authority: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, maxRuntimeManagerAuthoritySize+1))
	decoder.DisallowUnknownFields()
	var document runtimeManagerAuthorityDocument
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("decode runtime manager authority: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, errors.New("decode runtime manager authority: trailing content")
	}
	options := &RuntimeManagerOptions{Capabilities: make(map[string]runtimeconfig.ProviderCapabilities, len(document.Providers))}
	for key, provider := range document.Providers {
		capabilities := runtimeconfig.ProviderCapabilities{
			Version: runtimeconfig.Version(provider.Version), Provider: runtimeconfig.ProviderID(provider.Provider),
			Models: make(map[runtimeconfig.ModelID]runtimeconfig.ModelCapabilities, len(provider.Models)),
		}
		for modelID, model := range provider.Models {
			efforts := make([]runtimeconfig.ReasoningEffort, len(model.ReasoningEfforts))
			for index, effort := range model.ReasoningEfforts {
				efforts[index] = runtimeconfig.ReasoningEffort(effort)
			}
			capabilities.Models[runtimeconfig.ModelID(modelID)] = runtimeconfig.ModelCapabilities{
				ReasoningEfforts: efforts, MaxInputTokens: model.MaxInputTokens,
				MaxOutputTokens: model.MaxOutputTokens, ContextWindowTokens: model.ContextWindowTokens,
				MaxToolCalls: model.MaxToolCalls,
			}
		}
		options.Capabilities[key] = capabilities
	}
	if document.Platform != nil {
		delegability := make(map[runtimeconfig.Field]bool, len(document.Platform.Delegability))
		for field, allowed := range document.Platform.Delegability {
			delegability[runtimeconfig.Field(field)] = allowed
		}
		options.Platform = &runtimeconfig.Config{Version: runtimeconfig.Version(document.Platform.Version), Values: document.Platform.Values, Delegability: delegability}
	}
	return options, nil
}

func newRuntimeManagerComposition(pool *pgxpool.Pool, queries *db.Queries, options *RuntimeManagerOptions) (*handler.C3RuntimeManagerComposition, error) {
	if options == nil {
		return nil, nil
	}
	store, err := handler.NewPostgresRuntimeManagerStore(pool)
	if err != nil {
		return nil, err
	}
	catalog, err := handler.NewStaticCapabilityCatalog(options.Capabilities)
	if err != nil {
		return nil, err
	}
	platform, err := handler.NewStaticPlatformLayerSource(options.Platform)
	if err != nil {
		return nil, err
	}
	standards := handler.NewRuntimeStandardAPI(store, catalog, platform)
	configuration := handler.NewRuntimeConfigurationAPI(store, catalog, platform)
	return handler.NewC3RuntimeManagerComposition(
		standards,
		configuration,
		runtimeManagerOwnerMiddleware(false),
		runtimeManagerOwnerMiddleware(true),
		runtimeManagerWorkspaceAdminMiddleware(queries),
	)
}

func runtimeManagerOwnerMiddleware(admin bool) handler.C3RuntimeManagerMiddleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := strings.TrimSpace(r.Header.Get("X-User-ID"))
			if _, err := util.ParseUUID(userID); err != nil {
				writeRuntimeManagerMiddlewareError(w, r, http.StatusUnauthorized, "unauthenticated")
				return
			}
			if admin && r.Header.Get("X-Actor-Source") == "task_token" {
				writeRuntimeManagerMiddlewareError(w, r, http.StatusForbidden, "forbidden")
				return
			}
			ctx := handler.WithRuntimeManagerActor(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func runtimeManagerWorkspaceAdminMiddleware(queries *db.Queries) handler.C3RuntimeManagerMiddleware {
	return func(next http.Handler) http.Handler {
		return middleware.RequireWorkspaceRoleFromURL(queries, "workspaceId", "owner", "admin")(next)
	}
}

func writeRuntimeManagerMiddlewareError(w http.ResponseWriter, r *http.Request, status int, code string) {
	requestID := chimw.GetReqID(r.Context())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{
		"code": code, "message": code, "request_id": requestID, "retryable": false,
	}})
}

func mountRuntimeManager(router chi.Router, composition *handler.C3RuntimeManagerComposition) error {
	if composition == nil {
		return nil
	}
	if router == nil {
		return errors.New("runtime manager: protected router is required")
	}
	return composition.Mount(router)
}

// Compile-time assertion that the DB-backed store satisfies both durable C3 contracts.
var _ handler.RuntimeConfigurationStore = (*handler.PostgresRuntimeManagerStore)(nil)
var _ handler.RuntimeStandardStore = (*handler.PostgresRuntimeManagerStore)(nil)
