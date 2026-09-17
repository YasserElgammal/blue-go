package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	stdhttp "net/http"

	"github.com/yasserelgammal/blue-go/pagination"
)

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
	return c.JSON(stdhttp.StatusOK, struct {
		Data any                 `json:"data"`
		Meta pagination.Metadata `json:"meta"`
	}{Data: data, Meta: page.Metadata()})
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
