# Azure Functions Go Worker

A native Go language worker for Azure Functions, enabling first-class Go support (not Custom Handlers).

> ⚠️ **Early Development** - This project is in active development and not yet ready for production use.

## Overview

This worker allows you to write Azure Functions in Go with an idiomatic API:

```go
package main

import (
    "fmt"
    "github.com/Azure/azure-functions-go-worker/pkg/azfunc"
)

func init() {
    azfunc.RegisterHttpFunction("HelloWorld", handleHello)
}

func main() {
    azfunc.Start()
}

func handleHello(ctx *azfunc.Context, req *azfunc.HttpRequest) (*azfunc.HttpResponse, error) {
    name := req.GetQuery("name")
    if name == "" {
        name = "World"
    }
    return azfunc.OK(fmt.Sprintf("Hello, %s!", name)), nil
}
```

## Features

- **Native Go support** - Write functions in pure Go, compiled to a single binary
- **Idiomatic API** - Familiar patterns for Go developers
- **HTTP triggers** - Full support for HTTP request/response handling
- **Fast cold starts** - Go's quick startup time benefits serverless workloads
- **Type safety** - Compile-time checks, no runtime reflection magic

## Architecture

```
Azure Functions Host <──gRPC──> Go Worker <──> Your Go Functions
```

The worker communicates with the Azure Functions Host via gRPC using the standard [language worker protocol](https://github.com/Azure/azure-functions-language-worker-protobuf).

## Getting Started

### Prerequisites

- Go 1.21 or later
- Azure Functions Core Tools v4
- Protocol Buffers compiler (for development only)

### Building

```bash
# Clone the repository
git clone https://github.com/Azure/azure-functions-go-worker.git
cd azure-functions-go-worker

# Generate protobuf code (if needed)
make generate

# Build the worker
make build

# Build the example
make example
```

### Running the Example

```bash
cd examples/httpTrigger
func start
```

Then visit: http://localhost:7071/api/HttpTrigger?name=Gopher

## Project Structure

```
azure-functions-go-worker/
├── cmd/worker/          # Worker entry point
├── pkg/azfunc/          # Public SDK (stable API)
├── internal/
│   ├── rpc/             # gRPC client and handlers
│   ├── registry/        # Function registration
│   └── bindings/        # Type converters
├── proto/               # Protobuf definitions
├── examples/            # Example functions
└── docs/design/         # Architecture documentation
```

## API Reference

### Registering Functions

```go
// HTTP trigger
azfunc.RegisterHttpFunction("FunctionName", handler)
```

### HTTP Handler Signature

```go
func handler(ctx *azfunc.Context, req *azfunc.HttpRequest) (*azfunc.HttpResponse, error)
```

### HttpRequest

```go
req.Method              // HTTP method (GET, POST, etc.)
req.URL                 // Request URL
req.Headers             // map[string]string
req.GetHeader("name")   // Get header (case-insensitive)
req.GetQuery("param")   // Get query parameter
req.GetParam("route")   // Get route parameter
req.Body                // []byte
req.BodyAsString()      // string
```

### HttpResponse

```go
// Quick responses
azfunc.OK(body)                    // 200 OK
azfunc.Created(body)               // 201 Created
azfunc.BadRequest("message")       // 400 Bad Request
azfunc.NotFound("message")         // 404 Not Found
azfunc.InternalServerError("msg")  // 500 Internal Server Error

// Custom response
resp := &azfunc.HttpResponse{
    StatusCode: 201,
    Headers:    map[string]string{"X-Custom": "value"},
    Body:       myData,  // string, []byte, or any JSON-serializable value
}

// Fluent API
azfunc.OK(data).WithHeader("X-Custom", "value").WithContentType("application/json")
```

### Context

```go
ctx.InvocationID    // Unique ID for this invocation
ctx.FunctionID      // Function identifier
ctx.Log("message")  // Log at Information level
ctx.LogDebug("msg") // Log at Debug level
ctx.LogWarning("m") // Log at Warning level
ctx.LogError("msg") // Log at Error level
```

## Roadmap

### Phase 1: MVP ✅
- [x] Project structure
- [x] gRPC client implementation
- [x] Worker lifecycle (init, load, invoke)
- [x] HTTP trigger support
- [x] Basic SDK

### Phase 2: Polish
- [ ] End-to-end testing
- [ ] Panic recovery
- [ ] Improved error messages
- [ ] CI/CD pipeline

### Phase 3: More Bindings
- [ ] Timer trigger
- [ ] Queue trigger (Storage Queue, Service Bus)
- [ ] Blob input/output
- [ ] Cosmos DB bindings

### Phase 4: Production Ready
- [ ] Performance optimization
- [ ] Documentation
- [ ] Azure deployment support
- [ ] VS Code integration

## Contributing

Contributions are welcome! Please read our contributing guidelines (coming soon).

### Development Setup

```bash
# Install development tools
make tools

# Run tests
make test

# Format code
make fmt

# Run linter
make lint
```

## Design Documentation

See [docs/design/ARCHITECTURE.md](docs/design/ARCHITECTURE.md) for detailed architecture and design decisions.

## Related Projects

- [Azure Functions Host](https://github.com/Azure/azure-functions-host)
- [Azure Functions Python Worker](https://github.com/Azure/azure-functions-python-worker)
- [Azure Functions Node.js Worker](https://github.com/Azure/azure-functions-nodejs-worker)
- [Language Worker Protocol](https://github.com/Azure/azure-functions-language-worker-protobuf)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.