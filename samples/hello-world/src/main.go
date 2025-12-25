// Sample Azure Functions Go Worker Application
// This is a complete, deployable example using Custom Handlers.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	port := os.Getenv("FUNCTIONS_CUSTOMHANDLER_PORT")
	if port == "" {
		port = "8080"
	}

	// Register function handlers
	http.HandleFunc("/api/hello", handleHello)
	http.HandleFunc("/api/health", handleHealth)
	http.HandleFunc("/api/echo", handleEcho)

	log.Printf("Go Azure Functions worker starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// handleHello - HTTP GET/POST trigger that greets the user
func handleHello(w http.ResponseWriter, r *http.Request) {
	log.Printf("Hello function invoked: %s %s", r.Method, r.URL.String())

	// Get name from query string or request body
	name := r.URL.Query().Get("name")
	if name == "" && r.Body != nil {
		body, _ := io.ReadAll(r.Body)
		if len(body) > 0 {
			var req struct {
				Name string `json:"name"`
			}
			if json.Unmarshal(body, &req) == nil && req.Name != "" {
				name = req.Name
			} else {
				name = strings.TrimSpace(string(body))
			}
		}
	}
	if name == "" {
		name = "World"
	}

	response := map[string]interface{}{
		"message":   fmt.Sprintf("Hello, %s!", name),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"runtime":   "Go Azure Functions Worker",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleHealth - Health check endpoint
func handleHealth(w http.ResponseWriter, r *http.Request) {
	log.Printf("Health check invoked")

	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleEcho - Echoes back the request details
func handleEcho(w http.ResponseWriter, r *http.Request) {
	log.Printf("Echo function invoked: %s %s", r.Method, r.URL.String())

	body, _ := io.ReadAll(r.Body)

	// Build headers map
	headers := make(map[string]string)
	for key, values := range r.Header {
		headers[key] = strings.Join(values, ", ")
	}

	// Build query params map
	queryParams := make(map[string]string)
	for key, values := range r.URL.Query() {
		queryParams[key] = strings.Join(values, ", ")
	}

	response := map[string]interface{}{
		"method":      r.Method,
		"url":         r.URL.String(),
		"headers":     headers,
		"queryParams": queryParams,
		"body":        string(body),
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
