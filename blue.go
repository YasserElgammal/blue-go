// Package blue provides a small, explicit framework for building REST APIs.
// Its root API is a stable façade over focused implementation packages.
package blue

import (
	"github.com/yasserelgammal/blue-go/app"
	bluehttp "github.com/yasserelgammal/blue-go/http"
	standard "github.com/yasserelgammal/blue-go/middleware"
	page "github.com/yasserelgammal/blue-go/pagination"
	"github.com/yasserelgammal/blue-go/router"
)

type App = app.App
type Context = bluehttp.Context
type HandlerFunc = bluehttp.HandlerFunc
type Middleware = bluehttp.Middleware
type ErrorHandler = bluehttp.ErrorHandler
type Response = bluehttp.Response
type ResponseError = bluehttp.ResponseError
type ResponseFormatter = bluehttp.ResponseFormatter
type JSONConfig = bluehttp.JSONConfig
type HTTPError = bluehttp.HTTPError
type Pagination = page.Pagination
type Group = router.Group
type CORSConfig = standard.CORSConfig
type ServerConfig = app.ServerConfig

func New() *App { return app.New() }

func Logger() Middleware                   { return standard.Logger() }
func Recover() Middleware                  { return standard.Recover() }
func RequestID() Middleware                { return standard.RequestID() }
func CORS(config ...CORSConfig) Middleware { return standard.CORS(config...) }
func BodyLimit(maxBytes int64) Middleware  { return standard.BodyLimit(maxBytes) }

func DefaultServerConfig() ServerConfig { return app.DefaultServerConfig() }
func DefaultJSONConfig() JSONConfig     { return bluehttp.DefaultJSONConfig() }

func NewHTTPError(status int, message string) *HTTPError {
	return bluehttp.NewHTTPError(status, message)
}
func BadRequest(message string) *HTTPError       { return bluehttp.BadRequest(message) }
func Unauthorized(message string) *HTTPError     { return bluehttp.Unauthorized(message) }
func Forbidden(message string) *HTTPError        { return bluehttp.Forbidden(message) }
func NotFound(message string) *HTTPError         { return bluehttp.NotFound(message) }
func MethodNotAllowed(message string) *HTTPError { return bluehttp.MethodNotAllowed(message) }
func PayloadTooLarge(message string) *HTTPError  { return bluehttp.PayloadTooLarge(message) }
func UnsupportedMediaType(message string) *HTTPError {
	return bluehttp.UnsupportedMediaType(message)
}
func InternalServerError(message string) *HTTPError { return bluehttp.InternalServerError(message) }
