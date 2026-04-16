# gofernq

[![Go Reference](https://pkg.go.dev/badge/github.com/fernq-org/gofernq.svg)](https://pkg.go.dev/github.com/fernq-org/gofernq)
[![Go Report Card](https://goreportcard.com/badge/github.com/fernq-org/gofernq)](https://goreportcard.com/report/github.com/fernq-org/gofernq)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

<div align="right">

**[English](README.md)** | **[中文](README_CN.md)**

</div>

---

gofernq is a Go implementation of the Fernq protocol client. It provides a simple and efficient way to communicate with Fernq servers, featuring a flexible routing system and robust connection management.

## Features

- **Simple API**: Easy-to-use client interface with minimal setup
- **Flexible Routing**: Built-in router with path-based request handling
- **Connection Management**: Automatic reconnection and graceful shutdown
- **Connection Monitoring**: Built-in Done() channel for tracking connection status
- **Protocol Support**: Full Fernq protocol implementation with JSON and form content types
- **Concurrent Safe**: Thread-safe operations with proper locking mechanisms
- **Error Handling**: Comprehensive error types and messages

## Installation

```bash
go get github.com/fernq-org/gofernq
```

## Quick Start

### Basic Usage

```go
package main

import (
    "fmt"
    "github.com/fernq-org/gofernq"
)

func main() {
    // Create a new client
    client := gofernq.NewClient("my-client")

    // Connect to the Fernq server
    // URL format: fernq://<host>[:<port>]/<uuid>#<room_name>?room_pass=<password>
    // Example: fernq://localhost:9147/550e8400-e29b-41d4-a716-446655440000#my-room?room_pass=123456
    if err := client.Connect("fernq://localhost:9147/550e8400-e29b-41d4-a716-446655440000#my-room?room_pass=123456"); err != nil {
        panic(err)
    }
    defer client.Close()

    // Make a request
    resp, err := client.Request("fernq://api/hello")
    if err != nil {
        panic(err)
    }

    fmt.Printf("Response: %+v\n", resp)
}
```

### Using Router

```go
package main

import (
    "github.com/fernq-org/gofernq"
)

func main() {
    // Create a router
    router := gofernq.NewRouter()

    // Add a route handler
    router.AddRoute("/api/hello", func(ctx *gofernq.Context) {
        ctx.JSON(gofernq.StatusOK, map[string]string{
            "message": "Hello, World!",
        })
    })

    // Create a client with the router
    client := gofernq.NewClient("my-client", router)

    // Connect and use
    // URL format: fernq://<host>[:<port>]/<uuid>#<room_name>?room_pass=<password>
    if err := client.Connect("fernq://localhost:9147/550e8400-e29b-41d4-a716-446655440000#my-room?room_pass=123456"); err != nil {
        panic(err)
    }
    defer client.Close()
}
```

### Request with Body

```go
// Send a request with JSON body
data := map[string]interface{}{
    "name": "Alice",
    "age":  30,
}

resp, err := client.Request("fernq://api/user", data)
if err != nil {
    panic(err)
}
```

### Monitoring Connection Status

```go
package main

import (
    "fmt"
    "time"
    "github.com/fernq-org/gofernq"
)

func main() {
    client := gofernq.NewClient("my-client")

    // URL format: fernq://<host>[:<port>]/<uuid>#<room_name>?room_pass=<password>
    if err := client.Connect("fernq://localhost:9147/550e8400-e29b-41d4-a716-446655440000#my-room?room_pass=123456"); err != nil {
        panic(err)
    }
    defer client.Close()

    // Monitor connection status
    go func() {
        <-client.Done()
        fmt.Println("Connection closed!")
        // Handle reconnection logic here
    }()

    // Your main logic here...
    time.Sleep(30 * time.Second)
}
```

## API Reference

### Client

#### `NewClient(name string, routers ...*Router) *Client`

Creates a new client instance with the given name and optional routers.

**Parameters:**
- `name`: Client name identifier
- `routers`: Optional router instances for handling incoming requests

**Returns:** A new Client instance

#### `Connect(url string) error`

Connects to the Fernq server at the specified URL.

**Parameters:**
- `url`: Fernq protocol URL of the server (e.g., "fernq://localhost:9147/550e8400-e29b-41d4-a716-446655440000#my-room?room_pass=123456")
  - Format: fernq://<host>[:<port>]/<uuid>#<room_name>?room_pass=<password>

**Returns:** An error if connection fails

#### `Request(path string, body ...any) (*ResponseMessage, error)`

Sends a request to the specified path.

**Parameters:**
- `path`: Request path (e.g., "/api/users")
- `body`: Optional request body (will be serialized as JSON)

**Returns:** A ResponseMessage and an error

#### `Done() <-chan struct{}`

Returns a channel that will be closed when the client connection is terminated. Use this to monitor connection status and handle reconnection logic.

**Returns:** A read-only channel that closes when connection is lost

#### `Close()`

Closes the client connection gracefully.

### Router

#### `NewRouter() *Router`

Creates a new router instance.

**Returns:** A new Router instance

#### `AddRoute(path string, handler func(*Context)) error`

Adds a route handler for the specified path.

**Parameters:**
- `path`: Route path (supports dynamic segments)
- `handler`: Function to handle requests to this path

**Returns:** An error if the route is invalid

### Context

The Context object provides methods for handling requests:

#### `JSON(state int, body any)`

Sends a JSON response with the specified state code and body.

**Parameters:**
- `state`: HTTP-like status code (e.g., 200, 404, 500)
- `body`: Response body to be serialized as JSON

### Status Codes

Commonly used status codes:

- `StatusOK` (200)
- `StatusCreated` (201)
- `StatusBadRequest` (400)
- `StatusUnauthorized` (401)
- `StatusForbidden` (403)
- `StatusNotFound` (404)
- `StatusInternalServerError` (500)

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
