// Package main demonstrates a QR Code Generator Azure Function written in Go.
//
// This sample uses the Custom Handler pattern with standard net/http.
//
// Endpoints:
// - GET  /generate - Interactive web page for QR code generation
// - POST /generate - API endpoint to generate a QR code from text/URL
// - GET  /health   - Health check endpoint
// - GET  /         - Redirects to /generate
//
// To run locally:
// 1. Build: go build -o handler.exe . (Windows) or go build -o handler . (Linux)
// 2. Run: func start
package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
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

	http.HandleFunc("/generate", handleGenerate)
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/", handleRoot)

	log.Printf("QR Code Generator starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// GenerateRequest represents the input for QR code generation.
type GenerateRequest struct {
	Content string `json:"content"`
	Size    int    `json:"size,omitempty"`
}

// GenerateResponse represents the output with the generated QR code.
type GenerateResponse struct {
	Image   string `json:"image"`
	Content string `json:"content"`
	Size    int    `json:"size"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error string `json:"error"`
}

// handleGenerate creates a QR code from the provided content.
func handleGenerate(w http.ResponseWriter, r *http.Request) {
	// GET request: serve the interactive landing page
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, landingPageHTML)
		return
	}

	// Only accept POST for API
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "Method not allowed. Use GET or POST."})
		return
	}

	// Parse the request body
	var genReq GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&genReq); err != nil {
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
	jsonResponse(w, http.StatusOK, map[string]string{
		"status":    "healthy",
		"service":   "qr-generator",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleRoot serves a redirect or info for the root path
func handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/generate", http.StatusFound)
}

// jsonResponse writes a JSON response with the given status code.
func jsonResponse(w http.ResponseWriter, statusCode int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(body)
}

// landingPageHTML is the interactive web page for QR code generation.
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
                const response = await fetch('/generate', {
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
