package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AbuDubu/amethyst/server/internal/api/apigen"
)

func newTestAPI(t *testing.T, ready ReadinessCheck) http.Handler {
	t.Helper()
	if ready == nil {
		ready = func(context.Context) error { return nil }
	}
	h, err := NewHandler(Deps{
		Ready:           ready,
		CanonicalOrigin: "https://a.example",
		Version:         "v1.2.3-test",
		Logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	return h
}

func do(h http.Handler, method, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return v
}

// requireProblem asserts an RFC 9457 response with the given status and code.
func requireProblem(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) apigen.Problem {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d (body %s)", rec.Code, status, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("Content-Type = %q, want application/problem+json", ct)
	}
	p := decode[apigen.Problem](t, rec)
	if p.Status != status || p.Code != code || p.Type == "" || p.Title == "" {
		t.Fatalf("problem = %+v, want status %d and code %q with type and title", p, status, code)
	}
	return p
}

func TestHealthz(t *testing.T) {
	rec := do(newTestAPI(t, nil), http.MethodGet, "/api/healthz")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := decode[apigen.Status](t, rec).Status; got != apigen.Ok {
		t.Errorf("status = %q, want ok", got)
	}
}

func TestReadyzWhenDependenciesAreReady(t *testing.T) {
	rec := do(newTestAPI(t, nil), http.MethodGet, "/api/readyz")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := decode[apigen.Status](t, rec).Status; got != apigen.Ready {
		t.Errorf("status = %q, want ready", got)
	}
}

func TestReadyzFailsButHealthzPassesWhenDatabaseIsDown(t *testing.T) {
	h := newTestAPI(t, func(context.Context) error {
		return errors.New("dial tcp 10.0.0.5:5432: connection refused")
	})

	ready := do(h, http.MethodGet, "/api/readyz")
	if ready.Code != http.StatusServiceUnavailable {
		t.Errorf("readyz status = %d, want 503", ready.Code)
	}
	if strings.Contains(ready.Body.String(), "10.0.0.5") {
		t.Error("readyz response leaked the internal error")
	}

	if live := do(h, http.MethodGet, "/api/healthz"); live.Code != http.StatusOK {
		t.Errorf("healthz status = %d, want 200 (liveness must not depend on the database)", live.Code)
	}
}

func TestReadyzGivesUpOnAHungDependency(t *testing.T) {
	h := newTestAPI(t, func(ctx context.Context) error {
		<-ctx.Done() // Simulates a dependency that never answers.
		return ctx.Err()
	})

	if rec := do(h, http.MethodGet, "/api/readyz"); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
}

func TestServerInfo(t *testing.T) {
	rec := do(newTestAPI(t, nil), http.MethodGet, "/api/server")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	info := decode[apigen.ServerInfo](t, rec)
	if info.CanonicalOrigin != "https://a.example" || info.Software != "amethyst" || info.Version != "v1.2.3-test" {
		t.Errorf("info = %+v, want https://a.example running amethyst v1.2.3-test", info)
	}
}

func TestUnknownPathIsProblem404(t *testing.T) {
	for _, target := range []string{"/api/nope", "/api/communities/x", "/api/"} {
		t.Run(target, func(t *testing.T) {
			requireProblem(t, do(newTestAPI(t, nil), http.MethodGet, target), http.StatusNotFound, codeNotFound)
		})
	}
}

func TestWrongMethodIsProblem405WithAllow(t *testing.T) {
	rec := do(newTestAPI(t, nil), http.MethodPost, "/api/healthz")

	requireProblem(t, rec, http.StatusMethodNotAllowed, codeMethodNotAllowed)
	if allow := rec.Header().Get("Allow"); allow != "GET" {
		t.Errorf("Allow = %q, want GET", allow)
	}
}
