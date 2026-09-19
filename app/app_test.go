package app_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/yasserelgammal/blue-go/app"
	bluehttp "github.com/yasserelgammal/blue-go/http"
)

func request(t *testing.T, handler http.Handler, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(method, target, nil))
	return recorder
}

func TestHTTPMethods(t *testing.T) {
	application := app.New()
	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
	for _, method := range methods {
		method := method
		application.Handle(method, "/resource", func(c *bluehttp.Context) error {
			return c.String(http.StatusOK, method)
		})
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			response := request(t, application, method, "/resource")
			if response.Code != http.StatusOK || response.Body.String() != method {
				t.Fatalf("got status %d body %q", response.Code, response.Body.String())
			}
		})
	}
}

func TestMiddlewareExecutionOrder(t *testing.T) {
	application := app.New()
	var calls []string
	middleware := func(name string) bluehttp.Middleware {
		return func(next bluehttp.HandlerFunc) bluehttp.HandlerFunc {
			return func(c *bluehttp.Context) error {
				calls = append(calls, name+":before")
				err := next(c)
				calls = append(calls, name+":after")
				return err
			}
		}
	}
	application.Use(middleware("app-1"), middleware("app-2"))
	api := application.Group("/api")
	api.GET("/hello", func(c *bluehttp.Context) error {
		calls = append(calls, "handler")
		return c.NoContent(http.StatusNoContent)
	})
	api.Use(middleware("group"))

	response := request(t, application, http.MethodGet, "/api/hello")
	if response.Code != http.StatusNoContent {
		t.Fatalf("got status %d", response.Code)
	}
	want := []string{"app-1:before", "app-2:before", "group:before", "handler", "group:after", "app-2:after", "app-1:after"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %#v, want %#v", calls, want)
	}
}

func TestErrorHandling(t *testing.T) {
	t.Run("HTTP error", func(t *testing.T) {
		application := app.New()
		application.GET("/users/:id", func(*bluehttp.Context) error {
			return bluehttp.NotFound("User not found")
		})
		response := request(t, application, http.MethodGet, "/users/9")
		if response.Code != http.StatusNotFound {
			t.Fatalf("got status %d", response.Code)
		}
		assertErrorMessage(t, response, "User not found")
	})

	t.Run("ordinary error is hidden", func(t *testing.T) {
		application := app.New()
		application.GET("/", func(*bluehttp.Context) error {
			return errors.New("database password leaked")
		})
		response := request(t, application, http.MethodGet, "/")
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("got status %d", response.Code)
		}
		assertErrorMessage(t, response, "Internal Server Error")
	})

	t.Run("custom handler", func(t *testing.T) {
		application := app.New()
		application.SetErrorHandler(func(c *bluehttp.Context, err error) {
			_ = c.JSON(http.StatusTeapot, map[string]string{"custom": err.Error()})
		})
		application.GET("/", func(*bluehttp.Context) error { return errors.New("tea") })
		response := request(t, application, http.MethodGet, "/")
		if response.Code != http.StatusTeapot || response.Body.String() != "{\"custom\":\"tea\"}\n" {
			t.Fatalf("got status %d body %q", response.Code, response.Body.String())
		}
	})
}

func TestCustomResponseFormatter(t *testing.T) {
	application := app.New()
	application.SetResponseFormatter(func(response bluehttp.Response) any {
		body := map[string]any{"ok": response.Success}
		if response.Data != nil {
			body["result"] = response.Data
		}
		if response.Error != nil {
			body["problem"] = response.Error.Message
		}
		return body
	})
	application.GET("/success", func(c *bluehttp.Context) error {
		return c.Respond(http.StatusOK, map[string]int{"id": 7})
	})
	application.GET("/failure", func(*bluehttp.Context) error {
		return bluehttp.BadRequest("Invalid request")
	})

	success := request(t, application, http.MethodGet, "/success")
	if success.Body.String() != "{\"ok\":true,\"result\":{\"id\":7}}\n" {
		t.Fatalf("success body = %q", success.Body.String())
	}
	failure := request(t, application, http.MethodGet, "/failure")
	if failure.Code != http.StatusBadRequest || failure.Body.String() != "{\"ok\":false,\"problem\":\"Invalid request\"}\n" {
		t.Fatalf("failure status %d body %q", failure.Code, failure.Body.String())
	}
}

func TestNotFound(t *testing.T) {
	response := request(t, app.New(), http.MethodGet, "/missing")
	if response.Code != http.StatusNotFound {
		t.Fatalf("got status %d", response.Code)
	}
	assertErrorMessage(t, response, "Not Found")
}

func TestMethodNotAllowed(t *testing.T) {
	application := app.New()
	application.GET("/users", func(*bluehttp.Context) error { return nil })
	application.POST("/users", func(*bluehttp.Context) error { return nil })
	response := request(t, application, http.MethodDelete, "/users")
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("got status %d", response.Code)
	}
	if allow := response.Header().Get("Allow"); allow != "GET, POST" {
		t.Fatalf("Allow = %q", allow)
	}
	assertErrorMessage(t, response, "Method Not Allowed")
}

func assertErrorMessage(t *testing.T, response *httptest.ResponseRecorder, want string) {
	t.Helper()
	var body struct {
		Success bool `json:"success"`
		Error   struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Message != want {
		t.Fatalf("message = %q, want %q", body.Error.Message, want)
	}
	if body.Success {
		t.Fatal("error response reported success")
	}
}
