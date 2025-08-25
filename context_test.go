package mig_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/levmv/mig"
)

func TestContext_BindJSON(t *testing.T) {
	type TestUser struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	testCases := []struct {
		name           string
		reqBody        string
		expectedStatus int
		expectedBody   string
		expectBindErr  bool
	}{
		{
			name:           "Successful bind",
			reqBody:        `{"name": "John Doe", "age": 30}`,
			expectedStatus: http.StatusOK,
			expectedBody:   `OK: John Doe is 30`,
			expectBindErr:  false,
		},
		{
			name:           "Malformed JSON",
			reqBody:        `{"name": "John Doe", "age": 30,}`, // Extra comma makes it invalid
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Bad Request\n",
			expectBindErr:  true,
		},
		{
			name:           "Unknown field in JSON",
			reqBody:        `{"name": "John Doe", "age": 30, "email": "j@d.com"}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Bad Request\n",
			expectBindErr:  true,
		},
		{
			name:           "Wrong data type",
			reqBody:        `{"name": "John Doe", "age": "thirty"}`, // age should be an int
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Bad Request\n",
			expectBindErr:  true,
		},
		{
			name:           "Empty request body",
			reqBody:        ``,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Bad Request\n",
			expectBindErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := mig.New(context.Background())

			restoreLogger := setupSilentLogger(m)
			defer restoreLogger()

			testHandler := func(c *mig.Context) error {
				var user TestUser
				if err := c.BindJSON(&user); err != nil {
					return err
				}
				res := "OK: " + user.Name + " is " + strconv.Itoa(user.Age)
				return c.Raw([]byte(res))
			}

			m.POST("/", testHandler)

			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.reqBody))
			req.Header.Set("Content-Type", "application/json")
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

func TestContext_GetPut(t *testing.T) {
	m := mig.New(context.Background())

	var retrievedValue any
	var nonExistentValue any

	handler := func(c *mig.Context) error {
		type testKey struct{}

		// Test Case 1: Put a value and then get it back.
		c.Put(testKey{}, "hello from test")
		retrievedValue = c.Get(testKey{})

		// Test Case 2: Get a value that was never put.
		type anotherKey struct{}
		nonExistentValue = c.Get(anotherKey{})

		return c.String(http.StatusOK, "OK")
	}

	m.GET("/", handler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	m.Mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("handler did not run successfully, status code: %d", rec.Code)
	}

	assertEqual(t, "hello from test", retrievedValue, "Value retrieved should match value put")
	assertEqual(t, nil, nonExistentValue, "Getting a non-existent key should return nil")
}

func TestContext_QueryParam(t *testing.T) {
	m := mig.New(context.Background())

	t.Run("standart params", func(t *testing.T) {
		handler := func(c *mig.Context) error {
			name := c.QueryParam("name", "default")
			assertEqual(t, "mig", name, "Should get existing query param")

			author := c.QueryParam("author", "unknown")
			assertEqual(t, "unknown", author, "Should fall back to default for missing param")

			assertEqual(t, true, c.HasQueryParam("name"), "'name' should exist in query")
			assertEqual(t, false, c.HasQueryParam("missing"), "'missing' param should return false")
			return c.String(http.StatusOK, "OK")
		}
		m.GET("/test", handler)
		req := httptest.NewRequest(http.MethodGet, "/test?name=mig&version=1", nil)
		rec := httptest.NewRecorder()
		m.Mux.ServeHTTP(rec, req)
		assertEqual(t, http.StatusOK, rec.Code, "Request with standard params failed")
	})

	t.Run("explicitly empty param", func(t *testing.T) {
		handlerEmpty := func(c *mig.Context) error {
			emptyName := c.QueryParam("name", "default")
			assertEqual(t, "", emptyName, "Explicit empty param should return empty string")
			return c.String(http.StatusOK, "OK")
		}
		m.GET("/empty", handlerEmpty)
		reqEmpty := httptest.NewRequest(http.MethodGet, "/empty?name=", nil)
		recEmpty := httptest.NewRecorder()
		m.Mux.ServeHTTP(recEmpty, reqEmpty)
		assertEqual(t, http.StatusOK, recEmpty.Code, "Request with empty param failed")
	})
}

func TestContext_PathValue(t *testing.T) {
	m := mig.New(context.Background())
	var receivedID string

	m.GET("/users/{id}", func(c *mig.Context) error {
		receivedID = c.PathValue("id")
		return c.Raw([]byte("OK"))
	})

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	rec := httptest.NewRecorder()

	m.Mux.ServeHTTP(rec, req)

	assertEqual(t, http.StatusOK, rec.Code, "Request should succeed")
	assertEqual(t, "123", receivedID, "PathValue should correctly extract the parameter")
}

func TestContext_ResponseWriters(t *testing.T) {
	m := mig.New(context.Background())

	t.Run("JSON response", func(t *testing.T) {
		type User struct {
			Name string `json:"name"`
		}
		m.GET("/json", func(c *mig.Context) error {
			return c.JSON(User{Name: "mig"})
		})
		req := httptest.NewRequest(http.MethodGet, "/json", nil)
		rec := httptest.NewRecorder()
		m.Mux.ServeHTTP(rec, req)

		assertEqual(t, http.StatusOK, rec.Code, "JSON status code should be 200")
		assertEqual(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"), "JSON Content-Type header is incorrect")
		assertEqual(t, `{"name":"mig"}`, strings.TrimSpace(rec.Body.String()), "JSON body is incorrect")
	})

	t.Run("Redirect response", func(t *testing.T) {
		m.GET("/redirect", func(c *mig.Context) error {
			return c.Redirect(http.StatusFound, "/new-path")
		})
		req := httptest.NewRequest(http.MethodGet, "/redirect", nil)
		rec := httptest.NewRecorder()
		m.Mux.ServeHTTP(rec, req)

		assertEqual(t, http.StatusFound, rec.Code, "Redirect status code is incorrect")
		assertEqual(t, "/new-path", rec.Header().Get("Location"), "Redirect Location header is incorrect")
	})

	t.Run("HTML response", func(t *testing.T) {
		htmlContent := "<h1>Hello</h1>"
		m.GET("/html", func(c *mig.Context) error {
			return c.HTML(htmlContent)
		})
		req := httptest.NewRequest(http.MethodGet, "/html", nil)
		rec := httptest.NewRecorder()
		m.Mux.ServeHTTP(rec, req)

		assertEqual(t, http.StatusOK, rec.Code, "HTML status code should be 200")
		assertEqual(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"), "HTML Content-Type header is incorrect")
		assertEqual(t, htmlContent, rec.Body.String(), "HTML body is incorrect")
	})
}
