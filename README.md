# Mig

A small web framework for Go, built on the modern `net/http.ServeMux` (Go 1.22+).

`mig` provides a cleaner API for common web development tasks (routing, middleware, request handling) without unnecessary complexity or third-party dependencies. 

[![Go Reference](https://pkg.go.dev/badge/github.com/levmv/mig.svg)](https://pkg.go.dev/github.com/levmv/mig)
[![Go Report Card](https://goreportcard.com/badge/github.com/levmv/mig)](https://goreportcard.com/report/github.com/levmv/mig)
[![Go Version](https://img.shields.io/badge/Go-1.22%2B-blue.svg)](https://go.dev/doc/go1.22)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Features

- **Router**: Uses stdlib `http.ServeMux` (Go 1.22+) 
- **API**: Verb-based helpers (`GET`, `POST`, etc.) and route grouping with prefixes.
- **Middleware**: Standard `func(Handler) Handler` pattern.
- **Context**: Request-scoped `Context` object (pooled via `sync.Pool`) with helpers for JSON binding, responses, and path/query access.
- **Error Handling**: Unified error type, panic recovery, JSON or plain text responses.
- **Shutdown**: Helpers for graceful shutdown on `SIGINT` / `SIGTERM`.
- **Dependencies**: None (only standard library).


## Quick Start

Create a `main.go` file with the following code:

```go
package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/levmv/mig"
	"github.com/levmv/mig/middleware"
)

func main() {
	// 1. Initialize Mig with a background context.
	m := mig.New(context.Background())

	// 2. Register global middleware.
	// These will run for every request.
	m.Use(middleware.RequestID())
	m.Use(middleware.RequestLogger())

	// 3. Define your routes and handlers.
	m.GET("/", func(c *mig.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})

	// Route with a path parameter.
	m.GET("/users/{id}", func(c *mig.Context) error {
		id := c.PathValue("id")
		return c.JSON(map[string]string{"user_id": id})
	})

	// 4. Create a route group for better organization.
	api := m.Group("/api")
	api.GET("/status", func(c *mig.Context) error {
		return c.JSON(map[string]string{"status": "ok"})
	})

	// 5. Start the server. Run() blocks until a shutdown signal is received.
	fmt.Println("Server starting on http://localhost:8080")
	if err := m.Run(":8080"); err != nil {
		fmt.Printf("Server failed to start: %v\n", err)
	}
}
```

## Installation

```bash
go get github.com/levmv/mig
```


---

## License

This project is licensed under the MIT License.