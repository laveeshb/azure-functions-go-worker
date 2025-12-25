# Azure Functions Go Worker - Architecture Design

## Overview

This document describes the architecture for a native Go language worker for Azure Functions. The worker implements an out-of-process model, communicating with the Azure Functions Host via gRPC.

## Goals

1. **First-class Go support** - Native Go experience, not Custom Handlers
2. **Familiar patterns** - Idiomatic Go APIs that feel natural to Go developers
3. **Performance** - Leverage Go's efficiency and fast startup times
4. **Compatibility** - Support all major Azure Functions bindings over time

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Azure Functions Host                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │   Triggers  │  │  Bindings   │  │  Worker Management      │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
└────────────────────────────┬────────────────────────────────────┘
                             │ gRPC (Bidirectional Streaming)
                             │ FunctionRpc.proto
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Go Worker Process                           │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                    gRPC Client                               ││
│  │  - Streaming connection to host                              ││
│  │  - Message routing                                           ││
│  └─────────────────────────────────────────────────────────────┘│
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                  Message Handlers                            ││
│  │  - WorkerInitRequest    → WorkerInitResponse                 ││
│  │  - FunctionLoadRequest  → FunctionLoadResponse               ││
│  │  - InvocationRequest    → InvocationResponse                 ││
│  │  - WorkerStatusRequest  → WorkerStatusResponse               ││
│  └─────────────────────────────────────────────────────────────┘│
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                 Function Registry                            ││
│  │  - Function metadata storage                                 ││
│  │  - Handler lookup                                            ││
│  └─────────────────────────────────────────────────────────────┘│
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                 Binding Handlers                             ││
│  │  - HTTP Trigger/Response                                     ││
│  │  - Timer Trigger                                             ││
│  │  - Queue Trigger/Output                                      ││
│  │  - Blob Input/Output                                         ││
│  │  - (more bindings...)                                        ││
│  └─────────────────────────────────────────────────────────────┘│
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                   User Functions                             ││
│  │  - Compiled into worker binary                               ││
│  │  - Registered at startup                                     ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

## Project Structure

```
azure-functions-go-worker/
├── cmd/
│   └── worker/
│       └── main.go              # Worker entry point
├── internal/
│   ├── rpc/
│   │   ├── client.go            # gRPC client implementation
│   │   ├── handlers.go          # Message handlers
│   │   └── stream.go            # Streaming logic
│   ├── registry/
│   │   └── registry.go          # Function registry
│   ├── executor/
│   │   └── executor.go          # Function execution engine
│   └── bindings/
│       ├── http.go              # HTTP trigger/response
│       ├── timer.go             # Timer trigger
│       ├── queue.go             # Queue bindings
│       └── blob.go              # Blob bindings
├── pkg/
│   └── azfunc/
│       ├── azfunc.go            # Public SDK entry point
│       ├── context.go           # Function context
│       ├── http.go              # HTTP request/response types
│       ├── triggers.go          # Trigger definitions
│       └── bindings.go          # Binding definitions
├── proto/
│   ├── FunctionRpc.proto        # Copied from azure-functions-language-worker-protobuf
│   └── generate.go              # go:generate directive
├── api/
│   └── v1/
│       └── *.pb.go              # Generated protobuf code
├── docs/
│   └── design/
│       └── ARCHITECTURE.md      # This document
├── examples/
│   └── httpTrigger/
│       └── main.go              # Example HTTP function
├── scripts/
│   ├── generate-proto.ps1       # Windows proto generation
│   └── generate-proto.sh        # Unix proto generation
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Core Components

### 1. gRPC Client (`internal/rpc/`)

Manages the bidirectional streaming connection with the Azure Functions Host.

**Responsibilities:**
- Establish connection using host-provided parameters (host, port, worker ID)
- Maintain the streaming channel
- Route incoming messages to appropriate handlers
- Send outgoing responses

**Connection Flow:**
```
Worker Start
    │
    ▼
Parse CLI args (--host, --port, --workerId, --requestId, --grpcMaxMessageLength)
    │
    ▼
Connect to host:port via gRPC
    │
    ▼
Call FunctionRpc.EventStream() - bidirectional stream
    │
    ▼
Send StartStream message with workerId
    │
    ▼
Enter message loop (receive/handle/respond)
```

### 2. Message Handlers (`internal/rpc/handlers.go`)

Handle specific message types from the host.

#### WorkerInitRequest → WorkerInitResponse
- Received once at startup
- Contains host version, capabilities
- Respond with worker version, capabilities

#### FunctionLoadRequest → FunctionLoadResponse
- Received for each function to load
- Contains function metadata (ID, name, bindings)
- Load and validate function, respond with success/failure

#### InvocationRequest → InvocationResponse
- Received when a function is triggered
- Contains input data, trigger metadata
- Execute function, return output data and status

#### WorkerStatusRequest → WorkerStatusResponse  
- Health check from host
- Respond with current worker status

### 3. Function Registry (`internal/registry/`)

Maintains a mapping of function IDs to handlers.

```go
type FunctionInfo struct {
    ID        string
    Name      string
    Handler   FunctionHandler
    Bindings  []BindingInfo
}

type Registry struct {
    functions map[string]*FunctionInfo  // key: functionId
}
```

### 4. Binding Handlers (`internal/bindings/`)

Convert between protobuf TypedData and Go types.

**HTTP Trigger Example:**
```go
// Incoming: RpcHttp from protobuf
// Convert to: azfunc.HttpRequest (user-friendly Go type)

// Outgoing: azfunc.HttpResponse from user
// Convert to: TypedData with RpcHttp for protobuf
```

### 5. Public SDK (`pkg/azfunc/`)

Developer-facing API for writing functions.

```go
package main

import (
    "github.com/Azure/azure-functions-go-worker/pkg/azfunc"
)

func main() {
    azfunc.RegisterFunction("HttpTrigger", azfunc.HttpTrigger(), handleHttp)
    azfunc.Start()
}

func handleHttp(ctx *azfunc.Context, req *azfunc.HttpRequest) (*azfunc.HttpResponse, error) {
    name := req.Query("name")
    if name == "" {
        name = "World"
    }
    
    return &azfunc.HttpResponse{
        StatusCode: 200,
        Body:       fmt.Sprintf("Hello, %s!", name),
    }, nil
}
```

## gRPC Protocol Details

### Streaming Model

The worker uses **bidirectional streaming** via `FunctionRpc.EventStream()`:

```protobuf
service FunctionRpc {
    rpc EventStream(stream StreamingMessage) returns (stream StreamingMessage);
}
```

All communication happens over this single stream using `StreamingMessage` wrapper:

```protobuf
message StreamingMessage {
    string request_id = 1;
    oneof content {
        StartStream start_stream = 20;
        WorkerInitRequest worker_init_request = 17;
        WorkerInitResponse worker_init_response = 16;
        FunctionLoadRequest function_load_request = 8;
        FunctionLoadResponse function_load_response = 9;
        InvocationRequest invocation_request = 4;
        InvocationResponse invocation_response = 5;
        // ... more message types
    }
}
```

### Message Sequence

```
Host                                    Worker
  │                                        │
  │◄──────── StartStream ─────────────────│  (Worker announces itself)
  │                                        │
  │──────── WorkerInitRequest ───────────►│
  │◄─────── WorkerInitResponse ───────────│
  │                                        │
  │──────── FunctionLoadRequest ─────────►│  (For each function)
  │◄─────── FunctionLoadResponse ─────────│
  │                                        │
  │──────── InvocationRequest ───────────►│  (When triggered)
  │◄─────── InvocationResponse ───────────│
  │                                        │
  │──────── WorkerStatusRequest ─────────►│  (Periodic health check)
  │◄─────── WorkerStatusResponse ─────────│
```

## Function Discovery Mechanism

### Approach: Code Registration + function.json

Functions are registered in code and discovered via `function.json` files:

**1. Developer registers function in code:**
```go
azfunc.RegisterFunction("MyHttpFunc", azfunc.HttpTrigger(), handler)
```

**2. Each function has a folder with function.json:**
```
MyHttpFunc/
└── function.json
```

```json
{
  "bindings": [
    {
      "type": "httpTrigger",
      "direction": "in",
      "name": "req",
      "methods": ["get", "post"]
    },
    {
      "type": "http",
      "direction": "out",
      "name": "$return"
    }
  ]
}
```

**3. Host reads function.json and sends FunctionLoadRequest**

**4. Worker matches function ID to registered handler**

## Type Mapping

### HTTP Types

| Protobuf (RpcHttp)     | Go SDK (azfunc)        |
|------------------------|------------------------|
| Method                 | HttpRequest.Method     |
| Url                    | HttpRequest.URL        |
| Headers                | HttpRequest.Headers    |
| Body (TypedData)       | HttpRequest.Body       |
| Query                  | HttpRequest.Query()    |
| Params                 | HttpRequest.Params     |
| StatusCode             | HttpResponse.StatusCode|
| Headers                | HttpResponse.Headers   |
| Body                   | HttpResponse.Body      |

### TypedData Conversion

```go
// TypedData can contain different data types
message TypedData {
    oneof data {
        string string = 1;
        bytes bytes = 2;
        sint64 int = 3;
        double double = 4;
        RpcHttp http = 5;
        // ... more types
    }
}

// Go conversion functions
func TypedDataToBytes(td *TypedData) []byte
func TypedDataToString(td *TypedData) string  
func BytesToTypedData(b []byte) *TypedData
func StringToTypedData(s string) *TypedData
```

## Error Handling

### Function Execution Errors

- Catch panics in function execution
- Return `InvocationResponse` with `Result.Status = Failure`
- Include exception message and stack trace

### Connection Errors

- Implement reconnection logic with backoff
- Log connection issues for debugging

## MVP Implementation Phases

### Phase 1: Core Infrastructure ✓ (Current)
- [x] Project structure
- [ ] Proto generation setup
- [ ] gRPC client skeleton
- [ ] Basic message routing

### Phase 2: Worker Lifecycle
- [ ] WorkerInitRequest/Response handling
- [ ] FunctionLoadRequest/Response handling
- [ ] Function registry

### Phase 3: HTTP Trigger
- [ ] InvocationRequest handling
- [ ] HTTP binding conversion
- [ ] InvocationResponse with HTTP output
- [ ] End-to-end HTTP function test

### Phase 4: Developer Experience
- [ ] Clean SDK API
- [ ] Error messages and logging
- [ ] Example functions
- [ ] Documentation

## Configuration

Worker receives configuration via CLI arguments from the host:

| Argument | Description |
|----------|-------------|
| `--host` | gRPC server host |
| `--port` | gRPC server port |
| `--workerId` | Unique worker identifier |
| `--requestId` | Request correlation ID |
| `--grpcMaxMessageLength` | Max gRPC message size |

## Future Considerations

### Additional Bindings (Post-MVP)
- Timer Trigger
- Queue Trigger/Output (Storage Queue, Service Bus)
- Blob Input/Output
- Cosmos DB Input/Output
- Event Hub Trigger/Output
- Durable Functions support

### Performance Optimizations
- Connection pooling
- Concurrent invocation handling
- Memory management for large payloads

### Tooling
- CLI tool for function scaffolding
- VS Code extension integration
- Azure Functions Core Tools integration

## References

- [Azure Functions Language Worker Protocol](https://github.com/Azure/azure-functions-language-worker-protobuf)
- [Python Worker Implementation](https://github.com/Azure/azure-functions-python-worker)
- [Node.js Worker Implementation](https://github.com/Azure/azure-functions-nodejs-worker)
- [Azure Functions Host](https://github.com/Azure/azure-functions-host)
