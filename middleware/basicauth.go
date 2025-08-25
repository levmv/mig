package middleware

import (
	"net/http"
	"strconv"

	"github.com/levmv/mig"
)

type BasicAuthConfig struct {
	IsAllowed func(user string, password string) bool
	Realm     string
}

func BasicAuthWithConfig(cfg BasicAuthConfig) mig.MiddlewareFunc {
	if cfg.IsAllowed == nil {
		panic("BasicAuth middleware requires IsAllowed function")
	}
	var realm string
	if cfg.Realm == "" {
		realm = `"restricted"`
	} else {
		realm = strconv.Quote(cfg.Realm)
	}

	return func(next mig.Handler) mig.Handler {
		return func(m *mig.Context) error {
			u, p, ok := m.Request.BasicAuth()
			if ok && cfg.IsAllowed(u, p) {
				return next(m)
			}

			m.Response.Header().Set("WWW-Authenticate", "Basic realm="+realm)
			return mig.NewHTTPError(http.StatusUnauthorized)
		}
	}
}
