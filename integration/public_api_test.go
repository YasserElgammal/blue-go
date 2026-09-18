package integration_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	blue "github.com/yasserelgammal/blue-go"
)

func TestPublicAPI(t *testing.T) {
	application := blue.New()
	application.GET("/health", func(c *blue.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	recorder := httptest.NewRecorder()
	application.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))
	if recorder.Code != http.StatusOK || recorder.Body.String() != "{\"status\":\"ok\"}\n" {
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
