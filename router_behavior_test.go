package mig_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/levmv/mig"
)

func TestRouterStdlibBehavior(t *testing.T) {
	m := mig.New(context.Background())

	// A handler for GET and PUT on the same path. This is important for testing the Allow header.
	m.GET("/resource", func(c *mig.Context) error { return c.Raw([]byte("GET OK")) })
	m.PUT("/resource", func(c *mig.Context) error { return c.Raw([]byte("PUT OK")) })

	// A handler with a trailing slash to test redirects.
	m.GET("/admin/", func(c *mig.Context) error { return c.Raw([]byte("ADMIN OK")) })

	testCases := []struct {
		name            string
		method          string
		path            string
		expectedStatus  int
		expectedBody    string
		expectedHeaders map[string]string
	}{
		{
			name:           "HEAD request to GET route",
			method:         http.MethodHead,
			path:           "/resource",
			expectedStatus: http.StatusOK,
			expectedBody:   "", // Body should be empty for HEAD
		},
		{
			name:           "Method Not Allowed (405)",
			method:         http.MethodPost, // POST is not registered for /resource
			path:           "/resource",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "",
			// The Allow header should list all valid methods for this path.
			// Note: We will sort these for stable comparison.
			expectedHeaders: map[string]string{"Allow": "GET, HEAD, PUT"},
		},
		{
			name:            "OPTIONS request",
			method:          http.MethodOptions,
			path:            "/resource",
			expectedStatus:  http.StatusMethodNotAllowed,
			expectedBody:    "",
			expectedHeaders: map[string]string{"Allow": "GET, HEAD, PUT"},
		},
		{
			name:            "Trailing Slash Redirect (301)",
			method:          http.MethodGet,
			path:            "/admin", // Request without slash
			expectedStatus:  http.StatusMovedPermanently,
			expectedBody:    "<a href=\"/admin/\">Moved Permanently</a>.\n\n",
			expectedHeaders: map[string]string{"Location": "/admin/"},
		},
		{
			name:           "Successful Trailing Slash Request",
			method:         http.MethodGet,
			path:           "/admin/", // Request with slash
			expectedStatus: http.StatusOK,
			expectedBody:   "ADMIN OK",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()

			m.Mux.ServeHTTP(rec, req)

			assertEqual(t, tc.expectedStatus, rec.Code, "Status code mismatch")

			// Only check body if one is expected, as some are empty.
			if tc.expectedBody != "" {
				assertEqual(t, tc.expectedBody, rec.Body.String(), "Response body mismatch")
			}

			// Check for expected headers.
			for key, val := range tc.expectedHeaders {
				if key == "Allow" {
					// Special handling for Allow header, as order is not guaranteed.
					assertAllowHeader(t, val, rec.Header().Get(key))
				} else {
					assertEqual(t, val, rec.Header().Get(key), "Header mismatch for "+key)
				}
			}
		})
	}
}

// assertAllowHeader is a special helper to compare Allow headers in a deterministic way.
func assertAllowHeader(t *testing.T, expected, actual string) {
	t.Helper()

	if actual == "" {
		t.Fatalf("Allow header was unexpectedly empty")
	}

	expectedMethods := strings.Split(expected, ", ")
	sort.Strings(expectedMethods)

	actualMethods := strings.Split(actual, ", ")
	sort.Strings(actualMethods)

	assertEqual(t, strings.Join(expectedMethods, ", "), strings.Join(actualMethods, ", "), "Allow header mismatch")
}
