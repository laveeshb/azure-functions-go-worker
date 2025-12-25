// Package integration contains integration tests that simulate the Azure Functions Host.
package integration

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-functions-go-worker/internal/bindings"
	"github.com/Azure/azure-functions-go-worker/internal/registry"
	"github.com/Azure/azure-functions-go-worker/internal/rpc"
	pb "github.com/Azure/azure-functions-go-worker/internal/rpc/proto"
	"google.golang.org/grpc"
)

// TestEndToEndHttpTrigger tests the complete flow from host to worker and back
func TestEndToEndHttpTrigger(t *testing.T) {
	// Create a registry and register a test function
	reg := registry.NewRegistry()

	err := reg.RegisterHandler("TestHttpFunction", func(ctx context.Context, req *pb.InvocationRequest) (*pb.InvocationResponse, error) {
		// Extract name from HTTP request
		var name string
		for _, binding := range req.InputData {
			if data, ok := binding.RpcData.(*pb.ParameterBinding_Data); ok {
				if httpData, ok := data.Data.Data.(*pb.TypedData_Http); ok {
					if httpData.Http.Query != nil {
						name = httpData.Http.Query["name"]
					}
				}
			}
		}
		if name == "" {
			name = "World"
		}

		// Create response
		resp := bindings.OK(fmt.Sprintf("Hello, %s!", name))
		rpcHttp, _ := resp.ToRpcHttp()

		return &pb.InvocationResponse{
			InvocationId: req.InvocationId,
			ReturnValue: &pb.TypedData{
				Data: &pb.TypedData_Http{Http: rpcHttp},
			},
			Result: &pb.StatusResult{Status: pb.StatusResult_Success},
		}, nil
	})
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	// Start mock host server
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer lis.Close()

	port := lis.Addr().(*net.TCPAddr).Port
	t.Logf("Mock host listening on port %d", port)

	// Create mock host
	host := &mockHost{
		t:         t,
		responses: make(chan *pb.StreamingMessage, 10),
		done:      make(chan struct{}),
	}

	grpcServer := grpc.NewServer()
	pb.RegisterFunctionRpcServer(grpcServer, host)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			t.Logf("Server stopped: %v", err)
		}
	}()
	defer grpcServer.Stop()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Create and start our worker client
	handlers := rpc.NewHandlers(reg)
	cfg := rpc.Config{
		Host:     "127.0.0.1",
		Port:     port,
		WorkerID: "test-worker-001",
	}
	client := rpc.NewClient(cfg, handlers)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to mock host
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Start the worker in background
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := client.Start(ctx); err != nil && ctx.Err() == nil {
			t.Logf("Worker stopped: %v", err)
		}
	}()

	// Wait for invocation response
	select {
	case resp := <-host.responses:
		if invokeResp, ok := resp.Content.(*pb.StreamingMessage_InvocationResponse); ok {
			t.Logf("Got invocation response: %s", invokeResp.InvocationResponse.InvocationId)

			if invokeResp.InvocationResponse.Result.Status != pb.StatusResult_Success {
				t.Errorf("Expected success, got: %v", invokeResp.InvocationResponse.Result.Status)
			}

			// Verify response body
			if invokeResp.InvocationResponse.ReturnValue != nil {
				if httpData, ok := invokeResp.InvocationResponse.ReturnValue.Data.(*pb.TypedData_Http); ok {
					if httpData.Http.Body != nil {
						if strData, ok := httpData.Http.Body.Data.(*pb.TypedData_String_); ok {
							expected := "Hello, E2ETest!"
							if strData.String_ != expected {
								t.Errorf("Expected '%s', got '%s'", expected, strData.String_)
							} else {
								t.Logf("SUCCESS: Got expected response: %s", strData.String_)
							}
						}
					}
				}
			}
		}
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for invocation response")
	}

	// Stop worker
	client.Stop()
	cancel()
	wg.Wait()

	t.Log("End-to-end test completed successfully")
}

// mockHost simulates the Azure Functions Host
type mockHost struct {
	pb.UnimplementedFunctionRpcServer
	t         *testing.T
	responses chan *pb.StreamingMessage
	done      chan struct{}
}

func (h *mockHost) EventStream(stream pb.FunctionRpc_EventStreamServer) error {
	// Wait for StartStream from worker
	msg, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to receive start stream: %w", err)
	}
	if startStream, ok := msg.Content.(*pb.StreamingMessage_StartStream); ok {
		h.t.Logf("Host: Received StartStream from worker: %s", startStream.StartStream.WorkerId)
	} else {
		return fmt.Errorf("expected StartStream, got %T", msg.Content)
	}

	// Send WorkerInitRequest
	if err := stream.Send(&pb.StreamingMessage{
		RequestId: "init-1",
		Content: &pb.StreamingMessage_WorkerInitRequest{
			WorkerInitRequest: &pb.WorkerInitRequest{
				HostVersion: "4.0.0-test",
			},
		},
	}); err != nil {
		return err
	}
	h.t.Log("Host: Sent WorkerInitRequest")

	// Process messages from worker
	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}

		switch content := msg.Content.(type) {
		case *pb.StreamingMessage_WorkerInitResponse:
			h.t.Logf("Host: Received WorkerInitResponse, version=%s",
				content.WorkerInitResponse.WorkerVersion)

			// Send FunctionLoadRequest
			if err := stream.Send(&pb.StreamingMessage{
				RequestId: "funcload-1",
				Content: &pb.StreamingMessage_FunctionLoadRequest{
					FunctionLoadRequest: &pb.FunctionLoadRequest{
						FunctionId: "func-001",
						Metadata: &pb.RpcFunctionMetadata{
							Name:       "TestHttpFunction",
							EntryPoint: "TestHttpFunction",
							Bindings: map[string]*pb.BindingInfo{
								"req": {Type: "httpTrigger", Direction: pb.BindingInfo_in},
								"res": {Type: "http", Direction: pb.BindingInfo_out},
							},
						},
					},
				},
			}); err != nil {
				return err
			}
			h.t.Log("Host: Sent FunctionLoadRequest")

		case *pb.StreamingMessage_FunctionLoadResponse:
			h.t.Logf("Host: Received FunctionLoadResponse, success=%v",
				content.FunctionLoadResponse.Result.Status == pb.StatusResult_Success)

			// Send InvocationRequest
			if err := stream.Send(&pb.StreamingMessage{
				RequestId: "invoke-1",
				Content: &pb.StreamingMessage_InvocationRequest{
					InvocationRequest: &pb.InvocationRequest{
						InvocationId: "inv-001",
						FunctionId:   "func-001",
						InputData: []*pb.ParameterBinding{
							{
								Name: "req",
								RpcData: &pb.ParameterBinding_Data{
									Data: &pb.TypedData{
										Data: &pb.TypedData_Http{
											Http: &pb.RpcHttp{
												Method: "GET",
												Url:    "http://localhost/api/test?name=E2ETest",
												Query:  map[string]string{"name": "E2ETest"},
											},
										},
									},
								},
							},
						},
					},
				},
			}); err != nil {
				return err
			}
			h.t.Log("Host: Sent InvocationRequest")

		case *pb.StreamingMessage_InvocationResponse:
			h.t.Logf("Host: Received InvocationResponse for %s",
				content.InvocationResponse.InvocationId)
			h.responses <- msg
			return nil

		case *pb.StreamingMessage_RpcLog:
			h.t.Logf("Host: Log from worker: %s", content.RpcLog.Message)

		default:
			h.t.Logf("Host: Received message type: %T", content)
		}
	}
}
