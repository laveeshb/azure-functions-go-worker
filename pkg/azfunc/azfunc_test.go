package azfunc

import (
	"context"
	"testing"

	"github.com/laveeshb/azure-functions-go-worker/internal/rpc/proto"
)

// TestRegisterHttpFunction tests function registration
func TestRegisterHttpFunction(t *testing.T) {
	// Reset global registry for each test
	globalRegistry = globalRegistry.Clone()

	handler := func(ctx *Context, req *HttpRequest) (*HttpResponse, error) {
		return OK("test"), nil
	}

	err := RegisterHttpFunction("TestFunc", handler)
	if err != nil {
		t.Fatalf("RegisterHttpFunction failed: %v", err)
	}
}

// TestRegisterHttpFunctionDuplicate tests duplicate registration fails
func TestRegisterHttpFunctionDuplicate(t *testing.T) {
	// Reset global registry for each test
	globalRegistry = globalRegistry.Clone()

	handler := func(ctx *Context, req *HttpRequest) (*HttpResponse, error) {
		return OK("test"), nil
	}

	err := RegisterHttpFunction("DuplicateFunc", handler)
	if err != nil {
		t.Fatalf("First registration failed: %v", err)
	}

	err = RegisterHttpFunction("DuplicateFunc", handler)
	if err == nil {
		t.Fatal("Expected error for duplicate registration, got nil")
	}
}

// TestContextLogging tests context logging methods don't panic when logger is nil
func TestContextLoggingNilLogger(t *testing.T) {
	ctx := &Context{
		Context:      context.Background(),
		InvocationID: "test-123",
		logger:       nil,
	}

	// These should not panic even with nil logger
	ctx.Log("test message")
	ctx.LogDebug("debug message")
	ctx.LogWarning("warning message")
	ctx.LogError("error message")
}

// TestContextFields tests context field initialization
func TestContextFields(t *testing.T) {
	ctx := &Context{
		Context:      context.Background(),
		InvocationID: "inv-123",
		FunctionID:   "func-456",
		FunctionName: "MyFunction",
		TraceContext: &TraceContext{
			TraceParent: "00-trace-span-01",
			TraceState:  "state",
			Attributes:  map[string]string{"key": "value"},
		},
		RetryContext: &RetryContext{
			RetryCount:    1,
			MaxRetryCount: 3,
		},
	}

	if ctx.InvocationID != "inv-123" {
		t.Errorf("Expected InvocationID 'inv-123', got '%s'", ctx.InvocationID)
	}
	if ctx.FunctionID != "func-456" {
		t.Errorf("Expected FunctionID 'func-456', got '%s'", ctx.FunctionID)
	}
	if ctx.FunctionName != "MyFunction" {
		t.Errorf("Expected FunctionName 'MyFunction', got '%s'", ctx.FunctionName)
	}
	if ctx.TraceContext.TraceParent != "00-trace-span-01" {
		t.Errorf("Expected TraceParent '00-trace-span-01', got '%s'", ctx.TraceContext.TraceParent)
	}
	if ctx.RetryContext.RetryCount != 1 {
		t.Errorf("Expected RetryCount 1, got %d", ctx.RetryContext.RetryCount)
	}
}

// TestExecuteHttpHandler tests the HTTP handler execution
func TestExecuteHttpHandler(t *testing.T) {
	handler := func(ctx *Context, req *HttpRequest) (*HttpResponse, error) {
		name := req.GetQuery("name")
		if name == "" {
			name = "World"
		}
		return OK("Hello, " + name + "!"), nil
	}

	// Create a mock invocation request
	req := &proto.InvocationRequest{
		InvocationId: "test-inv-123",
		FunctionId:   "test-func",
		InputData: []*proto.ParameterBinding{
			{
				Name: "req",
				RpcData: &proto.ParameterBinding_Data{
					Data: &proto.TypedData{
						Data: &proto.TypedData_Http{
							Http: &proto.RpcHttp{
								Method: "GET",
								Url:    "http://localhost/api/test?name=Azure",
								Query: map[string]string{
									"name": "Azure",
								},
								Headers: map[string]string{
									"Content-Type": "application/json",
								},
							},
						},
					},
				},
			},
		},
	}

	resp, err := executeHttpHandler(context.Background(), req, handler)
	if err != nil {
		t.Fatalf("executeHttpHandler failed: %v", err)
	}

	if resp.InvocationId != "test-inv-123" {
		t.Errorf("Expected InvocationId 'test-inv-123', got '%s'", resp.InvocationId)
	}

	if resp.Result.Status != proto.StatusResult_Success {
		t.Errorf("Expected Success status, got %v", resp.Result.Status)
	}

	// Verify the response body
	if resp.ReturnValue == nil {
		t.Fatal("Expected ReturnValue, got nil")
	}

	httpResp, ok := resp.ReturnValue.Data.(*proto.TypedData_Http)
	if !ok {
		t.Fatal("Expected HTTP response in ReturnValue")
	}

	if httpResp.Http.StatusCode != "200" {
		t.Errorf("Expected status code '200', got '%s'", httpResp.Http.StatusCode)
	}
}

// TestExecuteHttpHandlerError tests handler error handling
func TestExecuteHttpHandlerError(t *testing.T) {
	handler := func(ctx *Context, req *HttpRequest) (*HttpResponse, error) {
		return nil, context.DeadlineExceeded
	}

	req := &proto.InvocationRequest{
		InvocationId: "test-inv-error",
		FunctionId:   "test-func",
		InputData:    []*proto.ParameterBinding{},
	}

	resp, err := executeHttpHandler(context.Background(), req, handler)
	if err != nil {
		t.Fatalf("executeHttpHandler should not return error: %v", err)
	}

	if resp.Result.Status != proto.StatusResult_Failure {
		t.Errorf("Expected Failure status, got %v", resp.Result.Status)
	}

	if resp.Result.Exception == nil {
		t.Error("Expected exception in result")
	}
}

// TestExecuteHttpHandlerNoInput tests handling when no HTTP input is provided
func TestExecuteHttpHandlerNoInput(t *testing.T) {
	handler := func(ctx *Context, req *HttpRequest) (*HttpResponse, error) {
		// Should receive empty request
		if req == nil {
			return nil, context.Canceled
		}
		return OK("no input"), nil
	}

	req := &proto.InvocationRequest{
		InvocationId: "test-inv-no-input",
		FunctionId:   "test-func",
		InputData:    []*proto.ParameterBinding{},
	}

	resp, err := executeHttpHandler(context.Background(), req, handler)
	if err != nil {
		t.Fatalf("executeHttpHandler failed: %v", err)
	}

	if resp.Result.Status != proto.StatusResult_Success {
		t.Errorf("Expected Success status, got %v", resp.Result.Status)
	}
}

// TestExecuteHttpHandlerWithTraceContext tests trace context propagation
func TestExecuteHttpHandlerWithTraceContext(t *testing.T) {
	var capturedCtx *Context

	handler := func(ctx *Context, req *HttpRequest) (*HttpResponse, error) {
		capturedCtx = ctx
		return OK("traced"), nil
	}

	req := &proto.InvocationRequest{
		InvocationId: "test-inv-trace",
		FunctionId:   "test-func",
		TraceContext: &proto.RpcTraceContext{
			TraceParent: "00-abc123-def456-01",
			TraceState:  "azure=value",
			Attributes: map[string]string{
				"key1": "value1",
			},
		},
		InputData: []*proto.ParameterBinding{},
	}

	_, err := executeHttpHandler(context.Background(), req, handler)
	if err != nil {
		t.Fatalf("executeHttpHandler failed: %v", err)
	}

	if capturedCtx.TraceContext == nil {
		t.Fatal("Expected TraceContext, got nil")
	}

	if capturedCtx.TraceContext.TraceParent != "00-abc123-def456-01" {
		t.Errorf("Expected TraceParent '00-abc123-def456-01', got '%s'", capturedCtx.TraceContext.TraceParent)
	}

	if capturedCtx.TraceContext.Attributes["key1"] != "value1" {
		t.Errorf("Expected attribute 'key1'='value1', got '%s'", capturedCtx.TraceContext.Attributes["key1"])
	}
}

// TestExecuteHttpHandlerWithRetryContext tests retry context propagation
func TestExecuteHttpHandlerWithRetryContext(t *testing.T) {
	var capturedCtx *Context

	handler := func(ctx *Context, req *HttpRequest) (*HttpResponse, error) {
		capturedCtx = ctx
		return OK("retried"), nil
	}

	req := &proto.InvocationRequest{
		InvocationId: "test-inv-retry",
		FunctionId:   "test-func",
		RetryContext: &proto.RetryContext{
			RetryCount:    2,
			MaxRetryCount: 5,
		},
		InputData: []*proto.ParameterBinding{},
	}

	_, err := executeHttpHandler(context.Background(), req, handler)
	if err != nil {
		t.Fatalf("executeHttpHandler failed: %v", err)
	}

	if capturedCtx.RetryContext == nil {
		t.Fatal("Expected RetryContext, got nil")
	}

	if capturedCtx.RetryContext.RetryCount != 2 {
		t.Errorf("Expected RetryCount 2, got %d", capturedCtx.RetryContext.RetryCount)
	}

	if capturedCtx.RetryContext.MaxRetryCount != 5 {
		t.Errorf("Expected MaxRetryCount 5, got %d", capturedCtx.RetryContext.MaxRetryCount)
	}
}

// TestReExportedTypes tests that re-exported types work correctly
func TestReExportedTypes(t *testing.T) {
	// Test OK function
	resp := OK("test body")
	if resp.StatusCode != 200 {
		t.Errorf("Expected StatusCode 200, got %d", resp.StatusCode)
	}

	// Test Created function
	resp = Created("created body")
	if resp.StatusCode != 201 {
		t.Errorf("Expected StatusCode 201, got %d", resp.StatusCode)
	}

	// Test BadRequest function
	resp = BadRequest("bad request")
	if resp.StatusCode != 400 {
		t.Errorf("Expected StatusCode 400, got %d", resp.StatusCode)
	}

	// Test NotFound function
	resp = NotFound("not found")
	if resp.StatusCode != 404 {
		t.Errorf("Expected StatusCode 404, got %d", resp.StatusCode)
	}

	// Test InternalServerError function
	resp = InternalServerError("server error")
	if resp.StatusCode != 500 {
		t.Errorf("Expected StatusCode 500, got %d", resp.StatusCode)
	}

	// Test NewHttpResponse function
	resp = NewHttpResponse()
	if resp.StatusCode != 200 {
		t.Errorf("Expected default StatusCode 200, got %d", resp.StatusCode)
	}
}

// TestHttpRequestMethods tests HttpRequest methods
func TestHttpRequestMethods(t *testing.T) {
	req := &HttpRequest{
		Method: "POST",
		URL:    "http://localhost/api/test",
		Headers: map[string]string{
			"content-type":  "application/json",
			"authorization": "Bearer token123",
		},
		Query: map[string]string{
			"id":   "123",
			"name": "test",
		},
		Params: map[string]string{
			"userId": "456",
		},
		Body: []byte(`{"key": "value"}`),
	}

	// Test GetHeader method
	if req.GetHeader("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", req.GetHeader("Content-Type"))
	}
	if req.GetHeader("NonExistent") != "" {
		t.Error("Expected empty string for non-existent header")
	}

	// Test GetQuery method
	if req.GetQuery("id") != "123" {
		t.Errorf("Expected query param 'id'='123', got '%s'", req.GetQuery("id"))
	}
	if req.GetQuery("nonexistent") != "" {
		t.Error("Expected empty string for non-existent query param")
	}

	// Test GetParam method
	if req.GetParam("userId") != "456" {
		t.Errorf("Expected param 'userId'='456', got '%s'", req.GetParam("userId"))
	}
}

// TestHttpResponseChaining tests response builder pattern
func TestHttpResponseChaining(t *testing.T) {
	resp := NewHttpResponse()
	resp.StatusCode = 201
	resp.Body = "response body"
	resp.WithHeader("X-Custom-Header", "custom-value")

	if resp.StatusCode != 201 {
		t.Errorf("Expected StatusCode 201, got %d", resp.StatusCode)
	}
	if resp.Headers["X-Custom-Header"] != "custom-value" {
		t.Errorf("Expected header 'X-Custom-Header'='custom-value', got '%s'", resp.Headers["X-Custom-Header"])
	}
	if resp.Body != "response body" {
		t.Errorf("Expected Body 'response body', got '%v'", resp.Body)
	}
}
