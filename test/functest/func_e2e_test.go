// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

//go:build integration
// +build integration

// Package functest contains integration tests that use Azure Functions Core Tools (func.exe).
// Run with: go test -tags=integration ./test/functest/...
package functest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	funcPort    = "7071"
	funcBaseURL = "http://localhost:7071"
)

// TestFuncExeHttpTrigger tests the HTTP trigger function using func.exe
func TestFuncExeHttpTrigger(t *testing.T) {
	// Find project root
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}

	exampleDir := filepath.Join(projectRoot, "examples", "httpHandler")
	handlerExe := filepath.Join(exampleDir, "handler.exe")

	// Build the handler
	t.Log("Building handler...")
	buildCmd := exec.Command("go", "build", "-o", handlerExe, ".")
	buildCmd.Dir = exampleDir
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build handler: %v", err)
	}
	defer os.Remove(handlerExe)

	// Start func.exe
	t.Log("Starting Azure Functions Host...")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	funcCmd := exec.CommandContext(ctx, "func", "start", "--port", funcPort)
	funcCmd.Dir = exampleDir
	funcCmd.Stdout = os.Stdout
	funcCmd.Stderr = os.Stderr

	if err := funcCmd.Start(); err != nil {
		t.Fatalf("Failed to start func: %v", err)
	}
	defer func() {
		funcCmd.Process.Kill()
		funcCmd.Wait()
	}()

	// Wait for func to be ready
	if err := waitForFunc(ctx, funcBaseURL); err != nil {
		t.Fatalf("Func did not become ready: %v", err)
	}
	t.Log("Azure Functions Host is ready")

	// Test HttpTrigger with name parameter
	t.Run("HttpTrigger with name", func(t *testing.T) {
		resp, err := http.Get(funcBaseURL + "/api/HttpTrigger?name=IntegrationTest")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		expected := "Hello, IntegrationTest! This is an Azure Function running in Go."
		if string(body) != expected {
			t.Errorf("Expected '%s', got '%s'", expected, string(body))
		}
		t.Logf("Response: %s", string(body))
	})

	// Test HttpTrigger without name (default)
	t.Run("HttpTrigger default name", func(t *testing.T) {
		resp, err := http.Get(funcBaseURL + "/api/HttpTrigger")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		expected := "Hello, World! This is an Azure Function running in Go."
		if string(body) != expected {
			t.Errorf("Expected '%s', got '%s'", expected, string(body))
		}
		t.Logf("Response: %s", string(body))
	})

	// Test HelloWorld endpoint
	t.Run("HelloWorld JSON response", func(t *testing.T) {
		resp, err := http.Get(funcBaseURL + "/api/HelloWorld")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			t.Errorf("Expected JSON content type, got '%s'", contentType)
		}

		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "Hello from Azure Functions Go Worker") {
			t.Errorf("Response doesn't contain expected message: %s", string(body))
		}
		t.Logf("Response: %s", string(body))
	})

	// Test POST with body
	t.Run("HttpTrigger POST with body", func(t *testing.T) {
		resp, err := http.Post(funcBaseURL+"/api/HttpTrigger", "text/plain", strings.NewReader("PostTest"))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		expected := "Hello, PostTest! This is an Azure Function running in Go."
		if string(body) != expected {
			t.Errorf("Expected '%s', got '%s'", expected, string(body))
		}
		t.Logf("Response: %s", string(body))
	})
}

// waitForFunc waits for the Functions host to be ready
func waitForFunc(ctx context.Context, baseURL string) error {
	client := &http.Client{Timeout: 2 * time.Second}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			resp, err := client.Get(baseURL + "/api/HttpTrigger")
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return nil
				}
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}

// findProjectRoot finds the project root by looking for go.mod
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}
