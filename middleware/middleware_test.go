package middleware_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

// ── Rate Limiter ─────────────────────────────────────────────────────────────

func TestRateLimitAllowsRequestsUnderLimit(t *testing.T) {
	mw := middleware.RateLimit(middleware.RateLimitConfig{
		MaxRequests: 3,
		Window:      time.Second,
	})
	handler := mw(func(c *bluehttp.Context) error {
		return c.NoContent(http.StatusOK)
	})

	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		c := bluehttp.NewContext(rec, httptest.NewRequest(http.MethodGet, "/", nil), nil)
		if err := handler(c); err != nil {
			t.Fatalf("request %d: unexpected error: %v", i+1, err)
		}
	}
}

func TestRateLimitBlocksRequestsOverLimit(t *testing.T) {
	mw := middleware.RateLimit(middleware.RateLimitConfig{
		MaxRequests: 2,
		Window:      time.Second,
	})
	handler := mw(func(c *bluehttp.Context) error { return nil })

	for i := 0; i < 2; i++ {
		c := bluehttp.NewContext(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil), nil)
		_ = handler(c)
	}
	c := bluehttp.NewContext(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil), nil)
	err := handler(c)
	if status := bluehttp.StatusForError(err); status != 429 {
		t.Fatalf("expected 429, got %d", status)
	}
}

func TestRateLimitCustomKeyFunc(t *testing.T) {
	mw := middleware.RateLimit(middleware.RateLimitConfig{
		MaxRequests: 1,
		Window:      time.Second,
		KeyFunc:     func(c *bluehttp.Context) string { return c.Request.Header.Get("X-API-Key") },
	})
	handler := mw(func(c *bluehttp.Context) error { return nil })

	makeReq := func(key string) *bluehttp.Context {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("X-API-Key", key)
		return bluehttp.NewContext(httptest.NewRecorder(), r, nil)
	}

	// First request for each key: allowed.
	if err := handler(makeReq("key-a")); err != nil {
		t.Fatal("key-a first request should pass")
	}
	if err := handler(makeReq("key-b")); err != nil {
		t.Fatal("key-b first request should pass")
	}
	// Second request for key-a: blocked.
	if status := bluehttp.StatusForError(handler(makeReq("key-a"))); status != 429 {
		t.Fatalf("key-a second request: expected 429, got %d", status)
	}
}

// ── Timeout ───────────────────────────────────────────────────────────────────

func TestTimeoutPassesThroughFastHandler(t *testing.T) {
	mw := middleware.Timeout(100 * time.Millisecond)
	handler := mw(func(c *bluehttp.Context) error {
		return c.NoContent(http.StatusOK)
	})
	c := bluehttp.NewContext(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil), nil)
	if err := handler(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTimeoutReturns503ForSlowHandler(t *testing.T) {
	mw := middleware.Timeout(10 * time.Millisecond)
	handler := mw(func(c *bluehttp.Context) error {
		select {
		case <-time.After(200 * time.Millisecond):
			return nil
		case <-c.Request.Context().Done():
			return c.Request.Context().Err()
		}
	})
	c := bluehttp.NewContext(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil), nil)
	err := handler(c)
	if status := bluehttp.StatusForError(err); status != 503 {
		t.Fatalf("expected 503, got %d (err: %v)", status, err)
	}
}

// ── ETag ─────────────────────────────────────────────────────────────────────

func TestETagAddsHeaderToGetResponse(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := bluehttp.NewContext(rec, req, nil)

	handler := middleware.ETag()(func(c *bluehttp.Context) error {
		return c.String(http.StatusOK, "hello world")
	})
	if err := handler(c); err != nil {
		t.Fatal(err)
	}
	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Fatal("ETag header missing")
	}
	if !strings.HasPrefix(etag, `W/"`) {
		t.Fatalf("ETag not weak: %q", etag)
	}
}

func TestETagReturns304OnMatch(t *testing.T) {
	// First request: get the ETag.
	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	c1 := bluehttp.NewContext(rec1, req1, nil)
	handler := middleware.ETag()(func(c *bluehttp.Context) error {
		return c.String(http.StatusOK, "hello world")
	})
	_ = handler(c1)
	etag := rec1.Header().Get("ETag")

	// Second request: send If-None-Match.
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("If-None-Match", etag)
	c2 := bluehttp.NewContext(rec2, req2, nil)
	if err := handler(c2); err != nil {
		t.Fatal(err)
	}
	if rec2.Code != http.StatusNotModified {
		t.Fatalf("expected 304, got %d", rec2.Code)
	}
	if rec2.Body.Len() != 0 {
		t.Fatal("304 response must have no body")
	}
}

func TestETagSkipsNonGetMethods(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	c := bluehttp.NewContext(rec, req, nil)

	handler := middleware.ETag()(func(c *bluehttp.Context) error {
		return c.String(http.StatusOK, "data")
	})
	_ = handler(c)
	if rec.Header().Get("ETag") != "" {
		t.Fatal("ETag must not be set on POST responses")
	}
}

// ── LoggerWith ───────────────────────────────────────────────────────────────

func TestLoggerWithCustomLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	c := bluehttp.NewContext(rec, req, nil)

	handler := middleware.LoggerWith(middleware.LoggerConfig{Logger: logger})(func(c *bluehttp.Context) error {
		return c.NoContent(http.StatusOK)
	})
	if err := handler(c); err != nil {
		t.Fatal(err)
	}

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("log not valid JSON: %v\noutput: %s", err, buf.String())
	}
	if record["path"] != "/ping" {
		t.Fatalf("path = %v", record["path"])
	}
}

func TestLoggerWithCustomLevelForStatus(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	levelFunc := func(status int) slog.Level {
		if status >= 400 {
			return slog.LevelWarn
		}
		return slog.LevelInfo
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/bad", nil)
	c := bluehttp.NewContext(rec, req, nil)

	handler := middleware.LoggerWith(middleware.LoggerConfig{
		Logger:         logger,
		LevelForStatus: levelFunc,
	})(func(c *bluehttp.Context) error {
		return bluehttp.BadRequest("oops")
	})
	_ = handler(c)

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("log not valid JSON: %v", err)
	}
	if record["level"] != "WARN" {
		t.Fatalf("level = %v, want WARN", record["level"])
	}
}

func TestDefaultLevelForStatus(t *testing.T) {
	if middleware.DefaultLevelForStatus(200) != slog.LevelInfo {
		t.Fatal("200 should be Info")
	}
	if middleware.DefaultLevelForStatus(404) != slog.LevelInfo {
		t.Fatal("404 should be Info")
	}
	if middleware.DefaultLevelForStatus(500) != slog.LevelError {
		t.Fatal("500 should be Error")
	}
	if middleware.DefaultLevelForStatus(503) != slog.LevelError {
		t.Fatal("503 should be Error")
	}
}
