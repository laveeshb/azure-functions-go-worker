// Package main demonstrates a QR Code Generator Azure Function written in Go.
//
// This sample shows how to:
// - Handle HTTP POST requests with JSON payloads
// - Generate QR codes using a pure Go library
// - Return binary (PNG) responses
//
// Endpoints:
// - POST /api/generate - Generate a QR code from text/URL
// - GET  /api/health   - Health check endpoint
//
// To run locally:
// 1. Build: go build -o worker.exe ./samples/qr-generator
// 2. Run: func start
package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"

	"github.com/laveeshb/azure-functions-go-worker/pkg/azfunc"
	"github.com/skip2/go-qrcode"
)

func init() {
	// Register the QR code generator endpoint
	if err := azfunc.RegisterHttpFunction("Generate", handleGenerate); err != nil {
		log.Fatalf("Failed to register Generate: %v", err)
	}

	// Register a simple health check endpoint
	if err := azfunc.RegisterHttpFunction("Health", handleHealth); err != nil {
		log.Fatalf("Failed to register Health: %v", err)
	}
}

func main() {
	log.Println("Starting QR Code Generator Azure Functions app...")

	if err := azfunc.Start(); err != nil {
		log.Fatalf("Worker failed: %v", err)
	}
}

// GenerateRequest represents the input for QR code generation.
type GenerateRequest struct {
	// Content is the text or URL to encode in the QR code
	Content string `json:"content"`
	// Size is the image size in pixels (default: 256)
	Size int `json:"size,omitempty"`
}

// GenerateResponse represents the output with the generated QR code.
type GenerateResponse struct {
	// Image is the base64-encoded PNG image
	Image string `json:"image"`
	// Content is the original content that was encoded
	Content string `json:"content"`
	// Size is the size of the generated image
	Size int `json:"size"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error string `json:"error"`
}

// handleGenerate creates a QR code from the provided content.
func handleGenerate(ctx *azfunc.Context, req *azfunc.HttpRequest) (*azfunc.HttpResponse, error) {
	ctx.Log("Processing QR code generation request")

	// Only accept POST requests
	if req.Method != "POST" {
		return jsonResponse(405, ErrorResponse{Error: "Method not allowed. Use POST."}), nil
	}

	// Parse the request body
	var genReq GenerateRequest
	if err := json.Unmarshal(req.Body, &genReq); err != nil {
		ctx.Log(fmt.Sprintf("Failed to parse request body: %v", err))
		return jsonResponse(400, ErrorResponse{Error: "Invalid JSON payload"}), nil
	}

	// Validate content
	if genReq.Content == "" {
		return jsonResponse(400, ErrorResponse{Error: "Content is required"}), nil
	}

	// Set default size if not provided
	size := genReq.Size
	if size <= 0 {
		size = 256
	}
	if size > 1024 {
		return jsonResponse(400, ErrorResponse{Error: "Size cannot exceed 1024 pixels"}), nil
	}

	// Generate the QR code
	png, err := qrcode.Encode(genReq.Content, qrcode.Medium, size)
	if err != nil {
		ctx.Log(fmt.Sprintf("Failed to generate QR code: %v", err))
		return jsonResponse(500, ErrorResponse{Error: "Failed to generate QR code"}), nil
	}

	ctx.Log(fmt.Sprintf("Generated QR code for content: %s (size: %d)", genReq.Content, size))

	// Return the response with base64-encoded image
	response := GenerateResponse{
		Image:   base64.StdEncoding.EncodeToString(png),
		Content: genReq.Content,
		Size:    size,
	}

	return jsonResponse(200, response), nil
}

// handleHealth returns a simple health check response.
func handleHealth(ctx *azfunc.Context, req *azfunc.HttpRequest) (*azfunc.HttpResponse, error) {
	ctx.Log("Health check request")

	return jsonResponse(200, map[string]string{
		"status":  "healthy",
		"service": "qr-generator",
	}), nil
}

// jsonResponse creates an HTTP response with JSON content.
func jsonResponse(statusCode int, body interface{}) *azfunc.HttpResponse {
	jsonBody, _ := json.Marshal(body)
	return &azfunc.HttpResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(jsonBody),
	}
}
