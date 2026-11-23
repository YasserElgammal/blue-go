package middleware

import (
	"log/slog"
	stdhttp "net/http"
	"time"

	bluehttp "github.com/yasserelgammal/blue-go/v2/http"
)

// LoggerConfig customizes how the logger middleware emits log records.
type LoggerConfig struct {
	// Logger is the slog.Logger to write to. When nil, slog.Default() is used.
	Logger *slog.Logger

	// LevelForStatus maps an HTTP status code to a slog.Level.
	// Called once per request with the final response status.
	// When nil the middleware uses Info for statuses below 500 and Error
	// for 5xx responses.
	LevelForStatus func(status int) slog.Level
}

// DefaultLevelForStatus returns Info for responses below 500 and Error for
// 5xx responses.
func DefaultLevelForStatus(status int) slog.Level {
	if status >= stdhttp.StatusInternalServerError {
		return slog.LevelError
	}
	return slog.LevelInfo
}

// Logger logs one structured record for each request using slog.Default().
func Logger() Middleware {
	return LoggerWith(LoggerConfig{})
}

// LoggerWith returns a logger middleware configured by cfg.
//
//	// Use a dedicated logger and emit warnings for 4xx responses.
//	app.Use(blue.LoggerWith(blue.LoggerConfig{
//	    Logger: myLogger,
//	    LevelForStatus: func(status int) slog.Level {
//	        switch {
//	        case status >= 500: return slog.LevelError
//	        case status >= 400: return slog.LevelWarn
//	        default:            return slog.LevelInfo
//	        }
//	    },
//	}))
func LoggerWith(cfg LoggerConfig) Middleware {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	levelFor := cfg.LevelForStatus
	if levelFor == nil {
		levelFor = DefaultLevelForStatus
	}

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

			logger.Log(c.Request.Context(), levelFor(status), "http request", attributes...)
			return err
		}
	}
}
