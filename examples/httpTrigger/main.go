// Package main demonstrates a simple Azure Functions app with an HTTP trigger.
//
// To use this example:
// 1. Build: go build -o worker.exe ./examples/httpTrigger
// 2. Create a function app with the appropriate host.json and function.json files
// 3. Run with Azure Functions Core Tools: func start
package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/laveeshb/azure-functions-go-worker/pkg/azfunc"
)

func init() {
	// Register HTTP function handlers
	if err := azfunc.RegisterHttpFunction("HttpTrigger", handleHttpTrigger); err != nil {
		log.Fatalf("Failed to register HttpTrigger: %v", err)
	}

	if err := azfunc.RegisterHttpFunction("HelloWorld", handleHelloWorld); err != nil {
		log.Fatalf("Failed to register HelloWorld: %v", err)
	}
}

func main() {
	log.Println("Starting example Azure Functions Go app...")

	if err := azfunc.Start(); err != nil {
		log.Fatalf("Worker failed: %v", err)
	}
}

// handleHttpTrigger handles HTTP requests with name parameter.
func handleHttpTrigger(ctx *azfunc.Context, req *azfunc.HttpRequest) (*azfunc.HttpResponse, error) {
	ctx.Log(fmt.Sprintf("Processing request: %s %s", req.Method, req.URL))

	// Get name from query string or body
	name := req.GetQuery("name")
	if name == "" {
		name = strings.TrimSpace(req.BodyAsString())
	}
	if name == "" {
		name = "World"
	}

	message := fmt.Sprintf("Hello, %s! This is an Azure Function running in Go.", name)

	return azfunc.OK(message), nil
}

// handleHelloWorld is a simple hello world function.
func handleHelloWorld(ctx *azfunc.Context, req *azfunc.HttpRequest) (*azfunc.HttpResponse, error) {
	response := map[string]interface{}{
		"message":      "Hello from Azure Functions Go Worker!",
		"method":       req.Method,
		"url":          req.URL,
		"invocationId": ctx.InvocationID,
		"headers":      req.Headers,
		"query":        req.Query,
	}

	return azfunc.OK(response).WithContentType("application/json"), nil
}
