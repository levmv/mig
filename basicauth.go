package mig

import (
	"net/http"
	"strconv"
)

type BasicAuthConfig struct {
	IsAllowed func(user string, password string) bool
	Realm     string
}

func BasicAuthWithConfig(cfg BasicAuthConfig) MiddlewareFunc {
	if cfg.IsAllowed == nil {
		panic("BasicAuth middleware requires IsAllowed function")
	}
	if cfg.Realm == "" {
		cfg.Realm = "restricted"
	} else {
		cfg.Realm = strconv.Quote(cfg.Realm)
	}

	return func(next Handler) Handler {
		return func(m *Context) error {
			u, p, ok := m.Request.BasicAuth()
			if ok && cfg.IsAllowed(u, p) {
				return next(m)
			}

			m.ResponseWriter.Header().Set("WWW-Authenticate", "Basic realm="+cfg.Realm)
			return NewHTTPError(http.StatusUnauthorized)
		}
	}
}

func BasicAuth(next Handler) Handler {
	m := BasicAuthWithConfig(BasicAuthConfig{})
	return m(next)
}
