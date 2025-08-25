package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"io"

	"github.com/levmv/mig"
)

// RequestID middleware ensures every request has an ID.
// It prefers the incoming X-Request-ID header, otherwise generates a random ID.
func RequestID() mig.MiddlewareFunc {
	return func(next mig.Handler) mig.Handler {
		return func(c *mig.Context) error {
			id := c.Request.Header.Get(mig.RequestIDHeader)
			if id == "" {
				var b [8]byte
				if _, err := io.ReadFull(rand.Reader, b[:]); err == nil {
					id = hex.EncodeToString(b[:])
				}
			}
			c.SetRequestID(id)
			return next(c)
		}
	}
}
