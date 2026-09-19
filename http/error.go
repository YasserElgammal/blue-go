package http

import (
	"errors"
	stdhttp "net/http"
)

// HTTPError is an error with an HTTP response status and safe client message.
type HTTPError struct {
	Status  int
	Message string
	Cause   error
}

func (e *HTTPError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return stdhttp.StatusText(e.Status)
}

func (e *HTTPError) Unwrap() error { return e.Cause }

// NewHTTPError creates an error rendered as JSON by DefaultErrorHandler.
func NewHTTPError(status int, message string) *HTTPError {
	return &HTTPError{Status: status, Message: message}
}

func BadRequest(message string) *HTTPError {
	return NewHTTPError(stdhttp.StatusBadRequest, message)
}
func Unauthorized(message string) *HTTPError {
	return NewHTTPError(stdhttp.StatusUnauthorized, message)
}
func Forbidden(message string) *HTTPError {
	return NewHTTPError(stdhttp.StatusForbidden, message)
}
func NotFound(message string) *HTTPError {
	return NewHTTPError(stdhttp.StatusNotFound, message)
}
func MethodNotAllowed(message string) *HTTPError {
	return NewHTTPError(stdhttp.StatusMethodNotAllowed, message)
}
func PayloadTooLarge(message string) *HTTPError {
	return NewHTTPError(stdhttp.StatusRequestEntityTooLarge, message)
}
func UnsupportedMediaType(message string) *HTTPError {
	return NewHTTPError(stdhttp.StatusUnsupportedMediaType, message)
}
func InternalServerError(message string) *HTTPError {
	return NewHTTPError(stdhttp.StatusInternalServerError, message)
}

func withCause(httpError *HTTPError, cause error) *HTTPError {
	httpError.Cause = cause
	return httpError
}

// ErrorHandler writes an error returned by a handler.
type ErrorHandler func(*Context, error)

// DefaultErrorHandler renders HTTP errors as JSON and hides ordinary internal
// errors behind a generic 500 response.
func DefaultErrorHandler(c *Context, err error) {
	if c.Committed() {
		return
	}
	status := stdhttp.StatusInternalServerError
	message := stdhttp.StatusText(status)
	var httpError *HTTPError
	if errors.As(err, &httpError) {
		if httpError.Status >= 400 && httpError.Status <= 599 {
			status = httpError.Status
		}
		if httpError.Message != "" {
			message = httpError.Message
		} else if text := stdhttp.StatusText(status); text != "" {
			message = text
		}
	}
	_ = c.Error(status, message)
}

// StatusForError returns the response status represented by err.
func StatusForError(err error) int {
	var httpError *HTTPError
	if errors.As(err, &httpError) && httpError.Status >= 400 && httpError.Status <= 599 {
		return httpError.Status
	}
	return stdhttp.StatusInternalServerError
}
