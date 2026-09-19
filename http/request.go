package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	stdhttp "net/http"
	"strconv"
	"strings"
)

// JSONConfig controls how Bind and BindJSON validate JSON request bodies.
type JSONConfig struct {
	// RequireContentType rejects bodies whose media type is not application/json
	// or an application media type with a +json structured suffix.
	RequireContentType bool
	// DisallowUnknownFields rejects JSON object keys that do not map to the
	// destination value.
	DisallowUnknownFields bool
}

// DefaultJSONConfig returns the JSON binding defaults used by new applications
// and contexts.
func DefaultJSONConfig() JSONConfig {
	return JSONConfig{RequireContentType: true}
}

// HandlerFunc handles an HTTP request using a framework Context.
type HandlerFunc func(*Context) error

// Middleware wraps a HandlerFunc with additional behavior.
type Middleware func(HandlerFunc) HandlerFunc

// QueryInt returns an integer query parameter. It returns fallback when the
// parameter is absent or is not a valid base-10 integer.
func (c *Context) QueryInt(name string, fallback int) int {
	value := c.Query(name)
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return n
}

// BindJSON decodes exactly one JSON value from the request body into dst.
func (c *Context) BindJSON(dst any) error {
	if dst == nil {
		return fmt.Errorf("blue: BindJSON destination is nil")
	}
	if c.jsonConfig.RequireContentType && !isJSONContentType(c.Request.Header.Get("Content-Type")) {
		return withCause(
			UnsupportedMediaType("Content-Type must be application/json"),
			fmt.Errorf("blue: unsupported Content-Type %q", c.Request.Header.Get("Content-Type")),
		)
	}
	decoder := json.NewDecoder(c.Request.Body)
	if c.jsonConfig.DisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	if err := decoder.Decode(dst); err != nil {
		return jsonRequestError(err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return withCause(
				BadRequest("Request body must contain one JSON value"),
				fmt.Errorf("blue: request body contains multiple JSON values"),
			)
		}
		return jsonRequestError(err)
	}
	return nil
}

// Bind is an alias for BindJSON.
func (c *Context) Bind(dst any) error { return c.BindJSON(dst) }

func isJSONContentType(value string) bool {
	mediaType, _, err := mime.ParseMediaType(value)
	if err != nil {
		return false
	}
	mediaType = strings.ToLower(mediaType)
	return mediaType == "application/json" ||
		(strings.HasPrefix(mediaType, "application/") && strings.HasSuffix(mediaType, "+json"))
}

func jsonRequestError(err error) error {
	var maxBytesError *stdhttp.MaxBytesError
	if errors.As(err, &maxBytesError) {
		return withCause(PayloadTooLarge("Request body is too large"), err)
	}
	if errors.Is(err, io.EOF) {
		return withCause(BadRequest("Request body is required"), err)
	}
	var invalidDestination *json.InvalidUnmarshalError
	if errors.As(err, &invalidDestination) {
		return err
	}
	message := "Invalid JSON body"
	var syntaxError *json.SyntaxError
	var typeError *json.UnmarshalTypeError
	switch {
	case errors.As(err, &syntaxError), errors.Is(err, io.ErrUnexpectedEOF):
		message = "Request body contains malformed JSON"
	case errors.As(err, &typeError):
		message = "Request body contains a value of the wrong type"
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		message = "Request body contains an unknown field"
	}
	return withCause(BadRequest(message), err)
}
