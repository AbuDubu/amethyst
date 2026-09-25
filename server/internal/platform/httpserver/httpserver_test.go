package httpserver

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
	"testing/fstest"
)

const indexHTML = `<!doctype html><div id="root"></div>`

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	return newTestHandlerWithReadiness(t, func(context.Context) error { return nil })
}

func newTestHandlerWithReadiness(t *testing.T, ready ReadinessCheck) http.Handler {
	t.Helper()
	web := fstest.MapFS{
		"index.html":    {Data: []byte(indexHTML)},
		"assets/app.js": {Data: []byte("console.log('app')")},
	}
	h, err := New(web, ready, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return h
}

func get(h http.Handler, method, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec
}

func TestHealthz(t *testing.T) {
	rec := get(newTestHandler(t), http.MethodGet, "/api/healthz")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status field = %q, want ok", body["status"])
	}
}

func TestReadyzWhenDependenciesAreReady(t *testing.T) {
	rec := get(newTestHandler(t), http.MethodGet, "/api/readyz")

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestReadyzFailsButHealthzPassesWhenDatabaseIsDown(t *testing.T) {
	h := newTestHandlerWithReadiness(t, func(context.Context) error {
		return errors.New("dial tcp 10.0.0.5:5432: connection refused")
	})

	ready := get(h, http.MethodGet, "/api/readyz")
	if ready.Code != http.StatusServiceUnavailable {
		t.Errorf("readyz status = %d, want 503", ready.Code)
	}
	if strings.Contains(ready.Body.String(), "10.0.0.5") {
		t.Error("readyz response leaked the internal error")
	}

	if live := get(h, http.MethodGet, "/api/healthz"); live.Code != http.StatusOK {
		t.Errorf("healthz status = %d, want 200 (liveness must not depend on the database)", live.Code)
	}
}

func TestReadyzGivesUpOnAHungDependency(t *testing.T) {
	h := newTestHandlerWithReadiness(t, func(ctx context.Context) error {
		<-ctx.Done() // Simulates a dependency that never answers.
		return ctx.Err()
	})

	if rec := get(h, http.MethodGet, "/api/readyz"); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
}

func TestUnknownAPIPathIsJSON404NotFrontend(t *testing.T) {
	for _, target := range []string{"/api/nope", "/api/v1/communities/x", "/api/"} {
		t.Run(target, func(t *testing.T) {
			rec := get(newTestHandler(t), http.MethodGet, target)

			if rec.Code != http.StatusNotFound {
				t.Errorf("status = %d, want 404", rec.Code)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}
			if strings.Contains(rec.Body.String(), "<!doctype html>") {
				t.Error("API miss returned the frontend shell")
			}
		})
	}
}

func TestClientRoutesReceiveIndex(t *testing.T) {
	for _, target := range []string{"/", "/communities/gardening", "/c/a/discussions/42"} {
		t.Run(target, func(t *testing.T) {
			rec := get(newTestHandler(t), http.MethodGet, target)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			if rec.Body.String() != indexHTML {
				t.Errorf("body = %q, want index.html", rec.Body.String())
			}
		})
	}
}

func TestStaticAssetIsServed(t *testing.T) {
	rec := get(newTestHandler(t), http.MethodGet, "/assets/app.js")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "console.log") {
		t.Errorf("body = %q, want app.js contents", rec.Body.String())
	}
}

func TestMissingAssetIs404NotIndex(t *testing.T) {
	rec := get(newTestHandler(t), http.MethodGet, "/assets/stale-1234.js")

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestNonGETToFrontendIsRejected(t *testing.T) {
	rec := get(newTestHandler(t), http.MethodPost, "/communities")

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestNewFailsWithoutFrontendBuild(t *testing.T) {
	_, err := New(fstest.MapFS{}, func(context.Context) error { return nil }, slog.Default())

	if err == nil {
		t.Fatal("New succeeded without index.html, want error")
	}
	if !strings.Contains(err.Error(), "make web-build") {
		t.Errorf("error %q should tell the developer how to fix it", err)
	}
}
