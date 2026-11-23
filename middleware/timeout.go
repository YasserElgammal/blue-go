package middleware

import (
	"context"
	"errors"
	"time"

	bluehttp "github.com/yasserelgammal/blue-go/v2/http"
)

// Timeout cancels a request's context after the given duration and returns
// 503 Service Unavailable when the handler does not finish in time.
//
// Handlers that respect context cancellation will stop early; others will
// still run to completion but their response will not be sent.
//
//	app.Use(blue.Timeout(5 * time.Second))
func Timeout(d time.Duration) Middleware {
	if d <= 0 {
		panic("blue: Timeout duration must be greater than zero")
	}
	return func(next bluehttp.HandlerFunc) bluehttp.HandlerFunc {
		return func(c *bluehttp.Context) error {
			ctx, cancel := context.WithTimeout(c.Request.Context(), d)
			defer cancel()

			c.Request = c.Request.WithContext(ctx)

			done := make(chan error, 1)
			go func() { done <- next(c) }()

			select {
			case err := <-done:
				return err
			case <-ctx.Done():
				if errors.Is(ctx.Err(), context.DeadlineExceeded) {
					return bluehttp.NewHTTPError(503, "Request timed out")
				}
				return bluehttp.NewHTTPError(503, "Request cancelled")
			}
		}
	}
}
