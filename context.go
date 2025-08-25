package mig

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
)

// Context represents the context of an HTTP request. It provides methods to
// access request data and send responses.
//
// WARNING: A Context is created for each incoming request and is reused via a sync.Pool.
// For this reason, a Context is only valid for the lifetime of a request.
// It MUST NOT be stored or used across multiple goroutines.
// Doing so will lead to data races and unpredictable behavior, such as reading
// data from another request. For background operations, extract needed values first.
type Context struct {
	Request  *http.Request
	Response *Response
	Logger   *slog.Logger
	Mig      *Mig
	query    url.Values
}

// Reset reuses the context instance for a new request.
func (c *Context) Reset(r *http.Request, rw http.ResponseWriter) {
	c.Request = r
	if c.Response == nil {
		c.Response = &Response{}
	}
	c.Response.ResponseWriter = rw
	c.Response.status = 0
	c.Response.written = 0
	c.query = nil
	c.Logger = c.Mig.Logger // Reset to base logger
}

// Put stores a value in the context for the current request.
func (c *Context) Put(name any, value any) {
	newCtx := context.WithValue(c.Request.Context(), name, value)
	c.Request = c.Request.WithContext(newCtx)
}

// Get retrieves a value from the context.
func (c *Context) Get(name any) any {
	return c.Request.Context().Value(name)
}

// PathValue returns the value of a URL path parameter by its name.
// For example, for a route registered as "/users/{id}", PathValue("id") will
// return the corresponding value from the request URL.
func (c *Context) PathValue(name string) string {
	return c.Request.PathValue(name)
}

// QueryParam returns a query parameter value.
// If the parameter is missing, it returns the provided default.
// If the parameter is present but empty (?foo=), it returns the empty string.
func (c *Context) QueryParam(name string, def string) string {
	if c.query == nil {
		c.query = c.Request.URL.Query()
	}
	if val, ok := c.query[name]; ok {
		// Parameter exists, even if value is empty
		if len(val) > 0 {
			return val[0]
		}
		return ""
	}
	return def
}

// HasQueryParam reports whether the given query parameter exists at all.
func (c *Context) HasQueryParam(name string) bool {
	if c.query == nil {
		c.query = c.Request.URL.Query()
	}
	_, ok := c.query[name]
	return ok
}

func (c *Context) BindJSON(out any) error {
	decoder := json.NewDecoder(c.Request.Body)

	// This is an opinionated but very robust default.
	// It prevents clients from sending junk data or typos without realizing it.
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(out); err != nil {
		e := NewHTTPError(http.StatusBadRequest)
		e.Internal = err
		return e
	}
	return nil
}

func (c *Context) JSON(out any) error {
	res, err := json.Marshal(out)
	if err != nil {
		return err
	}
	c.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.Response.WriteHeader(http.StatusOK)
	_, err = c.Response.Write(res)
	return err
}

func (c *Context) View(name string, data any) error {
	buf := new(bytes.Buffer)
	if err := c.Mig.Renderer.Render(buf, name, data); err != nil {
		return err
	}
	return c.HTML(buf.String())
}

func (c *Context) HTML(html string) error {
	c.Response.Header().Set("content-type", "text/html; charset=utf-8")
	return c.Raw([]byte(html))
}

// Raw sends a raw response without setting content type.
func (c *Context) Raw(data []byte) error {
	_, err := c.Response.Write(data)
	return err
}

// String sends a plain text response with a given status code.
func (c *Context) String(code int, s string) error {
	c.Response.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.Response.WriteHeader(code)
	_, err := c.Response.Write([]byte(s))
	return err
}

// NoContent sends an empty response with the given status code.
func (c *Context) NoContent(code int) error {
	c.Response.WriteHeader(code)
	return nil
}

// Redirect sends an HTTP redirect response.
func (c *Context) Redirect(code int, url string) error {
	http.Redirect(c.Response, c.Request, url, code)
	return nil
}

// The key for storing the request ID in a context.Context.
type requestIDKey struct{}

// RequestIDHeader is the name of the HTTP header used for the request ID.
const RequestIDHeader = "X-Request-ID"

// SetRequestID attaches the request ID to this Context, its Request.Context, and the response header.
func (c *Context) SetRequestID(id string) {
	if id == "" {
		return
	}
	ctx := context.WithValue(c.Request.Context(), requestIDKey{}, id)
	c.Request = c.Request.WithContext(ctx)

	c.Response.Header().Set(RequestIDHeader, id)

	if c.Logger != nil {
		c.Logger = c.Logger.With(slog.String("id", id))
	}
}

// RequestID returns the request ID, or "" if none set.
func (c *Context) RequestID() string {
	return RequestIDFromContext(c.Request.Context())
}

// RequestIDFromContext extracts a request ID from a stdlib context.Context.
func RequestIDFromContext(ctx context.Context) string {
	if v := ctx.Value(requestIDKey{}); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
