package middleware

import (
	"fmt"
	"log/slog"
	stdhttp "net/http"
	"runtime/debug"

	bluehttp "github.com/yasserelgammal/blue-go/http"
)

// Recover converts panics into an internal server error. http.ErrAbortHandler
// is re-panicked so net/http can preserve its documented connection behavior.
func Recover() Middleware {
	return func(next bluehttp.HandlerFunc) bluehttp.HandlerFunc {
		return func(c *bluehttp.Context) (err error) {
			defer func() {
				if recovered := recover(); recovered != nil {
					if recovered == stdhttp.ErrAbortHandler {
						panic(recovered)
					}
					slog.Default().Error("panic recovered",
						"panic", recovered,
						"method", c.Request.Method,
						"path", c.Request.URL.Path,
						"stack", string(debug.Stack()),
					)
					err = &bluehttp.HTTPError{
						Status:  stdhttp.StatusInternalServerError,
						Message: stdhttp.StatusText(stdhttp.StatusInternalServerError),
						Cause:   fmt.Errorf("panic: %v", recovered),
					}
				}
			}()
			return next(c)
		}
	}
}
