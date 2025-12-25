package bindings

import (
	"testing"

	pb "github.com/Azure/azure-functions-go-worker/internal/rpc/proto"
)

func TestNewHttpRequest(t *testing.T) {
	rpcHttp := &pb.RpcHttp{
		Method: "POST",
		Url:    "https://example.com/api/test?name=gopher",
		Headers: map[string]string{
			"Content-Type": "application/json",
			"X-Custom":     "custom-value",
		},
		Query: map[string]string{
			"name": "gopher",
		},
		Params: map[string]string{
			"id": "123",
		},
		Body: &pb.TypedData{
			Data: &pb.TypedData_String_{String_: `{"message":"hello"}`},
		},
	}

	req := NewHttpRequest(rpcHttp)

	if req.Method != "POST" {
		t.Errorf("Expected method 'POST', got '%s'", req.Method)
	}
	if req.URL != "https://example.com/api/test?name=gopher" {
		t.Errorf("Expected URL 'https://example.com/api/test?name=gopher', got '%s'", req.URL)
	}
	if req.GetHeader("content-type") != "application/json" {
		t.Errorf("Expected header 'application/json', got '%s'", req.GetHeader("content-type"))
	}
	if req.GetQuery("name") != "gopher" {
		t.Errorf("Expected query 'gopher', got '%s'", req.GetQuery("name"))
	}
	if req.GetParam("id") != "123" {
		t.Errorf("Expected param '123', got '%s'", req.GetParam("id"))
	}
	if req.BodyAsString() != `{"message":"hello"}` {
		t.Errorf("Expected body '{\"message\":\"hello\"}', got '%s'", req.BodyAsString())
	}
}

func TestNewHttpRequestNil(t *testing.T) {
	req := NewHttpRequest(nil)
	if req == nil {
		t.Fatal("Expected non-nil request for nil input")
	}
	if req.Headers == nil || req.Query == nil || req.Params == nil {
		t.Error("Expected initialized maps")
	}
}

func TestHttpRequestGetters(t *testing.T) {
	req := &HttpRequest{
		Headers: map[string]string{"content-type": "text/plain"},
		Query:   map[string]string{"page": "1"},
		Params:  map[string]string{"id": "abc"},
	}

	// Test existing keys
	if req.GetHeader("Content-Type") != "text/plain" {
		t.Error("GetHeader should be case-insensitive")
	}
	if req.GetQuery("page") != "1" {
		t.Error("GetQuery failed")
	}
	if req.GetParam("id") != "abc" {
		t.Error("GetParam failed")
	}

	// Test missing keys
	if req.GetHeader("x-missing") != "" {
		t.Error("GetHeader should return empty for missing")
	}
	if req.GetQuery("missing") != "" {
		t.Error("GetQuery should return empty for missing")
	}
	if req.GetParam("missing") != "" {
		t.Error("GetParam should return empty for missing")
	}
}

func TestHttpResponseToRpcHttp(t *testing.T) {
	tests := []struct {
		name     string
		response *HttpResponse
		checkFn  func(*testing.T, *pb.RpcHttp)
	}{
		{
			name: "string body",
			response: &HttpResponse{
				StatusCode: 200,
				Body:       "Hello, World!",
			},
			checkFn: func(t *testing.T, rpc *pb.RpcHttp) {
				if rpc.StatusCode != "200" {
					t.Errorf("Expected status '200', got '%s'", rpc.StatusCode)
				}
				if TypedDataToString(rpc.Body) != "Hello, World!" {
					t.Error("Body mismatch")
				}
			},
		},
		{
			name: "byte body",
			response: &HttpResponse{
				StatusCode: 201,
				Body:       []byte("binary data"),
			},
			checkFn: func(t *testing.T, rpc *pb.RpcHttp) {
				if rpc.StatusCode != "201" {
					t.Errorf("Expected status '201', got '%s'", rpc.StatusCode)
				}
			},
		},
		{
			name: "json body",
			response: &HttpResponse{
				StatusCode: 200,
				Body:       map[string]string{"key": "value"},
			},
			checkFn: func(t *testing.T, rpc *pb.RpcHttp) {
				if rpc.Headers["Content-Type"] != "application/json" {
					t.Error("Expected Content-Type to be application/json for map body")
				}
			},
		},
		{
			name: "custom headers",
			response: &HttpResponse{
				StatusCode: 200,
				Headers:    map[string]string{"X-Custom": "value"},
				Body:       "test",
			},
			checkFn: func(t *testing.T, rpc *pb.RpcHttp) {
				if rpc.Headers["X-Custom"] != "value" {
					t.Error("Custom header not set")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rpc, err := tt.response.ToRpcHttp()
			if err != nil {
				t.Fatalf("ToRpcHttp failed: %v", err)
			}
			tt.checkFn(t, rpc)
		})
	}
}

func TestConvenienceFunctions(t *testing.T) {
	tests := []struct {
		name           string
		response       *HttpResponse
		expectedStatus int
	}{
		{"OK", OK("success"), 200},
		{"Created", Created("created"), 201},
		{"BadRequest", BadRequest("bad"), 400},
		{"NotFound", NotFound("not found"), 404},
		{"InternalServerError", InternalServerError("error"), 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.response.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, tt.response.StatusCode)
			}
		})
	}
}

func TestFluentAPI(t *testing.T) {
	resp := OK("test").
		WithHeader("X-Custom", "value").
		WithContentType("text/html")

	if resp.Headers["X-Custom"] != "value" {
		t.Error("WithHeader failed")
	}
	if resp.Headers["Content-Type"] != "text/html" {
		t.Error("WithContentType failed")
	}
}

func TestNewHttpResponse(t *testing.T) {
	resp := NewHttpResponse()
	if resp.StatusCode != 200 {
		t.Errorf("Expected default status 200, got %d", resp.StatusCode)
	}
	if resp.Headers == nil {
		t.Error("Headers should be initialized")
	}
}
