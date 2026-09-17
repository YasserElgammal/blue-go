package app

import bluehttp "github.com/yasserelgammal/blue-go/http"

// HandlerFunc, Middleware, and ErrorHandler are aliases for Blue's HTTP
// primitives. They live here as part of the application-facing API.
type HandlerFunc = bluehttp.HandlerFunc
type Middleware = bluehttp.Middleware
type ErrorHandler = bluehttp.ErrorHandler
