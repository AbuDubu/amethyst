package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

const indexHTML = `<!doctype html><div id="root"></div>`

// apiStub stands in for the real API so these tests cover only routing between
// the API and the frontend; the API has its own tests.
var apiStub = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("X-Handled-By", "api")
	w.WriteHeader(http.StatusTeapot)
})

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	web := fstest.MapFS{
		"index.html":    {Data: []byte(indexHTML)},
		"assets/app.js": {Data: []byte("console.log('app')")},
	}
	h, err := New(web, apiStub)
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

func TestEveryAPIPathGoesToTheAPIHandler(t *testing.T) {
	for _, target := range []string{"/api/healthz", "/api/nope", "/api/", "/api/communities/x"} {
		for _, method := range []string{http.MethodGet, http.MethodPost} {
			t.Run(method+" "+target, func(t *testing.T) {
				rec := get(newTestHandler(t), method, target)

				if rec.Header().Get("X-Handled-By") != "api" {
					t.Errorf("not routed to the API (status %d)", rec.Code)
				}
			})
		}
	}
}

func TestClientRoutesReceiveIndex(t *testing.T) {
	for _, target := range []string{"/", "/communities/gardening", "/c/a/discussions/42", "/apiary"} {
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
	_, err := New(fstest.MapFS{}, apiStub)

	if err == nil {
		t.Fatal("New succeeded without index.html, want error")
	}
	if !strings.Contains(err.Error(), "make web-build") {
		t.Errorf("error %q should tell the developer how to fix it", err)
	}
}
