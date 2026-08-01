package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// C3RuntimeManagerMiddleware is an authorization boundary supplied by shared
// production composition. Runtime Manager never mounts a route without one.
type C3RuntimeManagerMiddleware func(http.Handler) http.Handler

// C3RuntimeManagerComposition owns the complete Runtime Manager route surface
// and the authorization class assigned to each scope.
type C3RuntimeManagerComposition struct {
	Standards      *RuntimeStandardAPI
	Configuration  *RuntimeConfigurationAPI
	OwnerRead      C3RuntimeManagerMiddleware
	OwnerAdmin     C3RuntimeManagerMiddleware
	WorkspaceAdmin C3RuntimeManagerMiddleware
}

// NewC3RuntimeManagerComposition validates all production dependencies before
// returning a mountable composition. Platform sources are required here even
// though the standalone admission engine supports a nil platform layer.
func NewC3RuntimeManagerComposition(
	standards *RuntimeStandardAPI,
	configuration *RuntimeConfigurationAPI,
	ownerRead C3RuntimeManagerMiddleware,
	ownerAdmin C3RuntimeManagerMiddleware,
	workspaceAdmin C3RuntimeManagerMiddleware,
) (*C3RuntimeManagerComposition, error) {
	composition := &C3RuntimeManagerComposition{
		Standards: standards, Configuration: configuration,
		OwnerRead: ownerRead, OwnerAdmin: ownerAdmin, WorkspaceAdmin: workspaceAdmin,
	}
	if err := composition.validate(); err != nil {
		return nil, err
	}
	return composition, nil
}

func (c *C3RuntimeManagerComposition) validate() error {
	if c == nil || c.Standards == nil || c.Configuration == nil {
		return errors.New("runtime manager composition: APIs are required")
	}
	if c.Standards.Store == nil || c.Standards.Capabilities == nil || c.Standards.Platform == nil ||
		c.Configuration.Store == nil || c.Configuration.Capabilities == nil || c.Configuration.Platform == nil {
		return errors.New("runtime manager composition: API dependencies are required")
	}
	if c.OwnerRead == nil || c.OwnerAdmin == nil || c.WorkspaceAdmin == nil {
		return errors.New("runtime manager composition: authorization middleware is required")
	}
	return nil
}

// Mount installs every route under the caller's existing request-ID and
// recovery stack. Owner-global reads and mutations use distinct authorization
// boundaries; every workspace binding route requires workspace admin.
func (c *C3RuntimeManagerComposition) Mount(router chi.Router) error {
	if router == nil {
		return errors.New("runtime manager composition: router is required")
	}
	if err := c.validate(); err != nil {
		return err
	}
	for _, route := range c.Standards.Routes() {
		middleware := c.OwnerAdmin
		if route.Method == http.MethodGet {
			middleware = c.OwnerRead
		}
		router.With(middleware).MethodFunc(route.Method, route.Pattern, route.Handler)
	}
	for _, route := range c.Configuration.Routes() {
		router.With(c.WorkspaceAdmin).MethodFunc(route.Method, route.Pattern, route.Handler)
	}
	return nil
}
