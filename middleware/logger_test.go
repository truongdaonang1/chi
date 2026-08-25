package middleware_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type testLogger struct {
	buf bytes.Buffer
	mu  sync.Mutex
}

func (l *testLogger) Print(v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, val := range v {
		if s, ok := val.(string); ok {
			l.buf.WriteString(s)
		}
	}
}

func (l *testLogger) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.String()
}

func TestLoggerNestedRouterPattern(t *testing.T) {
	tl := &testLogger{}
	formatter := &middleware.DefaultLogFormatterImpl{
		Logger:  tl,
		NoColor: true,
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestLogger(formatter))

	subRouter := chi.NewRouter()
	subRouter.Get("/users/{id}", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("user response"))
	})

	r.Mount("/api/v1", subRouter)

	req := httptest.NewRequest("GET", "/api/v1/users/42", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	logOutput := tl.String()
	if !strings.Contains(logOutput, "/api/v1/users/{id}") {
		t.Fatalf("expected log output to contain /api/v1/users/{id}, got: %s", logOutput)
	}
	if strings.Contains(logOutput, "/api/v1/*") {
		t.Fatalf("log output should not contain parent wildcard /api/v1/*, got: %s", logOutput)
	}
}

func TestLoggerDeeplyNestedRouter(t *testing.T) {
	tl := &testLogger{}
	formatter := &middleware.DefaultLogFormatterImpl{
		Logger:  tl,
		NoColor: true,
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestLogger(formatter))

	r.Route("/api", func(r chi.Router) {
		r.Route("/v1", func(r chi.Router) {
			r.Route("/orgs/{orgID}", func(r chi.Router) {
				r.Get("/teams/{teamID}", func(w http.ResponseWriter, req *http.Request) {
					w.Write([]byte("team response"))
				})
			})
		})
	})

	req := httptest.NewRequest("GET", "/api/v1/orgs/apple/teams/design", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	logOutput := tl.String()
	expectedPattern := "/api/v1/orgs/{orgID}/teams/{teamID}"
	if !strings.Contains(logOutput, expectedPattern) {
		t.Fatalf("expected log to contain %q, got: %s", expectedPattern, logOutput)
	}
}

func TestLoggerSubrouter404And405(t *testing.T) {
	tl := &testLogger{}
	formatter := &middleware.DefaultLogFormatterImpl{
		Logger:  tl,
		NoColor: true,
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestLogger(formatter))

	subRouter := chi.NewRouter()
	subRouter.Post("/users/{id}", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("ok"))
	})
	r.Mount("/api/v1", subRouter)

	req404 := httptest.NewRequest("GET", "/api/v1/nonexistent", nil)
	rec404 := httptest.NewRecorder()
	r.ServeHTTP(rec404, req404)
	if rec404.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec404.Code)
	}

	req405 := httptest.NewRequest("GET", "/api/v1/users/42", nil)
	rec405 := httptest.NewRecorder()
	r.ServeHTTP(rec405, req405)
	if rec405.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", rec405.Code)
	}
}
