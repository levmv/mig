package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/levmv/mig"
)

func TestBasicAuth(t *testing.T) {
	m := mig.New(context.Background())

	restoreLogger := setupSilentLogger(m)
	defer restoreLogger()

	protectedHandler := func(c *mig.Context) error {
		return c.Raw([]byte("Welcome, authorized user"))
	}

	authConfig := BasicAuthConfig{
		IsAllowed: func(user, password string) bool {
			return user == "mig" && password == "secret"
		},
		Realm: "Test Realm",
	}
	authMiddleware := BasicAuthWithConfig(authConfig)

	finalHandler := authMiddleware(protectedHandler)

	m.GET("/protected", finalHandler)

	testCases := []struct {
		name                string
		username            string
		password            string
		expectStatus        int
		expectBody          string
		expectWwwAuthHeader string
	}{
		{
			name:                "Valid credentials",
			username:            "mig",
			password:            "secret",
			expectStatus:        http.StatusOK,
			expectBody:          "Welcome, authorized user",
			expectWwwAuthHeader: "", // No header on success
		},
		{
			name:                "Invalid password",
			username:            "mig",
			password:            "wrong",
			expectStatus:        http.StatusUnauthorized,
			expectBody:          "Unauthorized\n",
			expectWwwAuthHeader: `Basic realm="Test Realm"`,
		},
		{
			name:                "Invalid user",
			username:            "guest",
			password:            "secret",
			expectStatus:        http.StatusUnauthorized,
			expectBody:          "Unauthorized\n",
			expectWwwAuthHeader: `Basic realm="Test Realm"`,
		},
		{
			name:                "No credentials",
			username:            "",
			password:            "",
			expectStatus:        http.StatusUnauthorized,
			expectBody:          "Unauthorized\n",
			expectWwwAuthHeader: `Basic realm="Test Realm"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)

			if tc.username != "" || tc.password != "" {
				req.SetBasicAuth(tc.username, tc.password)
			}
			rec := httptest.NewRecorder()

			m.Mux.ServeHTTP(rec, req)

			assertEqual(t, tc.expectStatus, rec.Code, "Status code mismatch")
			assertEqual(t, tc.expectBody, rec.Body.String(), "Response body mismatch")

			if tc.expectWwwAuthHeader != "" {
				assertEqual(t, tc.expectWwwAuthHeader, rec.Header().Get("WWW-Authenticate"), "WWW-Authenticate header mismatch")
			}
		})
	}
}

func TestBasicAuth_PanicOnNil(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("BasicAuthWithConfig should have panicked on nil IsAllowed function, but did not")
		}
	}()

	_ = BasicAuthWithConfig(BasicAuthConfig{
		IsAllowed: nil,
	})
}
