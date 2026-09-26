package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"

	"github.com/AbuDubu/amethyst/server/internal/api/apigen"
)

// Stable problem codes the frontend may branch on. Add new ones here; never
// rename an existing one.
const (
	codeBadRequest       = "bad_request"
	codeNotFound         = "not_found"
	codeMethodNotAllowed = "method_not_allowed"
	codeInternal         = "internal"
)

// problems writes RFC 9457 Problem Details responses for every error the API
// produces, so clients handle one error shape.
type problems struct {
	spec   *openapi3.T
	logger *slog.Logger
}

// validation handles requests rejected by the OpenAPI validator.
func (p problems) validation(_ context.Context, err error, w http.ResponseWriter, r *http.Request, opts nethttpmiddleware.ErrorHandlerOpts) {
	switch opts.StatusCode {
	case http.StatusNotFound, http.StatusMethodNotAllowed:
		// The validator reports "no matching operation" as 404 even when the
		// path exists with other methods, so consult the contract directly.
		allow := p.allowedMethods(r.URL.Path)
		if allow == "" {
			write(w, http.StatusNotFound, codeNotFound, "No such API endpoint.")
			return
		}
		w.Header().Set("Allow", allow)
		write(w, http.StatusMethodNotAllowed, codeMethodNotAllowed, r.Method+" is not supported for this endpoint.")
	default:
		// Validation messages describe the caller's own request, so they are safe to return.
		write(w, http.StatusBadRequest, codeBadRequest, err.Error())
	}
}

// badRequest handles parameters the generated code could not decode.
func (p problems) badRequest(w http.ResponseWriter, _ *http.Request, err error) {
	write(w, http.StatusBadRequest, codeBadRequest, err.Error())
}

// internal handles errors returned by handlers. Details are logged, never sent.
func (p problems) internal(w http.ResponseWriter, r *http.Request, err error) {
	p.logger.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", err)
	write(w, http.StatusInternalServerError, codeInternal, "")
}

// allowedMethods lists the methods the contract defines for path, for the
// Allow header that must accompany a 405.
func (p problems) allowedMethods(path string) string {
	item := p.spec.Paths.Find(path)
	if item == nil {
		return ""
	}
	var methods []string
	for m := range item.Operations() {
		methods = append(methods, m)
	}
	slices.Sort(methods)
	return strings.Join(methods, ", ")
}

func write(w http.ResponseWriter, status int, code, detail string) {
	body := apigen.Problem{
		Type:   "about:blank",
		Title:  http.StatusText(status),
		Status: status,
		Code:   code,
	}
	if detail != "" {
		body.Detail = &detail
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
