package chi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestRoutePattern(t *testing.T) {
	rctx := chi.NewRouteContext()
	if rctx.RoutePattern() != "" {
		t.Fatalf("expected empty string, got %q", rctx.RoutePattern())
	}

	rctx.RoutePatterns = []string{"/users"}
	if rctx.RoutePattern() != "/users" {
		t.Fatalf("expected /users, got %q", rctx.RoutePattern())
	}

	rctx.RoutePatterns = []string{"/api/*", "/users/{id}"}
	if rctx.RoutePattern() != "/api/users/{id}" {
		t.Fatalf("expected /api/users/{id}, got %q", rctx.RoutePattern())
	}

	rctx.RoutePatterns = []string{"/api/*", "/v1/*", "/orgs/{orgID}/*", "/teams/{teamID}"}
	if rctx.RoutePattern() != "/api/v1/orgs/{orgID}/teams/{teamID}" {
		t.Fatalf("expected /api/v1/orgs/{orgID}/teams/{teamID}, got %q", rctx.RoutePattern())
	}

	rctx.RoutePatterns = []string{"/*", "/users"}
	if rctx.RoutePattern() != "/users" {
		t.Fatalf("expected /users, got %q", rctx.RoutePattern())
	}

	rctx.RoutePatterns = []string{"/api/*", "/v1/*"}
	if rctx.RoutePattern() != "/api/v1" {
		t.Fatalf("expected /api/v1, got %q", rctx.RoutePattern())
	}
}

func TestNestedRouteContext(t *testing.T) {
	r := chi.NewRouter()

	subRouter := chi.NewRouter()
	var capturedPattern string
	var capturedID string

	subRouter.Get("/users/{id}", func(w http.ResponseWriter, req *http.Request) {
		capturedPattern = chi.RouteContext(req.Context()).RoutePattern()
		capturedID = chi.URLParam(req, "id")
		w.WriteHeader(http.StatusOK)
	})

	r.Mount("/api/v1", subRouter)

	req := httptest.NewRequest("GET", "/api/v1/users/42", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if capturedPattern != "/api/v1/users/{id}" {
		t.Fatalf("expected pattern /api/v1/users/{id}, got %q", capturedPattern)
	}
	if capturedID != "42" {
		t.Fatalf("expected param id 42, got %q", capturedID)
	}
}
