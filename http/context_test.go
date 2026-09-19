package http_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	bluehttp "github.com/yasserelgammal/blue-go/http"
	"github.com/yasserelgammal/blue-go/pagination"
)

func context(method, target string, body io.Reader) (*bluehttp.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, target, body)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	return bluehttp.NewContext(recorder, request, nil), recorder
}

func TestQueryParameters(t *testing.T) {
	c, _ := context(stdhttp.MethodGet, "/search?q=trees&page=3&invalid=nope", nil)
	if c.Query("q") != "trees" || c.QueryInt("page", 1) != 3 || c.QueryInt("invalid", 7) != 7 || c.QueryInt("missing", 9) != 9 {
		t.Fatal("query helpers returned unexpected values")
	}
}

func TestJSONResponse(t *testing.T) {
	c, response := context(stdhttp.MethodGet, "/", nil)
	if err := c.JSON(stdhttp.StatusCreated, map[string]string{"status": "ok"}); err != nil {
		t.Fatal(err)
	}
	if response.Code != stdhttp.StatusCreated || response.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("unexpected response: status %d headers %#v", response.Code, response.Header())
	}
	if response.Body.String() != "{\"status\":\"ok\"}\n" {
		t.Fatalf("body = %q", response.Body.String())
	}
}

func TestUnifiedResponse(t *testing.T) {
	c, response := context(stdhttp.MethodGet, "/", nil)
	if err := c.RespondWithMessage(stdhttp.StatusCreated, "User created", map[string]int{"id": 42}); err != nil {
		t.Fatal(err)
	}
	if response.Code != stdhttp.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, stdhttp.StatusCreated)
	}
	if response.Body.String() != "{\"success\":true,\"message\":\"User created\",\"data\":{\"id\":42}}\n" {
		t.Fatalf("body = %q", response.Body.String())
	}
}

func TestContextResponseFormatter(t *testing.T) {
	c, response := context(stdhttp.MethodGet, "/", nil)
	c.SetResponseFormatter(func(value bluehttp.Response) any {
		return map[string]any{
			"code":    value.Status,
			"payload": value.Data,
		}
	})
	if err := c.Respond(stdhttp.StatusAccepted, "queued"); err != nil {
		t.Fatal(err)
	}
	if response.Body.String() != "{\"code\":202,\"payload\":\"queued\"}\n" {
		t.Fatalf("body = %q", response.Body.String())
	}
}

func TestStringAndNoContentResponses(t *testing.T) {
	stringContext, stringResponse := context(stdhttp.MethodGet, "/", nil)
	if err := stringContext.String(stdhttp.StatusOK, "hello %s", "blue"); err != nil {
		t.Fatal(err)
	}
	if stringResponse.Body.String() != "hello blue" || stringResponse.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Fatalf("unexpected string response: %#v %q", stringResponse.Header(), stringResponse.Body.String())
	}

	emptyContext, emptyResponse := context(stdhttp.MethodDelete, "/", nil)
	if err := emptyContext.NoContent(stdhttp.StatusNoContent); err != nil {
		t.Fatal(err)
	}
	if emptyResponse.Code != stdhttp.StatusNoContent || emptyResponse.Body.Len() != 0 {
		t.Fatalf("unexpected no-content response: %d %q", emptyResponse.Code, emptyResponse.Body.String())
	}
}

func TestBindJSON(t *testing.T) {
	type input struct {
		Name string `json:"name"`
	}
	c, _ := context(stdhttp.MethodPost, "/", bytes.NewBufferString(`{"name":"oak"}`))
	var value input
	if err := c.Bind(&value); err != nil {
		t.Fatal(err)
	}
	if value.Name != "oak" {
		t.Fatalf("name = %q", value.Name)
	}
}

func TestBindJSONValidation(t *testing.T) {
	type input struct {
		Name string `json:"name"`
	}

	t.Run("requires JSON content type", func(t *testing.T) {
		request := httptest.NewRequest(stdhttp.MethodPost, "/", bytes.NewBufferString(`{"name":"oak"}`))
		c := bluehttp.NewContext(httptest.NewRecorder(), request, nil)
		var value input
		err := c.BindJSON(&value)
		if status := bluehttp.StatusForError(err); status != stdhttp.StatusUnsupportedMediaType {
			t.Fatalf("status = %d, want %d", status, stdhttp.StatusUnsupportedMediaType)
		}
	})

	t.Run("accepts structured JSON suffix", func(t *testing.T) {
		c, _ := context(stdhttp.MethodPost, "/", bytes.NewBufferString(`{"name":"oak"}`))
		c.Request.Header.Set("Content-Type", "application/problem+json; charset=utf-8")
		var value input
		if err := c.BindJSON(&value); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("rejects unknown fields when configured", func(t *testing.T) {
		c, _ := context(stdhttp.MethodPost, "/", bytes.NewBufferString(`{"name":"oak","secret":true}`))
		c.SetJSONConfig(bluehttp.JSONConfig{
			RequireContentType:    true,
			DisallowUnknownFields: true,
		})
		var value input
		err := c.BindJSON(&value)
		if status := bluehttp.StatusForError(err); status != stdhttp.StatusBadRequest {
			t.Fatalf("status = %d, want %d", status, stdhttp.StatusBadRequest)
		}
		if err == nil || err.Error() != "Request body contains an unknown field" {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("returns safe syntax error", func(t *testing.T) {
		c, _ := context(stdhttp.MethodPost, "/", bytes.NewBufferString(`{"name":`))
		var value input
		err := c.BindJSON(&value)
		if err == nil || err.Error() != "Request body contains malformed JSON" {
			t.Fatalf("error = %v", err)
		}
		if !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Fatalf("error does not retain its cause: %v", err)
		}
	})
}

func TestPaginatedResponse(t *testing.T) {
	c, response := context(stdhttp.MethodGet, "/users", nil)
	if err := c.Paginated([]string{"a", "b"}, pagination.Pagination{Page: 2, PerPage: 20, Total: 150}); err != nil {
		t.Fatal(err)
	}
	var body struct {
		Success bool                `json:"success"`
		Data    []string            `json:"data"`
		Meta    pagination.Metadata `json:"meta"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Success || len(body.Data) != 2 || body.Meta.LastPage != 8 || body.Meta.CurrentPage != 2 {
		t.Fatalf("unexpected pagination response: %#v", body)
	}
}

func TestDefaultErrorHandler(t *testing.T) {
	t.Run("HTTP error", func(t *testing.T) {
		c, response := context(stdhttp.MethodGet, "/", nil)
		bluehttp.DefaultErrorHandler(c, bluehttp.NotFound("User not found"))
		if response.Code != stdhttp.StatusNotFound || response.Body.String() != "{\"success\":false,\"error\":{\"message\":\"User not found\"}}\n" {
			t.Fatalf("unexpected response: %d %q", response.Code, response.Body.String())
		}
	})
	t.Run("ordinary error", func(t *testing.T) {
		c, response := context(stdhttp.MethodGet, "/", nil)
		bluehttp.DefaultErrorHandler(c, errors.New("secret"))
		if response.Code != stdhttp.StatusInternalServerError || response.Body.String() != "{\"success\":false,\"error\":{\"message\":\"Internal Server Error\"}}\n" {
			t.Fatalf("unexpected response: %d %q", response.Code, response.Body.String())
		}
	})
}
