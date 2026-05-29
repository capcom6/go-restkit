# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2025-10-20

### Added

- Core HTTP client with JSON marshal/unmarshal support
  - `Do`, `DoWithHeaders`, `DoRAW`, `DoRAWWithHeaders` methods
  - Automatic `Accept: application/json` and `Content-Type: application/json` defaults
  - Base URL resolution via `Config.BaseURL`
  - Custom `http.Client` injection via `Config.Client`
- Three-tier error classification
  - `InternalError` — request construction failures (marshal, URL parse)
  - `InfrastructureError` — network-level failures (timeout, DNS, TLS)
  - `APIError` — server error responses with `StatusCode`, `Body`, `Headers`
  - `ErrorWithBody` interface with `RawBody()` and `ParseError()` methods
- Error detection helpers
  - `AsAPIError`, `IsInternalError`, `IsInfrastructureError`, `IsAPIError`
  - `IsClientError` (4xx), `IsServerError` (5xx)
- Sentinel errors: `ErrInvalidConfig`, `ErrEmptyMethod`, `ErrEmptyErrorBody`, `ErrUnmarshalJSON`
- Comprehensive test suite with table-driven tests
  - HTTP 204 No Content handling
  - Corrupt response/request handling
  - Response header access on both success and error paths
  - Error body parsing via `ParseError`
- CI/CD pipeline
  - GitHub Actions workflow with lint, test (race + coverage), and benchmark jobs
  - golangci-lint with strict golden config
  - Codecov coverage upload
- Development tooling
  - Makefile with `fmt`, `lint`, `test`, `benchmark`, `deps` targets
  - `.golangci.yml` with 50+ linters enabled
  - `.gitignore` for Go, macOS, Linux, Windows, VS Code
- Documentation
  - `ERROR_HANDLING.md` with comprehensive error handling guide and best practices
  - License: Apache 2.0

### Changed

- Headers type from `map[string]string` to `net/http.Header` (breaking)
- Path resolution to not encode query parameters
- Error body reading capped at 1 MiB to prevent memory exhaustion
- HTTP method is now required (`ErrEmptyMethod` on empty string)

### Fixed

- Lint issues across all source files (errcheck, naming, exhaustruct, etc.)

[0.1.0]: https://github.com/capcom6/go-restkit/releases/tag/v0.1.0
