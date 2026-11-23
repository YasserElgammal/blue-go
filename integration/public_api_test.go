package integration_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	blue "github.com/yasserelgammal/blue-go/v2"
)

func TestPublicAPI(t *testing.T) {
	application := blue.New()
	application.GET("/health", func(c *blue.Context) error {
		return c.Respond(http.StatusOK, map[string]string{"status": "ok"})
	})
	recorder := httptest.NewRecorder()
	application.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))
	if recorder.Code != http.StatusOK || recorder.Body.String() != "{\"success\":true,\"data\":{\"status\":\"ok\"}}\n" {
		t.Fatalf("got status %d body %q", recorder.Code, recorder.Body.String())
	}
}

func TestPublicMiddlewareAPI(t *testing.T) {
	application := blue.New()
	application.Use(
		blue.RequestID(),
		blue.CORS(blue.CORSConfig{
			AllowedOrigins: []string{"https://example.com"},
			AllowedMethods: []string{http.MethodPost},
		}),
	)

	request := httptest.NewRequest(http.MethodOptions, "/users", nil)
	request.Header.Set("Origin", "https://example.com")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	recorder := httptest.NewRecorder()
	application.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("got status %d body %q", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
	if got := recorder.Header().Get("X-Request-ID"); got == "" {
		t.Fatal("X-Request-ID is empty")
	}
}

func TestPublicPriority2Middleware(t *testing.T) {
	application := blue.New()
	application.Use(
		blue.ETag(),
		blue.RateLimit(blue.RateLimitConfig{MaxRequests: 5}),
		blue.Logger(),
	)

	application.GET("/items/*path", func(c *blue.Context) error {
		return c.String(http.StatusOK, "path: "+c.Param("path"))
	})

	// Test wildcard + ETag
	req := httptest.NewRequest(http.MethodGet, "/items/docs/intro.md", nil)
	rec := httptest.NewRecorder()
	application.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d", rec.Code)
	}
	if rec.Body.String() != "path: docs/intro.md" {
		t.Fatalf("unexpected body: %q", rec.Body.String())
	}
	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Fatal("expected ETag header")
	}

	// Test 304 with If-None-Match
	req2 := httptest.NewRequest(http.MethodGet, "/items/docs/intro.md", nil)
	req2.Header.Set("If-None-Match", etag)
	rec2 := httptest.NewRecorder()
	application.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusNotModified {
		t.Fatalf("expected 304, got %d", rec2.Code)
	}
}
