package middleware_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	bluehttp "github.com/yasserelgammal/blue-go/v2/http"
	"github.com/yasserelgammal/blue-go/v2/middleware"
)

func TestRecoverConvertsPanicToHTTPError(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/panic", nil)
	context := bluehttp.NewContext(httptest.NewRecorder(), request, nil)
	handler := middleware.Recover()(func(*bluehttp.Context) error {
		panic("boom")
	})
	err := handler(context)
	if status := bluehttp.StatusForError(err); status != http.StatusInternalServerError {
		t.Fatalf("status = %d", status)
	}
}

func TestRequestID(t *testing.T) {
	t.Run("generates an ID", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		context := bluehttp.NewContext(recorder, httptest.NewRequest(http.MethodGet, "/", nil), nil)
		var handlerID string
		err := middleware.RequestID()(func(c *bluehttp.Context) error {
			handlerID = c.RequestID()
			return c.NoContent(http.StatusNoContent)
		})(context)
		if err != nil {
			t.Fatal(err)
		}
		responseID := recorder.Header().Get("X-Request-ID")
		if handlerID == "" || responseID != handlerID || len(handlerID) != 32 {
			t.Fatalf("handler ID = %q, response ID = %q", handlerID, responseID)
		}
	})

	t.Run("preserves a valid incoming ID", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set("X-Request-ID", "client-request-42")
		recorder := httptest.NewRecorder()
		context := bluehttp.NewContext(recorder, request, nil)
		err := middleware.RequestID()(func(c *bluehttp.Context) error {
			if c.RequestID() != "client-request-42" {
				t.Fatalf("RequestID = %q", c.RequestID())
			}
			return nil
		})(context)
		if err != nil {
			t.Fatal(err)
		}
		if got := recorder.Header().Get("X-Request-ID"); got != "client-request-42" {
			t.Fatalf("X-Request-ID = %q", got)
		}
	})
}

func TestLoggerIncludesRequestID(t *testing.T) {
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	request := httptest.NewRequest(http.MethodGet, "/users", nil)
	request.Header.Set("X-Request-ID", "log-request-7")
	context := bluehttp.NewContext(httptest.NewRecorder(), request, nil)
	handler := middleware.RequestID()(middleware.Logger()(func(c *bluehttp.Context) error {
		return c.NoContent(http.StatusNoContent)
	}))
	if err := handler(context); err != nil {
		t.Fatal(err)
	}
	var record map[string]any
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if record["request_id"] != "log-request-7" {
		t.Fatalf("request_id = %#v", record["request_id"])
	}
}

func TestBodyLimit(t *testing.T) {
	t.Run("rejects known oversized body before handler", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("12345"))
		context := bluehttp.NewContext(httptest.NewRecorder(), request, nil)
		called := false
		err := middleware.BodyLimit(4)(func(*bluehttp.Context) error {
			called = true
			return nil
		})(context)
		if called {
			t.Fatal("oversized request reached handler")
		}
		if status := bluehttp.StatusForError(err); status != http.StatusRequestEntityTooLarge {
			t.Fatalf("status = %d", status)
		}
	})

	t.Run("limits body with unknown length", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"too long"}`))
		request.ContentLength = -1
		request.Header.Set("Content-Type", "application/json")
		context := bluehttp.NewContext(httptest.NewRecorder(), request, nil)
		err := middleware.BodyLimit(8)(func(c *bluehttp.Context) error {
			var value struct {
				Name string `json:"name"`
			}
			return c.BindJSON(&value)
		})(context)
		if status := bluehttp.StatusForError(err); status != http.StatusRequestEntityTooLarge {
			t.Fatalf("status = %d, error = %v", status, err)
		}
	})

	t.Run("allows body within limit", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("1234"))
		context := bluehttp.NewContext(httptest.NewRecorder(), request, nil)
		called := false
		err := middleware.BodyLimit(4)(func(*bluehttp.Context) error {
			called = true
			return nil
		})(context)
		if err != nil || !called {
			t.Fatalf("called = %v, error = %v", called, err)
		}
	})
}

func TestCORS(t *testing.T) {
	config := middleware.CORSConfig{
		AllowedOrigins: []string{"https://example.com"},
		AllowedMethods: []string{http.MethodGet, http.MethodPost},
		AllowedHeaders: []string{"Content-Type", "X-Token"},
		ExposedHeaders: []string{"X-Request-ID"},
		MaxAge:         600,
	}

	t.Run("adds headers to an allowed request", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set("Origin", "https://example.com")
		recorder := httptest.NewRecorder()
		context := bluehttp.NewContext(recorder, request, nil)
		err := middleware.CORS(config)(func(c *bluehttp.Context) error {
			return c.NoContent(http.StatusNoContent)
		})(context)
		if err != nil {
			t.Fatal(err)
		}
		if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
			t.Fatalf("Access-Control-Allow-Origin = %q", got)
		}
		if got := recorder.Header().Get("Access-Control-Expose-Headers"); got != "X-Request-ID" {
			t.Fatalf("Access-Control-Expose-Headers = %q", got)
		}
	})

	t.Run("handles an allowed preflight", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodOptions, "/", nil)
		request.Header.Set("Origin", "https://example.com")
		request.Header.Set("Access-Control-Request-Method", http.MethodPost)
		request.Header.Set("Access-Control-Request-Headers", "Content-Type, X-Token")
		recorder := httptest.NewRecorder()
		context := bluehttp.NewContext(recorder, request, nil)
		called := false
		err := middleware.CORS(config)(func(*bluehttp.Context) error {
			called = true
			return nil
		})(context)
		if err != nil {
			t.Fatal(err)
		}
		if called {
			t.Fatal("preflight reached the next handler")
		}
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("status = %d", recorder.Code)
		}
		if got := recorder.Header().Get("Access-Control-Max-Age"); got != "600" {
			t.Fatalf("Access-Control-Max-Age = %q", got)
		}
		if got := recorder.Header().Get("Vary"); !strings.Contains(got, "Origin") {
			t.Fatalf("Vary = %q", got)
		}
	})

	t.Run("omits CORS headers for a disallowed simple method", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodDelete, "/", nil)
		request.Header.Set("Origin", "https://example.com")
		recorder := httptest.NewRecorder()
		context := bluehttp.NewContext(recorder, request, nil)
		err := middleware.CORS(config)(func(c *bluehttp.Context) error {
			return c.NoContent(http.StatusNoContent)
		})(context)
		if err != nil {
			t.Fatal(err)
		}
		if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Fatalf("Access-Control-Allow-Origin = %q", got)
		}
	})

	t.Run("rejects a disallowed preflight", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodOptions, "/", nil)
		request.Header.Set("Origin", "https://attacker.example")
		request.Header.Set("Access-Control-Request-Method", http.MethodPost)
		context := bluehttp.NewContext(httptest.NewRecorder(), request, nil)
		err := middleware.CORS(config)(func(*bluehttp.Context) error { return nil })(context)
		if status := bluehttp.StatusForError(err); status != http.StatusForbidden {
			t.Fatalf("status = %d", status)
		}
	})
}
