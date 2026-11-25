# Changelog

All notable changes to Blue are documented in this file.

## [2.2.0] - 2026-11-25

### Changed

- Compare route specificity from left to right across segments (static, then
  parameter, then wildcard) instead of by total static segment count.
- Trim all trailing slashes during route and request path normalization while
  keeping repeated leading and internal slashes significant.
- Split request paths once per `MatchRequest` call rather than per candidate
  route.

## [2.1.0] - 2026-11-23

### Added

- Wildcard route segments (`*name`) for greedy catch-all matching. A wildcard
  segment must be the last segment in the path and captures all remaining
  path parts joined with `/` into `c.Param("name")`.
  Example: `app.GET("/files/*path", handler)` matches `/files/docs/api/v1.md`
  and sets `path` to `"docs/api/v1.md"`.
- Wildcard routes always lose to static and `:param` routes with the same
  prefix, preserving existing precedence rules.
- `RateLimit(RateLimitConfig)` middleware: fixed-window in-memory rate limiter
  with a configurable `KeyFunc` (defaults to `RemoteAddr`). Returns 429 when
  the limit is exceeded.
- `Timeout(time.Duration)` middleware: cancels the request context after the
  given duration and returns 503 Service Unavailable.
- `ETag()` middleware: computes a weak SHA-256 ETag for GET/HEAD responses and
  handles `If-None-Match` → 304 Not Modified automatically.
- `LoggerWith(LoggerConfig)` middleware: accepts a custom `*slog.Logger` and a
  `LevelForStatus` function to control the log level per response status.
  `Logger()` is unchanged and continues to use `slog.Default()` with
  Info/Error level selection.
- `DefaultLevelForStatus(status int) slog.Level` helper: Info below 500,
  Error for 5xx.

## [2.0.1] - 2026-09-20

### Fixed

- Update Go module path and internal/documentation imports to
  `github.com/yasserelgammal/blue-go/v2`.

## [2.0.0] - 2026-09-20

### Added

- Add unified response helpers through `Context.Respond`,
  `Context.RespondWithMessage`, and `Context.Error`.
- Add application- and request-level response customization with
  `SetResponseFormatter`.
- Add safe HTTP server defaults and configurable `ServerConfig` timeouts and
  header limits.
- Add `BodyLimit` middleware for known-length and streamed request bodies.
- Add configurable JSON binding with content-type validation and optional
  unknown-field rejection.
- Add safe client errors for malformed JSON, invalid JSON value types, empty
  bodies, unsupported media types, and oversized request bodies.

### Changed

- Route paginated and default error responses through the unified response
  formatter.
- Require `application/json` or an `application/*+json` media type when binding
  JSON with the default configuration.

### Compatibility

- Keep `Context.JSON` available for raw, unwrapped JSON responses.

## [1.0.0] - 2026-09-19

### Added

- Configurable CORS middleware with preflight request handling.
- Request ID middleware with incoming ID propagation, secure ID generation,
  response headers, and structured logging support.
- Graceful server lifecycle APIs: `Start`, `Shutdown`, and
  `RunWithGracefulShutdown`.
- Continuous integration checks for Go 1.22 and the latest stable Go release.

### Changed

- Expanded the README with middleware and server lifecycle documentation.
- Updated the project introduction to describe the framework's current
  features and database-independent design.

## [0.1.0] - 2026-09-17

### Added

- HTTP routing with static and parameterized paths.
- Route groups and application- or group-level middleware.
- Request binding and JSON, string, and empty response helpers.
- Centralized HTTP error handling.
- Structured request logging and panic recovery middleware.
- Database-agnostic pagination metadata and response helpers.
- Unit and public API integration tests.
