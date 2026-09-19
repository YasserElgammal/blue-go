package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	stdhttp "net/http"

	"github.com/yasserelgammal/blue-go/v2/pagination"
)

// Response is the framework's unified representation of an API response.
// Status is available to custom formatters but is not included in the default
// JSON body.
type Response struct {
	Status  int            `json:"-"`
	Success bool           `json:"success"`
	Message string         `json:"message,omitempty"`
	Data    any            `json:"data,omitempty"`
	Meta    any            `json:"meta,omitempty"`
	Error   *ResponseError `json:"error,omitempty"`
}

// ResponseError contains safe error information returned to an API client.
type ResponseError struct {
	Message string `json:"message"`
}

// ResponseFormatter converts a unified Response into the value serialized as
// JSON. It can be configured per application or per request.
type ResponseFormatter func(Response) any

// DefaultResponseFormatter returns Blue's standard response envelope.
func DefaultResponseFormatter(response Response) any { return response }

// JSON writes value as a JSON response.
func (c *Context) JSON(status int, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	c.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.Response.WriteHeader(status)
	_, err = io.Copy(c.Response, bytes.NewReader(payload))
	return err
}

// Respond writes a unified JSON response containing data.
func (c *Context) Respond(status int, data any) error {
	return c.writeResponse(Response{
		Status:  status,
		Success: status >= 200 && status < 400,
		Data:    data,
	})
}

// RespondWithMessage writes a unified JSON response containing a message and
// optional data.
func (c *Context) RespondWithMessage(status int, message string, data any) error {
	return c.writeResponse(Response{
		Status:  status,
		Success: status >= 200 && status < 400,
		Message: message,
		Data:    data,
	})
}

// Error writes a unified JSON error response. message must be safe to expose
// to the client.
func (c *Context) Error(status int, message string) error {
	return c.writeResponse(Response{
		Status:  status,
		Success: false,
		Error:   &ResponseError{Message: message},
	})
}

func (c *Context) writeResponse(response Response) error {
	formatter := c.responseFormatter
	if formatter == nil {
		formatter = DefaultResponseFormatter
	}
	return c.JSON(response.Status, formatter(response))
}

// String writes a UTF-8 plain-text response.
func (c *Context) String(status int, value string, args ...any) error {
	if len(args) > 0 {
		value = fmt.Sprintf(value, args...)
	}
	c.Response.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.Response.WriteHeader(status)
	_, err := io.WriteString(c.Response, value)
	return err
}

// NoContent writes a response with no body.
func (c *Context) NoContent(status int) error {
	c.Response.WriteHeader(status)
	return nil
}

// Paginated writes a 200 JSON response containing data and pagination metadata.
func (c *Context) Paginated(data any, page pagination.Pagination) error {
	return c.writeResponse(Response{
		Status:  stdhttp.StatusOK,
		Success: true,
		Data:    data,
		Meta:    page.Metadata(),
	})
}

type responseState struct {
	stdhttp.ResponseWriter
	status    int
	committed bool
}

func (w *responseState) WriteHeader(status int) {
	if w.committed {
		return
	}
	w.status = status
	w.committed = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseState) Write(payload []byte) (int, error) {
	if !w.committed {
		w.WriteHeader(stdhttp.StatusOK)
	}
	return w.ResponseWriter.Write(payload)
}

func (w *responseState) Unwrap() stdhttp.ResponseWriter { return w.ResponseWriter }
