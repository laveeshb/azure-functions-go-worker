# Hello World - gRPC Worker

A simple Azure Functions app demonstrating the **native Go gRPC worker**.

This sample uses the `pkg/azfunc` package which handles all gRPC communication with the Azure Functions host automatically.

## Table of Contents

- [Functions](#functions)
- [Quick Start](#quick-start)
- [Usage](#usage)
- [How It Works](#how-it-works)
- [Project Structure](#project-structure)
- [Comparison with Custom Handler](#comparison-with-custom-handler)

## Functions

| Function | Method | Route | Description |
|----------|--------|-------|-------------|
| Hello | GET/POST | `/api/hello` | Greets the user by name |
| Health | GET | `/api/health` | Health check endpoint |
| Echo | POST | `/api/echo` | Echoes back request details |

## Quick Start

```bash
# Build the worker
go build -o worker.exe .

# Start the function host
func start
```

## Usage

```bash
# Hello with default greeting
curl http://localhost:7071/api/hello

# Hello with name in query
curl "http://localhost:7071/api/hello?name=Azure"

# Hello with name in body
curl -X POST http://localhost:7071/api/hello -d '{"name":"Go"}'

# Health check
curl http://localhost:7071/api/health

# Echo
curl -X POST http://localhost:7071/api/echo \
  -H "Content-Type: application/json" \
  -d '{"message":"test"}'
```

## How It Works

The function app author writes simple handler functions:

```go
func handleHello(ctx *azfunc.Context, req *azfunc.HttpRequest) (*azfunc.HttpResponse, error) {
    return &azfunc.HttpResponse{
        StatusCode: 200,
        Body:       []byte("Hello, World!"),
    }, nil
}
```

The `azfunc` package handles:
- gRPC connection to Azure Functions host
- Worker initialization handshake
- Function loading
- Request/response translation
- Logging integration

## Project Structure

```
hello-world-grpc/
├── main.go              # Function implementations
├── go.mod               # Go module definition
├── host.json            # Azure Functions host config
├── local.settings.json  # Local development settings
├── worker.config.json   # Worker discovery config
├── Hello/
│   └── function.json    # Hello function binding
├── Health/
│   └── function.json    # Health function binding
└── Echo/
    └── function.json    # Echo function binding
```

## Comparison with Custom Handler

See the [hello-world-custom-handler](../hello-world-custom-handler/) sample for comparison. Key differences:

| Aspect | gRPC Worker | Custom Handler |
|--------|-------------|----------------|
| Protocol | gRPC (binary) | HTTP (JSON) |
| Developer API | `azfunc.HttpRequest/Response` | `http.Request/ResponseWriter` |
| Overhead | Lower | HTTP parsing overhead |
| Setup | Zero config | `customHandler` in host.json |
| Binding Support | All Azure bindings | HTTP only |

**Recommendation:** Use the gRPC worker for new projects. It provides better performance and will support all Azure Functions bindings.
