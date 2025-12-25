// Sample Azure Functions Go Worker Application using gRPC
//
// This example demonstrates the native Go worker that communicates
// with the Azure Functions host via gRPC - no Custom Handler needed.
//
// Endpoints:
// - GET/POST /api/hello  - Greet the user
// - GET      /api/health - Health check endpoint
// - POST     /api/echo   - Echo back request details
//
// To run locally:
// 1. Build: go build -o worker.exe .
// 2. Run: func start
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/laveeshb/azure-functions-go-worker/pkg/azfunc"
)

func init() {
	// Register function handlers using the gRPC worker
	if err := azfunc.RegisterHttpFunction("Hello", handleHello); err != nil {
		log.Fatalf("Failed to register Hello: %v", err)
	}

	if err := azfunc.RegisterHttpFunction("Health", handleHealth); err != nil {
		log.Fatalf("Failed to register Health: %v", err)
	}

	if err := azfunc.RegisterHttpFunction("Echo", handleEcho); err != nil {
		log.Fatalf("Failed to register Echo: %v", err)
	}
}

func main() {
	log.Println("Starting Hello World Azure Functions app (gRPC worker)...")

	if err := azfunc.Start(); err != nil {
		log.Fatalf("Worker failed: %v", err)
	}
}

// handleHello greets the user by name.
func handleHello(ctx *azfunc.Context, req *azfunc.HttpRequest) (*azfunc.HttpResponse, error) {
	ctx.Log(fmt.Sprintf("Hello function invoked: %s %s", req.Method, req.Url))

	// Get name from query string or request body
	name := req.Query["name"]
	if name == "" && len(req.Body) > 0 {
		var bodyReq struct {
			Name string `json:"name"`
		}
		if json.Unmarshal(req.Body, &bodyReq) == nil && bodyReq.Name != "" {
			name = bodyReq.Name
		} else {
			name = strings.TrimSpace(string(req.Body))
		}
	}
	if name == "" {
		name = "World"
	}

	response := map[string]interface{}{
		"message":   fmt.Sprintf("Hello, %s!", name),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"runtime":   "Go Azure Functions Worker (gRPC)",
	}

	return jsonResponse(200, response), nil
}

// handleHealth returns the health status of the function app.
func handleHealth(ctx *azfunc.Context, req *azfunc.HttpRequest) (*azfunc.HttpResponse, error) {
	ctx.Log("Health check invoked")

	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   "1.0.0",
		"worker":    "gRPC",
	}

	return jsonResponse(200, response), nil
}

// handleEcho echoes back the request details.
func handleEcho(ctx *azfunc.Context, req *azfunc.HttpRequest) (*azfunc.HttpResponse, error) {
	ctx.Log(fmt.Sprintf("Echo function invoked: %s %s", req.Method, req.Url))

	if req.Method != "POST" {
		return jsonResponse(405, map[string]string{"error": "Method not allowed. Use POST."}), nil
	}

	response := map[string]interface{}{
		"method":      req.Method,
		"url":         req.Url,
		"headers":     req.Headers,
		"query":       req.Query,
		"body":        string(req.Body),
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	}

	return jsonResponse(200, response), nil
}

// jsonResponse creates an HTTP response with JSON content.
func jsonResponse(statusCode int, data interface{}) *azfunc.HttpResponse {
	body, _ := json.Marshal(data)
	return &azfunc.HttpResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: body,
	}
}
