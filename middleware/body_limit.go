package middleware

import (
	stdhttp "net/http"

	bluehttp "github.com/yasserelgammal/blue-go/http"
)

// BodyLimit limits the number of request-body bytes a handler may read. It
// rejects a known oversized Content-Length before calling the next handler and
// also protects requests whose size is not known in advance.
func BodyLimit(maxBytes int64) Middleware {
	if maxBytes <= 0 {
		panic("blue: body limit must be greater than zero")
	}
	return func(next bluehttp.HandlerFunc) bluehttp.HandlerFunc {
		return func(c *bluehttp.Context) error {
			if c.Request.ContentLength > maxBytes {
				return bluehttp.PayloadTooLarge("Request body is too large")
			}
			if c.Request.Body != nil {
				c.Request.Body = stdhttp.MaxBytesReader(c.Response, c.Request.Body, maxBytes)
			}
			return next(c)
		}
	}
}
