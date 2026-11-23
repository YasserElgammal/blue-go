package middleware

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"strings"

	bluehttp "github.com/yasserelgammal/blue-go/v2/http"
)

// ETag adds a weak ETag header to every response that has a body and handles
// conditional requests. When the client sends an If-None-Match header that
// matches the computed ETag, the middleware responds with 304 Not Modified and
// no body — saving bandwidth for read-heavy endpoints.
//
// The ETag is derived from a SHA-256 hash of the response body. Only GET and
// HEAD requests are considered; POST, PUT, PATCH, and DELETE requests pass
// through unchanged.
//
//	app.Use(blue.ETag())
func ETag() Middleware {
	return func(next bluehttp.HandlerFunc) bluehttp.HandlerFunc {
		return func(c *bluehttp.Context) error {
			// Only GET and HEAD benefit from ETags.
			method := c.Request.Method
			if method != http.MethodGet && method != http.MethodHead {
				return next(c)
			}

			// Capture the response into a buffer so we can hash it.
			buf := &etagResponseWriter{ResponseWriter: c.Response, body: &bytes.Buffer{}}
			original := c.Response
			c.Response = buf

			err := next(c)

			// Restore the original writer regardless of outcome.
			c.Response = original

			if err != nil {
				return err
			}
			if buf.status == 0 {
				buf.status = http.StatusOK
			}

			// Only tag successful, non-empty responses.
			if buf.status < 200 || buf.status >= 300 || buf.body.Len() == 0 {
				writeCapture(original, buf)
				return nil
			}

			// Compute weak ETag: W/"<first12hexchars>"
			sum := sha256.Sum256(buf.body.Bytes())
			etag := fmt.Sprintf(`W/"%x"`, sum[:6])

			// Check If-None-Match.
			if match := c.Request.Header.Get("If-None-Match"); match != "" {
				for _, candidate := range strings.Split(match, ",") {
					if strings.TrimSpace(candidate) == etag {
						// Copy headers but skip body.
						copyHeaders(original, buf)
						original.Header().Set("ETag", etag)
						original.WriteHeader(http.StatusNotModified)
						return nil
					}
				}
			}

			buf.Header().Set("ETag", etag)
			writeCapture(original, buf)
			return nil
		}
	}
}

// etagResponseWriter captures the response status, headers, and body.
type etagResponseWriter struct {
	http.ResponseWriter
	status int
	body   *bytes.Buffer
}

func (w *etagResponseWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}

func (w *etagResponseWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(p)
}

func (w *etagResponseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// copyHeaders copies all captured response headers to dst.
func copyHeaders(dst http.ResponseWriter, src *etagResponseWriter) {
	for key, values := range src.Header() {
		for _, v := range values {
			dst.Header().Add(key, v)
		}
	}
}

// writeCapture flushes the captured status, headers, and body to dst.
func writeCapture(dst http.ResponseWriter, src *etagResponseWriter) {
	copyHeaders(dst, src)
	dst.WriteHeader(src.status)
	_, _ = io.Copy(dst, src.body)
}
