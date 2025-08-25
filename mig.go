// Package mig implements simple golang web framework.
package mig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Mig is the core framework instance that handles HTTP requests and routing.
type Mig struct {
	RouteGroup
	Mux    *http.ServeMux
	BindTo string
	// ErrorHandler is used to process any errors during requests.
	// By default, DefaultErrorHandler() is used
	ErrorHandler    HTTPErrorHandler
	Logger          *slog.Logger
	Renderer        Renderer
	pool            sync.Pool
	http            *http.Server
	ctx             context.Context
	ShutdownTimeout time.Duration
	// Request
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

type (
	MiddlewareFunc   func(Handler) Handler
	Handler          func(*Context) error
	HTTPErrorHandler func(error, *Context)
)

// Renderer is an interface for rendering templates.
type Renderer interface {
	Render(io.Writer, string, any) error
}

// ErrNotFound is a standard HTTP 404 error. To create a custom 404 handler, register `m.Any("/", ...)` last.
var ErrNotFound = NewHTTPError(http.StatusNotFound)

// HTTPError represents an error happened while handling request.
type HTTPError struct {
	Code     int    `json:"code"`
	Message  string `json:"message"`
	Internal error  `json:"-"`
	Stack    string `json:"-"`
}

func (e *HTTPError) Error() string {
	if e.Internal != nil {
		return e.Internal.Error()
	}
	return e.Message
}

func (e *HTTPError) Unwrap() error {
	return e.Internal
}

func NewHTTPError(code int) *HTTPError {
	e := &HTTPError{Code: code, Message: http.StatusText(code)}
	return e
}

func New(ctx context.Context) *Mig {
	m := Mig{
		ShutdownTimeout:   10 * time.Second,
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       75 * time.Second,
	}
	m.RouteGroup = RouteGroup{
		Mig: &m,
	}
	m.Mux = http.NewServeMux()

	m.Logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))

	m.pool = sync.Pool{
		New: func() any {
			return &Context{
				Logger: m.Logger,
				Mig:    &m,
			}
		},
	}
	m.ErrorHandler = m.DefaultErrorHandler
	// m.Renderer = &DefaultRenderer{}
	m.http = &http.Server{
		Addr:    ":8080",
		Handler: m.Mux,
	}
	m.ctx = ctx
	return &m
}

// Start begins listening for HTTP requests in a separate goroutine.
// It is a non-blocking method. Use the Shutdown method to stop the server.
func (m *Mig) Start(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err // bind failed
	}

	m.http.Addr = addr
	m.http.ReadTimeout = m.ReadTimeout
	m.http.ReadHeaderTimeout = m.ReadHeaderTimeout
	m.http.WriteTimeout = m.WriteTimeout
	m.http.IdleTimeout = m.IdleTimeout

	m.Logger.Info("Server starting", "addr", m.http.Addr)

	go func() {
		if err := m.http.Serve(ln); err != http.ErrServerClosed {
			m.Logger.Error("Server unexpectedly closed", "err", err)
		}
	}()
	return nil
}

// Shutdown gracefully stops the HTTP server.
// It waits for the duration of ShutdownTimeout for existing connections to finish.
func (m *Mig) Shutdown() error {
	m.Logger.Info("Server shutting down...")

	shutdownCtx, cancel := context.WithTimeout(m.ctx, m.ShutdownTimeout)
	defer cancel()

	if err := m.http.Shutdown(shutdownCtx); err != nil {
		m.Logger.Error("Server forced to shutdown", "err", err)
		return err
	}
	m.Logger.Info("Server gracefully stopped.")
	return nil
}

// Run starts the server, blocks until an OS signal is received, and then
// performs a graceful shutdown. This is the simplest way to run the server.
func (m *Mig) Run(addr string) error {
	if err := m.Start(addr); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(m.ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()

	return m.Shutdown()
}

func (m *Mig) Execute(handler Handler, rw http.ResponseWriter, r *http.Request) {
	cont, _ := m.pool.Get().(*Context)
	cont.Reset(r, rw)
	defer m.pool.Put(cont)

	defer func() {
		if rec := recover(); rec != nil {
			var err error

			// Try to assert if the recovered value is already an error.
			recErr, ok := rec.(error)
			if ok {
				err = recErr
			} else {
				// If it's not (e.g., a string from panic("foo")), wrap it in an error.
				err = fmt.Errorf("%v", rec)
			}

			if errors.Is(err, http.ErrAbortHandler) {
				panic(err)
			}

			httpErr := NewHTTPError(http.StatusInternalServerError)
			httpErr.Internal = err
			httpErr.Stack = string(debug.Stack())

			m.ErrorHandler(httpErr, cont)
		}
	}()

	err := handler(cont)
	if err != nil {
		m.ErrorHandler(err, cont)
	}
}

func (m *Mig) DefaultErrorHandler(err error, ctx *Context) {
	var e *HTTPError
	if !errors.As(err, &e) {
		e = &HTTPError{
			Code:     http.StatusInternalServerError,
			Message:  http.StatusText(http.StatusInternalServerError),
			Internal: err,
		}
	}

	// Logging: include internal error and stack if present, but never expose stack to clients.
	if e.Stack != "" {
		m.Logger.Error(
			"panic recovered",
			"id", ctx.RequestID(),
			"error", e.Internal,
			"stack", e.Stack,
		)
	} else {
		m.Logger.Error(
			"request error",
			"id", ctx.RequestID(),
			"code", e.Code,
			"error", e.Internal,
		)
	}

	// If response has already been partially written, we must not attempt to write again.
	if ctx.Response.Written() > 0 {
		return
	}

	// For special status codes that must not include a body, just set status.
	if e.Code == http.StatusNoContent || e.Code == http.StatusNotModified {
		ctx.Response.WriteHeader(e.Code)
		return
	}

	if strings.Contains(ctx.Request.Header.Get("Accept"), "application/json") {
		ctx.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
		ctx.Response.WriteHeader(e.Code)
		// Public-facing error payload
		payload := map[string]any{
			"code":    e.Code,
			"message": e.Message,
		}
		_ = json.NewEncoder(ctx.Response).Encode(payload)
		return
	}

	// Fallback: plain text
	http.Error(ctx.Response, e.Message, e.Code)
}
