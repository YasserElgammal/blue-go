package http

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

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
	decoder := json.NewDecoder(c.Request.Body)
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("blue: request body must contain one JSON value")
		}
		return err
	}
	return nil
}

// Bind is an alias for BindJSON.
func (c *Context) Bind(dst any) error { return c.BindJSON(dst) }
