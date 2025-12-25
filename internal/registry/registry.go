// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

// Package registry provides function registration and lookup functionality.
package registry

import (
	"context"
	"fmt"
	"log"
	"sync"

	pb "github.com/Azure/azure-functions-go-worker/internal/rpc/proto"
)

// FunctionHandler is the signature for a function handler.
// The handler receives the invocation context and input data, and returns output data or an error.
type FunctionHandler func(ctx context.Context, req *pb.InvocationRequest) (*pb.InvocationResponse, error)

// FunctionInfo contains metadata and handler for a registered function.
type FunctionInfo struct {
	// ID is the unique identifier assigned by the host.
	ID string
	// Name is the function name as registered by the developer.
	Name string
	// EntryPoint is the entry point specified in function.json.
	EntryPoint string
	// Handler is the function to execute.
	Handler FunctionHandler
	// Bindings contains the binding information from the host.
	Bindings map[string]*pb.BindingInfo
	// Metadata contains the full function metadata from the host.
	Metadata *pb.RpcFunctionMetadata
}

// Registry manages function registration and lookup.
type Registry struct {
	mu sync.RWMutex
	// functions maps function ID to function info (set after FunctionLoadRequest)
	functions map[string]*FunctionInfo
	// handlers maps function name to handler (set at startup via RegisterFunction)
	handlers map[string]FunctionHandler
}

// NewRegistry creates a new function registry.
func NewRegistry() *Registry {
	return &Registry{
		functions: make(map[string]*FunctionInfo),
		handlers:  make(map[string]FunctionHandler),
	}
}

// RegisterHandler registers a function handler by name.
// This is called at application startup before the worker connects to the host.
func (r *Registry) RegisterHandler(name string, handler FunctionHandler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.handlers[name]; exists {
		return fmt.Errorf("function handler already registered: %s", name)
	}

	r.handlers[name] = handler
	log.Printf("Registered function handler: %s", name)
	return nil
}

// LoadFunction loads a function with metadata from the host.
// This is called when the host sends a FunctionLoadRequest.
func (r *Registry) LoadFunction(ctx context.Context, functionID string, metadata *pb.RpcFunctionMetadata) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := metadata.Name
	entryPoint := metadata.EntryPoint
	if entryPoint == "" {
		entryPoint = name
	}

	// Look up the handler by name or entry point
	handler, exists := r.handlers[name]
	if !exists {
		handler, exists = r.handlers[entryPoint]
	}
	if !exists {
		return fmt.Errorf("no handler registered for function: %s (entry point: %s)", name, entryPoint)
	}

	info := &FunctionInfo{
		ID:         functionID,
		Name:       name,
		EntryPoint: entryPoint,
		Handler:    handler,
		Bindings:   metadata.Bindings,
		Metadata:   metadata,
	}

	r.functions[functionID] = info
	log.Printf("Loaded function: %s (ID: %s)", name, functionID)
	return nil
}

// Execute executes a function by its invocation request.
func (r *Registry) Execute(ctx context.Context, req *pb.InvocationRequest) (*pb.InvocationResponse, error) {
	r.mu.RLock()
	info, exists := r.functions[req.FunctionId]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("function not found: %s", req.FunctionId)
	}

	log.Printf("Executing function: %s (ID: %s, Invocation: %s)", info.Name, info.ID, req.InvocationId)

	// Call the registered handler
	return info.Handler(ctx, req)
}

// GetFunction returns the function info for a given function ID.
func (r *Registry) GetFunction(functionID string) (*FunctionInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	info, exists := r.functions[functionID]
	return info, exists
}

// GetFunctionByName returns the function info for a given function name.
func (r *Registry) GetFunctionByName(name string) (*FunctionInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, info := range r.functions {
		if info.Name == name {
			return info, true
		}
	}
	return nil, false
}

// ListFunctions returns all registered function names.
func (r *Registry) ListFunctions() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.functions))
	for _, info := range r.functions {
		names = append(names, info.Name)
	}
	return names
}

// ListHandlers returns all registered handler names.
func (r *Registry) ListHandlers() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.handlers))
	for name := range r.handlers {
		names = append(names, name)
	}
	return names
}

// Clear removes all registered functions (but not handlers).
// This is useful for testing or when reloading functions.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.functions = make(map[string]*FunctionInfo)
}
