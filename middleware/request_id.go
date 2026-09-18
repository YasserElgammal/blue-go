package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	bluehttp "github.com/yasserelgammal/blue-go/http"
)

const requestIDHeader = "X-Request-ID"

// RequestID assigns an identifier to every request. A valid incoming
// X-Request-ID is preserved; otherwise a cryptographically random ID is used.
// The identifier is available through Context.RequestID and is returned in
// the X-Request-ID response header.
func RequestID() Middleware {
	return func(next bluehttp.HandlerFunc) bluehttp.HandlerFunc {
		return func(c *bluehttp.Context) error {
			id := strings.TrimSpace(c.Request.Header.Get(requestIDHeader))
			if !validRequestID(id) {
				var err error
				id, err = newRequestID()
				if err != nil {
					return err
				}
			}
			c.SetRequestID(id)
			c.Response.Header().Set(requestIDHeader, id)
			return next(c)
		}
	}
}

func validRequestID(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for _, character := range id {
		if character < 0x21 || character > 0x7e {
			return false
		}
	}
	return true
}

func newRequestID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("blue: generate request ID: %w", err)
	}
	return hex.EncodeToString(value[:]), nil
}
