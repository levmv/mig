package mig

import (
	"log/slog"
	"time"
)

func RequestLogger() MiddlewareFunc {
	return func(next Handler) Handler {
		return func(c *Context) error {
			start := time.Now()
			err := next(c)
			c.Logger.Info("request", slog.String("request", c.Request.URL.Path), slog.Duration("t", time.Since(start)))
			return err
		}
	}
}
