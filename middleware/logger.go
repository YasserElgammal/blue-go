package middleware

import (
	"log/slog"
	stdhttp "net/http"
	"time"

	bluehttp "github.com/yasserelgammal/blue-go/v2/http"
)

// Logger logs one structured record for each request using slog.Default().
func Logger() Middleware {
	return func(next bluehttp.HandlerFunc) bluehttp.HandlerFunc {
		return func(c *bluehttp.Context) error {
			started := time.Now()
			err := next(c)
			status := c.Status()
			if status == 0 {
				if err != nil {
					status = bluehttp.StatusForError(err)
				} else {
					status = stdhttp.StatusOK
				}
			}
			attributes := []any{
				"method", c.Request.Method,
				"path", c.Request.URL.Path,
				"status", status,
				"duration", time.Since(started),
			}
			if requestID := c.RequestID(); requestID != "" {
				attributes = append(attributes, "request_id", requestID)
			}
			if err != nil {
				attributes = append(attributes, "error", err)
			}
			slog.Default().Info("http request", attributes...)
			return err
		}
	}
}
