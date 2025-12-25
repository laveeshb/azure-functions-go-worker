// Package main demonstrates a QR Code Generator Azure Function written in Go.
//
// This sample shows how to:
// - Handle HTTP POST requests with JSON payloads
// - Generate QR codes using a pure Go library
// - Return binary (PNG) responses
// - Serve an interactive HTML landing page
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
// For GET requests, it serves an interactive HTML page.
// For POST requests, it generates a QR code and returns JSON.
//
// PRIVACY: No user data is stored or logged. All processing is done in-memory.
func handleGenerate(ctx *azfunc.Context, req *azfunc.HttpRequest) (*azfunc.HttpResponse, error) {
	ctx.Log("Processing QR code generation request")

	// GET request: serve the interactive landing page
	if req.Method == "GET" {
		return &azfunc.HttpResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type": "text/html; charset=utf-8",
			},
			Body: landingPageHTML,
		}, nil
	}

	// Only accept POST requests for API
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
