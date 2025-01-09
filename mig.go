// Package mig implements simple golang web framework.
package mig

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"
)

// Mig is the core framework's instance.
// Create an instance of Mig by using New().
type Mig struct {
	RouteGroup
	Mux    *http.ServeMux
	BindTo string
	// ErrorHandler is used to process any errors during requests.
	// By default, DefaultErrorHandler() is used
	ErrorHandler HTTPErrorHandler
	Logger       *slog.Logger
	Renderer     Renderer
	pool         sync.Pool
	http         *http.Server
	ctx          context.Context
}

type (
	MiddlewareFunc   func(Handler) Handler
	Handler          func(*Context) error
	HTTPErrorHandler func(error, *Context)
)

var ErrNotFound = NewHTTPError(http.StatusNotFound)

// HTTPError represents an error happend while handling request.
type HTTPError struct {
	Code     int    `json:"-"`
	Message  string `json:"code"`
	Internal error  `json:"-"`
}

func (e *HTTPError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("code=%d, message=%v, err=%v", e.Code, e.Message, e.Internal)
	}
	return fmt.Sprintf("code=%d, message=%v", e.Code, e.Message)
}

func (e *HTTPError) Unwrap() error {
	return e.Internal
}

func NewHTTPError(code int) *HTTPError {
	e := &HTTPError{Code: code, Message: http.StatusText(code)}
	return e
}

func New(ctx context.Context) *Mig {
	m := Mig{}
	m.RouteGroup = RouteGroup{
		Mig: &m,
	}
	m.Mux = http.NewServeMux()
	m.pool = sync.Pool{
		New: func() any {
			return &Context{
				Logger: m.Logger,
				Mig:    &m,
			}
		},
	}
	m.Logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	m.ErrorHandler = m.DefaultErrorHandler
	// m.Renderer = &DefaultRenderer{}
	m.http = &http.Server{
		Addr:              ":8080",
		Handler:           m.Mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	m.ctx = ctx
	return &m
}

func (m *Mig) ListenAndServe(addr string) {
	m.http.Addr = addr
	go func() {
		if err := m.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			m.Logger.Error("ListenAndServe", slog.Any("err", err))
		}
	}()
}

func (m *Mig) WaitShutdown(stop context.CancelFunc) {
	<-m.ctx.Done()
	stop()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := m.http.Shutdown(ctx); err != nil {
		m.Logger.Error("Server forced to shutdown", slog.Any("err", err))
	}
}

func (m *Mig) Execute(handler Handler, rw http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			err, ok := err.(error)
			if !ok {
				err = fmt.Errorf("%v", r)
			}
			if errors.Is(err, http.ErrAbortHandler) {
				panic(r)
			}
			http.Error(rw, "Internal Server Error!", http.StatusInternalServerError)
			m.Logger.Error("Panic recover", slog.Any("err", err))
		}
	}()

	cont, _ := m.pool.Get().(*Context)
	cont.Reset(r, rw)

	err := handler(cont)
	if err != nil {
		m.ErrorHandler(err, cont)
	}
	m.pool.Put(cont)
}

func (m *Mig) DefaultErrorHandler(err error, ctx *Context) {
	e, ok := err.(*HTTPError)
	if !ok {
		e = &HTTPError{
			Code:     http.StatusInternalServerError,
			Message:  http.StatusText(http.StatusInternalServerError),
			Internal: err,
		}
	}

	http.Error(ctx.ResponseWriter, e.Message, e.Code)
	m.Logger.Error(e.Error())
}

func (m *Mig) Shutdown(ctx context.Context) error {
	return m.http.Shutdown(ctx)
}

func (m *Mig) HandleSignals() {
}
