# Changelog

All notable changes to Blue are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

[Unreleased]: https://github.com/YasserElgammal/blue-go/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/YasserElgammal/blue-go/releases/tag/v0.1.0
