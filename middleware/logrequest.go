package middleware

import (
	"log/slog"
	"time"

	"github.com/levmv/mig"
)

// RequestLogger returns a middleware that logs one line per request with timing info.
// It ensures a log entry is written even if the handler panics.
//
// NOTE: For request IDs to appear in logs, you must register the RequestID()
// middleware before RequestLogger, e.g.: m.Use(RequestID(), RequestLogger()).
func RequestLogger() mig.MiddlewareFunc {
	return func(next mig.Handler) mig.Handler {
		return func(c *mig.Context) error {
			start := time.Now()

			defer func() {
				c.Logger.Info("request",
					slog.String("method", c.Request.Method),
					slog.String("path", c.Request.URL.Path),
					slog.Int("status", c.Response.Status()),
					slog.Int("bytes", c.Response.Written()),
					slog.Duration("t", time.Since(start)),
				)
			}()

			return next(c)
		}
	}
}
