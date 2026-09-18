# Blue

Blue is a lightweight Go web framework for building REST APIs.

It provides routing, route groups, request binding, response helpers, centralized error handling, pagination, graceful shutdown, and ready-to-use middleware for logging, panic recovery, CORS, and request IDs.

Blue favors explicit code, the Go standard library, and a focused public API. It stays independent of application architecture and persistence choices, allowing you to use any database driver, ORM, validation library, or project structure.

Authentication, dependency injection, templating, and code generation remain application-level choices rather than framework requirements.


## Package structure

The implementation is separated by responsibility while the root `blue`
package remains the convenient application-facing API:

```text
blue-go/
|-- app/
|   |-- app.go                # application and HTTP server
|   |-- handler.go            # application-facing handler aliases
|   `-- app_test.go
|-- router/
|   |-- router.go             # registration and matching
|   |-- route.go              # route parsing and parameters
|   |-- group.go              # route groups
|   `-- router_test.go
|-- middleware/
|   |-- middleware.go         # middleware type alias
|   |-- logger.go             # structured request logging
|   |-- cors.go               # CORS headers and preflight requests
|   |-- request_id.go         # request ID generation and propagation
|   |-- recover.go            # panic recovery
|   `-- middleware_test.go
|-- http/
|   |-- context.go            # request context
|   |-- request.go            # query and JSON body helpers
|   |-- response.go           # response helpers
|   |-- error.go              # HTTP errors and error handling
|   `-- context_test.go
|-- pagination/
|   |-- pagination.go         # pagination values and metadata
|   `-- pagination_test.go
|-- integration/
|   `-- public_api_test.go    # root API integration test
|-- blue.go                   # stable root-package facade
|-- go.mod
|-- LICENSE
`-- README.md
```

Dependencies flow from the small HTTP and pagination primitives toward the
router, middleware, and application packages; no package imports back up the
chain.

## Requirements and installation

Blue requires Go 1.22 or newer.

```sh
go get github.com/yasserelgammal/blue-go
```

## Quick start

```go
package main

import (
    "net/http"

    blue "github.com/yasserelgammal/blue-go"
)

func main() {
    app := blue.New()

    app.Use(blue.Logger(), blue.Recover())

    app.GET("/health", func(c *blue.Context) error {
        return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
    })

    if err := app.Run(":8080"); err != nil {
        panic(err)
    }
}
```

An `*App` implements `http.Handler`, so it also works directly with
`http.Server`, `httptest`, and other standard-library HTTP tools.

## Defining routes

Blue supports `GET`, `POST`, `PUT`, `PATCH`, and `DELETE`, as well as a generic
`Handle` method.

```go
app.GET("/users", listUsers)
app.POST("/users", createUser)
app.PUT("/users/:id", replaceUser)
app.PATCH("/users/:id", updateUser)
app.DELETE("/users/:id", deleteUser)
app.Handle("OPTIONS", "/users", usersOptions)
```

Handlers return an error so failures flow through one centralized error
handler:

```go
func listUsers(c *blue.Context) error {
    return c.JSON(http.StatusOK, []User{})
}
```

Invalid, duplicate, or ambiguous route declarations panic during registration,
so configuration mistakes fail early. Static routes take precedence over
parameter routes.

## Path and query parameters

A parameter occupies a complete path segment and begins with `:`.

```go
app.GET("/users/:id", func(c *blue.Context) error {
    id := c.Param("id")
    page := c.QueryInt("page", 1)
    filter := c.Query("filter")

    return c.JSON(http.StatusOK, map[string]any{
        "id": id, "page": page, "filter": filter,
    })
})
```

`QueryInt` returns its fallback when the value is missing or is not an integer.

Decode a JSON request body with `Bind` or `BindJSON`:

```go
var input CreateUserInput
if err := c.Bind(&input); err != nil {
    return blue.BadRequest("Invalid JSON body")
}
```

The underlying `*http.Request` and `http.ResponseWriter` remain available as
`c.Request` and `c.Response`.

## Route groups

Groups apply a shared prefix without imposing a controller or directory layout.
Groups can be nested.

```go
api := app.Group("/api/v1")
api.GET("/users", listUsers)
api.GET("/users/:id", showUser)

admin := api.Group("/admin")
admin.DELETE("/users/:id", deleteUser)
```

## Middleware

A middleware is a function that wraps a handler. Application middleware also
runs for framework-generated 404 and 405 responses.

```go
func AddHeader() blue.Middleware {
    return func(next blue.HandlerFunc) blue.HandlerFunc {
        return func(c *blue.Context) error {
            c.Response.Header().Set("X-App", "example")
            return next(c)
        }
    }
}

app.Use(blue.Logger(), blue.Recover(), AddHeader())
```

Groups have their own middleware. Parent middleware runs before nested group
middleware.

```go
api := app.Group("/api")
api.Use(Auth())
api.GET("/profile", profile)
```

Middleware executes in the order passed to `Use`; its code after `next` runs in
reverse order. Blue includes optional structured request logging via
`log/slog`, request IDs, CORS, and panic recovery middleware.

### Request IDs

`RequestID` preserves a valid incoming `X-Request-ID` or generates a random
one. The value is returned in the response header and is automatically added
to records written by `Logger`:

```go
app.Use(blue.RequestID(), blue.Logger())

app.GET("/request-id", func(c *blue.Context) error {
    return c.JSON(http.StatusOK, map[string]string{
        "request_id": c.RequestID(),
    })
})
```

### CORS

`CORS` adds cross-origin response headers and handles valid browser preflight
requests before routing:

```go
app.Use(blue.RequestID(), blue.CORS(blue.CORSConfig{
    AllowedOrigins: []string{"https://example.com"},
    AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
    AllowedHeaders: []string{"Authorization", "Content-Type"},
    ExposedHeaders: []string{"X-Request-ID"},
    MaxAge:         600,
}))
```

Calling `blue.CORS()` without a configuration allows all origins without
credentials and uses the common REST methods and JSON request headers.
Place `RequestID` before `CORS` when preflight responses should also receive a
request ID.

## Server lifecycle

`Run` and `Start` both block while the server is running. `Shutdown` can be
called from another goroutine to wait for active requests to finish:

```go
go func() {
    if err := app.Start(":8080"); err != nil {
        log.Printf("server stopped: %v", err)
    }
}()

// Later, after receiving your application's shutdown signal:
shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
if err := app.Shutdown(shutdownCtx); err != nil {
    log.Printf("shutdown failed: %v", err)
}
```

For the common case, `RunWithGracefulShutdown` handles `Ctrl+C` and `SIGTERM`
and gives active requests up to ten seconds to complete:

```go
if err := app.RunWithGracefulShutdown(":8080"); err != nil {
    panic(err)
}
```

## Responses

Context response helpers return write or serialization errors for the central
error handler:

```go
return c.JSON(http.StatusOK, value)
return c.String(http.StatusOK, "hello %s", name)
return c.NoContent(http.StatusNoContent)
```

JSON responses use `encoding/json` and the
`application/json; charset=utf-8` content type.

## Error handling

Return an `HTTPError` when a message is safe to send to the client:

```go
func showUser(c *blue.Context) error {
    user, err := findUser(c.Param("id"))
    if errors.Is(err, ErrNotFound) {
        return blue.NotFound("User not found")
    }
    if err != nil {
        return err
    }
    return c.JSON(http.StatusOK, user)
}
```

Helpers include `BadRequest`, `Unauthorized`, `Forbidden`, `NotFound`,
`MethodNotAllowed`, and `InternalServerError`. `NewHTTPError` accepts any HTTP
error status.

By default, errors are returned as:

```json
{"error":{"message":"User not found"}}
```

Ordinary errors receive a generic 500 message so internal details are not
exposed. Replace the policy when needed:

```go
app.SetErrorHandler(func(c *blue.Context, err error) {
    // Log err and write the response used by your API.
    _ = c.JSON(http.StatusInternalServerError, map[string]string{
        "error": "request failed",
    })
})
```

## Pagination

Pagination metadata is database- and ORM-agnostic. Your application remains
responsible for querying the records and total count.

```go
page := c.QueryInt("page", 1)
perPage := c.QueryInt("per_page", 20)

return c.Paginated(users, blue.Pagination{
    Page: page,
    PerPage: perPage,
    Total: total,
})
```

The response has a consistent shape:

```json
{
  "data": [],
  "meta": {
    "current_page": 1,
    "per_page": 20,
    "total": 150,
    "last_page": 8
  }
}
```

Invalid pagination values are normalized: page and per-page are at least 1,
total is at least 0, and an empty result has a last page of 0.

`Pagination.Metadata()` can also be used when an application needs normalized
metadata without writing a response:

```go
pageInfo := blue.Pagination{Page: page, PerPage: perPage, Total: total}
meta := pageInfo.Metadata()
```

## Testing

Because an app is an `http.Handler`, use `httptest` without starting a server:

```go
recorder := httptest.NewRecorder()
request := httptest.NewRequest(http.MethodGet, "/health", nil)
app.ServeHTTP(recorder, request)

if recorder.Code != http.StatusOK {
    t.Fatalf("got status %d", recorder.Code)
}
```

Run the framework test suite with:

```sh
go test ./...
go vet ./...
```

Tests are colocated with the package they cover:

- `app/app_test.go` covers HTTP methods, middleware composition, centralized
  errors, 404 responses, and 405 responses.
- `router/router_test.go` covers registration, matching, path parameters,
  static-route precedence, and nested groups.
- `http/context_test.go` covers query parsing, body binding, responses, and
  default error rendering.
- `middleware/middleware_test.go` covers panic recovery.
- `pagination/pagination_test.go` covers metadata calculation and value
  normalization.
- `integration/public_api_test.go` verifies the public `blue` package as an
  application would use it.

There are intentionally no `_test.go` files in the repository root.

## Philosophy

- Prefer explicit code and composition over magic.
- Keep the core small and use the standard library first.
- Make common REST API operations convenient.
- Keep business logic, persistence, and application organization in the user's
  application.
- Add abstractions only when they solve a demonstrated problem.

## License

Blue is available under the [MIT License](LICENSE).
