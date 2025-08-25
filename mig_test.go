package mig_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/levmv/mig"
	"github.com/levmv/mig/middleware"
)

func assertEqual(t *testing.T, expected, actual any, msg string) {
	t.Helper()
	if expected != actual {
		t.Fatalf("%s: mismatch\n\texp: %v\n\tgot: %v", msg, expected, actual)
	}
}

func assertNoError(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: unexpected error: %v", msg, err)
	}
}

func setupSilentLogger(m *mig.Mig) (restore func()) {
	originalLogger := m.Logger
	m.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	return func() {
		m.Logger = originalLogger
	}
}

func testMiddleware(name string) mig.MiddlewareFunc {
	return func(next mig.Handler) mig.Handler {
		return func(c *mig.Context) error {
			c.Raw([]byte(name + " > "))
			return next(c)
		}
	}
}

func TestRoutingAndMiddleware(t *testing.T) {
	m := mig.New(context.Background())

	restoreLogger := setupSilentLogger(m)
	defer restoreLogger()

	finalHandler := func(c *mig.Context) error {
		return c.Raw([]byte("OK"))
	}

	m.Use(testMiddleware("M_ROOT"))
	apiGroup := m.Group("/api", testMiddleware("M_API"))
	v1Group := apiGroup.Group("/v1")
	v1Group.Any("/users", finalHandler)
	adminGroup := m.Group("/admin", testMiddleware("M_ADMIN"))
	adminGroup.Any("/status", finalHandler)

	testCases := []struct {
		name           string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Successful nested group route",
			path:           "/api/v1/users",
			expectedStatus: http.StatusOK,
			expectedBody:   "M_ROOT > M_API > OK",
		},
		{
			name:           "Successful admin route",
			path:           "/admin/status",
			expectedStatus: http.StatusOK,
			expectedBody:   "M_ROOT > M_ADMIN > OK",
		},
		{
			name:           "Route not found",
			path:           "/api/v2/users",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "404 page not found\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()

			m.Mux.ServeHTTP(rec, req)

			res := rec.Result()
			defer res.Body.Close()

			body, err := io.ReadAll(res.Body)
			assertNoError(t, err, "Failed to read response body")

			assertEqual(t, tc.expectedStatus, res.StatusCode, "Status code mismatch")
			assertEqual(t, tc.expectedBody, string(body), "Response body mismatch")
		})
	}
}

func TestPanicRecovery(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, nil))

	m := mig.New(context.Background())
	m.Logger = logger

	m.Use(middleware.RequestID())

	m.GET("/panic", func(*mig.Context) error {
		panic("oh no, a panic!")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	m.Mux.ServeHTTP(rec, req)

	assertEqual(t, http.StatusInternalServerError, rec.Code, "Status code should be 500")

	// Check that the response body is the correct JSON error.
	var jsonBody map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &jsonBody)
	assertNoError(t, err, "Failed to unmarshal JSON response body")
	assertEqual(t, float64(500), jsonBody["code"], "JSON response code is incorrect") // JSON numbers are float64
	assertEqual(t, "Internal Server Error", jsonBody["message"], "JSON response message is incorrect")

	logOutput := logBuffer.String()

	// Get the request ID from the response header to verify it's in the log.
	requestID := rec.Header().Get(mig.RequestIDHeader)
	if requestID == "" {
		t.Fatal("X-Request-ID header was not set on response")
	}

	// Verify the log contains the key elements.
	if !strings.Contains(logOutput, "panic recovered") {
		t.Errorf("Log output should contain 'panic recovered', but got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "oh no, a panic!") {
		t.Errorf("Log output should contain the panic message, but got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "stack=") {
		t.Errorf("Log output should contain the stack trace, but got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "id="+requestID) {
		t.Errorf("Log output should contain the correct request ID, but got: %s", logOutput)
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	m := mig.New(context.Background())
	var capturedID string

	m.Use(middleware.RequestID())

	m.GET("/", func(c *mig.Context) error {
		capturedID = c.RequestID()
		return c.Raw([]byte("OK"))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	m.Mux.ServeHTTP(rec, req)

	assertEqual(t, http.StatusOK, rec.Code, "Status code should be 200")

	headerID := rec.Header().Get(mig.RequestIDHeader)
	if headerID == "" {
		t.Fatal("X-Request-ID header should be set")
	}

	assertEqual(t, headerID, capturedID, "ID in context should match header ID")
}
