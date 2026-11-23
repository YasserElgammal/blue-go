package middleware

import (
	"sync"
	"time"

	bluehttp "github.com/yasserelgammal/blue-go/v2/http"
)

// RateLimitConfig configures the RateLimit middleware.
//
// The limiter uses a fixed-window algorithm. Each unique key gets its own
// counter that resets every Window duration. When the counter exceeds
// MaxRequests the middleware returns 429 Too Many Requests.
type RateLimitConfig struct {
	// MaxRequests is the maximum number of requests allowed per window.
	// Defaults to 100 when zero.
	MaxRequests int

	// Window is the duration of each counting window.
	// Defaults to 1 second when zero.
	Window time.Duration

	// KeyFunc extracts the rate-limit key from the request. The key is
	// typically the client IP, an API key, or a user identifier.
	// Defaults to the RemoteAddr of the request when nil.
	KeyFunc func(*bluehttp.Context) string
}

// RateLimit returns a middleware that limits the request rate per key.
//
//	app.Use(blue.RateLimit(blue.RateLimitConfig{
//	    MaxRequests: 60,
//	    Window:      time.Minute,
//	    KeyFunc:     func(c *blue.Context) string { return c.Request.RemoteAddr },
//	}))
func RateLimit(config RateLimitConfig) Middleware {
	if config.MaxRequests <= 0 {
		config.MaxRequests = 100
	}
	if config.Window <= 0 {
		config.Window = time.Second
	}
	if config.KeyFunc == nil {
		config.KeyFunc = func(c *bluehttp.Context) string {
			return c.Request.RemoteAddr
		}
	}

	type bucket struct {
		count     int
		windowEnd time.Time
	}

	var mu sync.Mutex
	buckets := make(map[string]*bucket)

	return func(next bluehttp.HandlerFunc) bluehttp.HandlerFunc {
		return func(c *bluehttp.Context) error {
			key := config.KeyFunc(c)
			now := time.Now()

			mu.Lock()
			b, ok := buckets[key]
			if !ok || now.After(b.windowEnd) {
				b = &bucket{count: 0, windowEnd: now.Add(config.Window)}
				buckets[key] = b
			}
			b.count++
			exceeded := b.count > config.MaxRequests
			mu.Unlock()

			if exceeded {
				return bluehttp.NewHTTPError(429, "Too Many Requests")
			}
			return next(c)
		}
	}
}
