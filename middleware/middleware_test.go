package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	bluehttp "github.com/yasserelgammal/blue-go/http"
	"github.com/yasserelgammal/blue-go/middleware"
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
