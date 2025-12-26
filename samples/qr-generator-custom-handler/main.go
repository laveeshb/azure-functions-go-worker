// Package main demonstrates a QR Code Generator Azure Function using Custom Handler.
//
// This sample uses the Custom Handler approach (HTTP-based) which deploys directly
// to Azure Functions without requiring container infrastructure.
//
// For the gRPC version with full binding support, see ../qr-generator-grpc/
//
// PRIVACY: This application does not store, log, or transmit any user data.
// All QR code generation happens in-memory and data is immediately discarded
// after the response is sent. No cookies, no tracking, no data retention.
//
// Endpoints:
// - GET  /api/generate - Interactive web page for QR code generation
// - POST /api/generate - API endpoint to generate a QR code from text/URL
// - GET  /api/health   - Health check endpoint
//
// To run locally:
// 1. Build: go build -o handler.exe .
// 2. Run: func start
//
// To deploy to Azure:
// 1. Run: .\deploy.ps1 -ResourceGroupName "my-rg" -Location "eastus"
package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/skip2/go-qrcode"
)

func main() {
	port := os.Getenv("FUNCTIONS_CUSTOMHANDLER_PORT")
	if port == "" {
		port = "8080"
	}

	// Register function handlers
	http.HandleFunc("/api/generate", handleGenerate)
	http.HandleFunc("/api/health", handleHealth)

	log.Printf("QR Code Generator starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
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
// For GET requests, it serves an interactive HTML page.
// For POST requests, it generates a QR code and returns JSON.
//
// PRIVACY: No user data is stored or logged. All processing is done in-memory.
func handleGenerate(w http.ResponseWriter, r *http.Request) {
	log.Printf("Generate function invoked: %s %s", r.Method, r.URL.String())

	// GET request: serve the interactive landing page
	if r.Method == "GET" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(landingPageHTML))
		return
	}

	// Only accept POST requests for API
	if r.Method != "POST" {
		jsonResponse(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "Method not allowed. Use POST."})
		return
	}

	// Parse the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, ErrorResponse{Error: "Failed to read request body"})
		return
	}

	var genReq GenerateRequest
	if err := json.Unmarshal(body, &genReq); err != nil {
		log.Printf("Failed to parse request body: %v", err)
		jsonResponse(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid JSON payload"})
		return
	}

	// Validate content
	if genReq.Content == "" {
		jsonResponse(w, http.StatusBadRequest, ErrorResponse{Error: "Content is required"})
		return
	}

	// Set default size if not provided
	size := genReq.Size
	if size <= 0 {
		size = 256
	}
	if size > 1024 {
		jsonResponse(w, http.StatusBadRequest, ErrorResponse{Error: "Size cannot exceed 1024 pixels"})
		return
	}

	// Generate the QR code
	png, err := qrcode.Encode(genReq.Content, qrcode.Medium, size)
	if err != nil {
		log.Printf("Failed to generate QR code: %v", err)
		jsonResponse(w, http.StatusInternalServerError, ErrorResponse{Error: "Failed to generate QR code"})
		return
	}

	log.Printf("Generated QR code for content: %s (size: %d)", genReq.Content, size)

	// Return the response with base64-encoded image
	response := GenerateResponse{
		Image:   base64.StdEncoding.EncodeToString(png),
		Content: genReq.Content,
		Size:    size,
	}

	jsonResponse(w, http.StatusOK, response)
}

// handleHealth returns a simple health check response.
func handleHealth(w http.ResponseWriter, r *http.Request) {
	log.Printf("Health check request")

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"service":   "qr-generator",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"runtime":   "Custom Handler (Go)",
	})
}

// jsonResponse writes a JSON response with the given status code and body.
func jsonResponse(w http.ResponseWriter, statusCode int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(body)
}

// landingPageHTML is the interactive web page for QR code generation.
// PRIVACY: This page runs entirely in the browser. No data is stored or tracked.
const landingPageHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>QR Code Generator</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            justify-content: center;
            align-items: center;
            padding: 20px;
        }
        .container {
            background: white;
            border-radius: 16px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            padding: 40px;
            max-width: 500px;
            width: 100%;
        }
        h1 {
            color: #333;
            margin-bottom: 8px;
            font-size: 28px;
        }
        .subtitle {
            color: #666;
            margin-bottom: 24px;
            font-size: 14px;
        }
        .badge {
            display: inline-block;
            background: #e8f5e9;
            color: #2e7d32;
            padding: 4px 8px;
            border-radius: 4px;
            font-size: 11px;
            margin-bottom: 16px;
        }
        .form-group {
            margin-bottom: 20px;
        }
        label {
            display: block;
            margin-bottom: 8px;
            color: #444;
            font-weight: 500;
        }
        input[type="text"], select {
            width: 100%;
            padding: 12px 16px;
            border: 2px solid #e0e0e0;
            border-radius: 8px;
            font-size: 16px;
            transition: border-color 0.2s;
        }
        input[type="text"]:focus, select:focus {
            outline: none;
            border-color: #667eea;
        }
        button {
            width: 100%;
            padding: 14px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            border-radius: 8px;
            font-size: 16px;
            font-weight: 600;
            cursor: pointer;
            transition: transform 0.2s, box-shadow 0.2s;
        }
        button:hover {
            transform: translateY(-2px);
            box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
        }
        button:disabled {
            opacity: 0.6;
            cursor: not-allowed;
            transform: none;
        }
        .result {
            margin-top: 24px;
            text-align: center;
            display: none;
        }
        .result.show { display: block; }
        .qr-image {
            max-width: 100%;
            border-radius: 8px;
            box-shadow: 0 4px 12px rgba(0,0,0,0.1);
        }
        .download-btn {
            margin-top: 16px;
            background: #28a745;
        }
        .download-btn:hover {
            box-shadow: 0 4px 12px rgba(40, 167, 69, 0.4);
        }
        .error {
            background: #fee;
            color: #c00;
            padding: 12px;
            border-radius: 8px;
            margin-top: 16px;
            display: none;
        }
        .error.show { display: block; }
        .privacy-notice {
            margin-top: 24px;
            padding: 16px;
            background: #f0f7ff;
            border-radius: 8px;
            border-left: 4px solid #667eea;
        }
        .privacy-notice h3 {
            color: #667eea;
            font-size: 14px;
            margin-bottom: 8px;
        }
        .privacy-notice p {
            color: #555;
            font-size: 13px;
            line-height: 1.5;
        }
        .powered-by {
            margin-top: 20px;
            text-align: center;
            color: #999;
            font-size: 12px;
        }
        .powered-by a {
            color: #667eea;
            text-decoration: none;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🔳 QR Code Generator</h1>
        <span class="badge">☁️ Azure Functions + Custom Handler</span>
        <p class="subtitle">Generate QR codes instantly from any text or URL</p>

        <div class="form-group">
            <label for="content">Text or URL</label>
            <input type="text" id="content" placeholder="Enter text or URL to encode..." autofocus>
        </div>

        <div class="form-group">
            <label for="size">Size</label>
            <select id="size">
                <option value="128">Small (128px)</option>
                <option value="256" selected>Medium (256px)</option>
                <option value="512">Large (512px)</option>
                <option value="1024">Extra Large (1024px)</option>
            </select>
        </div>

        <button id="generate" onclick="generateQR()">Generate QR Code</button>

        <div class="error" id="error"></div>

        <div class="result" id="result">
            <img class="qr-image" id="qrImage" alt="Generated QR Code">
            <button class="download-btn" onclick="downloadQR()">⬇️ Download QR Code</button>
        </div>

        <div class="privacy-notice">
            <h3>🔒 Privacy First</h3>
            <p>
                <strong>Your data is never stored.</strong> This service does not log, save, or transmit 
                your input to any third party. All QR code generation happens in-memory on the server 
                and your data is immediately discarded after the response is sent. No cookies, no tracking, 
                no data retention.
            </p>
        </div>

        <p class="powered-by">
            Powered by <a href="https://github.com/laveeshb/azure-functions-go-worker" target="_blank">Azure Functions Go Worker</a>
        </p>
    </div>

    <script>
        let currentImageData = null;

        async function generateQR() {
            const content = document.getElementById('content').value.trim();
            const size = parseInt(document.getElementById('size').value);
            const btn = document.getElementById('generate');
            const error = document.getElementById('error');
            const result = document.getElementById('result');

            if (!content) {
                showError('Please enter some text or a URL');
                return;
            }

            btn.disabled = true;
            btn.textContent = 'Generating...';
            error.classList.remove('show');
            result.classList.remove('show');

            try {
                const response = await fetch('/api/generate', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ content, size })
                });

                const data = await response.json();

                if (!response.ok) {
                    throw new Error(data.error || 'Failed to generate QR code');
                }

                currentImageData = data.image;
                document.getElementById('qrImage').src = 'data:image/png;base64,' + data.image;
                result.classList.add('show');
            } catch (err) {
                showError(err.message);
            } finally {
                btn.disabled = false;
                btn.textContent = 'Generate QR Code';
            }
        }

        function showError(message) {
            const error = document.getElementById('error');
            error.textContent = message;
            error.classList.add('show');
        }

        function downloadQR() {
            if (!currentImageData) return;
            
            const link = document.createElement('a');
            link.href = 'data:image/png;base64,' + currentImageData;
            link.download = 'qrcode.png';
            document.body.appendChild(link);
            link.click();
            document.body.removeChild(link);
        }

        // Generate on Enter key
        document.getElementById('content').addEventListener('keypress', (e) => {
            if (e.key === 'Enter') generateQR();
        });
    </script>
</body>
</html>`
