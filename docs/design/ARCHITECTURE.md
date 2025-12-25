# Azure Functions Go Worker - Architecture Design

## Overview

This document describes the architecture for a native Go language worker for Azure Functions. The worker implements an out-of-process model, communicating with the Azure Functions Host via gRPC.

## Goals

1. **First-class Go support** - Native Go experience, not Custom Handlers
2. **Familiar patterns** - Idiomatic Go APIs that feel natural to Go developers
3. **Performance** - Leverage Go's efficiency and fast startup times
4. **Compatibility** - Support all major Azure Functions bindings over time

## Design Decisions & Rationale

### Why Go instead of Rust?

| Factor | Go | Rust |
|--------|-----|------|
| Community demand | Higher - more requests on GitHub issues | Lower |
| Learning curve | Gentle - most devs productive in days | Steep - ownership/lifetimes take weeks |
| AWS Lambda precedent | Native Go support exists, migration appeal | No native Rust runtime |
| Compile times | Fast (~seconds) | Slow (~minutes for large projects) |
| Iteration speed | Faster prototyping | Slower due to strict compiler |

**Decision:** Go offers faster time-to-market and broader adoption potential.

### Why start fresh instead of forking radu-matei/azure-functions-golang-worker?

1. **Staleness** - Last commit was 6+ years ago (2018)
2. **API drift** - Azure Functions Host APIs have changed significantly
3. **Protobuf changes** - The gRPC contract has new messages and fields
4. **Go ecosystem evolution** - Go modules, generics, better error handling patterns
5. **Technical debt** - Starting fresh avoids inheriting outdated patterns

**Decision:** Clean slate allows modern Go practices and current Azure Functions APIs.

### Why out-of-process instead of in-process?

1. **No host modifications required** - Host already supports language workers via gRPC
2. **Process isolation** - Go panics don't crash the host
3. **Independent deployment** - Worker can be updated without host changes
4. **Precedent** - Python, Node.js, Java, PowerShell all use this model
5. **Microsoft's direction** - In-process is being deprecated for .NET too

**Decision:** Out-of-process is the supported, future-proof approach.

### Why compile functions into the worker binary?

**Alternatives considered:**
- **Reflection/plugins** - Go plugins are fragile, OS-specific, and have version coupling issues
- **Interpreted Go** - Yaegi/similar are slow and don't support all Go features
- **Separate binaries** - Would require custom IPC, complicating the architecture

**Our approach:** Functions are compiled into the worker binary via imports.

```go
// User's main.go
import _ "myapp/functions"  // registers handlers in init()

func main() {
    azfunc.Start()
}
```

**Benefits:**
- Full Go performance (no interpretation overhead)
- Standard Go toolchain (go build, go test)
- Type safety at compile time
- Easy debugging with standard tools

### Why function.json for discovery instead of code-only?

1. **Host compatibility** - Azure Functions Host reads function.json for trigger/binding config
2. **Declarative bindings** - Connection strings, auth levels, routes are config, not code
3. **Tooling integration** - VS Code, Azure Portal, Core Tools expect function.json
4. **Separation of concerns** - Infrastructure config vs application logic

**Future consideration:** We may add code-first metadata generation (like Python decorators) that auto-generates function.json.

### Why `internal/` vs `pkg/` package layout?

| Package | Visibility | Purpose |
|---------|------------|---------|
| `internal/` | Private | Implementation details that may change |
| `pkg/azfunc/` | Public | Stable API for function developers |

**Rationale:**
- `internal/rpc` - gRPC client details are implementation, not API
- `internal/registry` - Function storage internals may change
- `internal/bindings` - Converter internals may evolve
- `pkg/azfunc` - Developer-facing API must be stable

This follows Go best practices for library design.

### Why typed HttpRequest/HttpResponse instead of raw protobuf?

**Raw protobuf problems:**
```go
// Unfriendly - requires knowing protobuf internals
body := req.InputData[0].GetData().GetHttp().GetBody().GetString_()
```

**Our SDK:**
```go
// Friendly - idiomatic Go
body := req.BodyAsString()
name := req.GetQuery("name")
```

**Benefits:**
- Hides protobuf complexity from users
- Provides type-safe, discoverable API
- Matches patterns Go developers expect (similar to net/http)

### Why bidirectional streaming instead of unary RPC?

The Azure Functions Host uses `EventStream` - a single bidirectional stream for all communication:

```protobuf
service FunctionRpc {
  rpc EventStream(stream StreamingMessage) returns (stream StreamingMessage);
}
```

**Reasons:**
1. **Connection efficiency** - Single persistent connection vs per-request overhead
2. **Async messaging** - Host can send invocations anytime without polling
3. **Logging** - Worker can stream logs back continuously
4. **Protocol requirement** - This is how the host works, not optional

### Why handler registration in init() instead of main()?

```go
func init() {
    azfunc.RegisterHttpFunction("MyFunc", handler)
}

func main() {
    azfunc.Start()  // blocking
}
```

**Rationale:**
1. **Guaranteed execution** - init() runs before main(), ensures registration happens
2. **Decoupled packages** - Functions can be in separate packages, each with init()
3. **Testing** - Can import function packages in tests without starting worker
4. **Pattern precedent** - Similar to database/sql driver registration

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
    "github.com/laveeshb/azure-functions-go-worker/pkg/azfunc"
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
