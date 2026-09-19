# Changelog

All notable changes to Blue are documented in this file.

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