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

func TestSpecificityIsComparedFromLeftToRight(t *testing.T) {
	r := router.New()
	r.Register(http.MethodGet, "/:section/settings", taggedHandler("later-static"))
	r.Register(http.MethodGet, "/users/:action", taggedHandler("earlier-static"))

	matched, _ := r.MatchRequest(http.MethodGet, "/users/settings")
	if matched == nil || matched.Handler(nil).Error() != "earlier-static" {
		t.Fatal("the route with the earlier static segment should take precedence")
	}
}

func TestAllTrailingSlashesAreIgnored(t *testing.T) {
	r := router.New()
	r.Register(http.MethodGet, "/users///", taggedHandler("users"))

	for _, path := range []string{"/users", "/users/", "/users///"} {
		matched, _ := r.MatchRequest(http.MethodGet, path)
		if matched == nil || matched.Handler(nil).Error() != "users" {
			t.Fatalf("path %q did not match the normalized route", path)
		}
	}
}

func TestRepeatedInternalSlashesRemainSignificant(t *testing.T) {
	r := router.New()
	r.Register(http.MethodGet, "/users/:id", taggedHandler("normal"))
	r.Register(http.MethodGet, "/users//:id", taggedHandler("repeated"))

	matched, _ := r.MatchRequest(http.MethodGet, "/users//42")
	if matched == nil || matched.Handler(nil).Error() != "repeated" {
		t.Fatal("repeated internal slash should remain a distinct path segment")
	}
}

func TestRepeatedLeadingSlashesRemainSignificant(t *testing.T) {
	r := router.New()
	r.Register(http.MethodGet, "/users", taggedHandler("normal"))
	r.Register(http.MethodGet, "//users", taggedHandler("repeated"))

	matched, _ := r.MatchRequest(http.MethodGet, "//users")
	if matched == nil || matched.Handler(nil).Error() != "repeated" {
		t.Fatal("repeated leading slash should remain a distinct path segment")
	}
}

func TestWildcardRouteMatching(t *testing.T) {
	r := router.New()
	r.Register(http.MethodGet, "/files/*path", taggedHandler("wildcard"))

	tests := []struct {
		requestPath  string
		wantMatch    bool
		wantWildcard string
	}{
		{"/files/readme.txt", true, "readme.txt"},
		{"/files/docs/api/v1.md", true, "docs/api/v1.md"},
		{"/files/", true, ""},
		{"/other", false, ""},
	}
	for _, tt := range tests {
		matched, _ := r.MatchRequest(http.MethodGet, tt.requestPath)
		if tt.wantMatch && matched == nil {
			t.Fatalf("path %q: expected match, got none", tt.requestPath)
		}
		if !tt.wantMatch && matched != nil {
			t.Fatalf("path %q: expected no match, got one", tt.requestPath)
		}
		if tt.wantMatch && matched.Params["path"] != tt.wantWildcard {
			t.Fatalf("path %q: wildcard param = %q, want %q", tt.requestPath, matched.Params["path"], tt.wantWildcard)
		}
	}
}

func TestWildcardLosesToStaticAndParam(t *testing.T) {
	r := router.New()
	r.Register(http.MethodGet, "/files/*path", taggedHandler("wildcard"))
	r.Register(http.MethodGet, "/files/:name", taggedHandler("param"))
	r.Register(http.MethodGet, "/files/readme", taggedHandler("static"))

	if matched, _ := r.MatchRequest(http.MethodGet, "/files/readme"); matched == nil || matched.Handler(nil).Error() != "static" {
		t.Fatal("static route should beat wildcard")
	}
	if matched, _ := r.MatchRequest(http.MethodGet, "/files/something"); matched == nil || matched.Handler(nil).Error() != "param" {
		t.Fatal(":param route should beat wildcard")
	}
	if matched, _ := r.MatchRequest(http.MethodGet, "/files/a/b/c"); matched == nil || matched.Handler(nil).Error() != "wildcard" {
		t.Fatal("multi-segment path should match wildcard")
	}
}

func TestWildcardWithParamPrefix(t *testing.T) {
	r := router.New()
	r.Register(http.MethodGet, "/users/:id/files/*path", taggedHandler("user-files"))

	matched, _ := r.MatchRequest(http.MethodGet, "/users/42/files/docs/readme.txt")
	if matched == nil {
		t.Fatal("expected match")
	}
	if matched.Params["id"] != "42" {
		t.Fatalf("id = %q, want %q", matched.Params["id"], "42")
	}
	if matched.Params["path"] != "docs/readme.txt" {
		t.Fatalf("path = %q, want %q", matched.Params["path"], "docs/readme.txt")
	}
}

func TestWildcardPanicsWhenNotLast(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for mid-route wildcard")
		}
	}()
	r := router.New()
	r.Register(http.MethodGet, "/files/*path/extra", taggedHandler("bad"))
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
