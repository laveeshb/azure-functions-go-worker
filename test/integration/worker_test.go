// Package integration contains integration tests that simulate the Azure Functions Host.
package integration

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	pb "github.com/laveeshb/azure-functions-go-worker/internal/rpc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// mockFunctionRpcServer implements a mock Azure Functions Host
type mockFunctionRpcServer struct {
	pb.UnimplementedFunctionRpcServer
	messages chan *pb.StreamingMessage
	t        *testing.T
}

func (s *mockFunctionRpcServer) EventStream(stream pb.FunctionRpc_EventStreamServer) error {
	// Send start stream message
	startMsg := &pb.StreamingMessage{
		RequestId: "start-123",
		Content: &pb.StreamingMessage_StartStream{
			StartStream: &pb.StartStream{
				WorkerId: "test-worker-001",
			},
		},
	}
	if err := stream.Send(startMsg); err != nil {
		return fmt.Errorf("failed to send start stream: %w", err)
	}

	// Wait for worker init request from client
	msg, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to receive message: %w", err)
	}
	s.t.Logf("Received message type: %T", msg.Content)

	// Check if it's a worker init request
	if initReq, ok := msg.Content.(*pb.StreamingMessage_WorkerInitRequest); ok {
		s.t.Logf("Worker init request received: HostVersion=%s", initReq.WorkerInitRequest.HostVersion)

		// Send worker init response
		initResp := &pb.StreamingMessage{
			RequestId: msg.RequestId,
			Content: &pb.StreamingMessage_WorkerInitResponse{
				WorkerInitResponse: &pb.WorkerInitResponse{
					WorkerVersion: "1.0.0",
					Result: &pb.StatusResult{
						Status: pb.StatusResult_Success,
					},
				},
			},
		}
		if err := stream.Send(initResp); err != nil {
			return fmt.Errorf("failed to send init response: %w", err)
		}
	}

	// Send function load request
	funcLoadReq := &pb.StreamingMessage{
		RequestId: "funcload-123",
		Content: &pb.StreamingMessage_FunctionLoadRequest{
			FunctionLoadRequest: &pb.FunctionLoadRequest{
				FunctionId: "func-001",
				Metadata: &pb.RpcFunctionMetadata{
					Name:       "HttpTrigger",
					EntryPoint: "HttpTrigger",
					Bindings: map[string]*pb.BindingInfo{
						"req": {
							Type:      "httpTrigger",
							Direction: pb.BindingInfo_in,
						},
						"res": {
							Type:      "http",
							Direction: pb.BindingInfo_out,
						},
					},
				},
			},
		},
	}
	if err := stream.Send(funcLoadReq); err != nil {
		return fmt.Errorf("failed to send function load request: %w", err)
	}

	// Wait for function load response
	msg, err = stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to receive function load response: %w", err)
	}
	if loadResp, ok := msg.Content.(*pb.StreamingMessage_FunctionLoadResponse); ok {
		s.t.Logf("Function load response: FunctionID=%s, Success=%v",
			loadResp.FunctionLoadResponse.FunctionId,
			loadResp.FunctionLoadResponse.Result.Status == pb.StatusResult_Success)
	}

	// Send invocation request
	invokeReq := &pb.StreamingMessage{
		RequestId: "invoke-123",
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
										Url:    "http://localhost/api/HttpTrigger?name=IntegrationTest",
										Query: map[string]string{
											"name": "IntegrationTest",
										},
										Headers: map[string]string{
											"Content-Type": "application/json",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	if err := stream.Send(invokeReq); err != nil {
		return fmt.Errorf("failed to send invocation request: %w", err)
	}

	// Wait for invocation response
	msg, err = stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to receive invocation response: %w", err)
	}
	if invokeResp, ok := msg.Content.(*pb.StreamingMessage_InvocationResponse); ok {
		s.t.Logf("Invocation response: InvocationID=%s, Success=%v",
			invokeResp.InvocationResponse.InvocationId,
			invokeResp.InvocationResponse.Result.Status == pb.StatusResult_Success)

		// Check the return value
		if invokeResp.InvocationResponse.ReturnValue != nil {
			if httpData, ok := invokeResp.InvocationResponse.ReturnValue.Data.(*pb.TypedData_Http); ok {
				s.t.Logf("HTTP Response: Status=%s", httpData.Http.StatusCode)
				if httpData.Http.Body != nil {
					if strData, ok := httpData.Http.Body.Data.(*pb.TypedData_String_); ok {
						s.t.Logf("Response Body: %s", strData.String_)

						// Verify the response contains our test name
						if strData.String_ != "Hello, IntegrationTest! This is an Azure Function running in Go." {
							s.t.Errorf("Unexpected response body: %s", strData.String_)
						}
					}
				}
			}
		}

		s.messages <- msg
	}

	return nil
}

// TestWorkerIntegration tests the worker by simulating an Azure Functions Host
func TestWorkerIntegration(t *testing.T) {
	// Start a mock gRPC server (simulating Azure Functions Host)
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer lis.Close()

	port := lis.Addr().(*net.TCPAddr).Port
	t.Logf("Mock host listening on port %d", port)

	mockServer := &mockFunctionRpcServer{
		messages: make(chan *pb.StreamingMessage, 10),
		t:        t,
	}

	grpcServer := grpc.NewServer()
	pb.RegisterFunctionRpcServer(grpcServer, mockServer)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			t.Logf("Server stopped: %v", err)
		}
	}()
	defer grpcServer.Stop()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Connect our worker client to the mock server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, fmt.Sprintf("127.0.0.1:%d", port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		t.Fatalf("Failed to dial mock server: %v", err)
	}
	defer conn.Close()

	t.Log("Worker connected to mock host")

	// Use the gRPC client to establish the event stream
	client := pb.NewFunctionRpcClient(conn)
	stream, err := client.EventStream(ctx)
	if err != nil {
		t.Fatalf("Failed to create event stream: %v", err)
	}

	// Receive start stream from host
	startMsg, err := stream.Recv()
	if err != nil {
		t.Fatalf("Failed to receive start stream: %v", err)
	}
	if _, ok := startMsg.Content.(*pb.StreamingMessage_StartStream); !ok {
		t.Fatalf("Expected StartStream, got %T", startMsg.Content)
	}
	t.Log("Received start stream from host")

	// This is a simplified test - in reality the worker would handle the full protocol
	// For now we're just verifying the gRPC connection works
	t.Log("Integration test passed - gRPC communication working")
}
