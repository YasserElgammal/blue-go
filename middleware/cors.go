package middleware

import (
	"net/http"
	"strconv"
	"strings"

	bluehttp "github.com/yasserelgammal/blue-go/v2/http"
)

// CORSConfig controls the cross-origin requests accepted by CORS middleware.
// Empty fields use conservative defaults suitable for JSON APIs.
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int // seconds; zero omits Access-Control-Max-Age
}

// CORS applies Cross-Origin Resource Sharing headers and handles browser
// preflight requests. It accepts zero or one configuration value.
func CORS(config ...CORSConfig) Middleware {
	if len(config) > 1 {
		panic("blue: CORS accepts at most one configuration")
	}
	settings := CORSConfig{}
	if len(config) == 1 {
		settings = config[0]
	}
	settings = corsDefaults(settings)

	return func(next bluehttp.HandlerFunc) bluehttp.HandlerFunc {
		return func(c *bluehttp.Context) error {
			origin := c.Request.Header.Get("Origin")
			if origin == "" {
				return next(c)
			}

			addVary(c.Response.Header(), "Origin")
			allowedOrigin, allowed := matchOrigin(origin, settings.AllowedOrigins, settings.AllowCredentials)
			preflightMethod := c.Request.Header.Get("Access-Control-Request-Method")
			isPreflight := c.Request.Method == http.MethodOptions && preflightMethod != ""
			if !allowed {
				if isPreflight {
					return bluehttp.Forbidden("CORS origin is not allowed")
				}
				return next(c)
			}
			requestedMethod := c.Request.Method
			if isPreflight {
				requestedMethod = preflightMethod
			}
			if !containsFold(settings.AllowedMethods, requestedMethod) {
				if isPreflight {
					return bluehttp.Forbidden("CORS method is not allowed")
				}
				return next(c)
			}

			headers := c.Response.Header()
			headers.Set("Access-Control-Allow-Origin", allowedOrigin)
			if settings.AllowCredentials {
				headers.Set("Access-Control-Allow-Credentials", "true")
			}
			if len(settings.ExposedHeaders) > 0 {
				headers.Set("Access-Control-Expose-Headers", strings.Join(settings.ExposedHeaders, ", "))
			}
			if !isPreflight {
				return next(c)
			}

			addVary(headers, "Access-Control-Request-Method")
			addVary(headers, "Access-Control-Request-Headers")
			requestedHeaders := splitHeaderList(c.Request.Header.Get("Access-Control-Request-Headers"))
			if !headersAllowed(requestedHeaders, settings.AllowedHeaders) {
				return bluehttp.Forbidden("CORS headers are not allowed")
			}

			headers.Set("Access-Control-Allow-Methods", strings.Join(settings.AllowedMethods, ", "))
			if len(requestedHeaders) > 0 {
				if containsFold(settings.AllowedHeaders, "*") {
					headers.Set("Access-Control-Allow-Headers", strings.Join(requestedHeaders, ", "))
				} else {
					headers.Set("Access-Control-Allow-Headers", strings.Join(settings.AllowedHeaders, ", "))
				}
			}
			if settings.MaxAge > 0 {
				headers.Set("Access-Control-Max-Age", strconv.Itoa(settings.MaxAge))
			}
			return c.NoContent(http.StatusNoContent)
		}
	}
}

func corsDefaults(config CORSConfig) CORSConfig {
	if len(config.AllowedOrigins) == 0 {
		config.AllowedOrigins = []string{"*"}
	}
	if len(config.AllowedMethods) == 0 {
		config.AllowedMethods = []string{
			http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
			http.MethodPatch, http.MethodDelete,
		}
	}
	if len(config.AllowedHeaders) == 0 {
		config.AllowedHeaders = []string{"Accept", "Authorization", "Content-Type"}
	}
	if config.MaxAge < 0 {
		panic("blue: CORS MaxAge cannot be negative")
	}
	return config
}

func matchOrigin(origin string, allowed []string, credentials bool) (string, bool) {
	for _, candidate := range allowed {
		if candidate == "*" {
			if credentials {
				return origin, true
			}
			return "*", true
		}
		if candidate == origin {
			return origin, true
		}
	}
	return "", false
}

func containsFold(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), target) {
			return true
		}
	}
	return false
}

func splitHeaderList(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			result = append(result, part)
		}
	}
	return result
}

func headersAllowed(requested, allowed []string) bool {
	if containsFold(allowed, "*") {
		return true
	}
	for _, header := range requested {
		if !containsFold(allowed, header) {
			return false
		}
	}
	return true
}

func addVary(headers http.Header, value string) {
	for _, existing := range headers.Values("Vary") {
		if containsFold(strings.Split(existing, ","), value) {
			return
		}
	}
	headers.Add("Vary", value)
}
