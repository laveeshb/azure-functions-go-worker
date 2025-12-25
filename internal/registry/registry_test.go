// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package registry

import (
	"context"
	"testing"

	pb "github.com/Azure/azure-functions-go-worker/internal/rpc/proto"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}
	if r.functions == nil {
		t.Error("functions map is nil")
	}
	if r.handlers == nil {
		t.Error("handlers map is nil")
	}
}

func TestRegisterHandler(t *testing.T) {
	r := NewRegistry()

	handler := func(ctx context.Context, req *pb.InvocationRequest) (*pb.InvocationResponse, error) {
		return &pb.InvocationResponse{InvocationId: req.InvocationId}, nil
	}

	// Register a handler
	err := r.RegisterHandler("TestFunc", handler)
	if err != nil {
		t.Fatalf("RegisterHandler failed: %v", err)
	}

	// Verify it's registered
	handlers := r.ListHandlers()
	if len(handlers) != 1 {
		t.Errorf("Expected 1 handler, got %d", len(handlers))
	}
	if handlers[0] != "TestFunc" {
		t.Errorf("Expected handler name 'TestFunc', got '%s'", handlers[0])
	}

	// Try to register duplicate
	err = r.RegisterHandler("TestFunc", handler)
	if err == nil {
		t.Error("Expected error when registering duplicate handler")
	}
}

func TestLoadFunction(t *testing.T) {
	r := NewRegistry()

	handler := func(ctx context.Context, req *pb.InvocationRequest) (*pb.InvocationResponse, error) {
		return &pb.InvocationResponse{InvocationId: req.InvocationId}, nil
	}

	// Register handler first
	r.RegisterHandler("MyFunc", handler)

	// Load function
	metadata := &pb.RpcFunctionMetadata{
		Name:       "MyFunc",
		EntryPoint: "MyFunc",
		Bindings:   make(map[string]*pb.BindingInfo),
	}

	err := r.LoadFunction(context.Background(), "func-123", metadata)
	if err != nil {
		t.Fatalf("LoadFunction failed: %v", err)
	}

	// Verify function is loaded
	info, exists := r.GetFunction("func-123")
	if !exists {
		t.Fatal("Function not found after loading")
	}
	if info.Name != "MyFunc" {
		t.Errorf("Expected function name 'MyFunc', got '%s'", info.Name)
	}
	if info.ID != "func-123" {
		t.Errorf("Expected function ID 'func-123', got '%s'", info.ID)
	}
}

func TestLoadFunctionNotRegistered(t *testing.T) {
	r := NewRegistry()

	metadata := &pb.RpcFunctionMetadata{
		Name:       "UnknownFunc",
		EntryPoint: "UnknownFunc",
	}

	err := r.LoadFunction(context.Background(), "func-456", metadata)
	if err == nil {
		t.Error("Expected error when loading unregistered function")
	}
}

func TestExecute(t *testing.T) {
	r := NewRegistry()

	expectedResponse := &pb.InvocationResponse{
		InvocationId: "inv-123",
		Result: &pb.StatusResult{
			Status: pb.StatusResult_Success,
		},
	}

	handler := func(ctx context.Context, req *pb.InvocationRequest) (*pb.InvocationResponse, error) {
		return expectedResponse, nil
	}

	r.RegisterHandler("ExecFunc", handler)
	r.LoadFunction(context.Background(), "func-exec", &pb.RpcFunctionMetadata{
		Name: "ExecFunc",
	})

	req := &pb.InvocationRequest{
		InvocationId: "inv-123",
		FunctionId:   "func-exec",
	}

	resp, err := r.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if resp.InvocationId != expectedResponse.InvocationId {
		t.Errorf("Expected invocation ID '%s', got '%s'", expectedResponse.InvocationId, resp.InvocationId)
	}
}

func TestExecuteFunctionNotFound(t *testing.T) {
	r := NewRegistry()

	req := &pb.InvocationRequest{
		InvocationId: "inv-123",
		FunctionId:   "nonexistent",
	}

	_, err := r.Execute(context.Background(), req)
	if err == nil {
		t.Error("Expected error when executing nonexistent function")
	}
}

func TestExecutePanicRecovery(t *testing.T) {
	r := NewRegistry()

	handler := func(ctx context.Context, req *pb.InvocationRequest) (*pb.InvocationResponse, error) {
		panic("something went wrong!")
	}

	r.RegisterHandler("PanicFunc", handler)
	r.LoadFunction(context.Background(), "func-panic", &pb.RpcFunctionMetadata{
		Name: "PanicFunc",
	})

	req := &pb.InvocationRequest{
		InvocationId: "inv-panic",
		FunctionId:   "func-panic",
	}

	// Should not panic, should return error response
	resp, err := r.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("Execute should not return error on panic, got: %v", err)
	}
	if resp == nil {
		t.Fatal("Response should not be nil")
	}
	if resp.Result.Status != pb.StatusResult_Failure {
		t.Errorf("Expected Failure status, got %v", resp.Result.Status)
	}
	if resp.Result.Exception == nil {
		t.Fatal("Expected exception in result")
	}
	if resp.Result.Exception.Message == "" {
		t.Error("Expected exception message")
	}
	if resp.Result.Exception.Type != "PanicException" {
		t.Errorf("Expected exception type 'PanicException', got '%s'", resp.Result.Exception.Type)
	}
}

func TestGetFunctionByName(t *testing.T) {
	r := NewRegistry()

	handler := func(ctx context.Context, req *pb.InvocationRequest) (*pb.InvocationResponse, error) {
		return nil, nil
	}

	r.RegisterHandler("NamedFunc", handler)
	r.LoadFunction(context.Background(), "func-named", &pb.RpcFunctionMetadata{
		Name: "NamedFunc",
	})

	info, exists := r.GetFunctionByName("NamedFunc")
	if !exists {
		t.Fatal("Function not found by name")
	}
	if info.ID != "func-named" {
		t.Errorf("Expected ID 'func-named', got '%s'", info.ID)
	}

	_, exists = r.GetFunctionByName("NonexistentFunc")
	if exists {
		t.Error("Should not find nonexistent function")
	}
}

func TestClear(t *testing.T) {
	r := NewRegistry()

	handler := func(ctx context.Context, req *pb.InvocationRequest) (*pb.InvocationResponse, error) {
		return nil, nil
	}

	r.RegisterHandler("ClearFunc", handler)
	r.LoadFunction(context.Background(), "func-clear", &pb.RpcFunctionMetadata{
		Name: "ClearFunc",
	})

	// Verify function exists
	funcs := r.ListFunctions()
	if len(funcs) != 1 {
		t.Fatalf("Expected 1 function, got %d", len(funcs))
	}

	// Clear
	r.Clear()

	// Functions should be empty, but handlers should remain
	funcs = r.ListFunctions()
	if len(funcs) != 0 {
		t.Errorf("Expected 0 functions after clear, got %d", len(funcs))
	}

	handlers := r.ListHandlers()
	if len(handlers) != 1 {
		t.Errorf("Expected 1 handler after clear, got %d", len(handlers))
	}
}
