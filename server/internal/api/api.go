// Package api implements the browser API described by api/openapi.yaml.
//
// Routing, request/response types, and request validation all come from the
// contract: apigen is generated from it, and every request is validated
// against it before a handler runs. Handlers here only translate between the
// contract's types and the application's modules.
package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"

	"github.com/AbuDubu/amethyst/server/internal/api/apigen"
)

// ReadinessCheck reports whether the server's dependencies can serve traffic.
// A nil error means ready.
type ReadinessCheck func(ctx context.Context) error

// readinessTimeout bounds how long a readiness probe may take, so a hung
// dependency reports "not ready" instead of hanging the probe.
const readinessTimeout = 2 * time.Second

// Deps are what the API handlers need from the rest of the application.
type Deps struct {
	Ready           ReadinessCheck
	CanonicalOrigin string
	Version         string
	Logger          *slog.Logger
}

// NewHandler returns the handler for every /api/ request.
func NewHandler(deps Deps) (http.Handler, error) {
	spec, err := apigen.GetSpec()
	if err != nil {
		return nil, fmt.Errorf("load embedded OpenAPI spec: %w", err)
	}

	p := problems{spec: spec, logger: deps.Logger}
	strict := apigen.NewStrictHandlerWithOptions(&server{deps: deps}, nil, apigen.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  p.badRequest,
		ResponseErrorHandlerFunc: p.internal,
	})
	routes := apigen.HandlerWithOptions(strict, apigen.StdHTTPServerOptions{
		BaseRouter:       http.NewServeMux(),
		ErrorHandlerFunc: p.badRequest,
	})

	// The validator runs first: unknown paths (404), wrong methods (405), and
	// requests that break the contract (400) never reach a handler.
	validate := nethttpmiddleware.OapiRequestValidatorWithOptions(spec, &nethttpmiddleware.Options{
		ErrorHandlerWithOpts: p.validation,
	})
	return validate(routes), nil
}

// server implements apigen.StrictServerInterface.
type server struct {
	deps Deps
}

var _ apigen.StrictServerInterface = (*server)(nil)

// GetHealthz reports liveness only: the process is up and serving HTTP.
// It deliberately checks no dependencies; that is readiness.
func (s *server) GetHealthz(context.Context, apigen.GetHealthzRequestObject) (apigen.GetHealthzResponseObject, error) {
	return apigen.GetHealthz200JSONResponse{Status: apigen.Ok}, nil
}

// GetReadyz reports whether this instance can do useful work right now. The
// failure reason is logged, not returned: probes are public, and database
// errors can reveal internal hostnames and configuration.
func (s *server) GetReadyz(ctx context.Context, _ apigen.GetReadyzRequestObject) (apigen.GetReadyzResponseObject, error) {
	ctx, cancel := context.WithTimeout(ctx, readinessTimeout)
	defer cancel()
	if err := s.deps.Ready(ctx); err != nil {
		s.deps.Logger.Warn("not ready", "error", err)
		return apigen.GetReadyz503JSONResponse{Status: apigen.NotReady}, nil
	}
	return apigen.GetReadyz200JSONResponse{Status: apigen.Ready}, nil
}

func (s *server) GetServerInfo(context.Context, apigen.GetServerInfoRequestObject) (apigen.GetServerInfoResponseObject, error) {
	return apigen.GetServerInfo200JSONResponse{
		CanonicalOrigin: s.deps.CanonicalOrigin,
		Software:        "amethyst",
		Version:         s.deps.Version,
	}, nil
}
