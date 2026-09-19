package router_test

import (
	"errors"
	"net/http"
	"reflect"
	"testing"

	bluehttp "github.com/yasserelgammal/blue-go/v2/http"
	"github.com/yasserelgammal/blue-go/v2/router"
)

func taggedHandler(tag string) bluehttp.HandlerFunc {
	return func(*bluehttp.Context) error { return errors.New(tag) }
}

func TestRegistrationAndHTTPMethods(t *testing.T) {
	r := router.New()
	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
	for _, method := range methods {
		r.Register(method, "/resource", taggedHandler(method))
	}
	for _, method := range methods {
		matched, _ := r.MatchRequest(method, "/resource")
		if matched == nil || matched.Handler(nil).Error() != method {
			t.Fatalf("method %s did not match its handler", method)
		}
	}
}

func TestDuplicateAndAmbiguousRoutesPanic(t *testing.T) {
	r := router.New()
	r.Register(http.MethodGet, "/users/:id", taggedHandler("first"))
	defer func() {
		if recover() == nil {
			t.Fatal("expected ambiguous route registration to panic")
		}
	}()
	r.Register(http.MethodGet, "/users/:name", taggedHandler("second"))
}

func TestRouteMatchingAndPathParameters(t *testing.T) {
	r := router.New()
	r.Register(http.MethodGet, "/users/:id/books/:book", taggedHandler("matched"))
	matched, _ := r.MatchRequest(http.MethodGet, "/users/42/books/blue")
	if matched == nil {
		t.Fatal("expected route to match")
	}
	want := map[string]string{"id": "42", "book": "blue"}
	if !reflect.DeepEqual(matched.Params, want) {
		t.Fatalf("params = %#v, want %#v", matched.Params, want)
	}
}

func TestStaticRouteTakesPrecedence(t *testing.T) {
	r := router.New()
	r.Register(http.MethodGet, "/users/:id", taggedHandler("parameter"))
	r.Register(http.MethodGet, "/users/me", taggedHandler("static"))
	matched, _ := r.MatchRequest(http.MethodGet, "/users/me")
	if matched == nil || matched.Handler(nil).Error() != "static" {
		t.Fatal("static route did not take precedence")
	}
}

func TestNestedRouteGroups(t *testing.T) {
	r := router.New()
	api := r.Group("/api/v1")
	admin := api.Group("/admin")
	admin.GET("/users/:id", taggedHandler("admin-user"))

	matched, _ := r.MatchRequest(http.MethodGet, "/api/v1/admin/users/12")
	if matched == nil || matched.Params["id"] != "12" {
		t.Fatalf("unexpected match: %#v", matched)
	}
}
