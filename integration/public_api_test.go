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
