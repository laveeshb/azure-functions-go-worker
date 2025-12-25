// This example uses Custom Handler HTTP mode for local testing with func.exe.
// The worker runs an HTTP server that receives forwarded requests from the host.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	port := os.Getenv("FUNCTIONS_CUSTOMHANDLER_PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/api/HttpTrigger", handleHttpTrigger)
	http.HandleFunc("/api/HelloWorld", handleHelloWorld)

	log.Printf("Go Custom Handler listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func handleHttpTrigger(w http.ResponseWriter, r *http.Request) {
	log.Printf("HttpTrigger: %s %s", r.Method, r.URL.String())

	// Get name from query string or body
	name := r.URL.Query().Get("name")
	if name == "" {
		body, _ := io.ReadAll(r.Body)
		name = strings.TrimSpace(string(body))
	}
	if name == "" {
		name = "World"
	}

	message := fmt.Sprintf("Hello, %s! This is an Azure Function running in Go.", name)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(message))
}

func handleHelloWorld(w http.ResponseWriter, r *http.Request) {
	log.Printf("HelloWorld: %s %s", r.Method, r.URL.String())

	response := map[string]interface{}{
		"message":   "Hello from Azure Functions Go Worker!",
		"method":    r.Method,
		"timestamp": "2025-12-24T17:00:00Z",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
