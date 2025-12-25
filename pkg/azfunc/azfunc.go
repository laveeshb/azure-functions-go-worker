// Package azfunc provides the public SDK for writing Azure Functions in Go.
//
// Example usage:
//
//	package main
//
//	import (
//		"github.com/laveeshb/azure-functions-go-worker/pkg/azfunc"
//	)
//
//	func main() {
//		azfunc.RegisterHttpFunction("HttpTrigger", handleHttp)
//		azfunc.Start()
//	}
//
//	func handleHttp(ctx *azfunc.Context, req *azfunc.HttpRequest) (*azfunc.HttpResponse, error) {
//		name := req.Query("name")
//		if name == "" {
//			name = "World"
//		}
//		return azfunc.OK(fmt.Sprintf("Hello, %s!", name)), nil
//	}
package azfunc

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/laveeshb/azure-functions-go-worker/internal/bindings"
	"github.com/laveeshb/azure-functions-go-worker/internal/registry"
	"github.com/laveeshb/azure-functions-go-worker/internal/rpc"
	pb "github.com/laveeshb/azure-functions-go-worker/internal/rpc/proto"
)

// Re-export types from internal packages for public use
type (
	// HttpRequest represents an incoming HTTP request.
	HttpRequest = bindings.HttpRequest
	// HttpResponse represents an HTTP response to send back.
	HttpResponse = bindings.HttpResponse
)

// Re-export convenience functions
var (
	// OK creates an HTTP 200 response with the given body.
	OK = bindings.OK
	// Created creates an HTTP 201 response with the given body.
	Created = bindings.Created
	// BadRequest creates an HTTP 400 response with the given message.
	BadRequest = bindings.BadRequest
	// NotFound creates an HTTP 404 response with the given message.
	NotFound = bindings.NotFound
	// InternalServerError creates an HTTP 500 response with the given message.
	InternalServerError = bindings.InternalServerError
	// NewHttpResponse creates a new HttpResponse with default values.
	NewHttpResponse = bindings.NewHttpResponse
)

// Context provides function execution context.
type Context struct {
	context.Context

	// InvocationID is the unique identifier for this invocation.
	InvocationID string
	// FunctionID is the unique identifier for the function.
	FunctionID string
	// FunctionName is the name of the function.
	FunctionName string
	// TraceContext contains distributed tracing information.
	TraceContext *TraceContext
	// RetryContext contains retry information if this is a retry.
	RetryContext *RetryContext

	// Logger for this invocation
	logger *invocationLogger
}

// TraceContext contains distributed tracing information.
type TraceContext struct {
	TraceParent string
	TraceState  string
	Attributes  map[string]string
}

// RetryContext contains information about retries.
type RetryContext struct {
	RetryCount    int
	MaxRetryCount int
}

// Log logs a message at the Information level.
func (c *Context) Log(message string) {
	if c.logger != nil {
		c.logger.Log(pb.RpcLog_Information, message)
	}
}

// LogDebug logs a message at the Debug level.
func (c *Context) LogDebug(message string) {
	if c.logger != nil {
		c.logger.Log(pb.RpcLog_Debug, message)
	}
}

// LogWarning logs a message at the Warning level.
func (c *Context) LogWarning(message string) {
	if c.logger != nil {
		c.logger.Log(pb.RpcLog_Warning, message)
	}
}

// LogError logs a message at the Error level.
func (c *Context) LogError(message string) {
	if c.logger != nil {
		c.logger.Log(pb.RpcLog_Error, message)
	}
}

// HttpHandler is the signature for HTTP-triggered function handlers.
type HttpHandler func(ctx *Context, req *HttpRequest) (*HttpResponse, error)

// Global worker instance
var (
	globalRegistry *registry.Registry
	globalClient   *rpc.Client
)

func init() {
	globalRegistry = registry.NewRegistry()
}

// RegisterHttpFunction registers an HTTP-triggered function handler.
func RegisterHttpFunction(name string, handler HttpHandler) error {
	// Wrap the HTTP handler to work with the internal registry
	wrappedHandler := func(ctx context.Context, req *pb.InvocationRequest) (*pb.InvocationResponse, error) {
		return executeHttpHandler(ctx, req, handler)
	}

	return globalRegistry.RegisterHandler(name, wrappedHandler)
}

// executeHttpHandler executes an HTTP handler and converts the result.
func executeHttpHandler(ctx context.Context, req *pb.InvocationRequest, handler HttpHandler) (*pb.InvocationResponse, error) {
	// Create function context
	funcCtx := &Context{
		Context:      ctx,
		InvocationID: req.InvocationId,
		FunctionID:   req.FunctionId,
	}

	// Set up trace context
	if req.TraceContext != nil {
		funcCtx.TraceContext = &TraceContext{
			TraceParent: req.TraceContext.TraceParent,
			TraceState:  req.TraceContext.TraceState,
			Attributes:  req.TraceContext.Attributes,
		}
	}

	// Set up retry context
	if req.RetryContext != nil {
		funcCtx.RetryContext = &RetryContext{
			RetryCount:    int(req.RetryContext.RetryCount),
			MaxRetryCount: int(req.RetryContext.MaxRetryCount),
		}
	}

	// Extract HTTP request from input bindings
	var httpRequest *HttpRequest
	for _, binding := range req.InputData {
		if data, ok := binding.RpcData.(*pb.ParameterBinding_Data); ok {
			if httpData, ok := data.Data.Data.(*pb.TypedData_Http); ok {
				httpRequest = bindings.NewHttpRequest(httpData.Http)
				break
			}
		}
	}

	if httpRequest == nil {
		httpRequest = &HttpRequest{
			Headers: make(map[string]string),
			Query:   make(map[string]string),
			Params:  make(map[string]string),
		}
	}

	// Call the user's handler
	httpResponse, err := handler(funcCtx, httpRequest)
	if err != nil {
		return &pb.InvocationResponse{
			InvocationId: req.InvocationId,
			Result: &pb.StatusResult{
				Status: pb.StatusResult_Failure,
				Exception: &pb.RpcException{
					Message: err.Error(),
				},
			},
		}, nil
	}

	// Convert response to protobuf
	rpcHttp, err := httpResponse.ToRpcHttp()
	if err != nil {
		return &pb.InvocationResponse{
			InvocationId: req.InvocationId,
			Result: &pb.StatusResult{
				Status: pb.StatusResult_Failure,
				Exception: &pb.RpcException{
					Message: fmt.Sprintf("failed to convert response: %v", err),
				},
			},
		}, nil
	}

	// Build response with HTTP output
	return &pb.InvocationResponse{
		InvocationId: req.InvocationId,
		ReturnValue: &pb.TypedData{
			Data: &pb.TypedData_Http{
				Http: rpcHttp,
			},
		},
		Result: &pb.StatusResult{
			Status: pb.StatusResult_Success,
		},
	}, nil
}

// Start starts the Azure Functions worker.
// This function blocks until the worker is terminated.
func Start() error {
	// Parse command line arguments
	// The host sends both legacy (--port) and new (--functions-uri) flags
	host := flag.String("host", "127.0.0.1", "gRPC server host")
	port := flag.Int("port", 0, "gRPC server port")
	workerID := flag.String("workerId", "", "Worker ID")
	requestID := flag.String("requestId", "", "Request ID")
	maxMsgLength := flag.Int("grpcMaxMessageLength", 0, "Max gRPC message length")

	// Also accept the newer --functions-* prefixed arguments that the host sends
	functionsURI := flag.String("functions-uri", "", "Functions URI (alternative to host/port)")
	functionsWorkerID := flag.String("functions-worker-id", "", "Functions Worker ID")
	functionsRequestID := flag.String("functions-request-id", "", "Functions Request ID")
	functionsMsgLength := flag.Int("functions-grpc-max-message-length", 0, "Functions Max gRPC message length")

	flag.Parse()

	// Prefer new --functions-* flags if provided
	if *functionsWorkerID != "" && *workerID == "" {
		*workerID = *functionsWorkerID
	}
	if *functionsRequestID != "" && *requestID == "" {
		*requestID = *functionsRequestID
	}
	if *functionsMsgLength > 0 && *maxMsgLength == 0 {
		*maxMsgLength = *functionsMsgLength
	}

	// Parse functions-uri to extract host/port if provided
	if *functionsURI != "" && *port == 0 {
		// Parse URI like "http://127.0.0.1:12345/"
		var uriHost string
		var uriPort int
		if _, err := fmt.Sscanf(*functionsURI, "http://%99[^:]:%d", &uriHost, &uriPort); err == nil {
			*host = uriHost
			*port = uriPort
			log.Printf("Parsed functions-uri: host=%s, port=%d", *host, *port)
		}
	}

	// Mark these as used to avoid compiler warnings
	_ = functionsURI
	_ = functionsWorkerID
	_ = functionsRequestID
	_ = functionsMsgLength

	// Fall back to environment variables if args not provided
	if *port == 0 {
		if envPort := os.Getenv("FUNCTIONS_GRPC_PORT"); envPort != "" {
			if p, err := fmt.Sscanf(envPort, "%d", port); err == nil && p == 1 {
				log.Printf("Using FUNCTIONS_GRPC_PORT from environment: %d", *port)
			}
		}
	}
	if *workerID == "" {
		if envWorkerID := os.Getenv("FUNCTIONS_WORKER_ID"); envWorkerID != "" {
			*workerID = envWorkerID
			log.Printf("Using FUNCTIONS_WORKER_ID from environment: %s", *workerID)
		}
	}
	if *requestID == "" {
		if envRequestID := os.Getenv("AZURE_FUNCTIONS_REQUEST_ID"); envRequestID != "" {
			*requestID = envRequestID
		}
	}

	if *port == 0 {
		return fmt.Errorf("port is required (use --port or set FUNCTIONS_GRPC_PORT)")
	}
	if *workerID == "" {
		return fmt.Errorf("workerId is required (use --workerId or set FUNCTIONS_WORKER_ID)")
	}

	log.Printf("Starting Azure Functions Go Worker")
	log.Printf("  Host: %s", *host)
	log.Printf("  Port: %d", *port)
	log.Printf("  Worker ID: %s", *workerID)
	log.Printf("  Request ID: %s", *requestID)
	log.Printf("  Max Message Length: %d", *maxMsgLength)

	// Create handlers with registry as executor
	handlers := rpc.NewHandlers(globalRegistry)

	// Create client
	cfg := rpc.Config{
		Host:             *host,
		Port:             *port,
		WorkerID:         *workerID,
		RequestID:        *requestID,
		MaxMessageLength: *maxMsgLength,
	}
	globalClient = rpc.NewClient(cfg, handlers)

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v", sig)
		cancel()
		globalClient.Stop()
	}()

	// Connect to the host
	if err := globalClient.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Start the message loop
	if err := globalClient.Start(ctx); err != nil {
		return fmt.Errorf("worker stopped with error: %w", err)
	}

	log.Printf("Azure Functions Go Worker stopped")
	return nil
}

// invocationLogger handles logging for a specific invocation.
type invocationLogger struct {
	invocationID string
}

// Log sends a log message to the host.
func (l *invocationLogger) Log(level pb.RpcLog_Level, message string) {
	if globalClient != nil {
		globalClient.SendLog(l.invocationID, level, message)
	}
}
