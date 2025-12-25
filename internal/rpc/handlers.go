package rpc

import (
	"context"
	"fmt"
	"log"
	"runtime"

	pb "github.com/laveeshb/azure-functions-go-worker/internal/rpc/proto"
)

const (
	// WorkerVersion is the version of this Go worker.
	WorkerVersion = "0.1.0"
	// RuntimeName identifies this as a Go worker.
	RuntimeName = "go"
)

// FunctionExecutor is the interface for executing functions.
type FunctionExecutor interface {
	// LoadFunction loads a function with the given metadata.
	LoadFunction(ctx context.Context, functionID string, metadata *pb.RpcFunctionMetadata) error
	// Execute executes a function with the given invocation request.
	Execute(ctx context.Context, req *pb.InvocationRequest) (*pb.InvocationResponse, error)
}

// Handlers contains the message handlers for the worker.
type Handlers struct {
	executor FunctionExecutor
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(executor FunctionExecutor) *Handlers {
	return &Handlers{
		executor: executor,
	}
}

// HandleWorkerInit handles the WorkerInitRequest from the host.
func (h *Handlers) HandleWorkerInit(ctx context.Context, requestID string, req *pb.WorkerInitRequest) (*pb.StreamingMessage, error) {
	log.Printf("Received WorkerInitRequest - Host version: %s, Worker directory: %s", req.HostVersion, req.WorkerDirectory)

	// Build capabilities map
	capabilities := map[string]string{
		"SupportsLoadResponseCollection": "true",
		"RpcHttpBodyOnly":                "true",
		"RpcHttpTriggerMetadataRemoved":  "true",
		"UseNullableValueDictionaryForHttp": "true",
	}

	// Build worker metadata
	metadata := &pb.WorkerMetadata{
		RuntimeName:    RuntimeName,
		RuntimeVersion: runtime.Version(),
		WorkerVersion:  WorkerVersion,
		WorkerBitness:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		CustomProperties: map[string]string{
			"GoVersion": runtime.Version(),
		},
	}

	response := &pb.StreamingMessage{
		RequestId: requestID,
		Content: &pb.StreamingMessage_WorkerInitResponse{
			WorkerInitResponse: &pb.WorkerInitResponse{
				WorkerVersion:  WorkerVersion,
				Capabilities:   capabilities,
				WorkerMetadata: metadata,
				Result: &pb.StatusResult{
					Status: pb.StatusResult_Success,
				},
			},
		},
	}

	log.Printf("Sending WorkerInitResponse - Worker version: %s", WorkerVersion)
	return response, nil
}

// HandleFunctionLoad handles the FunctionLoadRequest from the host.
func (h *Handlers) HandleFunctionLoad(ctx context.Context, requestID string, req *pb.FunctionLoadRequest) (*pb.StreamingMessage, error) {
	log.Printf("Received FunctionLoadRequest - Function ID: %s, Name: %s", req.FunctionId, req.Metadata.Name)

	var result *pb.StatusResult

	if h.executor != nil {
		err := h.executor.LoadFunction(ctx, req.FunctionId, req.Metadata)
		if err != nil {
			log.Printf("Error loading function %s: %v", req.Metadata.Name, err)
			result = &pb.StatusResult{
				Status: pb.StatusResult_Failure,
				Exception: &pb.RpcException{
					Message: err.Error(),
				},
			}
		} else {
			log.Printf("Successfully loaded function: %s", req.Metadata.Name)
			result = &pb.StatusResult{
				Status: pb.StatusResult_Success,
			}
		}
	} else {
		log.Printf("No executor configured, marking function as loaded: %s", req.Metadata.Name)
		result = &pb.StatusResult{
			Status: pb.StatusResult_Success,
		}
	}

	response := &pb.StreamingMessage{
		RequestId: requestID,
		Content: &pb.StreamingMessage_FunctionLoadResponse{
			FunctionLoadResponse: &pb.FunctionLoadResponse{
				FunctionId: req.FunctionId,
				Result:     result,
			},
		},
	}

	return response, nil
}

// HandleInvocation handles the InvocationRequest from the host.
func (h *Handlers) HandleInvocation(ctx context.Context, requestID string, req *pb.InvocationRequest) (*pb.StreamingMessage, error) {
	log.Printf("Received InvocationRequest - Invocation ID: %s, Function ID: %s", req.InvocationId, req.FunctionId)

	var invocationResponse *pb.InvocationResponse

	if h.executor != nil {
		resp, err := h.executor.Execute(ctx, req)
		if err != nil {
			log.Printf("Error executing function: %v", err)
			invocationResponse = &pb.InvocationResponse{
				InvocationId: req.InvocationId,
				Result: &pb.StatusResult{
					Status: pb.StatusResult_Failure,
					Exception: &pb.RpcException{
						Message:    err.Error(),
						StackTrace: "", // TODO: capture stack trace
					},
				},
			}
		} else {
			invocationResponse = resp
		}
	} else {
		log.Printf("No executor configured, returning empty response")
		invocationResponse = &pb.InvocationResponse{
			InvocationId: req.InvocationId,
			Result: &pb.StatusResult{
				Status: pb.StatusResult_Success,
			},
		}
	}

	response := &pb.StreamingMessage{
		RequestId: requestID,
		Content: &pb.StreamingMessage_InvocationResponse{
			InvocationResponse: invocationResponse,
		},
	}

	return response, nil
}

// HandleWorkerStatus handles the WorkerStatusRequest from the host.
func (h *Handlers) HandleWorkerStatus(ctx context.Context, requestID string, req *pb.WorkerStatusRequest) (*pb.StreamingMessage, error) {
	log.Printf("Received WorkerStatusRequest")

	response := &pb.StreamingMessage{
		RequestId: requestID,
		Content: &pb.StreamingMessage_WorkerStatusResponse{
			WorkerStatusResponse: &pb.WorkerStatusResponse{},
		},
	}

	return response, nil
}

// HandleEnvironmentReload handles the FunctionEnvironmentReloadRequest from the host.
func (h *Handlers) HandleEnvironmentReload(ctx context.Context, requestID string, req *pb.FunctionEnvironmentReloadRequest) (*pb.StreamingMessage, error) {
	log.Printf("Received FunctionEnvironmentReloadRequest - Function app directory: %s", req.FunctionAppDirectory)

	// TODO: Reload environment variables if needed

	response := &pb.StreamingMessage{
		RequestId: requestID,
		Content: &pb.StreamingMessage_FunctionEnvironmentReloadResponse{
			FunctionEnvironmentReloadResponse: &pb.FunctionEnvironmentReloadResponse{
				Result: &pb.StatusResult{
					Status: pb.StatusResult_Success,
				},
			},
		},
	}

	return response, nil
}
