// Package rpc provides the gRPC client implementation for communicating with the Azure Functions Host.
package rpc

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	pb "github.com/laveeshb/azure-functions-go-worker/internal/rpc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client represents a gRPC client for communicating with the Azure Functions Host.
type Client struct {
	host     string
	port     int
	workerID string

	conn   *grpc.ClientConn
	client pb.FunctionRpcClient
	stream pb.FunctionRpc_EventStreamClient

	handlers *Handlers

	mu       sync.RWMutex
	running  bool
	stopChan chan struct{}
}

// Config holds the configuration for the gRPC client.
type Config struct {
	Host               string
	Port               int
	WorkerID           string
	RequestID          string
	MaxMessageLength   int
}

// NewClient creates a new gRPC client with the given configuration.
func NewClient(cfg Config, handlers *Handlers) *Client {
	return &Client{
		host:     cfg.Host,
		port:     cfg.Port,
		workerID: cfg.WorkerID,
		handlers: handlers,
		stopChan: make(chan struct{}),
	}
}

// Connect establishes a connection to the Azure Functions Host.
func (c *Client) Connect(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", c.host, c.port)
	log.Printf("Connecting to Azure Functions Host at %s", addr)

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to host: %w", err)
	}
	c.conn = conn
	c.client = pb.NewFunctionRpcClient(conn)

	log.Printf("Connected to Azure Functions Host")
	return nil
}

// Start begins the bidirectional streaming communication with the host.
func (c *Client) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return fmt.Errorf("client is already running")
	}
	c.running = true
	c.mu.Unlock()

	// Establish the event stream
	stream, err := c.client.EventStream(ctx)
	if err != nil {
		return fmt.Errorf("failed to create event stream: %w", err)
	}
	c.stream = stream

	// Send StartStream message to identify this worker
	startMsg := &pb.StreamingMessage{
		Content: &pb.StreamingMessage_StartStream{
			StartStream: &pb.StartStream{
				WorkerId: c.workerID,
			},
		},
	}
	if err := c.stream.Send(startMsg); err != nil {
		return fmt.Errorf("failed to send start stream message: %w", err)
	}
	log.Printf("Sent StartStream message with worker ID: %s", c.workerID)

	// Start the message loop
	return c.messageLoop(ctx)
}

// messageLoop continuously receives and processes messages from the host.
func (c *Client) messageLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			log.Printf("Context cancelled, stopping message loop")
			return ctx.Err()
		case <-c.stopChan:
			log.Printf("Stop signal received, stopping message loop")
			return nil
		default:
			msg, err := c.stream.Recv()
			if err == io.EOF {
				log.Printf("Stream closed by host")
				return nil
			}
			if err != nil {
				return fmt.Errorf("error receiving message: %w", err)
			}

			// Handle the message asynchronously
			go c.handleMessage(ctx, msg)
		}
	}
}

// handleMessage routes incoming messages to the appropriate handler.
func (c *Client) handleMessage(ctx context.Context, msg *pb.StreamingMessage) {
	var response *pb.StreamingMessage
	var err error

	switch content := msg.Content.(type) {
	case *pb.StreamingMessage_WorkerInitRequest:
		response, err = c.handlers.HandleWorkerInit(ctx, msg.RequestId, content.WorkerInitRequest)

	case *pb.StreamingMessage_FunctionLoadRequest:
		response, err = c.handlers.HandleFunctionLoad(ctx, msg.RequestId, content.FunctionLoadRequest)

	case *pb.StreamingMessage_FunctionLoadRequestCollection:
		response, err = c.handlers.HandleFunctionLoadCollection(ctx, msg.RequestId, content.FunctionLoadRequestCollection)

	case *pb.StreamingMessage_InvocationRequest:
		response, err = c.handlers.HandleInvocation(ctx, msg.RequestId, content.InvocationRequest)

	case *pb.StreamingMessage_WorkerStatusRequest:
		response, err = c.handlers.HandleWorkerStatus(ctx, msg.RequestId, content.WorkerStatusRequest)

	case *pb.StreamingMessage_FunctionEnvironmentReloadRequest:
		response, err = c.handlers.HandleEnvironmentReload(ctx, msg.RequestId, content.FunctionEnvironmentReloadRequest)

	case *pb.StreamingMessage_WorkerTerminate:
		log.Printf("Received worker terminate request")
		c.Stop()
		return

	default:
		log.Printf("Unknown message type received: %T", content)
		return
	}

	if err != nil {
		log.Printf("Error handling message: %v", err)
		return
	}

	if response != nil {
		if err := c.Send(response); err != nil {
			log.Printf("Error sending response: %v", err)
		}
	}
}

// Send sends a message to the host.
func (c *Client) Send(msg *pb.StreamingMessage) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.stream == nil {
		return fmt.Errorf("stream is not initialized")
	}

	return c.stream.Send(msg)
}

// SendLog sends a log message to the host.
func (c *Client) SendLog(invocationID string, level pb.RpcLog_Level, message string) error {
	msg := &pb.StreamingMessage{
		Content: &pb.StreamingMessage_RpcLog{
			RpcLog: &pb.RpcLog{
				InvocationId: invocationID,
				Level:        level,
				Message:      message,
				Category:     "Function",
				LogCategory:  pb.RpcLog_User,
			},
		},
	}
	return c.Send(msg)
}

// Stop gracefully stops the client.
func (c *Client) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return
	}

	c.running = false
	close(c.stopChan)

	if c.stream != nil {
		c.stream.CloseSend()
	}

	if c.conn != nil {
		c.conn.Close()
	}

	log.Printf("Client stopped")
}

// WaitForConnection waits for the connection to be established with a timeout.
func (c *Client) WaitForConnection(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for connection")
		default:
			c.mu.RLock()
			running := c.running
			c.mu.RUnlock()
			if running {
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}
