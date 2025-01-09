package mig

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
)

type Context struct {
	mu             sync.RWMutex
	store          map[string]any
	Request        *http.Request
	ResponseWriter http.ResponseWriter
	Logger         *slog.Logger
	Mig            *Mig
	query          url.Values
}

func (c *Context) Reset(r *http.Request, rw http.ResponseWriter) {
	c.Request = r
	c.ResponseWriter = rw
	c.query = nil
	// c.mu.Unlock()
}

func (c *Context) Put(name string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.store == nil {
		c.store = make(map[string]any)
	}
	c.store[name] = value
}

func (c *Context) Get(name string) any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.store[name]
}

func (c *Context) PathValue(name string) string {
	return c.Request.PathValue(name)
}

func (c *Context) QueryParam(name string, def string) string {
	if c.query == nil {
		c.query = c.Request.URL.Query()
	}
	if val := c.query.Get(name); val != "" {
		return val
	}
	return def
}

func (c *Context) JSON(out any) error {
	res, err := json.Marshal(out)
	if err != nil {
		return err
	}
	c.ResponseWriter.Header().Add("Content-Type", "application/json; charset=utf-8")
	c.ResponseWriter.WriteHeader(200)
	_, err = c.ResponseWriter.Write(res)
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
	c.ResponseWriter.Header().Set("content-type", "text/html; charset=utf-8")
	return c.Raw([]byte(html))
}

func (c *Context) Raw(data []byte) error {
	_, err := c.ResponseWriter.Write(data)
	return err
}

func (c *Context) Redirect(code int, url string) error {
	c.ResponseWriter.Header().Set("Location", url)
	c.ResponseWriter.WriteHeader(code)
	return nil
}
